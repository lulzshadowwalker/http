package request

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

const bufferSize = 4096

type State string

const (
	StateInit State = "initialized"
	StateDone State = "done"
)

type Request struct {
	RequestLine RequestLine
	State       State
}

func NewRequest() *Request {
	return &Request{
		State: StateInit,
	}
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	request := NewRequest()

	buffer := make([]byte, bufferSize)

	// keeps track of how much data we have read from io.Reader to the buffer but still have not parsed yet
	buffered := 0 // Exclusive [x, readToIndex)
	for request.State != StateDone {
		if buffered == len(buffer) {
			bigger := make([]byte, 2*len(buffer))
			copy(bigger, buffer)
			buffer = bigger
		}

		n, err := reader.Read(buffer[buffered:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				request.State = StateDone
				break
			}

			return nil, err
		}

		buffered += n
		n, err = request.parse(buffer[0:buffered])
		if err != nil {
			return nil, err
		}

		// shift
		if n > 0 {
			copy(buffer, buffer[n:buffered])
			buffered -= n
		}
	}

	return request, nil
}

func (r *Request) parse(data []byte) (int, error) {
	if r.State == StateDone {
		return 0, nil
	}

	switch r.State {
	case StateInit:
		n, err := r.RequestLine.Parse(data)
		if err != nil {
			return 0, err
		}

		if n == 0 {
			// i.e. need more data
			return 0, nil
		}

		r.State = StateDone

		return n, nil

	default:
		return 0, fmt.Errorf("unrecognized request state %q", r.State)
	}
}

type RequestLine struct {
	HTTPVersion   string
	RequestTarget string
	Method        string
}

func (l *RequestLine) Parse(data []byte) (int, error) {
	idx := bytes.Index(data, []byte("\r\n"))
	if idx == -1 {
		return 0, nil
	}

	line := string(data[:idx])
	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return 0, errors.New("expected request-line to have exactly 3 parts")
	}

	method, target, version := parts[0], parts[1], parts[2]

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}
	if !slices.Contains(methods, method) {
		return 0, errors.New("invalid http method")
	}

	if !strings.HasPrefix(version, "HTTP/") {
		return 0, errors.New("invalid http version")
	}

	if version = strings.TrimPrefix(version, "HTTP/"); version != "1.1" {
		return 0, errors.New("unsupported http version")
	}

	l.Method = method
	l.RequestTarget = target
	l.HTTPVersion = version

	return idx + 2, nil

}
