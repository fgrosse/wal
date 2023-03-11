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
	payload  []byte
	err      error

	loaders map[EntryType]NewEntryFunc
}

type NewEntryFunc func() Entry

func NewSegmentReader(r io.Reader, entryLoaders []NewEntryFunc) (*SegmentReader, error) {
	if len(entryLoaders) == 0 {
		return nil, fmt.Errorf("missing entry loaders")
	}

	loaders := make(map[EntryType]NewEntryFunc, len(entryLoaders))
	for _, newEntry := range entryLoaders {
		entry := newEntry()
		typ := entry.Type()
		if _, ok := loaders[typ]; ok {
			return nil, fmt.Errorf("type %v was registered twice", typ)
		}

		loaders[typ] = newEntry
	}

	return &SegmentReader{
		r:       bufio.NewReader(r),
		loaders: loaders,
	}, nil
}

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

	newEntry, ok := r.loaders[r.typ]
	if !ok {
		r.err = fmt.Errorf("unknown WAL entry type %x", byte(r.typ))
		return false
	}

	entry := newEntry()
	r.payload, r.err = entry.ReadPayload(r.r)
	return true
}

// Read, decodes the next Entry and returns it together with its WAL offset.
// Before any entry can be read, the caller must call SegmentReader.Next() first.
//
// The complete usage pattern looks like this:
//
//	r, err := NewSegmentReader(…)
//  …
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

	if r.payload == nil {
		return nil, r.offset, errors.New("must call SegmentReader.Next() first")
	}

	if r.checksum != crc32.ChecksumIEEE(r.payload) {
		return nil, r.offset, fmt.Errorf("detected WAL Entry corruption at WAL offset %d", r.offset)
	}

	newEntry := r.loaders[r.typ]
	entry = newEntry()
	err = entry.DecodePayload(r.payload)

	return entry, r.offset, err
}

// Err returns any error that happened when calling Next(). This function must
// be called even if there was not a single Entry to read.
//
// See SegmentReader.Read() to see the full usage pattern.
func (r *SegmentReader) Err() error {
	return r.err
}
