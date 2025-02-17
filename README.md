# `gtar` Testing TARs

[![PkgGoDev](https://img.shields.io/badge/-reference-blue?logo=go&logoColor=white&labelColor=505050)](https://pkg.go.dev/github.com/thediveo/gtar)
[![GitHub](https://img.shields.io/github/license/thediveo/gtar)](https://img.shields.io/github/license/thediveo/gtar)
![build and test](https://github.com/thediveo/gtar/actions/workflows/buildandtest.yaml/badge.svg?branch=master)
![Coverage](https://img.shields.io/badge/Coverage-84.6%25-brightgreen)
[![Go Report Card](https://goreportcard.com/badge/github.com/thediveo/gtar)](https://goreportcard.com/report/github.com/thediveo/gtar)

`gtar` (pronounced "gee-tahr") is a small Go module to primarily help with
examining tape archive "TAR" files in tests. `gtar` works only with tar _files_,
as opposed to tar (_reader_) _streams_; the reason is that `gtar` first scans
the complete TAR file to build an index of the available files and directories,
and then test code can freely access the individual files inside the tape archive.

`gtar` requires Go 1.23 or later due Go iterator support.

## Restrictions

- TAR must be a file, and cannot be a stream (no `io.Writer` support) – the
  reason is that we need to seek within the file in order to read individual
  regular files.
- only files and directories are indexed, everything else is skipped over.
- no `fs.FS` support.
- no directory contents read support.

## Usage

```go
import "github.com/thediveo/gtar"

func main() {
  // error handling omitted
  tarf, _ := os.Open("my.tar")
  index, _ := gtar.New(tarf)
  for path := range index.AllRegularFilePaths() {
    println(path)
  }
}
```

## Tinkering

When tinkering with the `gtar` source code base, the recommended way is the
devcontainer environment. The devcontainer specified in this repository
contains:
- `gocover` command to run all tests with coverage, updating the README coverage
  badge automatically after successful runs.
- Go package documentation is served in the background on port TCP/HTTP `6060`
  of the devcontainer.
- [`go-mod-upgrade`](https://github.com/oligot/go-mod-upgrade) for interactive
  direct dependency upgrading in the terminal.
- [`goreportcard-cli`](https://github.com/gojp/goreportcard) – a report card
  for this Go module.
- [`pin-github-action`](https://github.com/mheap/pin-github-action) for
  maintaining Github Actions.

## Copyright and License

Copyright 2025 Harald Albrecht, licensed under the Apache License, Version 2.0.
