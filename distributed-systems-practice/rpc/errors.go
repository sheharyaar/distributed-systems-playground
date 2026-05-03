package rpc

import "errors"

// Sentinel errors returned by the package. Callers test for them with
// errors.Is.
var (
	// ErrServerClosed is returned by Server.Serve and Server.ListenAndServe
	// after a successful call to Server.Shutdown.
	ErrServerClosed = errors.New("rpc: server closed")

	// ErrClientClosed is returned by Client.Call after Client.Close has
	// returned.
	ErrClientClosed = errors.New("rpc: client closed")

	// ErrUnknownMethod is reported, via the wire Error type, when the
	// server has no handler registered for the requested method name.
	ErrUnknownMethod = errors.New("rpc: unknown method")

	// ErrInvalidHandler is reported when a value passed to Server.Register
	// does not satisfy the framework's handler shape.
	ErrInvalidHandler = errors.New("rpc: invalid handler")
)
