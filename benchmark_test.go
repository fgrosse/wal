package wal_test

import (
	"hash/crc32"
	"math/rand"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/fgrosse/wal"
	"github.com/fgrosse/wal/waltest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var (
	exampleBenchmarkEntries [1000]*waltest.ExampleEntry1
	initExampleSegmentFile  = new(sync.Once)
	exampleSegmentFileName  string
)

func init() {
	seed := time.Now().UnixMilli()
	rng := rand.New(rand.NewSource(seed))
	for i := range exampleBenchmarkEntries {
		exampleBenchmarkEntries[i] = &waltest.ExampleEntry1{
			ID:    uint32(i + 1),
			Point: []float32{rng.Float32() * 10, rng.Float32() * 10},
		}
	}
}

func BenchmarkWAL_Write(b *testing.B) {
	path := b.TempDir()
	conf := wal.DefaultConfiguration()
	w, err := wal.New(path, conf, waltest.ExampleEntries, zap.NewNop())
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := exampleBenchmarkEntries[i%len(exampleBenchmarkEntries)]
		_, _ = w.Write(e)
	}
}

func BenchmarkSegmentReader(b *testing.B) {
	initExampleSegmentFile.Do(func() {
		f, err := os.CreateTemp(b.TempDir(), "*.wal")
		require.NoError(b, err)

		w := wal.NewSegmentWriter(f)
		write := func(offset uint32, e wal.Entry) {
			payload := make([]byte, 4+2+4*2)
			e.EncodePayload(payload)
			checksum := crc32.ChecksumIEEE(payload)
			err := w.Write(offset, waltest.ExampleEntry1Type, checksum, payload)
			require.NoError(b, err)
		}

		for i, e := range exampleBenchmarkEntries {
			write(uint32(i+1), e)
		}

		require.NoError(b, w.Close())
		exampleSegmentFileName = f.Name()
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f, err := os.Open(exampleSegmentFileName)
		require.NoError(b, err)

		r, err := wal.NewSegmentReader(f, waltest.ExampleEntries)
		require.NoError(b, err)

		// var lastOffset uint32
		for r.Next() {
			_, _, err := r.Read()
			require.NoError(b, err)
			// lastOffset = offset
		}

		require.NoError(b, r.Err())
	}
}
