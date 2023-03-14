package wal

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSegmentWriter_Write(t *testing.T) {
	w := NewTestWriter()
	sw := NewSegmentWriter(w)

	offset := uint32(1234)
	typ := EntryType(0)
	payload := []byte{1, 2, 3, 4, 5}
	checksum := uint32(0x470b99f4)

	err := sw.Write(offset, typ, checksum, payload)
	require.NoError(t, err)

	// The segment writer should use buffered IO. Therefore, before we can assert
	// that the written bytes are actually visible in the writer, we must call
	// the Sync function.

	err = sw.Sync()
	require.NoError(t, err)

	var expected []byte
	expected = binary.BigEndian.AppendUint32(expected, offset) // Offset (4B)
	expected = append(expected, byte(typ))                     // Type (1B)
	expected = append(expected, 0x47, 0x0b, 0x99, 0xf4)        // CRC (4B)
	expected = append(expected, payload...)                    // Payload

	actual := w.Bytes()
	assert.Equal(t, expected, actual)
}

func TestSegmentWriter_Write_Size(t *testing.T) {
	w := NewTestWriter()
	sw := NewSegmentWriter(w)

	assert.Equal(t, 0, sw.size)

	err := sw.Write(42, EntryType(0), uint32(0x470b99f4), []byte{1, 2, 3, 4, 5})
	require.NoError(t, err)
	assert.Equal(t, 4+1+4+5, sw.size) // Offset + Type + CRC + Payload

	err = sw.Write(43, EntryType(0), uint32(0x470b99f4), []byte{'a', 'b', 'c'})
	require.NoError(t, err)
	assert.Equal(t, 14+4+1+4+3, sw.size) // Previous entry + Offset + Type + CRC + Payload
}

func TestNewSegmentWriter_Close(t *testing.T) {
	w := NewTestWriter()
	sw := NewSegmentWriter(w)

	var closed bool
	w.close = func() error {
		closed = true
		return nil
	}

	entry := []byte{1, 2, 3, 4, 5}
	checksum := uint32(0x470b99f4)
	err := sw.Write(1, EntryType(0), checksum, entry)
	require.NoError(t, err)

	// CLosing the segment writer should automatically also flush it

	err = sw.Close()
	require.NoError(t, err)

	actual := w.Bytes()
	assert.Len(t, actual, 14)
	assert.Equal(t, entry, actual[9:])

	assert.True(t, closed)
}

func TestNewSegmentWriter_CloseFile(t *testing.T) {
	f, err := os.CreateTemp("", t.Name())
	require.NoError(t, err)

	t.Logf("Using temporary file %q", f.Name())

	t.Cleanup(func() {
		t.Log("Removing temporary file")
		assert.NoError(t, os.Remove(f.Name()))
	})

	sw := NewSegmentWriter(f)

	entry := []byte{1, 2, 3, 4, 5}
	checksum := uint32(0x470b99f4)
	err = sw.Write(0, 0, checksum, entry)
	require.NoError(t, err)

	// CLosing the segment writer should automatically flush and sync it to disk

	err = sw.Close()
	require.NoError(t, err)

	actual, err := os.ReadFile(f.Name())
	require.NoError(t, err)

	assert.Len(t, actual, 14)
	assert.Equal(t, entry, actual[9:])

	err = f.Close()
	assert.Error(t, err)
	assert.True(t, errors.Is(err, os.ErrClosed))
}

type TestWriter struct {
	buf   *bytes.Buffer
	close func() error
}

func NewTestWriter() *TestWriter {
	return &TestWriter{
		buf: new(bytes.Buffer),
	}
}

func (w *TestWriter) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *TestWriter) Bytes() []byte {
	return w.buf.Bytes()
}

func (w *TestWriter) Close() error {
	if w.close != nil {
		return w.close()
	}
	return nil
}
