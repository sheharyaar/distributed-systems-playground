package rpc

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClient_CallAfterClose_ReturnsErrClientClosed(t *testing.T) {
	_, addr, stop := startServer(t, func(s *Server) {
		s.Register("Echo", echoService{})
	})
	defer stop()

	client, err := Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	err = client.Call(context.Background(), "Echo.Echo", &echoArgs{}, &echoReply{})
	if !errors.Is(err, ErrClientClosed) {
		t.Errorf("Call after Close: got %v, want ErrClientClosed", err)
	}
}

func TestClient_Dial_UnreachableAddr_ReturnsError(t *testing.T) {
	// 127.0.0.1:1 (tcpmux) is reserved by IANA and almost never in use.
	if _, err := Dial("127.0.0.1:1"); err == nil {
		t.Error("expected error dialing unreachable address, got nil")
	}
}

func TestClient_Call_DeadlineExceeded(t *testing.T) {
	b := newBlockingService()
	_, addr, stop := startServer(t, func(s *Server) {
		s.Register("Block", b)
	})
	// Release the handler before the server stops so Shutdown can drain
	// quickly; defers run LIFO.
	defer stop()
	defer close(b.release)

	client, err := Dial(addr)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	err = client.Call(ctx, "Block.Wait", &echoArgs{In: "x"}, &echoReply{})
	if err == nil {
		t.Error("expected error from deadline-exceeded call, got nil")
	}
}
