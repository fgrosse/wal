package wal

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
)

// ExampleEntry is an Entry implementation that is used in unit tests.
type ExampleEntry struct {
	ID    uint32
	Point []float32
}

// ExampleEntryType is the EntryType that is returned by an ExampleEntry.
const ExampleEntryType = EntryType(1)

var ExampleEntries = NewEntryRegistry(
	NewExampleEntry,
)

func NewExampleEntry() Entry {
	return new(ExampleEntry)
}

func (*ExampleEntry) Type() EntryType {
	return ExampleEntryType
}

func (e *ExampleEntry) EncodePayload(b []byte) []byte {
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

func (*ExampleEntry) ReadPayload(r io.Reader) ([]byte, error) {
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

func (e *ExampleEntry) DecodePayload(b []byte) error {
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
