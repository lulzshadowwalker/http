package response

import (
	"fmt"
	"io"
	"strconv"

	"github.com/lulzshadowwalker/http/internal/headers"
)

const (
	StatusOK                  = 200
	StatusBadRequest          = 400
	StatusInternalServerError = 500
)

var reasons = map[int]string{
	StatusOK:                  "OK",
	StatusBadRequest:          "Bad Request",
	StatusInternalServerError: "Internal Server Error",
}

type Writer struct {
	w io.Writer
}

func NewWriter(w io.Writer) Writer {
	return Writer{
		w: w,
	}
}

func (w *Writer) WriteStatusLine(status int) error {
	reason := reasons[status]
	line := fmt.Sprintf("HTTP/1.1 %d %s\r\n", status, reason)
	_, err := w.w.Write([]byte(line))
	return err
}

func (w *Writer) WriteHeaders(h headers.Headers) error {
	for k, v := range h {
		line := fmt.Sprintf("%s: %s\r\n", k, v)
		if _, err := w.w.Write([]byte(line)); err != nil {
			return err
		}
	}

	_, err := w.w.Write([]byte("\r\n"))
	return err
}

func (w *Writer) WriteTrailers(h headers.Headers) error {
	for k, v := range h {
		if _, err := fmt.Fprintf(w.w, "%s: %s\r\n", k, v); err != nil {
			return err
		}
	}
	_, err := w.w.Write([]byte("\r\n"))
	return err
}

func (w *Writer) WriteChunkedBodyDone() (int, error) {
	return w.w.Write([]byte("0\r\n"))
}

func (w *Writer) WriteBody(data []byte) (int, error) {
	return w.w.Write(data)
}

func (w *Writer) WriteChunkedBody(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	chunk := fmt.Appendf(nil, "%x\r\n", len(p))
	chunk = append(chunk, p...)
	chunk = append(chunk, '\r', '\n')

	if _, err := w.w.Write(chunk); err != nil {
		return 0, err
	}
	return len(p), nil
}

func GetChunkedHeaders() headers.Headers {
	h := headers.NewHeaders()
	for k, v := range map[string]string{
		"transfer-encoding": "chunked",
		"connection":        "close",
		"content-type":      "application/json",
	} {
		if err := h.Set(k, v); err != nil {
			panic(err)
		}
	}
	return h
}

func GetDefaultHeaders(contentLength int) headers.Headers {
	defaults := map[string]string{
		"content-length": strconv.Itoa(contentLength),
		"connection":     "close",
		"content-type":   "text/html",
	}

	h := headers.NewHeaders()
	for k, v := range defaults {
		if err := h.Set(k, v); err != nil {
			panic(err)
		}
	}

	return h
}
