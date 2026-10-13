package headers

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
)

var (
	ErrInvalidHeader = errors.New("invalid header")
)

type Headers map[Key]string

func NewHeaders() Headers {
	return make(Headers)
}

type Key string

func NewKey(s string) (Key, error) {
	regex := regexp.MustCompile("^[A-Za-z0-9!#$%&'*+\\-.^_`|~]+$")
	if !regex.MatchString(s) {
		return "", ErrInvalidHeader
	}

	return Key(strings.ToLower(s)), nil
}

// Parse parses one field-line (header) at a time
func (h Headers) Parse(data []byte) (int, bool, error) {
	crlf := bytes.Index(data, []byte("\r\n"))
	if crlf == -1 {
		return 0, false, nil
	}

	// no more headers to parse
	if crlf == 0 {
		return 2, true, nil
	}

	// field line
	line := string(data[:crlf])
	before, after, ok := strings.Cut(line, ":")
	if !ok {
		return 0, false, ErrInvalidHeader
	}

	key, err := NewKey(before)
	if err != nil {
		return 0, false, err
	}
	val := strings.TrimSpace(after)

	if existing, ok := h[key]; ok {
		h[key] = existing + ", " + val
	} else {
		h[key] = val
	}

	return crlf + 2, false, nil
}
