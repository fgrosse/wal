package wal

import "io"

// Entry is a single record of the Write Ahead Log.
type Entry interface {
	Type() EntryType

	// EncodePayload encodes the payload into the provided buffer. In case the
	// buffer is too small to fit the entire payload, this function can grow the
	// old and return a new slice. Otherwise, the old slice must be returned.
	EncodePayload([]byte) []byte

	// ReadPayload reads the payload from the reader but does not yet decode it.
	// Reading and decoding are separate steps for performance reasons. Sometimes
	// we might want to quickly seek through the WAL without having to decode
	// every entry.
	ReadPayload(r io.Reader) ([]byte, error)

	// DecodePayload decodes an entry from a payload that has previously been read
	// by ReadPayload(…).
	DecodePayload([]byte) error
}

// EntryType is used to distinguish different types of messages that we write
// to the WAL.
type EntryType uint8
