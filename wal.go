package wal

import (
	"errors"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"go.uber.org/atomic"
	"go.uber.org/zap"
)

// WAL is a write-ahead log implementation.
type WAL struct {
	logger *zap.Logger
	conf   Configuration

	buffers sync.Pool // byte buffers for creating new WAL entries
	path    string    // filesystem path to the WAL directory

	mu         sync.Mutex
	lastOffset uint32 // the last offset that has been written or zero if no writes occurred yet
	segmentID  int    // ID of the current WAL segment, used to create segment file names
	segment    *SegmentWriter

	syncScheduled *atomic.Bool
	syncWaiters   []chan<- error // goroutines waiting for the next fsync
	closing       chan struct{}  // channel to signal that the WAL was closed (by closing the channel)
}

// New creates a new WAL instance that writes and reads segment files to a
// directory at the provided path.
func New(path string, conf Configuration, entryLoaders []NewEntryFunc, logger *zap.Logger) (*WAL, error) {
	logger.Debug("Creating write-ahead log",
		zap.String("path", path),
		zap.Object("configuration", conf),
	)

	if err := os.MkdirAll(path, 0777); err != nil {
		return nil, fmt.Errorf("creating WAL directory: %w", err)
	}

	wal := &WAL{
		logger:        logger,
		conf:          conf,
		path:          path,
		syncScheduled: atomic.NewBool(false),
		closing:       make(chan struct{}),
		buffers: sync.Pool{
			New: func() interface{} {
				if conf.EntryPayloadSize > 0 {
					return make([]byte, conf.EntryPayloadSize) // TODO: remove this in favor of learning a value automatically
				}

				var b []byte
				return b
			},
		},
	}

	err := wal.load(path, entryLoaders, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to load WAL: %w", err)
	}

	return wal, nil
}

func (w *WAL) load(path string, entryLoaders []NewEntryFunc, logger *zap.Logger) error {
	logger = logger.With(zap.String("path", path))

	logger.Debug("Checking for existing WAL segment files")

	segments, err := segmentFileNames(path)
	if err != nil {
		return fmt.Errorf("checking existing segment files: %w", err)
	}

	if len(segments) == 0 {
		logger.Debug("Did not find any existing WAL segment files, proceeding with empty WAL")
		return nil
	}

	lastSegment := segments[len(segments)-1]

	logger.Info("Loading existing WAL segments",
		zap.Strings("segments", segments),
		zap.String("last_segment", lastSegment),
	)

	segmentWriter, lastOffset, err := w.openSegment(lastSegment, entryLoaders)
	if err != nil {
		return fmt.Errorf("opening last segment: %w", err)
	}

	logger.Info("Finished reading last WAL segment",
		zap.String("last_segment", lastSegment),
		zap.Uint32("last_offset", lastOffset),
	)

	w.segment = segmentWriter
	w.lastOffset = lastOffset

	return nil
}

// segmentFileNames will return all files that are WAL segment files in sorted order by ascending ID.
func segmentFileNames(dir string) ([]string, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.wal"))
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	return names, nil
}

func (w *WAL) openSegment(path string, entryLoaders []NewEntryFunc) (*SegmentWriter, uint32, error) {
	f, err := os.OpenFile(path, os.O_RDWR, 0666)
	if err != nil {
		return nil, 0, err
	}

	lastOffset, err := w.readSegment(f, entryLoaders)
	if err != nil {
		return nil, 0, err
	}

	sw := NewSegmentWriterSize(f, w.conf.WriteBufferSize)
	return sw, lastOffset, nil
}

func (w *WAL) readSegment(f *os.File, entryLoaders []NewEntryFunc) (lastOffset uint32, err error) {
	r, err := NewSegmentReader(f, entryLoaders)
	if err != nil {
		return 0, fmt.Errorf("failed to create WAL segment reader: %w", err)
	}

	for r.Next() {
		_, offset, err := r.Read()
		if err != nil {
			return 0, fmt.Errorf("failed to read WAL entry: %w", err)
		}

		lastOffset = offset
	}

	return lastOffset, r.Err()
}

func (w *WAL) Write(e Entry) (offset uint32, err error) {
	// TODO: limit how many concurrent encodings can be in flight.  Since we can only
	//	     write one at a time to disk, a slow disk can cause the allocations below
	//	     to increase quickly.  If we're backed up, wait until others have completed.

	// Serialize the new WAL entry first into a buffer and then flush it with a
	// single write operation to disk.
	entryPayload := e.EncodePayload(w.buffers.Get().([]byte))

	// Calculate checksum of the payload to enable detecting WAL entry corruption.
	entryChecksum := crc32.ChecksumIEEE(entryPayload)

	// Create a channel that will later receive the result from concurrently
	// syncing the WAL. The channel must be buffered because the reader of the
	// channel might abandon it if syncing takes too long or there was another
	// error. In this case we must ensure that the sync() function does not
	// block when delivering the sync results.
	syncResult := make(chan error, 1)

	offset, err = w.write(e.Type(), entryPayload, entryChecksum, syncResult)

	// First, put back the buffer. We don't have to clean it because it is
	// completely overwritten, the next time it is used.
	// TODO: we actually may have to clean it if we support different entry sizes?
	w.buffers.Put(entryPayload)

	// Now check the error from writing. We can return immediately if it failed.
	if err != nil {
		return 0, err
	}

	// Lastly, wait for the fsync to complete.
	return offset, <-syncResult
}

func (w *WAL) write(typ EntryType, payload []byte, checksum uint32, syncResult chan<- error) (offset uint32, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// While holding the lock, make sure the log has not been closed.
	if w.isClosed() {
		return 0, errors.New("WAL is already closed")
	}

	// First check if we need to roll over to a new segment because the current
	// one is full. It might also be that we do not yet have a segment file at
	// all, because this is the very first write to the WAL. In this case this
	// function is going to set up the segment writer for us now.
	err = w.rollSegment()
	if err != nil {
		return 0, fmt.Errorf("failed to roll WAL segment: %w", err)
	}

	offset = w.lastOffset + 1

	w.logger.Debug("Writing WAL entry",
		zap.Int("segment_id", w.segmentID),
		zap.Uint32("offset", offset),
		zap.Uint32("crc32", checksum),
	)

	err = w.segment.Write(offset, typ, checksum, payload)
	if err != nil {
		return 0, err
	}

	w.lastOffset = offset

	err = w.scheduleSync(syncResult)
	return offset, err
}

func (w *WAL) rollSegment() error {
	if w.segment != nil && w.segment.size < w.conf.MaxSegmentSize {
		return nil
	}

	if err := w.newSegmentFile(); err != nil {
		return fmt.Errorf("error opening new segment file for wal (2): %v", err)
	}

	return nil
}

func (w *WAL) newSegmentFile() error {
	w.segmentID++

	if w.segment != nil {
		// Sync all waiting writes to the old segment and then close it.
		w.sync()

		if err := w.segment.Close(); err != nil {
			return err
		}
	}

	fileName := filepath.Join(w.path, fmt.Sprintf("%d.wal", w.segmentID))
	fd, err := os.OpenFile(fileName, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return err
	}

	w.logger.Debug("Starting new WAL segment",
		zap.Int("segment_id", w.segmentID),
		zap.String("path", fileName),
	)

	w.segment = NewSegmentWriterSize(fd, w.conf.WriteBufferSize)

	return nil
}

// sync the segment writer and then notify all goroutines that currently wait
// for a WAL sync.
// The caller must ensure the WAL is write-locked before calling this function.
func (w *WAL) sync() {
	start := time.Now()
	err := w.segment.Sync()
	took := time.Since(start)

	if len(w.syncWaiters) == 0 {
		return
	}

	w.logger.Debug("Finished syncing WAL to disk",
		zap.NamedError("result", err),
		zap.Duration("took", took),
		zap.Int("waiting_inserts", len(w.syncWaiters)),
	)

	for _, resultChan := range w.syncWaiters {
		resultChan <- err
	}

	w.syncWaiters = nil
}

// Register the given syncResult channel and then schedule an asynchronous WAL
// sync.
// The caller must ensure the WAL is write-locked before calling this function.
func (w *WAL) scheduleSync(syncResult chan<- error) error {
	w.syncWaiters = append(w.syncWaiters, syncResult)

	// Check if we are already waiting for a sync. In this another goroutine
	// will handle the fsync for us.
	if w.syncScheduled.Swap(true) {
		return nil
	}

	// Concurrently fsync the WAL and then notify all pending waiters.
	go func() {
		defer func() {
			w.syncScheduled.Swap(false)
		}()

		if w.conf.SyncDelay > 0 {
			t := time.NewTimer(w.conf.SyncDelay)
			select {
			case <-t.C:
				t.Stop()
				// time is up
			case <-w.closing:
				// we are going down, return immediately and let Close() handle syncing.
				return
			}
		}

		// Make sure the WAL has not been closed concurrently and then finally
		// do the sync.
		w.mu.Lock()
		if !w.isClosed() {
			w.sync()
		}
		w.mu.Unlock()
	}()

	return nil
}

// Close gracefully shuts down the writeAheadLog by making sure that all pending
// writes are completed and synced to disk before then closing the WAL segment file.
// Any future writes after the WAL has been closed will lead to an error.
func (w *WAL) Close() error {
	// First squire the lock, so we know that no writes happen at the moment and
	// no new syncs can be scheduled.
	w.mu.Lock()
	defer w.mu.Unlock()

	w.logger.Info("Closing WAL")

	if w.segment == nil {
		// We never got a single write, so we can return immediately.
		return nil
	}

	// Stop sync goroutine and sync all waiting writes if there are any.
	close(w.closing)
	w.sync()

	// Shutdown the segment writer.
	err := w.segment.Close()
	w.segment = nil

	return err
}

// isClosed returns whether the WAL was closed.
// The caller must ensure the WAL is write-locked before calling this function.
func (w *WAL) isClosed() bool {
	select {
	case <-w.closing:
		// If the "closing" channel is closed itself, we know that the WAL was closed.
		return true
	default:
		// By default, the WAL is not closed.
		return false
	}
}

// Offset returns the last offset that the WAL has written to disk
func (w *WAL) Offset() uint32 {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isClosed() {
		w.sync()
	}

	return w.lastOffset
}
