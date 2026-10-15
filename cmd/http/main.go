package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lulzshadowwalker/http/internal/http"
	"github.com/lulzshadowwalker/http/internal/request"
	"github.com/lulzshadowwalker/http/internal/response"
)

const addr = ":8000"

func main() {
	s, err := http.Listen(addr, func(w response.Writer, r *request.Request) *http.HandlerError {
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
