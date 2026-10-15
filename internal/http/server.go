package http

import (
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"

	"github.com/lulzshadowwalker/http/internal/request"
	"github.com/lulzshadowwalker/http/internal/response"
)

type Server struct {
	Addr    string
	Handler Handler

	listener net.Listener
	closed   atomic.Bool
}

func Listen(addr string, handler Handler) (*Server, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to create tcp listener: %w", err)
	}

	s := &Server{
		Addr:     addr,
		listener: l,
		Handler:  handler,
	}

	go s.listen()

	return s, nil
}

func (s *Server) Close() error {
	s.closed.Store(true)
	if err := s.listener.Close(); err != nil {
		return err
	}

	return nil
}

func (s *Server) listen() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}

			slog.Error("failed to accept tcp connection", "err", err)
			continue
		}

		go s.handle(conn)
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	r, err := request.RequestFromReader(conn)
	if err != nil {
		slog.Error("failed to read http request from connection", "err", err)
		return
	}

	w := response.NewWriter(conn)
	if err := s.Handler(w, r); err != nil {
		slog.Info("incoming request", "target", r.RequestLine.RequestTarget, "status", err.Status, "message", err.Message)
		return
	}
}

type Handler func(w response.Writer, r *request.Request) *HandlerError

type HandlerError struct {
	Status  int
	Message string
}

func (e HandlerError) Error() string {
	return e.Message
}
