package wal_test

import (
	"fmt"
	"os"

	"github.com/fgrosse/wal"
	"go.uber.org/zap"
)

// walEntries is an unexported package level variable that is used to register
// your own wal.Entry implementations. Such an Entry contains the logic of how
// to encode and decode a WAL with your custom data. Each wal.Entry is also
// associated with a unique wa.EntryType so we are able to map the binary
// representation back to your original Go type.
//
// In the example below we use two example implementations which are only
// available in unit tests. You might want to look into their implementation
// (see entry_test.go) to understand how you can efficiently implement your
// own encoding and decoding logic.
var walEntries = wal.NewEntryRegistry(
	func() wal.Entry { return new(wal.ExampleEntry1) },
	func() wal.Entry { return new(wal.ExampleEntry2) },
)

func Example() {
	// The WAL will persist all written entries onto disk in an efficient
	// append-only log file. Entries are split over multiple WAL segment files.
	// To create a new WAL, you have to provide a path to the directory where
	// the segment files will be stored.
	path, err := os.MkdirTemp("", "WALExample")
	check(err)

	// There are a few runtime options for the WAL which have an impact on its
	// performance and durability guarantees. By default, the WAL prefers strong
	// durability and will fsync each write to disk immediately. Under high
	// throughput, such a configuration can make the WAL a bottleneck of your
	// application. Therefore, it might make sense to configure a SyncDelay to
	// let the WAL automatically badge up fyncs for multiple writes.
	conf := wal.DefaultConfiguration()

	// This library uses go.uber.org/zap for efficient structured logging.
	logger, err := zap.NewProduction()
	check(err)

	// When you create a new WAL instance, it will immediately try and load any
	// existent WAL segments from the path you provided. The `walEntries` parameter
	// that is passed to wal.New(…) is an EntryRegistry which lets the WAL know
	// about your own Entry implementation. This way, you can specify your own types
	// and encoding/decoding logic but the WAL is still able to load entries from
	// the last segment.
	w, err := wal.New(path, conf, walEntries, logger)
	check(err)

	// Now you can finally write your first WAL entry. When this function
	// returns without an error you can be sure that it was fully written to disk.
	offset, err := w.Write(&wal.ExampleEntry1{
		ID:    42,
		Point: []float32{1, 2, 3},
	})

	// You might use the offset in your application or ignore it altogether.
	fmt.Print(offset)

	// Finally, you need to close the WAL to release any resources and close the
	// open segment file.
	err = w.Close()
	check(err)
}

// check is a simple helper function to check errors in Example().
// In a real application, you should implement proper error handling.
func check(err error) {
	if err != nil {
		panic(err)
	}
}
