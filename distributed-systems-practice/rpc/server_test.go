package rpc

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestServer_SingleCall_RoundTrip(t *testing.T) {
	_, addr, stop := startServer(t, func(s *Server) {
		s.Register("Echo", echoService{})
	})
	defer stop()

	client, err := Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	args := &echoArgs{In: "hello"}
	reply := &echoReply{}
	if err := client.Call(ctx, "Echo.Echo", args, reply); err != nil {
		t.Fatalf("call: %v", err)
	}
	if reply.Out != "hello" {
		t.Errorf("Out = %q, want %q", reply.Out, "hello")
	}
}

func TestServer_ConcurrentCalls_PreserveArgsPerCall(t *testing.T) {
	_, addr, stop := startServer(t, func(s *Server) {
		s.Register("Echo", echoService{})
	})
	defer stop()

	client, err := Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	const N = 50
	var wg sync.WaitGroup
	var failures atomic.Int32

	for i := 0; i < N; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			in := fmt.Sprintf("msg-%d", i)
			args := &echoArgs{In: in}
			reply := &echoReply{}
			if err := client.Call(ctx, "Echo.Echo", args, reply); err != nil {
				t.Errorf("call %d: %v", i, err)
				failures.Add(1)
				return
			}
			if reply.Out != in {
				t.Errorf("call %d: Out = %q, want %q", i, reply.Out, in)
				failures.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := failures.Load(); got > 0 {
		t.Fatalf("%d/%d concurrent calls failed", got, N)
	}
}

func TestServer_HandlerError_PropagatesToClient(t *testing.T) {
	_, addr, stop := startServer(t, func(s *Server) {
		s.Register("E", errorService{})
	})
	defer stop()

	client, err := Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.Call(ctx, "E.Fail", &echoArgs{In: "x"}, &echoReply{})
	if err == nil {
		t.Fatal("expected handler error to propagate, got nil")
	}
	var rpcErr *Error
	if !errors.As(err, &rpcErr) {
		t.Errorf("expected *rpc.Error in chain, got %T: %v", err, err)
	}
}

func TestServer_UnknownMethod_ReturnsError(t *testing.T) {
	_, addr, stop := startServer(t, func(s *Server) {
		s.Register("Echo", echoService{})
	})
	defer stop()

	client, err := Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := client.Call(ctx, "Echo.Missing", &echoArgs{}, &echoReply{}); err == nil {
		t.Fatal("expected error calling unknown method, got nil")
	}
}

func TestServer_Shutdown_DrainsInFlightCalls(t *testing.T) {
	b := newBlockingService()
	s, addr, _ := startServer(t, func(s *Server) {
		s.Register("Block", b)
	})

	client, err := Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	callDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		callDone <- client.Call(ctx, "Block.Wait", &echoArgs{In: "ok"}, &echoReply{})
	}()

	select {
	case <-b.started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never started")
	}

	shutdownDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownDone <- s.Shutdown(ctx)
	}()

	select {
	case err := <-shutdownDone:
		t.Fatalf("Shutdown returned before in-flight handler completed: %v", err)
	case <-time.After(150 * time.Millisecond):
		// expected: shutdown is draining
	}

	close(b.release)

	select {
	case err := <-shutdownDone:
		if err != nil {
			t.Errorf("Shutdown after release: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown did not return after handler released")
	}

	select {
	case err := <-callDone:
		if err != nil {
			t.Errorf("Call after release: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Call did not return after handler released")
	}
}

func TestServer_Shutdown_RejectsNewConnections(t *testing.T) {
	_, addr, stop := startServer(t, func(s *Server) {
		s.Register("Echo", echoService{})
	})
	stop()

	if _, err := Dial(addr); err == nil {
		t.Error("Dial after Shutdown succeeded; expected error")
	}
}

func TestServer_Crash_InFlightCallReturnsError(t *testing.T) {
	b := newBlockingService()
	s := NewServer()
	s.Register("Block", b)

	inner, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	cl := &crashableListener{Listener: inner}
	addr := inner.Addr().String()

	serveErr := make(chan error, 1)
	go func() { serveErr <- s.Serve(cl) }()

	client, err := Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	callDone := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		callDone <- client.Call(ctx, "Block.Wait", &echoArgs{In: "x"}, &echoReply{})
	}()

	select {
	case <-b.started:
	case <-time.After(2 * time.Second):
		t.Fatal("handler never started")
	}

	cl.Crash()

	select {
	case err := <-callDone:
		if err == nil {
			t.Error("expected error from in-flight call after server crash, got nil")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("in-flight call did not return within 3s of crash")
	}

	<-serveErr
}
