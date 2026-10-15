package main

import (
	"crypto/sha256"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	nethttp "net/http"

	"github.com/lulzshadowwalker/http/internal/headers"
	"github.com/lulzshadowwalker/http/internal/http"
	"github.com/lulzshadowwalker/http/internal/request"
	"github.com/lulzshadowwalker/http/internal/response"
)

const addr = ":8000"

func main() {
	s, err := http.Listen(addr, func(w response.Writer, r *request.Request) *http.HandlerError {
		target := r.RequestLine.RequestTarget
		if strings.HasPrefix(target, "/httpbin/") {
			path := strings.TrimPrefix(target, "/httpbin")
			res, err := nethttp.Get("https://httpbingo.org" + path)
			if err != nil {
				return &http.HandlerError{
					Status:  response.StatusInternalServerError,
					Message: "internal server error",
				}
			}
			defer res.Body.Close()

			w.WriteStatusLine(response.StatusOK)

			h := response.GetChunkedHeaders()
			h.Set("Trailer", "X-Content-SHA256, X-Content-Length")
			w.WriteHeaders(h)

			buf := make([]byte, 1024)
			var full []byte
			for {
				n, err := res.Body.Read(buf)
				if n > 0 {
					full = append(full, buf[:n]...)

					if _, err := w.WriteChunkedBody(buf[:n]); err != nil {
						return nil
					}
				}
				if err != nil {
					break // io.EOF or a read error: either way, we're done
				}
			}

			w.WriteChunkedBodyDone()

			sum := sha256.Sum256(full)
			trailers := headers.NewHeaders()
			trailers.Set("X-Content-SHA256", fmt.Sprintf("%x", sum))
			trailers.Set("X-Content-Length", strconv.Itoa(len(full)))
			w.WriteTrailers(trailers)
			return nil
		}

		switch r.RequestLine.RequestTarget {
		case "/400":
			content := `<html>
  <head>
    <title>400 Bad Request</title>
  </head>
  <body>
    <h1>Bad Request</h1>
  </body>
</html>`
			w.WriteStatusLine(400)
			w.WriteHeaders(response.GetDefaultHeaders(len(content)))
			w.WriteBody([]byte(content))
		case "/500":
			content := `<html>
  <head>
    <title>500 Internal Server Error</title>
  </head>
  <body>
    <h1>Internal Server Error</h1>
  </body>
</html>`
			w.WriteStatusLine(500)
			w.WriteHeaders(response.GetDefaultHeaders(len(content)))
			w.WriteBody([]byte(content))
		default:
			content := `<html>
  <head>
    <title>200 OK</title>
  </head>
  <body>
    <h1>Success!</h1>
  </body>
</html>
			`
			w.WriteStatusLine(200)
			w.WriteHeaders(response.GetDefaultHeaders(len(content)))
			w.WriteBody([]byte(content))
		}

		return nil
	})
	if err != nil {
		panic(err)
	}
	defer s.Close()

	slog.Info("server started", "port", addr)

	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	slog.Info("server gracefully stopped")
}
