# Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
- Improve performance of `WAL.Write(…)` by reducing allocations (fgrosse/wal#10)
- Lint library using `golangci-lint` (fgrosse/wal#8)
- Fix bug that causes `WAL.Offset()` to panic when the WAL is empty (fgrosse/wal#7)
- Use `sync/atomic` instead of `go.uber.org/atomic` (fgrosse/wal#6)

## [v0.2.0] - 2023-03-14
- Introduce `EntryRegistry` to improve code readability (fgrosse/wal#3)
- Introduce `github.com/fgrosse/wal/waltest` package (fgrosse/wal#5)
- Refactor `SegmentReader` API for better performance and readability (fgrosse/wal#5)
- Drop direct dependency on `go.uber.org/atomic`

## [v0.1.0] - 2023-03-12
- Initial release

[Unreleased]: https://github.com/fgrosse/wal/compare/v0.2.0...HEAD
[v0.2.0]: https://github.com/fgrosse/wal/releases/tag/v0.2.0
[v0.1.0]: https://github.com/fgrosse/wal/releases/tag/v0.1.0

