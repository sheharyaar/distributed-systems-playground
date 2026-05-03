package rpc

// Codec serializes and deserializes values for transmission on the wire.
//
// Implementations must round-trip the package's wire types Request,
// Response, and Error, and any user-defined argument and reply values
// passed through registered handlers.
type Codec interface {
	// Encode serializes v into a byte slice.
	Encode(v any) ([]byte, error)

	// Decode populates v from data. v must be a non-nil pointer to a
	// value of a type compatible with the value originally encoded.
	Decode(data []byte, v any) error
}

// DefaultCodec returns the Codec used by NewServer and Dial when no
// codec option is supplied.
func DefaultCodec() Codec {
	panic("not implemented")
}
