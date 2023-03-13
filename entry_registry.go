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

// NewEntryRegistry creates a new EntryRegistry. If you pass any constructor to
// this function, each must create a unique Entry implementation (i.e. one which
// returns a unique EntryType). Otherwise, this function panics.
//
// Alternatively, you can register the constructor functions using EntryRegistry.Register(…).
func NewEntryRegistry(constructors ...EntryConstructor) *EntryRegistry {
	r := &EntryRegistry{constructors: map[EntryType]EntryConstructor{}}
	for _, newEntry := range constructors {
		err := r.Register(newEntry)
		if err != nil {
			panic(err)
		}
	}

	return r
}

// Register an EntryConstructor function. Each Entry will be registered with
// the EntryType that is returned by the corresponding Entry.Type().
//
// An error is returned if this constructor was already registered; i.e. a
// constructor was already registered that creates an Entry with the same
// EntryType as this constructor's Entry.
func (r *EntryRegistry) Register(constructor EntryConstructor) error {
	entry := constructor()
	typ := entry.Type()
	if existing, ok := r.constructors[typ]; ok {
		return fmt.Errorf(`EntryType %x was already registered to type "%T"`, typ, existing())
	}

	r.constructors[typ] = constructor
	return nil
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
