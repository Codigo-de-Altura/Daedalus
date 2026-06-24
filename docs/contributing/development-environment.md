# Development environment

[← Back to the manual index](../README.md)

This page is for working on Daedalus itself. It covers the Make targets, the
Docker image, and the Compose service that standardize build, test, and run.

## One-command onboarding

From a clean clone, prepare your environment with a single command:

```sh
make setup
```

This downloads the module dependencies and builds the binary, leaving you ready
to develop, build, and test.

## Make targets

| Command       | What it does                                  |
| ------------- | --------------------------------------------- |
| `make setup`  | One-command onboarding from a clean clone.    |
| `make build`  | Compile the `daedalus` binary.                |
| `make test`   | Run the test suite.                           |
| `make lint`   | Check formatting (`gofmt`) and run `go vet`.  |
| `make run`    | Build and run the binary.                     |
| `make fmt`    | Format the codebase in place.                 |
| `make tidy`   | Tidy module dependencies.                     |
| `make help`   | List the available targets.                   |

## Docker

Build an image and run Daedalus in a container:

```sh
docker build -t daedalus:dev .
docker run --rm -it daedalus:dev
```

The image is multi-stage: it compiles the binary with the pinned Go toolchain
and runs it from a minimal Alpine image as a non-root user.

## Docker Compose

Bring up the development service:

```sh
docker compose up --build
```

The `daedalus` service builds from the local `Dockerfile` and runs with an
interactive terminal attached.

## Cross-platform notes

These tools work on Windows, macOS, and Linux. On Windows, run the `make`
commands from a Unix-like shell (such as Git Bash or WSL), or call the
underlying `go` commands directly.
