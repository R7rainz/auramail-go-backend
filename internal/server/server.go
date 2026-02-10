package server

import (
	"context"
	"net/http"
	"time"
)

// Config holds server configuration options
type Config struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// DefaultConfig returns sensible default server configuration
func DefaultConfig(addr string) Config {
	return Config{
		Addr:         addr,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  5 * time.Minute,
	}
}

// Server wraps http.Server with additional functionality
type Server struct {
	httpServer *http.Server
	config     Config
}

// New creates a new server with the given configuration
func New(cfg Config, handler http.Handler) *Server {
	return &Server{
		config: cfg,
		httpServer: &http.Server{
			Addr:         cfg.Addr,
			Handler:      handler,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
			IdleTimeout:  cfg.IdleTimeout,
		},
	}
}

// NewWithDefaults creates a server with default timeouts
func NewWithDefaults(addr string, handler http.Handler) *Server {
	return New(DefaultConfig(addr), handler)
}

// Start begins listening for HTTP requests (blocking)
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// Addr returns the server address
func (s *Server) Addr() string {
	return s.config.Addr
}
