/*
Implementation of RFC 6143 §7.7 Encodings.
https://tools.ietf.org/html/rfc6143#section-7.7
*/
package vnc

import (
	"bytes"
	"fmt"

	"github.com/kward/go-vnc/encodings"
)

// An Encoding implements a method for encoding pixel data that is
// sent by the server to the client.
type Encoding interface {
	fmt.Stringer
	Marshaler

	// Read the contents of the encoded pixel data from the reader.
	// This should return a new Encoding implementation that contains
	// the proper data.
	Read(*ClientConn, *Rectangle) (Encoding, error)

	// The number that uniquely identifies this encoding type.
	Type() encodings.Encoding
}

// Encodings describes a slice of Encoding.
type Encodings []Encoding

// Verify that interfaces are honored.
var _ Marshaler = (*Encodings)(nil)

// Marshal implements the Marshaler interface.
func (e Encodings) Marshal() ([]byte, error) {
	buf := NewBuffer(nil)
	for _, enc := range e {
		if err := buf.Write(enc.Type()); err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

// RawEncoding is the simplest encoding type, which is raw pixel data.
// See RFC 6143 §7.7.1.
// https://tools.ietf.org/html/rfc6143#section-7.7.1
type RawEncoding struct {
	Colors []Color
}

// Verify that interfaces are honored.
var _ Encoding = (*RawEncoding)(nil)

// Marshal implements the Encoding interface.
func (e *RawEncoding) Marshal() ([]byte, error) {
	buf := NewBuffer(nil)

	for _, c := range e.Colors {
		bytes, err := c.Marshal()
		if err != nil {
			return nil, err
		}
		if err := buf.Write(bytes); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Read implements the Encoding interface.
func (*RawEncoding) Read(c *ClientConn, rect *Rectangle) (Encoding, error) {
	var buf bytes.Buffer
	bytesPerPixel := int(c.pixelFormat.BPP / 8)
	n := rect.Area() * bytesPerPixel
	if err := c.receiveN(&buf, n); err != nil {
		return nil, fmt.Errorf("unable to read rectangle with raw encoding: %s", err)
	}

	colors := make([]Color, rect.Area())
	for y := uint16(0); y < rect.Height; y++ {
		for x := uint16(0); x < rect.Width; x++ {
			color := NewColor(&c.pixelFormat, &c.colorMap)
			if err := color.Unmarshal(buf.Next(bytesPerPixel)); err != nil {
				return nil, err
			}
			colors[int(y)*int(rect.Width)+int(x)] = *color
		}
	}

	return &RawEncoding{colors}, nil
}

// String implements the fmt.Stringer interface.
func (*RawEncoding) String() string { return "RawEncoding" }

// Type implements the Encoding interface.
func (*RawEncoding) Type() encodings.Encoding { return encodings.Raw }

// DesktopSizePseudoEncoding enables desktop resize support.
// See RFC 6143 §7.8.2.
type DesktopSizePseudoEncoding struct{}

// Verify that interfaces are honored.
var _ Encoding = (*DesktopSizePseudoEncoding)(nil)

// Marshal implements the Marshaler interface.
func (e *DesktopSizePseudoEncoding) Marshal() ([]byte, error) {
	return []byte{}, nil
}

// Read implements the Encoding interface.
func (*DesktopSizePseudoEncoding) Read(c *ClientConn, rect *Rectangle) (Encoding, error) {
	c.fbWidth = rect.Width
	c.fbHeight = rect.Height

	return &DesktopSizePseudoEncoding{}, nil
}

// String implements the fmt.Stringer interface.
func (e *DesktopSizePseudoEncoding) String() string { return "DesktopSizePseudoEncoding" }

// Type implements the Encoding interface.
func (*DesktopSizePseudoEncoding) Type() encodings.Encoding { return encodings.DesktopSizePseudo }
