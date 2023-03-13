package wal

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

type SegmentReader struct {
	r        *bufio.Reader
	offset   uint32
	typ      EntryType
	checksum uint32
	entry    Entry
	payload  []byte
	err      error
	registry *EntryRegistry
}

func NewSegmentReader(r io.Reader, reg *EntryRegistry) (*SegmentReader, error) {
	return &SegmentReader{
		r:        bufio.NewReader(r),
		registry: reg,
	}, nil
}

// Next loads the data for the next Entry from the underlying reader.
// For efficiency reasons, this function neither checks the entry checksum,
// nor does it decode the entry bytes. This is done, so the caller can quickly
// seek through a WAL up to a specific offset without having to decode each WAL
// entry.
//
// In order to actually decode the read WAL entry, you need to use SegmentReader.Read(…).
func (r *SegmentReader) Next() bool {
	var header [9]byte // 4B offset + 1B type + 4B checksum
	n, err := io.ReadFull(r.r, header[:])
	if err == io.EOF {
		return false
	}

	if err != nil {
		r.err = err
		return false
	}

	if n != 9 {
		r.err = io.ErrUnexpectedEOF
		return false
	}

	r.offset = binary.BigEndian.Uint32(header[:4])
	r.typ = EntryType(header[4])
	r.checksum = binary.BigEndian.Uint32(header[5:9])

	r.entry, err = r.registry.New(r.typ)
	if err != nil {
		r.err = err
		return false
	}

	r.payload, r.err = r.entry.ReadPayload(r.r)
	return true
}

// Read decodes the next Entry and returns it together with its WAL offset.
// Before any entry can be read, the caller must call SegmentReader.Next() first.
//
// The complete usage pattern looks like this:
//
//	r, err := NewSegmentReader(…)
//	…
//
//	for r.Next() {
//	  e, offset, err := r.Read()
//	  …
//	}
//
//	if err := r.Err(); err != nil {
//	  …
//	}
func (r *SegmentReader) Read() (entry Entry, offset uint32, err error) {
	if r.err != nil {
		return nil, 0, r.err
	}

	if r.entry == nil {
		return nil, r.offset, errors.New("must call SegmentReader.Next() first")
	}

	if r.checksum != crc32.ChecksumIEEE(r.payload) {
		return nil, r.offset, fmt.Errorf("detected WAL Entry corruption at WAL offset %d", r.offset)
	}

	err = r.entry.DecodePayload(r.payload)
	return r.entry, r.offset, err
}

// Err returns any error that happened when calling Next(). This function must
// be called even if there was not a single Entry to read.
//
// See SegmentReader.Read() to see the full usage pattern.
func (r *SegmentReader) Err() error {
	return r.err
}
