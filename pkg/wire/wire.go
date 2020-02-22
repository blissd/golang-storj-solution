// Package wire proves protocol framing functions.
// The data types, byte, string, and int64 can be encoded
// into frames suitable for sending across a network connection.
// The frame format is as follows:
// byte 0 - frame length, max 255 bytes
// byte 1 - frame id, effectively the type of frame.
// byte 2+ - frame payload, up to 253 bytes
// Frames don't encode any type information It is up to the caller to
// understand what type correlates to what frame id.
package wire

import (
	"encoding/binary"
	"fmt"
	"io"
	"unsafe"
)

// Encodes data types to an underlying io.Writer
type FrameEncoder interface {
	EncodeByte(b byte) error
	EncodeString(s string) error
	EncodeInt64(i int64) error
}

// Decodes data types from an underlying io.Reader
type FrameDecoder interface {
	DecodeByte() (byte, error)
	DecodeString() (string, error)
	DecodeInt64() (int64, error)
}

type frameEncoder struct {
	io.Writer
}

type frameDecoder struct {
	io.Reader
}

func NewEncoder(w io.Writer) FrameEncoder {
	return &frameEncoder{
		Writer: w,
	}
}

func NewDecoder(r io.Reader) FrameDecoder {
	return &frameDecoder{
		Reader: r,
	}
}

func (enc *frameEncoder) EncodeByte(b byte) error {
	_, err := enc.Write([]byte{2, b})
	if err != nil {
		return fmt.Errorf("wire.WriteByte: %w", err)
	}
	return nil
}

func (enc *frameEncoder) EncodeString(s string) error {
	if len(s) > 254 {
		return fmt.Errorf("wire.WriteString: too long %v", len(s))
	}
	if _, err := enc.Write([]byte{byte(len(s) + 1)}); err != nil {
		return fmt.Errorf("wire.WriteString: write length: %w", err)
	}
	if _, err := enc.Write([]byte(s)); err != nil {
		return fmt.Errorf("wire.WriteString: write payload: %w", err)
	}
	return nil
}

func (enc *frameEncoder) EncodeInt64(i int64) error {
	if _, err := enc.Write([]byte{byte(unsafe.Sizeof(i) + 1)}); err != nil {
		return fmt.Errorf("wire.EncodeInt64: write length: %w", err)
	}
	if err := binary.Write(enc, binary.BigEndian, i); err != nil {
		return fmt.Errorf("wire.EncodeInt64: %w", err)
	}
	return nil
}

func (dec *frameDecoder) DecodeByte() (byte, error) {
	bs := make([]byte, 2)
	_, err := dec.Read(bs)
	if err != nil {
		return 0, fmt.Errorf("wire.DecodeByte: %w", err)
	}
	if bs[0] != 2 {
		return 0, fmt.Errorf("wire.DecodeString: bad length: %v", bs[0])
	}
	return bs[1], nil
}

func (dec *frameDecoder) DecodeString() (string, error) {
	bs := make([]byte, 1)
	if _, err := dec.Read(bs); err != nil {
		return "", fmt.Errorf("wire.DecodeString length: %w", err)
	}
	length := bs[0]
	if length < 1 {
		return "", fmt.Errorf("wire.DecodeString: bad length: %v", bs[0])
	}

	bs = make([]byte, length-1)

	_, err := dec.Read(bs)
	if err != nil {
		return "", fmt.Errorf("wire.DecodeString: read payload: %w", err)
	}
	return string(bs), nil
}

func (dec *frameDecoder) DecodeInt64() (int64, error) {
	bs := make([]byte, 1)
	_, err := dec.Read(bs)
	if err != nil {
		return 0, fmt.Errorf("wire.DecodeString: %w", err)
	}

	var i int64
	if bs[0] != 1+byte(unsafe.Sizeof(i)) {
		return 0, fmt.Errorf("wire.DecodeInt64: bad length: %v", bs[0])
	}

	if err := binary.Read(dec, binary.BigEndian, &i); err != nil {
		return 0, fmt.Errorf("wire.DecodeInt64: %w", err)
	}
	return i, nil
}
