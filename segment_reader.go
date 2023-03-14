package wal

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
)

// The SegmentReader is responsible for reading WAL entries from their binary
// representation, typically from disk. It is used by the WAL to automatically
// resume the last open segment upon startup, but it can also be used to manually
// iterate through WAL segments.
//
// The complete usage pattern looks like this:
//
//	r, err := NewSegmentReader(…)
//	…
//
//	for r.ReadNext() {
//	  offset := r.Offset()
//	  …
//	  entry, err := r.Decode()
//	  …
//	}
//
//	if err := r.Err(); err != nil {
//	  …
//	}
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

// NewSegmentReader creates a new SegmentReader that reads encoded WAL entries
// from the provided reader. The registry is used to map the entry types that
// have been read to their Entry implementations which contain the decoding logic.
func NewSegmentReader(r io.Reader, registry *EntryRegistry) (*SegmentReader, error) {
	return &SegmentReader{
		r:        bufio.NewReader(r),
		registry: registry,
	}, nil
}

// SeekEnd reads through the entire segment until the end and returns the last offset.
func (r *SegmentReader) SeekEnd() (lastOffset uint32, err error) {
	for r.ReadNext() {
		lastOffset = r.Offset()
	}

	return lastOffset, r.Err()
}

// ReadNext loads the data for the next Entry from the underlying reader.
// For efficiency reasons, this function neither checks the entry checksum,
// nor does it decode the entry bytes. This is done, so the caller can quickly
// seek through a WAL up to a specific offset without having to decode each WAL
// entry.
//
// You can get the offset of the current entry using SegmentReader.Offset().
// In order to actually decode the read WAL entry, you need to use SegmentReader.Decode(…).
func (r *SegmentReader) ReadNext() bool {
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

// Offset returns the offset of the last entry that was read by SegmentReader.ReadNext().
func (r *SegmentReader) Offset() uint32 {
	return r.offset
}

// Decode decodes the last entry that was read using SegmentReader.ReadNext().
func (r *SegmentReader) Decode() (Entry, error) {
	if r.err != nil {
		return nil, r.err
	}

	if r.entry == nil {
		return nil, errors.New("must call SegmentReader.ReadNext() first")
	}

	if r.checksum != crc32.ChecksumIEEE(r.payload) {
		return nil, fmt.Errorf("detected WAL Entry corruption at WAL offset %d", r.offset)
	}

	err := r.entry.DecodePayload(r.payload)
	return r.entry, err
}

// Err returns any error that happened when calling ReadNext(). This function must
// always be called even if ReadNext() never returned true.
//
// Please refer to the comment on the SegmentReader type to see the full usage pattern.
func (r *SegmentReader) Err() error {
	return r.err
}
