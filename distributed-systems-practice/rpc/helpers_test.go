package rpc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

// echoArgs and echoReply are the shared fixture types used by handler
// tests.
type echoArgs struct {
	In string
}

type echoReply struct {
	Out string
}

// echoService pins the framework's handler shape: methods on a value
// or pointer receiver of the form
//
//	func (T) Method(ctx context.Context, args *Args, reply *Reply) error
//
// The reflective discovery of such methods is part of the contract.
type echoService struct{}

// Echo copies args.In into reply.Out.
func (echoService) Echo(ctx context.Context, args *echoArgs, reply *echoReply) error {
	reply.Out = args.In
	return nil
}

// errorService returns a handler error to exercise error propagation.
type errorService struct{}

// Fail returns a non-nil error including args.In in the message.
func (errorService) Fail(ctx context.Context, args *echoArgs, reply *echoReply) error {
	return fmt.Errorf("boom: %s", args.In)
}

// blockingService blocks each call until release is closed or ctx is
// canceled. Used to put a call deliberately in flight while a test
// inspects the server.
type blockingService struct {
	started chan struct{}
	release chan struct{}
}

func newBlockingService() *blockingService {
	return &blockingService{
		started: make(chan struct{}, 64),
		release: make(chan struct{}),
	}
}

// Wait signals on started, then blocks until release is closed or ctx
// is canceled.
func (b *blockingService) Wait(ctx context.Context, args *echoArgs, reply *echoReply) error {
	select {
	case b.started <- struct{}{}:
	default:
	}
	select {
	case <-b.release:
	case <-ctx.Done():
		return ctx.Err()
	}
	reply.Out = args.In
	return nil
}

// startServer constructs a Server, runs register against it, and serves
// it on a TCP listener bound to a random local port. The address is
// captured before Serve is called so tests need not race the bind. The
// returned stop function performs a 5 s graceful Shutdown and waits for
// Serve to return; it is safe to call multiple times.
func startServer(t *testing.T, register func(*Server)) (s *Server, addr string, stop func()) {
	t.Helper()
	s = NewServer()
	register(s)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr = lis.Addr().String()

	serveErr := make(chan error, 1)
	go func() { serveErr <- s.Serve(lis) }()

	var once sync.Once
	stop = func() {
		once.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.Shutdown(ctx)
			<-serveErr
		})
	}
	return s, addr, stop
}

// crashableListener wraps a net.Listener and tracks accepted connections
// so they can be closed simultaneously with the listener via Crash. It
// simulates a hard server termination from outside the package.
type crashableListener struct {
	net.Listener

	mu     sync.Mutex
	conns  []net.Conn
	closed bool
}

// Accept returns connections from the underlying listener, recording
// each one so Crash can tear them down later.
func (c *crashableListener) Accept() (net.Conn, error) {
	conn, err := c.Listener.Accept()
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		_ = conn.Close()
		return nil, net.ErrClosed
	}
	c.conns = append(c.conns, conn)
	c.mu.Unlock()
	return conn, nil
}

// Crash closes the underlying listener and every accepted connection.
// Subsequent Accept calls return net.ErrClosed.
func (c *crashableListener) Crash() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	conns := c.conns
	c.conns = nil
	c.mu.Unlock()
	_ = c.Listener.Close()
	for _, conn := range conns {
		_ = conn.Close()
	}
}
