package wal_test

import (
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/fgrosse/wal"
	"github.com/fgrosse/wal/waltest"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var exampleBenchmarkEntries [1000]*waltest.ExampleEntry1

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

func BenchmarkSegmentReader_SeekEnd(b *testing.B) {
	for i := 0; i < b.N; i++ {
		f, err := os.Open("testdata/segment.wal")
		require.NoError(b, err)
		b.Cleanup(func() { _ = f.Close() })

		r, err := wal.NewSegmentReader(f, waltest.ExampleEntries)
		require.NoError(b, err)

		_, err = r.SeekEnd()
		require.NoError(b, err)
	}
}
