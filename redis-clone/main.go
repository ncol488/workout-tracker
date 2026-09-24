package main

import (
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
}