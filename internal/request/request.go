package request

import (
	"errors"
	"io"
	"slices"
	"strings"
)

type Request struct {
	RequestLine RequestLine
}

type RequestLine struct {
	HTTPVersion   string
	RequestTarget string
	Method        string
}

func RequestFromReader(reader io.Reader) (*Request, error) {
	bytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(bytes), "\r\n")
	if len(lines) == 0 {
		return nil, errors.New("request cannot be empty")
	}

	parts := strings.Split(lines[0], " ")
	if len(parts) != 3 {
		return nil, errors.New("expected request-line to have exactly 3 parts")
	}

	method, target, version := parts[0], parts[1], parts[2]

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD"}
	if !slices.Contains(methods, method) {
		return nil, errors.New("invalid http method")
	}

	if !strings.HasPrefix(version, "HTTP/") {
		return nil, errors.New("invalid http version")
	}

	if version = strings.TrimPrefix(version, "HTTP/"); version != "1.1" {
		return nil, errors.New("unsupported http version")
	}

	request := &Request{
		RequestLine: RequestLine{
			HTTPVersion:   version,
			RequestTarget: target,
			Method:        method,
		},
	}
	return request, nil
}
