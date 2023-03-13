package wal

import "fmt"

// The EntryRegistry keeps track of all known Entry implementations.
// This is necessary in order to instantiate the correct types when loading WAL
// segments.
type EntryRegistry struct {
	constructors map[EntryType]EntryConstructor
}

// EntryConstructor is the constructor function of a specific Entry implementation.
type EntryConstructor func() Entry

// NewEntryRegistry creates a new EntryRegistry. You can either pass the
// constructor functions of all known Entry implementations or you can later
// register them using EntryRegistry.Register(…).
func NewEntryRegistry(constructors ...EntryConstructor) *EntryRegistry {
	r := &EntryRegistry{constructors: map[EntryType]EntryConstructor{}}
	for _, newEntry := range constructors {
		r.Register(newEntry)
	}

	return r
}

// Register an EntryConstructor function. Each Entry will be registered with
// the EntryType that is returned by the corresponding Entry.Type().
func (r *EntryRegistry) Register(constructor EntryConstructor) {
	entry := constructor()
	typ := entry.Type()
	r.constructors[typ] = constructor // TODO: return error if this type is already registered
}

// New instantiates a new Entry implementation that was previously registered
// for the requested EntryType. An error is returned if no Entry was registered
// for this type.
func (r *EntryRegistry) New(typ EntryType) (Entry, error) {
	newEntry, ok := r.constructors[typ]
	if !ok {
		return nil, fmt.Errorf("unknown WAL entry type %x", byte(typ))
	}

	return newEntry(), nil
}
