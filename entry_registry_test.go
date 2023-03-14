package wal_test

import (
	"testing"

	"github.com/fgrosse/wal"
	"github.com/fgrosse/wal/waltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEntryRegistry(t *testing.T) {
	r := wal.NewEntryRegistry()
	assert.NotNil(t, r)
}

func TestNewEntryRegistry_PanicIfRegisterTwice(t *testing.T) {
	fun := func() {
		wal.NewEntryRegistry(
			func() wal.Entry { return new(waltest.ExampleEntry1) },
			func() wal.Entry { return new(waltest.ExampleEntry1) }, // same type registered twice
		)
	}

	assert.PanicsWithError(t, `EntryType 0 was already registered to type "*waltest.ExampleEntry1"`, fun)
}

func TestEntryRegistry_Register(t *testing.T) {
	r := wal.NewEntryRegistry()
	err := r.Register(func() wal.Entry { return new(waltest.ExampleEntry1) })
	assert.NoError(t, err)

	err = r.Register(func() wal.Entry { return new(waltest.ExampleEntry2) })
	assert.NoError(t, err)

	err = r.Register(func() wal.Entry { return new(waltest.ExampleEntry2) })
	assert.EqualError(t, err, `EntryType 1 was already registered to type "*waltest.ExampleEntry2"`)
}

func TestNewEntryRegistryNew(t *testing.T) {
	r := wal.NewEntryRegistry()
	e, err := r.New(waltest.ExampleEntry1Type)
	assert.EqualError(t, err, "unknown WAL entry type 0")
	assert.Nil(t, e)

	err = r.Register(func() wal.Entry { return new(waltest.ExampleEntry1) })
	require.NoError(t, err)

	e, err = r.New(waltest.ExampleEntry1Type)
	assert.NoError(t, err)
	assert.IsType(t, new(waltest.ExampleEntry1), e)

	e, err = r.New(wal.EntryType(255))
	assert.EqualError(t, err, "unknown WAL entry type 255")
	assert.Nil(t, e)
}
