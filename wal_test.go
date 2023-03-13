package wal

import (
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fgrosse/zaptest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Run("without existing dir", func(t *testing.T) {
		path := t.TempDir()
		conf := DefaultConfiguration()
		wal, err := New(path, conf, ExampleEntries, zaptest.Logger(t))
		require.NoError(t, err)
		require.NotNil(t, wal)

		s, err := os.Stat(path)
		require.NoError(t, err)
		assert.True(t, s.IsDir())
	})
}

func TestWAL(t *testing.T) {
	path := t.TempDir()
	conf := DefaultConfiguration()
	conf.SyncDelay = time.Millisecond // allow test inserts to be written together which cleans up the test logs a little
	logger := zaptest.Logger(t)

	wal, err := New(path, conf, ExampleEntries, logger)
	require.NoError(t, err)

	t.Log("Inserting first couple of entries")
	inserts := []*ExampleEntry1{
		{ID: 1, Point: []float32{1, 2}},
		{ID: 2, Point: []float32{3, 4}},
		{ID: 3, Point: []float32{5, 6}},
	}

	var lastSeq uint32
	for _, x := range inserts {
		seq, err := wal.Write(x)
		require.NoError(t, err)

		assert.Greater(t, seq, lastSeq)
		lastSeq = seq
	}

	t.Log("Closing WAL properly and then re-open it again.")
	require.NoError(t, wal.Close())

	wal, err = New(path, conf, ExampleEntries, logger)
	require.NoError(t, err)

	t.Log("Inserting another few entries")
	inserts2 := []*ExampleEntry1{
		{ID: 4, Point: []float32{7, 8}},
		{ID: 5, Point: []float32{9, 0}},
	}

	for _, x := range inserts2 {
		_, err := wal.Write(x)
		require.NoError(t, err)
	}

	t.Log("Closing WAL another time")
	require.NoError(t, wal.Close())

	t.Log("Checking if all entries are persisted to disk correctly")
	segments, err := segmentFileNames(path)
	require.NoError(t, err)
	require.Len(t, segments, 1)

	lastSegment, err := os.OpenFile(segments[0], os.O_RDWR, 0666)
	require.NoError(t, err)

	r, err := NewSegmentReader(lastSegment, ExampleEntries)
	require.NoError(t, err)

	expectedEntries := append(inserts, inserts2...)

	var i int
	for r.Next() {
		i++

		entry, offset, err := r.Read()
		require.NoError(t, err)
		assert.EqualValues(t, i, offset)

		t.Logf("Read WAL entry from disk %+v", entry)
		assert.Equal(t, expectedEntries[i-1], entry)
	}

	assert.EqualValues(t, len(inserts)+len(inserts2), i)
}

func TestWAL_Insert_Concurrent(t *testing.T) {
	path := t.TempDir()
	conf := DefaultConfiguration()
	conf.SyncDelay = 10 * time.Millisecond

	wal, err := New(path, conf, ExampleEntries, zaptest.Logger(t))
	require.NoError(t, err)

	n := 100
	inserts := make([]*ExampleEntry1, n)
	for i := range inserts {
		inserts[i] = &ExampleEntry1{
			ID:    uint32(i + 1),
			Point: []float32{rand.Float32() * 10, rand.Float32() * 10},
		}
	}

	var wg sync.WaitGroup
	for _, x := range inserts {
		wg.Add(1)
		go func(e *ExampleEntry1) {
			_, err := wal.Write(e)
			assert.NoError(t, err)
			wg.Done()
		}(x)
	}

	wg.Wait()

	err = wal.Close()
	assert.NoError(t, err)
}
