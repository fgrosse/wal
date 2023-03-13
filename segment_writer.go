package wal

import (
	"bufio"
	"io"
	"os"
)

// The SegmentWriter is responsible for writing WAL entry records to disk.
// This type handles the necessary buffered I/O as well as file system syncing.
//
// Every Entry is written, using the following binary layout (big endian format):
//
//	  ┌─────────────┬───────────┬──────────┬─────────┐
//	  │ Offset (4B) │ Type (1B) │ CRC (4B) │ Payload │
//	  └─────────────┴───────────┴──────────┴─────────┘
//
//		- Offset = 32bit WAL entry number for each record in order to implement a low-water mark
//		- Type = Type of WAL entry
//		- CRC = 32bit hash computed over the payload using CRC
//		- Payload = The actual WAL entry payload data
type SegmentWriter struct {
	w      *bufio.Writer
	size   int // current size of the WAL segment that this writer owns. Used to roll over segment files
	closer io.Closer
	sync   func() error // sync function when writing to a file, otherwise a no-op
}

// NewSegmentWriter returns a new SegmentWriter writing to w, using the default
// write buffer size.
func NewSegmentWriter(w io.WriteCloser) *SegmentWriter {
	return NewSegmentWriterSize(w, DefaultWriteBufferSize)
}

// NewSegmentWriterSize returns a new SegmentWriter writing to w whose buffer
// has at least the specified size.
func NewSegmentWriterSize(w io.WriteCloser, bufferSize int) *SegmentWriter {
	if bufferSize <= 0 {
		bufferSize = DefaultWriteBufferSize
	}

	sw := &SegmentWriter{
		w:      bufio.NewWriterSize(w, bufferSize),
		sync:   func() error { return nil }, // no-op, unless writing to a file
		closer: w,
	}

	if f, ok := w.(*os.File); ok {
		sw.sync = f.Sync
	}

	return sw
}

// Write a new WAL entry.
//
// Note, that we do not use the Entry interface here because encoding the
// payload is done at an earlier stage than actually writing data to the WAL
// segment.
func (w *SegmentWriter) Write(offset uint32, typ EntryType, checksum uint32, payload []byte) error {
	var err error
	writeByte := func(b byte) {
		if err != nil {
			return
		}

		err = w.w.WriteByte(b)
		w.size++
	}

	writeUint32 := func(v uint32) { // big endian format
		writeByte(byte(v >> 24))
		writeByte(byte(v >> 16))
		writeByte(byte(v >> 8))
		writeByte(byte(v))
	}

	writeUint32(offset)
	writeByte(byte(typ))
	writeUint32(checksum)

	if err != nil {
		return err
	}

	if _, err = w.w.Write(payload); err != nil {
		return err
	}

	w.size += len(payload)

	return nil
}

// Sync writes any buffered data to the underlying io.Writer and syncs the file
// systems in-memory copy of recently written data to disk if we are writing to
// an os.File.
func (w *SegmentWriter) Sync() error {
	if err := w.w.Flush(); err != nil {
		return err
	}

	return w.sync()
}

// Close ensures that all buffered data is flushed to disk before and then closes
// the associated writer or file.
func (w *SegmentWriter) Close() error {
	if err := w.Sync(); err != nil {
		return err
	}

	return w.closer.Close()
}
