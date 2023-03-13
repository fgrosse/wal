package wal

import (
	"fmt"
	"io"
)

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

// The EntryRegistry keeps track of all known Entry implementations.
// This is necessary in order to instantiate the correct types when loading WAL
// segments.
type EntryRegistry struct { // TODO: move into its own file
	newEntry map[EntryType]NewEntryFunc
}

// NewEntryFunc is the constructor function of a specific Entry implementation.
type NewEntryFunc func() Entry // TODO: rename to EntryConstructor

// NewEntryRegistry creates a new EntryRegistry. You can either pass the
// constructor functions of all known Entry implementations or you can later
// register them using EntryRegistry.Register(…).
func NewEntryRegistry(entries ...NewEntryFunc) *EntryRegistry {
	r := &EntryRegistry{newEntry: map[EntryType]NewEntryFunc{}}
	for _, newEntry := range entries {
		r.Register(newEntry)
	}

	return r
}

// Register an Entry constructor function. Each Entry will be registered with
// the EntryType that is returned by the corresponding Entry.Type().
func (r *EntryRegistry) Register(newEntry NewEntryFunc) {
	entry := newEntry()
	typ := entry.Type()
	r.newEntry[typ] = newEntry // TODO: return error if this type is already registered
}

// New instantiates a new Entry implementation that was previously registered
// for the requested EntryType. An error is returned if no Entry was registered
// for this type.
func (r *EntryRegistry) New(typ EntryType) (Entry, error) {
	newEntry, ok := r.newEntry[typ]
	if !ok {
		return nil, fmt.Errorf("unknown WAL entry type %x", byte(typ))
	}

	return newEntry(), nil
}
