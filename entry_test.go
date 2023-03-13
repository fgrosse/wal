package wal

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ExampleEntry1 is an Entry implementation that is used in unit tests.
type ExampleEntry1 struct {
	ID    uint32
	Point []float32
}

// ExampleEntry2 is another Entry implementation that is used in unit tests.
type ExampleEntry2 struct {
	Test bool
	Name string
}

// Constants for the example Entry implementations.
const (
	ExampleEntry1Type EntryType = iota
	ExampleEntry2Type
)

// ExampleEntries is the EntryRegistry that contains all known example Entry
// implementations.
var ExampleEntries = NewEntryRegistry(
	func() Entry { return new(ExampleEntry1) },
	func() Entry { return new(ExampleEntry2) },
)

func TestExampleEntry1(t *testing.T) {
	original := &ExampleEntry1{
		ID:    5546132,
		Point: []float32{1, 2, 3, 4, 5},
	}

	assert.Equal(t, ExampleEntry1Type, original.Type())

	var encoded []byte
	encoded = original.EncodePayload(encoded)
	r := bytes.NewBuffer(encoded)

	decoded := new(ExampleEntry1)
	input, err := decoded.ReadPayload(r)
	require.NoError(t, err)

	err = decoded.DecodePayload(input)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func (*ExampleEntry1) Type() EntryType { return ExampleEntry1Type }

func (e *ExampleEntry1) EncodePayload(b []byte) []byte {
	size := 4 + 2 + 4*len(e.Point)
	if len(b) < size {
		b = append(b, make([]byte, size-len(b))...)
	}

	binary.BigEndian.PutUint32(b[0:4], e.ID)                // 4 byte
	binary.BigEndian.PutUint16(b[4:], uint16(len(e.Point))) // 2 byte
	for i, p := range e.Point {
		v := math.Float32bits(p)
		binary.BigEndian.PutUint32(b[6+i*4:], v)
	}

	return b[:size]
}

func (*ExampleEntry1) ReadPayload(r io.Reader) ([]byte, error) {
	buffer := make([]byte, 6) // 4B ID + 2B Point Dimension
	n, err := io.ReadFull(r, buffer)
	if err == io.EOF || n != len(buffer) {
		return nil, io.ErrUnexpectedEOF
	}

	if err != nil {
		return nil, err
	}

	dimension := binary.BigEndian.Uint16(buffer[4:6])
	values := make([]byte, 4*dimension)
	n, err = io.ReadFull(r, values)
	if err == io.EOF || n != len(values) {
		return nil, io.ErrUnexpectedEOF
	}

	return append(buffer, values...), nil
}

func (e *ExampleEntry1) DecodePayload(b []byte) error {
	r := bytes.NewBuffer(b)

	var err error
	read := func(x any) {
		if err != nil {
			return
		}

		err = binary.Read(r, binary.BigEndian, x)
	}

	read(&e.ID)

	var n uint16
	read(&n)
	e.Point = make([]float32, n)

	for i := uint16(0); i < n; i++ {
		read(&e.Point[i])
	}

	return err
}

func TestExampleEntry2(t *testing.T) {
	original := &ExampleEntry2{
		Test: true,
		Name: "Ada Lovelace",
	}

	var encoded []byte
	encoded = original.EncodePayload(encoded)
	r := bytes.NewBuffer(encoded)

	decoded := new(ExampleEntry2)
	input, err := decoded.ReadPayload(r)
	require.NoError(t, err)

	err = decoded.DecodePayload(input)
	require.NoError(t, err)

	assert.Equal(t, original, decoded)
}

func (*ExampleEntry2) Type() EntryType { return ExampleEntry2Type }

func (e *ExampleEntry2) EncodePayload(b []byte) []byte {
	nameLen := uint16(len(e.Name))
	if len(e.Name) > math.MaxUint16 {
		// If the string is too long, cut it off. In the real world we would
		// handle this case via a setter that validates this constraint.
		nameLen = math.MaxUint16
	}

	// 1 byte: e.Test
	// 2 byte: nameLen
	// n byte: name bytes
	totalLen := 1 + 2 + int(nameLen)
	if len(b) < totalLen {
		b = append(b, make([]byte, totalLen-len(b))...)
	}

	if e.Test {
		b[0] = 1
	} else {
		b[0] = 0
	}

	binary.BigEndian.PutUint16(b[1:3], nameLen) // 2 byte
	copy(b[3:], e.Name)

	return b
}

func (*ExampleEntry2) ReadPayload(r io.Reader) ([]byte, error) {
	buffer := make([]byte, 3) // 1B e.Test + 2B len(b.Name)
	n, err := io.ReadFull(r, buffer)
	if err == io.EOF || n != len(buffer) {
		return nil, io.ErrUnexpectedEOF
	}

	if err != nil {
		return nil, err
	}

	nameLen := binary.BigEndian.Uint16(buffer[1:3])
	name := make([]byte, nameLen)
	n, err = io.ReadFull(r, name)
	if err == io.EOF || n != len(name) {
		return nil, io.ErrUnexpectedEOF
	}

	return append(buffer, name...), nil
}

func (e *ExampleEntry2) DecodePayload(b []byte) error {
	if b[0] == 1 {
		e.Test = true
	} else {
		e.Test = false
	}

	e.Name = string(b[3:])
	return nil
}
