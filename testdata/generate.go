package main

import (
	"fmt"
	"hash/crc32"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/fgrosse/wal"
	"github.com/fgrosse/wal/waltest"
)

func main() {
	n := 1000
	path := "testdata/segment.wal"
	log.SetFlags(0)
	log.SetPrefix("> ")
	seed := time.Now().UnixMilli()
	entries := randomEntries(n, seed)

	log.Printf("Writing example segment file to %s", path)
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}

	err = writeSegment(f, entries)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Validating segment file...")
	err = validateSegment(f.Name(), n)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf(`Success... \ʕ◔ϖ◔ʔ/`)
}

func randomEntries(n int, seed int64) []*waltest.ExampleEntry1 {
	rng := rand.New(rand.NewSource(seed))
	entries := make([]*waltest.ExampleEntry1, n)
	for i := range entries {
		entries[i] = &waltest.ExampleEntry1{
			ID:    uint32(i + 1),
			Point: []float32{rng.Float32() * 10, rng.Float32() * 10},
		}
	}
	return entries
}

func writeSegment(f *os.File, entries []*waltest.ExampleEntry1) error {
	var payloadSize int // learned from first entry
	w := wal.NewSegmentWriter(f)
	for i, e := range entries {
		offset := uint32(i + 1)

		payload := make([]byte, payloadSize)
		payload = e.EncodePayload(payload)
		payloadSize = len(payload)

		checksum := crc32.ChecksumIEEE(payload)
		err := w.Write(offset, waltest.ExampleEntry1Type, checksum, payload)
		if err != nil {
			return err
		}
	}

	return w.Close()
}

func validateSegment(path string, expectedLastOffset int) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}

	r, err := wal.NewSegmentReader(f, waltest.ExampleEntries)
	if err != nil {
		return err
	}

	var lastOffset uint32
	for {
		offset, ok := r.Next()
		if !ok {
			break
		}

		_, err = r.Read()
		if err != nil {
			return err
		}

		lastOffset = offset
	}

	if lastOffset != uint32(expectedLastOffset) {
		return fmt.Errorf("expected last offset to equal %d but it was %d", expectedLastOffset, lastOffset)
	}

	return nil
}
