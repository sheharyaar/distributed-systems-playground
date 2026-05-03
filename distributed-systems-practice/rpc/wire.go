package rpc

// Request is the wire format of an RPC invocation.
type Request struct {
	// ID uniquely identifies this request on the connection that sent it.
	// The server echoes ID in the matching Response.
	ID uint64

	// Method is the registered handler name in "Service.Method" form.
	Method string

	// Body is the codec-encoded arguments value.
	Body []byte
}

// Response is the wire format of an RPC reply.
type Response struct {
	// ID matches the ID of the Request being replied to.
	ID uint64

	// Body is the codec-encoded reply value. Empty when Error is non-nil.
	Body []byte

	// Error is non-nil when the handler reported an error.
	Error *Error
}

// Error is the structured error type returned over the wire and surfaced
// to callers of Client.Call when a handler reports an error.
type Error struct {
	// Code is a machine-readable error class. Zero means unspecified.
	Code int

	// Message is a human-readable description of the error.
	Message string
}

// Error implements the error interface.
func (e *Error) Error() string {
	panic("not implemented")
}
