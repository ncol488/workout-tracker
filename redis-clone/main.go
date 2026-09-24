package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	listener, err := net.Listen("tcp", ":6380")
	if err != nil {
		fmt.Println("Failed to bind to port 6380:", err)
		os.Exit(1)
	}
	defer listener.Close()

	fmt.Println("Listening on :6380")

	conn, err := listener.Accept()
	if err != nil {
		fmt.Println("Failed to accept connection:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	// Read lines from the client and write them straight back.
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			// Client disconnected, or something went wrong reading.
			fmt.Println("Connection closed:", err)
			return
		}

		fmt.Print("Received: ", line)

		_, err = conn.Write([]byte(line))
		if err != nil {
			fmt.Println("Failed to write back to client:", err)
			return
		}
	}
}