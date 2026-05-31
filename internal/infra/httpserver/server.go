// Package httpserver wraps chi.Router behind a minimal lifecycle. The
// orchestrator passes a single registrar that wires up REST routes and any
// static mounts.
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

type Server struct {
	host   string
	port   int
	router *chi.Mux
	srv    *http.Server
}

func New(host string, port int) *Server {
	return &Server{host: host, port: port, router: chi.NewRouter()}
}

func (s *Server) Mux() *chi.Mux { return s.router }

func (s *Server) Start(register func(r chi.Router)) error {
	register(s.router)
	s.srv = &http.Server{
		Addr:              fmt.Sprintf("%s:%d", s.host, s.port),
		Handler:           s.router,
		ReadHeaderTimeout: 10 * time.Second,
	}
	go func() {
		if err := s.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("[webtasks] http listen error:", err)
		}
	}()
	return nil
}

func (s *Server) Stop(timeout time.Duration) error {
	if s.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.srv.Shutdown(ctx)
}
