package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
)

func main() {
	listener, err := net.Listen("tcp", ":6380")
	if err != nil {
		fmt.Println("Failed to bind to port 6380:", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Listening on :6380")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Failed to accept connection:", err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	fmt.Println("Client connected:", conn.RemoteAddr())

	reader := bufio.NewReader(conn)
	for {
		args, err := readCommand(reader)
		if err != nil {
			fmt.Println("Connection closed:", conn.RemoteAddr())
			return
		}

		fmt.Println("Parsed command:", args)

		if len(args) == 0 {
			continue
		}

		switch args[0] {
		case "PING":
			// RESP simple string reply: '+' + text + \r\n
			conn.Write([]byte("+PONG\r\n"))
		default:
			// We don't know this command yet — send a RESP error reply.
			conn.Write([]byte("-ERR unknown command\r\n"))
		}
	}
}

// readCommand reads one full RESP command: an array header, followed by
// that many bulk strings, and returns them as a slice of strings.
// Example wire input:  *3\r\n$3\r\nSET\r\n$3\r\nfoo\r\n$3\r\nbar\r\n
// Returns:              []string{"SET", "foo", "bar"}
func readCommand(reader *bufio.Reader) ([]string, error) {
	// --- Read the array header line, e.g. "*3" ---
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = line[:len(line)-2] // strip trailing \r\n

	if len(line) == 0 || line[0] != '*' {
		return nil, fmt.Errorf("expected array, got: %q", line)
	}

	count, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, fmt.Errorf("bad array count: %q", line)
	}

	// --- Read `count` bulk strings, one at a time ---
	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
		// Each bulk string is TWO lines: a "$N" length header, then the
		// actual N bytes of data (also followed by \r\n).

		lengthLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		lengthLine = lengthLine[:len(lengthLine)-2]

		if len(lengthLine) == 0 || lengthLine[0] != '$' {
			return nil, fmt.Errorf("expected bulk string, got: %q", lengthLine)
		}

		strLen, err := strconv.Atoi(lengthLine[1:])
		if err != nil {
			return nil, fmt.Errorf("bad bulk string length: %q", lengthLine)
		}

		dataLine, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		dataLine = dataLine[:len(dataLine)-2]

		if len(dataLine) != strLen {
			return nil, fmt.Errorf("expected %d bytes, got %d", strLen, len(dataLine))
		}

		args = append(args, dataLine)
	}

	return args, nil
}