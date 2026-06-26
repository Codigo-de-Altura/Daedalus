# Installation

[← Back to the manual index](../README.md)

Daedalus ships as a single, self-contained Go binary. The fastest way to get it
is the **install script**; you can also **download a prebuilt binary** manually
or **build from source**. This page covers each option and how to verify the
result.

> Options A and B install from **GitHub Releases**, so they need a published
> release. If there isn't one yet, [build from source](#option-c-build-from-source).

## Option A: Install script (recommended)

The install script picks the right archive for your platform, verifies its
SHA-256 checksum, extracts the `daedalus` executable, and places it on your
`PATH` — in one command.

### Linux and macOS

```sh
curl -fsSL https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.ps1 | iex
```

By default the script installs the **latest** release. You can pin a version or
change the install directory:

| Setting           | Linux/macOS (env var)                | Windows (parameter)                |
| ----------------- | ------------------------------------ | ---------------------------------- |
| Specific version  | `DAEDALUS_VERSION=v0.1.0`            | `-Version v0.1.0`                  |
| Install directory | `DAEDALUS_INSTALL_DIR=~/.local/bin` | `-BinDir C:\tools\daedalus`        |

```sh
# Linux/macOS — pin a version and install to a custom directory:
DAEDALUS_VERSION=v0.1.0 DAEDALUS_INSTALL_DIR="$HOME/.local/bin" \
  sh -c "$(curl -fsSL https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.sh)"
```

```powershell
# Windows — pass parameters by building the script block first:
& ([scriptblock]::Create((irm https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.ps1))) -Version v0.1.0
```

> On macOS/Linux, if the script falls back to `~/.local/bin`, make sure that
> directory is on your `PATH`. On Windows, open a **new** terminal after
> installing so the updated `PATH` takes effect.

After installing, jump to [Verify the installation](#verify-the-installation).

## Option B: Download a prebuilt binary manually

Prefer to do it by hand? Each
[GitHub Release](https://github.com/Codigo-de-Altura/Daedalus/releases) attaches
prebuilt archives and a `*_checksums.txt` file. Download the archive for your
platform, extract the `daedalus` executable, and put it on your `PATH`.

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

## Option C: Build from source

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
[Option B](#option-b-download-a-prebuilt-binary-manually) if you want to run it
from anywhere.

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
