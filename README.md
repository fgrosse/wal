<h1 align="center">Go Write-Ahead Log 🏃🧾</h1>
<p align="center">A write-ahead logging (WAL) implementation in Go. </p>
<p align="center">
    <a href="https://github.com/fgrosse/wal/releases"><img src="https://img.shields.io/github/tag/fgrosse/wal.svg?label=version&color=brightgreen"></a>
    <a href="https://github.com/fgrosse/wal/actions/workflows/test.yml"><img src="https://github.com/fgrosse/wal/actions/workflows/test.yml/badge.svg"></a>
    <a href="https://goreportcard.com/report/github.com/fgrosse/wal"><img src="https://goreportcard.com/badge/github.com/fgrosse/wal"></a>
    <a href="https://pkg.go.dev/github.com/fgrosse/wal"><img src="https://img.shields.io/badge/godoc-reference-blue.svg?color=blue"></a>
    <a href="https://github.com/fgrosse/wal/blob/master/LICENSE"><img src="https://img.shields.io/badge/license-BSD--3--Clause-blue.svg"></a>
</p>

<p align="center"><b>THIS SOFTWARE IS STILL IN ALPHA AND THERE ARE NO GUARANTEES REGARDING API STABILITY YET.</b></p>

---

Package `wal` implements an efficient Write-ahead log for Go applications.

The main goal of a Write-ahead Log (WAL) is to make the application more durable,
so it does not lose data in case of a crash. WALs are used in applications such as
database systems to flush all written data to disk before the changes are written
to the database. In case of a crash, the WAL enables the application to recover
lost in-memory changes by reconstructed all required operations from the log.

## Example usage

The code below is a copy of [`example_test.go`](example_test.go). It shows the
general usage of this library together with some explanation.

[embedmd]:# (example_test.go)
```go
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
```

### Encoding your own WAL Entries

Your custom entries must implement the `wal.Entry` interface:

[embedmd]:# (entry.go /.*Entry is a single record of the Write Ahead Log.*/ $)
```go
// Entry is a single record of the Write Ahead Log.
// It is up to the application that uses the WAL to provide at least one concrete
// Entry implementation to the WAL via the EntryRegistry.
type Entry interface {
	Type() EntryType

	// EncodePayload encodes the payload into the provided buffer. In case the
	// buffer is too small to fit the entire payload, this function can grow the
	// old and return a new slice. Otherwise, the old slice must be returned.
	EncodePayload([]byte) []byte

	// ReadPayload reads the payload from the reader but does not yet decode it.
	// Reading and decoding are separate steps for performance reasons. Sometimes
	// we might want to quickly seek through the WAL without having to decode
	// every entry.
	ReadPayload(r io.Reader) ([]byte, error)

	// DecodePayload decodes an entry from a payload that has previously been read
	// by ReadPayload(…).
	DecodePayload([]byte) error
}

// EntryType is used to distinguish different types of messages that we write
// to the WAL.
type EntryType uint8
```

You can find an example implementation at [`entry_test.go`](entry_test.go).

## How it works

Each `WAL.Write(…)` call creates a binary encoding of the passed `wal.Entry` which 
we call the entry's _payload_. This payload is written to disk together with some
metadata such as the entry Type, a CRC checksum and an offset number.

The full binary layout looks like the following:

[embedmd]:# (segment_writer.go /.*the following binary layout.*/ /.*- Payload =.*/)
```go
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
```

This data is appended to a file and the WAL makes sure that it is actually
written to non-volatile storage rather than just being stored in a memory-based
write cache that would be lost if power failed (see [fsynced][fsync]).

When the WAL file reaches a configurable maximum size, it is closed and the WAL
starts to append its records to a new and empty file. These files are called WAL
_segments_. Typically, the WAL is split into multiple segments to enable other
processes to take care of cleaning old segments, implement WAL segment backups
and more. When the WAL is started, it will resume operation at the end of the
last open segment file.

## Installation

```sh
$ go get github.com/fgrosse/wal
```

## Built With

* [go.uber.org/zap](go.uber.org/zap) - Blazing fast, structured, leveled logging in Go
* [go.uber.org/atomic](go.uber.org/atomic) - Simple wrappers for primitive types to enforce atomic access.
* [testify](https://github.com/stretchr/testify) - A simple unit test library
* _[and more](go.mod)_

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of
conduct and on the process for submitting pull requests to this repository.

## Versioning

**THIS SOFTWARE IS STILL IN ALPHA AND THERE ARE NO GUARANTEES REGARDING API STABILITY YET.**

All significant (e.g. breaking) changes are documented in the [CHANGELOG.md](CHANGELOG.md).

After the v1.0 release we plan to use [SemVer](http://semver.org/) for versioning.
For the versions available, see the [releases page][releases].

## Authors

- **Friedrich Große** - *Initial work* - [fgrosse](https://github.com/fgrosse)

See also the list of [contributors][contributors] who participated in this project.

## License

This project is licensed under the BSD-3-Clause License - see the [LICENSE](LICENSE) file for details.

[releases]: https://github.com/fgrosse/wal/releases
[contributors]: https://github.com/fgrosse/wal/contributors
[fsync]: https://en.wikipedia.org/wiki/Fsync
