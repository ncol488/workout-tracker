package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strconv"
	"sync"
)

// store is our in-memory key-value data, shared across ALL client connections.
// Because multiple goroutines (one per client) can read/write it at the same
// time, we guard every access with mu.
var (
	store = make(map[string]string)
	mu    sync.Mutex
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

		if len(args) == 0 {
			continue
		}

		reply := handleCommand(args)
		conn.Write([]byte(reply))
	}
}

// handleCommand runs the given command against the store and returns
// a fully RESP-formatted reply string, ready to write straight to the client.
func handleCommand(args []string) string {
	switch args[0] {
	case "PING":
		return "+PONG\r\n"

	case "SET":
		if len(args) != 3 {
			return "-ERR wrong number of arguments for 'SET'\r\n"
		}
		mu.Lock()
		store[args[1]] = args[2]
		mu.Unlock()
		return "+OK\r\n"

	case "GET":
		if len(args) != 2 {
			return "-ERR wrong number of arguments for 'GET'\r\n"
		}
		mu.Lock()
		val, ok := store[args[1]]
		mu.Unlock()

		if !ok {
			// RESP's "nil" bulk string reply — how Redis says "no such key"
			return "$-1\r\n"
		}
		// A real bulk string reply: $<length>\r\n<data>\r\n
		return fmt.Sprintf("$%d\r\n%s\r\n", len(val), val)

	case "DEL":
		if len(args) != 2 {
			return "-ERR wrong number of arguments for 'DEL'\r\n"
		}
		mu.Lock()
		_, existed := store[args[1]]
		delete(store, args[1])
		mu.Unlock()

		if existed {
			return ":1\r\n" // RESP integer reply: 1 key deleted
		}
		return ":0\r\n" // 0 keys deleted (it didn't exist)

	default:
		return "-ERR unknown command\r\n"
	}
}

// readCommand reads one full RESP command: an array header, followed by
// that many bulk strings, and returns them as a slice of strings.
func readCommand(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = line[:len(line)-2]

	if len(line) == 0 || line[0] != '*' {
		return nil, fmt.Errorf("expected array, got: %q", line)
	}

	count, err := strconv.Atoi(line[1:])
	if err != nil {
		return nil, fmt.Errorf("bad array count: %q", line)
	}

	args := make([]string, 0, count)
	for i := 0; i < count; i++ {
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