package wal

import (
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var exampleBenchmarkEntries [1000]*ExampleEntry1

func init() {
	seed := time.Now().UnixMilli()
	rng := rand.New(rand.NewSource(seed))
	for i := range exampleBenchmarkEntries {
		exampleBenchmarkEntries[i] = &ExampleEntry1{
			ID:    uint32(i + 1),
			Point: []float32{rng.Float32() * 10, rng.Float32() * 10},
		}
	}
}

func BenchmarkWAL_Write(b *testing.B) {
	path := b.TempDir()
	conf := DefaultConfiguration()
	wal, err := New(path, conf, ExampleEntries, zap.NewNop())
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		e := exampleBenchmarkEntries[i%len(exampleBenchmarkEntries)]
		_, _ = wal.Write(e)
	}
}
