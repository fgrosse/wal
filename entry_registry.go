package wal

import "fmt"

// The EntryRegistry keeps track of all known Entry implementations.
// This is necessary in order to instantiate the correct types when loading WAL
// segments.
type EntryRegistry struct {
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
