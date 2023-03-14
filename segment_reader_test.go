package wal_test

import (
	"bytes"
	"hash/crc32"
	"testing"

	"github.com/fgrosse/wal"
	"github.com/fgrosse/wal/waltest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSegmentReader(t *testing.T) {
	entries := []*waltest.ExampleEntry1{
		{
			ID:    42,
			Point: []float32{1, 2},
		},
		{
			ID:    43,
			Point: []float32{3, 4},
		},
		{
			ID:    44,
			Point: []float32{5, 6},
		},
	}

	buf := wal.NewTestWriter()
	w := wal.NewSegmentWriter(buf)

	write := func(offset uint32, e wal.Entry) {
		payload := make([]byte, 4+2+4*2)
		e.EncodePayload(payload)
		checksum := crc32.ChecksumIEEE(payload)
		err := w.Write(offset, waltest.ExampleEntry1Type, checksum, payload)
		require.NoError(t, err)
	}

	for i, e := range entries {
		write(uint32(i+1), e)
	}

	require.NoError(t, w.Sync())
	input := bytes.NewReader(buf.Bytes())
	r, err := wal.NewSegmentReader(input, waltest.ExampleEntries)
	require.NoError(t, err)

	for i, expected := range entries {
		offset, ok := r.Next()
		if !assert.True(t, ok) {
			break
		}

		assert.Equal(t, uint32(i)+1, offset)

		entry, err := r.Read()
		require.NoError(t, err)
		assert.Equal(t, expected, entry)
	}

	assert.NoError(t, r.Err())
}
