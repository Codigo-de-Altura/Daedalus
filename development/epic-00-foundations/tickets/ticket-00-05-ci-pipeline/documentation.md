# Continuous integration

Every push to `main` and every pull request runs an automated check that keeps
the repository building, tested, and clean.

## What runs

The CI workflow (`.github/workflows/ci.yml`) runs three stages on each change:

1. **Build** — `go build ./...`
2. **Test** — `go test ./...`
3. **Lint** — a `gofmt` formatting check followed by `go vet ./...`

The pinned Go version in CI matches the `go` directive in `go.mod`, so CI and
local builds stay reproducible. If any stage fails, the check fails and acts as
a quality gate.

## Reproduce the checks locally

You can run exactly what CI runs before opening a pull request:

```sh
go build ./...
go test ./...
make lint
```

If all three succeed locally, the CI gate will pass.
