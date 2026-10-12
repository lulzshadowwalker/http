package main

import (
	"fmt"
	"net"

	"github.com/lulzshadowwalker/http/internal/request"
)

func main() {
	listener, err := net.Listen("tcp", ":42069")
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Printf("listening on :42069\n")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("failed to accept tcp connection: %s\n", err)
			continue
		}

		fmt.Printf("accepted tcp connection\n")

		r, err := request.RequestFromReader(conn)
		if err != nil {
			panic(err)
		}

		fmt.Printf("Request line:\n")
		fmt.Printf("- Method: %s\n", r.RequestLine.Method)
		fmt.Printf("- Target: %s\n", r.RequestLine.RequestTarget)
		fmt.Printf("- Version: %s\n", r.RequestLine.HTTPVersion)
	}
}
