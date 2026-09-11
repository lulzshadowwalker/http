package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	address, err := net.ResolveUDPAddr("udp", "localhost:42069")
	if err != nil {
		panic(err)
	}

	connection, err := net.DialUDP("udp", nil, address)
	if err != nil {
		panic(err)
	}
	defer connection.Close()

	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("> ")
		str, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("failed to read string: %q", err)
			continue
		}

		if _, err := connection.Write([]byte(str)); err != nil {
			fmt.Printf("failed to write: %q", err)
			continue
		}
	}

}
