# Installation

[← Back to the manual index](../README.md)

Daedalus ships as a single Go binary. This page covers what you need and how to
build and run it.

## Prerequisites

- [Go](https://go.dev/dl/) 1.23 or newer (the version is declared in `go.mod`).
- Optional, for the containerized workflow: `make`, Docker, and Docker Compose.

## Build from source

From the repository root:

```sh
go build -o daedalus ./cmd/daedalus
```

This produces a single executable named `daedalus` in the current directory. To
compile every package without producing a binary, use `go build ./...`.

If you have `make`, the same build is available as:

```sh
make build
```

## Run

```sh
./daedalus
```

Print the version:

```sh
./daedalus --version
# daedalus 0.1.0-dev
```

The binary exits with code `0` on a clean run.

## Next steps

- Follow the [Quickstart](quickstart.md) to initialize your first workspace.
- Learn the interface and flags in [Command line](../guide/command-line.md).
