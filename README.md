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

## Example usage

**TODO**

[embedmd]:# (example_test.go)

## How it works

**TODO**

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
