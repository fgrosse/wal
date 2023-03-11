package wal

import (
	"bytes"
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSegmentReader(t *testing.T) {
	entries := []*ExampleEntry{
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

	buf := NewTestWriter()
	w := NewSegmentWriter(buf)

	write := func(offset uint32, e Entry) {
		payload := make([]byte, 4+2+4*2)
		e.EncodePayload(payload)
		checksum := crc32.ChecksumIEEE(payload)
		err := w.Write(offset, ExampleEntryType, checksum, payload)
		require.NoError(t, err)
	}

	for i, e := range entries {
		write(uint32(i+1), e)
	}

	require.NoError(t, w.Sync())
	input := bytes.NewReader(buf.Bytes())
	r, err := NewSegmentReader(input, ExampleTypes)
	require.NoError(t, err)

	for i, expected := range entries {
		if !assert.True(t, r.Next()) {
			break
		}

		entry, offset, err := r.Read()
		require.NoError(t, err)

		assert.Equal(t, uint32(i)+1, offset)
		assert.Equal(t, expected, entry)
	}

	assert.NoError(t, r.Err())
}
