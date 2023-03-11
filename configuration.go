package wal

import (
	"time"

	"go.uber.org/zap/zapcore"
)

// Default configuration options. You can use the DefaultConfiguration() function
// to create a Configuration instance that uses these constants.
const (
	DefaultWriteBufferSize  = 16 * 1024
	DefaultMaxSegmentSize   = 10 * 1024 * 1024
	DefaultEntryPayloadSize = 128 // TODO: no clue if this a good default and if a *default* here makes sense generally
)

// Configuration contains all settings of a write-ahead log.
type Configuration struct {
	WriteBufferSize  int // the size of the segment write buffer in bytes
	MaxSegmentSize   int // the file size in bytes at which the segment files will be rotated
	EntryPayloadSize int // the default size for entry payloads. can be tuned to reduce allocations

	// SyncDelay is the duration to wait for syncing writes to disk. The default
	// value 0 will cause every write to be synced immediately.
	SyncDelay time.Duration
}

// MarshalLogObject implements the zapcore.ObjectMarshaler interface.
func (c Configuration) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddInt("write_buffer_bytes", c.WriteBufferSize)
	enc.AddInt("max_segment_bytes", c.MaxSegmentSize)
	enc.AddInt("entry_payload_bytes", c.EntryPayloadSize)
	enc.AddDuration("sync_delay", c.SyncDelay)

	return nil
}

// DefaultConfiguration returns a new Configuration instance that contains all
// default WAL parameters.
func DefaultConfiguration() Configuration {
	return Configuration{
		WriteBufferSize:  DefaultWriteBufferSize,
		MaxSegmentSize:   DefaultMaxSegmentSize,
		EntryPayloadSize: DefaultEntryPayloadSize,
		SyncDelay:        0, // sync every write to disk immediately
	}
}
