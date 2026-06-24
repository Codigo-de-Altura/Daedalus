# Building and running Daedalus

Daedalus ships as a single Go binary. This guide covers building it from source
and running it.

## Prerequisites

- [Go](https://go.dev/dl/) 1.23 or newer (the version is declared in `go.mod`).

## Build

From the repository root:

```sh
go build -o daedalus ./cmd/daedalus
```

This compiles the `daedalus` executable into the current directory. To compile
every package without producing a binary, use `go build ./...`.

## Run

```sh
./daedalus
```

Print version information and exit:

```sh
./daedalus --version
# daedalus 0.1.0-dev
```

Show usage:

```sh
./daedalus --help
```

The binary exits with code `0` on a clean run.

## Repository layout

| Path             | Purpose                                                        |
| ---------------- | -------------------------------------------------------------- |
| `cmd/daedalus/`  | Binary entry point (`package main`).                           |
| `internal/`      | Core packages, not importable from outside this module.        |
| `README.md`      | Project overview, build and run instructions.                  |

The compiled `daedalus` binary is ignored by Git and should not be committed.
