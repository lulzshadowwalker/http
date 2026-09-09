package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	file, err := os.Open("messages.txt")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	buffer := make([]byte, 8)
	current := ""
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			splits := strings.Split(string(buffer[:n]), "\n")
			for i := 0; i < len(splits)-1; i++ {
				fmt.Printf("read: %s\n", current+splits[i])
				current = ""
			}
			current += splits[len(splits)-1]
		}
		if err != nil {
			if errors.Is(io.EOF, err) {
				break
			}

			panic(err)
		}
	}
	if len(current) > 0 {
		fmt.Printf("read: %s\n", current)
	}
}
