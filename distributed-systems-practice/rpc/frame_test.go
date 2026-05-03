package rpc

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestFrame_RoundTrip(t *testing.T) {
	cases := []struct {
		name    string
		payload []byte
	}{
		{"empty", []byte{}},
		{"small", []byte("hello")},
		{"binary", []byte{0x00, 0xff, 0x80, 0x01}},
		{"large_64KiB", bytes.Repeat([]byte("x"), 1<<16)},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteFrame(&buf, tc.payload); err != nil {
				t.Fatalf("write: %v", err)
			}
			got, err := ReadFrame(&buf)
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			if !bytes.Equal(got, tc.payload) {
				t.Errorf("got %x, want %x", got, tc.payload)
			}
		})
	}
}

func TestFrame_MultipleFramesInStream(t *testing.T) {
	payloads := [][]byte{
		[]byte("first"),
		{},
		[]byte("third"),
	}
	var buf bytes.Buffer
	for _, p := range payloads {
		if err := WriteFrame(&buf, p); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	for i, want := range payloads {
		got, err := ReadFrame(&buf)
		if err != nil {
			t.Fatalf("read frame %d: %v", i, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("frame %d: got %x, want %x", i, got, want)
		}
	}
}

func TestFrame_EmptyReader_ReturnsEOF(t *testing.T) {
	var buf bytes.Buffer
	_, err := ReadFrame(&buf)
	if !errors.Is(err, io.EOF) {
		t.Errorf("ReadFrame on empty reader: got %v, want io.EOF", err)
	}
}

func TestFrame_TruncatedHeader_ReturnsUnexpectedEOF(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, []byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	full := buf.Bytes()
	// Take only the first byte of the framed stream.
	r := bytes.NewReader(full[:1])
	_, err := ReadFrame(r)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("truncated header: got %v, want io.ErrUnexpectedEOF", err)
	}
}

func TestFrame_TruncatedPayload_ReturnsUnexpectedEOF(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteFrame(&buf, []byte("hello")); err != nil {
		t.Fatalf("write: %v", err)
	}
	full := buf.Bytes()
	// Drop the last byte so the payload is truncated.
	r := bytes.NewReader(full[:len(full)-1])
	_, err := ReadFrame(r)
	if !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Errorf("truncated payload: got %v, want io.ErrUnexpectedEOF", err)
	}
}
