// Package rpc is a request/response RPC framework over TCP.
//
// A Server registers named handlers and serves RPC invocations. A Client
// dials a server and invokes registered methods by name. A single Client
// may be used concurrently from multiple goroutines; replies are
// demultiplexed back to the originating caller. Framing and serialization
// are pluggable through the Codec interface; length-prefixed framing
// helpers are exposed for codec implementations that need them.
package rpc

import (
	"context"
	"net"
)

// Server accepts connections, dispatches RPC requests to registered
// handlers, and writes responses back to callers.
type Server struct {
	// unexported
}

// ServerOption configures a Server.
type ServerOption func(*Server)

// WithServerCodec configures the Server to use codec for serializing
// requests and responses. The default codec is used otherwise.
func WithServerCodec(codec Codec) ServerOption {
	panic("not implemented")
}

// NewServer returns a Server ready to register handlers and serve.
func NewServer(opts ...ServerOption) *Server {
	panic("not implemented")
}

// Register exposes the methods of handler under name. Methods that match
// the framework's handler shape become reachable as "name.Method".
//
// Register panics if name is already registered or if handler does not
// satisfy the framework's handler shape; setup-time configuration
// errors are programmer errors and not recoverable at runtime.
func (s *Server) Register(name string, handler any) {
	panic("not implemented")
}

// ListenAndServe listens on addr and serves until Shutdown is called.
// It returns ErrServerClosed when shut down cleanly.
func (s *Server) ListenAndServe(addr string) error {
	panic("not implemented")
}

// Serve serves on lis until Shutdown is called. Serve takes ownership
// of lis and closes it. It returns ErrServerClosed when shut down
// cleanly.
func (s *Server) Serve(lis net.Listener) error {
	panic("not implemented")
}

// Shutdown stops accepting new connections and waits for in-flight
// handler invocations to complete. New requests on existing connections
// are rejected. If ctx expires before in-flight invocations drain,
// Shutdown returns ctx.Err() and the server closes any remaining
// connections.
func (s *Server) Shutdown(ctx context.Context) error {
	panic("not implemented")
}
