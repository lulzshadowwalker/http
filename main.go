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

	ch := lines(file)

	for line := range ch {
		fmt.Printf("read: %s\n", line)
	}
}

func lines(f io.ReadCloser) <-chan string {
	ch := make(chan string)

	go func() {
		defer f.Close()
		defer close(ch)

		buffer := make([]byte, 8)
		current := ""

		for {
			n, err := f.Read(buffer)
			// read can return both n and err at once
			if n > 0 {
				// [:n] to use the bytes that were actually read, not the full buffer
				splits := strings.Split(string(buffer[:n]), "\n")
				splits[0] = current + splits[0]
				current = ""

				for i := 0; i < len(splits)-1; i++ {
					ch <- splits[i]
				}
				current = splits[len(splits)-1]
			}

			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				panic(err)
			}
		}

		if len(current) > 0 {
			ch <- current
		}
	}()

	return ch
}
