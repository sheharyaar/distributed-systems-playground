package rpc

import "io"

// WriteFrame writes a length-prefixed frame containing payload to w.
// An empty payload produces a valid frame.
func WriteFrame(w io.Writer, payload []byte) error {
	panic("not implemented")
}

// ReadFrame reads one length-prefixed frame from r and returns its
// payload.
//
// A clean end-of-stream observed between frames is reported as io.EOF.
// A stream that ends inside a frame header or payload is reported as
// io.ErrUnexpectedEOF.
func ReadFrame(r io.Reader) ([]byte, error) {
	panic("not implemented")
}
