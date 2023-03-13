package wal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEntryRegistry(t *testing.T) {
	r := NewEntryRegistry()
	assert.NotNil(t, r)
}

func TestNewEntryRegistry_PanicIfRegisterTwice(t *testing.T) {
	fun := func() {
		NewEntryRegistry(
			func() Entry { return new(ExampleEntry1) },
			func() Entry { return new(ExampleEntry1) }, // same type registered twice
		)
	}

	assert.PanicsWithError(t, `EntryType 0 was already registered to type "*wal.ExampleEntry1"`, fun)
}

func TestEntryRegistry_Register(t *testing.T) {
	r := NewEntryRegistry()
	err := r.Register(func() Entry { return new(ExampleEntry1) })
	assert.NoError(t, err)

	err = r.Register(func() Entry { return new(ExampleEntry2) })
	assert.NoError(t, err)

	err = r.Register(func() Entry { return new(ExampleEntry2) })
	assert.EqualError(t, err, `EntryType 1 was already registered to type "*wal.ExampleEntry2"`)
}

func TestNewEntryRegistryNew(t *testing.T) {
	r := NewEntryRegistry()
	e, err := r.New(ExampleEntry1Type)
	assert.EqualError(t, err, "unknown WAL entry type 0")
	assert.Nil(t, e)

	err = r.Register(func() Entry { return new(ExampleEntry1) })
	require.NoError(t, err)

	e, err = r.New(ExampleEntry1Type)
	assert.NoError(t, err)
	assert.IsType(t, new(ExampleEntry1), e)

	e, err = r.New(EntryType(255))
	assert.EqualError(t, err, "unknown WAL entry type 255")
	assert.Nil(t, e)
}
