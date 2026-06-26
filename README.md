# Daedalus

Daedalus is a lightweight **TUI/CLI written in Go (with the [Charm](https://charm.sh) stack)** that automates the setup and management of a project's AI scaffolding — agents, prompts, DAG workflows, and an SDD backlog — in a **backend-agnostic** way, and **compiles** it to the native format of the tool you use (the first adapter targets Claude Code → `.claude/`).

Instead of re-creating global prompts, agents, workflows, and a backlog by hand in every repository, you point Daedalus at a repo, describe a brief, and get a versioned, ready-to-use SDD ecosystem. Daedalus does **not** execute agents — that stays with your runtime (e.g. Claude Code).

> This repository is in its foundations stage (epic-00). The binary currently launches a minimal Bubble Tea skeleton; product features arrive in later epics.

## Install

The install script downloads the right prebuilt binary for your platform from
GitHub Releases, verifies its checksum, and puts `daedalus` on your `PATH`:

```sh
# Linux and macOS
curl -fsSL https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.ps1 | iex
```

See the [installation guide](docs/getting-started/installation.md) for pinning a
version, choosing an install directory, manual downloads, and building from
source.

## Requirements

- [Go](https://go.dev/dl/) 1.23 or newer (matches the `go` directive in `go.mod`).
- Optional: `make`, Docker, and Docker Compose for the containerized dev workflow.

## Getting started

From a clean clone, a single command prepares the environment (downloads dependencies and builds the binary):

```sh
make setup
```

This is the onboarding command for new developers. It is equivalent to running `go mod download` followed by a build.

## Build

```sh
make build          # or: go build -o daedalus ./cmd/daedalus
```

The build produces a single executable named `daedalus`.

## Run

```sh
make run            # or: ./daedalus
```

Run it in an interactive terminal to launch the TUI; press `q` (or `Ctrl+C`) to quit cleanly. In a non-interactive context (piped input, CI) the binary prints a short notice and exits with code 0. Print the version with:

```sh
./daedalus --version
```

## Common tasks

| Command       | What it does                                  |
| ------------- | --------------------------------------------- |
| `make setup`  | One-command onboarding from a clean clone     |
| `make build`  | Compile the `daedalus` binary                 |
| `make test`   | Run the test suite                            |
| `make lint`   | Check formatting (`gofmt`) and run `go vet`   |
| `make run`    | Build and run the binary                      |
| `make fmt`    | Format the codebase in place                  |
| `make tidy`   | Tidy module dependencies                      |

## Docker

Build and run Daedalus in a container, or bring up the development service with Compose:

```sh
docker build -t daedalus:dev .
docker run --rm -it daedalus:dev

docker compose up --build
```

## Documentation

The full user manual lives in [`docs/`](docs/README.md) — installation,
quickstart, the command-line guide, configuration, and contributing notes. It is
organized as a manual with an index and grows alongside each feature.

The public website (marketing landing page + the manual) is a single-page app in
[`web/`](web/README.md), built with Vite + React and published to GitHub Pages.
It renders the same markdown from `docs/`, so that content stays the single
source of truth.

## Configuration

| Environment variable | Values                         | Default | Purpose                       |
| -------------------- | ------------------------------ | ------- | ----------------------------- |
| `DAEDALUS_LOG_LEVEL` | `debug`, `info`, `warn`, `error` | `info`  | Minimum structured-log level. |

Logs are structured (JSON) and written to `stderr`, so they never interfere with the TUI render on `stdout`.

## Project layout

```
cmd/daedalus/    Binary entry point (package main)
internal/        Core packages (not importable outside this module)
  buildinfo/     Static binary identification (name, version)
  logging/       Structured logging baseline (log/slog)
  tui/           Bubble Tea bootstrap skeleton
.github/workflows/  Continuous integration (build, test, lint)
docs/            User manual (index + chapters), maintained alongside features
scripts/         Install scripts (install.sh, install.ps1) for prebuilt binaries
web/             Marketing + docs website (Vite + React SPA) for GitHub Pages
development/     SDD planning artifacts (epics, tickets) — not shipped in the binary
```

## License

See repository for license details.
