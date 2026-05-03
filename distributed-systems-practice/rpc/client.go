package rpc

import "context"

// Client is the local representation of a connection to an RPC server.
// A single Client may be used concurrently from multiple goroutines.
type Client struct {
	// unexported
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// WithClientCodec configures the Client to use codec for serializing
// requests and responses. The default codec is used otherwise.
func WithClientCodec(codec Codec) ClientOption {
	panic("not implemented")
}

// Dial connects to the RPC server at addr and returns a ready Client.
func Dial(addr string, opts ...ClientOption) (*Client, error) {
	panic("not implemented")
}

// Call invokes the named method on the server, blocking until a
// response is received or ctx is canceled. args is encoded by the
// client's codec; reply must be a non-nil pointer and is decoded into
// on success.
//
// If the handler returns a Go error, Call returns an *Error whose
// Message carries the handler's error text. If the underlying transport
// fails, Call returns the transport error. If ctx is canceled or its
// deadline expires before a reply arrives, Call returns ctx.Err().
func (c *Client) Call(ctx context.Context, method string, args, reply any) error {
	panic("not implemented")
}

// Close closes the connection. In-flight Call invocations are unblocked
// with a non-nil error. Subsequent calls to Call return ErrClientClosed.
func (c *Client) Close() error {
	panic("not implemented")
}
