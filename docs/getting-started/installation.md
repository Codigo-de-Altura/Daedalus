# Installation

[← Back to the manual index](../README.md)

Daedalus ships as a single, self-contained Go binary. You can either **download
a prebuilt binary** or **build it from source**. This page covers both, per
platform, and how to verify the result.

## Option A: Download a prebuilt binary (recommended)

Prebuilt binaries are published on **GitHub Releases**. Download the archive for
your platform, extract the `daedalus` executable, and put it somewhere on your
`PATH`.

> The signed release archives and the exact download URL/tag are produced by the
> release pipeline. Until that ships, use [build from source](#option-b-build-from-source).

### Linux and macOS

```sh
# Extract the downloaded archive, then move the binary onto your PATH:
tar -xzf daedalus_*.tar.gz
sudo mv daedalus /usr/local/bin/
```

If `/usr/local/bin` is not writable, move it to any directory on your `PATH`
(for example `~/.local/bin`).

### Windows

Extract the downloaded `.zip`, then place `daedalus.exe` in a directory that is
on your `PATH` (for example a `bin` folder you add to the `Path` environment
variable). You can then run `daedalus` from any terminal.

After installing, jump to [Verify the installation](#verify-the-installation).

## Option B: Build from source

Building from source works on every platform Go supports.

### Prerequisites

- [Go](https://go.dev/dl/) 1.23 or newer (the version is declared in `go.mod`).
- A clone of the Daedalus repository.
- Optional, for the containerized contributor workflow: `make`, Docker, and
  Docker Compose.

### Build

From the repository root:

```sh
go build -o daedalus ./cmd/daedalus
```

This produces a single executable named `daedalus` (or `daedalus.exe` on
Windows) in the current directory. To compile every package without producing a
binary, use `go build ./...`.

If you have `make`, the same build is available as:

```sh
make build
```

Move the resulting binary onto your `PATH` as shown in
[Option A](#linux-and-macos) if you want to run it from anywhere.

## Verify the installation

Print the version:

```sh
daedalus --version
# daedalus 0.1.0-dev
```

If you see the version line and the process exits with code `0`, Daedalus is
installed correctly.

> If you have not put the binary on your `PATH`, run it from its directory with
> `./daedalus --version` (or `.\daedalus.exe --version` on Windows).

## Next steps

- Follow the [Quickstart](quickstart.md) to go from zero to a compiled
  workspace.
- Learn the ideas behind the workspace in [Concepts](../guide/concepts.md).
- See every command in the [Command reference](../guide/command-reference.md).
