package rpc

import (
	"bytes"
	"testing"
)

func TestCodec_RoundTrip_PreservesValues(t *testing.T) {
	type sub struct {
		N int
		S string
	}
	type msg struct {
		ID   uint64
		Name string
		Tags []string
		Sub  sub
	}

	cases := []struct {
		name string
		in   msg
	}{
		{"zero", msg{}},
		{"populated", msg{ID: 42, Name: "abc", Tags: []string{"a", "b"}, Sub: sub{N: 7, S: "x"}}},
		{"unicode", msg{Name: "héllo 🌍"}},
		{"long_slice", msg{Tags: makeStrings(1000)}},
	}

	codec := DefaultCodec()
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			data, err := codec.Encode(tc.in)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			var got msg
			if err := codec.Decode(data, &got); err != nil {
				t.Fatalf("decode: %v", err)
			}
			// Re-encode and compare bytes; this avoids reflect.DeepEqual
			// quirks for codecs that don't distinguish nil from empty
			// slices.
			data2, err := codec.Encode(got)
			if err != nil {
				t.Fatalf("re-encode: %v", err)
			}
			if !bytes.Equal(data, data2) {
				t.Errorf("round-trip mismatch:\nfirst : %x\nsecond: %x", data, data2)
			}
		})
	}
}

func TestCodec_DecodeIntoNonPointer_ReturnsError(t *testing.T) {
	codec := DefaultCodec()
	data, err := codec.Encode("hello")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	var dst string
	if err := codec.Decode(data, dst); err == nil {
		t.Error("expected error decoding into non-pointer, got nil")
	}
}

func TestCodec_RoundTripWireTypes(t *testing.T) {
	codec := DefaultCodec()

	t.Run("request", func(t *testing.T) {
		in := Request{ID: 7, Method: "Foo.Bar", Body: []byte{0x01, 0x02, 0x03}}
		data, err := codec.Encode(in)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		var got Request
		if err := codec.Decode(data, &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.ID != in.ID || got.Method != in.Method || !bytes.Equal(got.Body, in.Body) {
			t.Errorf("got %+v, want %+v", got, in)
		}
	})

	t.Run("response_ok", func(t *testing.T) {
		in := Response{ID: 7, Body: []byte("ok")}
		data, err := codec.Encode(in)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		var got Response
		if err := codec.Decode(data, &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.ID != in.ID || !bytes.Equal(got.Body, in.Body) || got.Error != nil {
			t.Errorf("got %+v, want %+v", got, in)
		}
	})

	t.Run("response_error", func(t *testing.T) {
		in := Response{ID: 9, Error: &Error{Code: 1, Message: "bad"}}
		data, err := codec.Encode(in)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		var got Response
		if err := codec.Decode(data, &got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.ID != in.ID || got.Error == nil ||
			got.Error.Code != in.Error.Code || got.Error.Message != in.Error.Message {
			t.Errorf("got %+v, want %+v", got, in)
		}
	})
}

func makeStrings(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = "x"
	}
	return out
}
