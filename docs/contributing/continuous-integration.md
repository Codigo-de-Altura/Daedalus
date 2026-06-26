# Continuous integration

[← Back to the manual index](../README.md)

Daedalus uses three GitHub Actions workflows: one keeps the repository building,
tested, and clean on every change; one publishes the documentation site; and one
publishes release binaries.

## The CI workflow

Every push to `main` and every pull request runs the CI workflow
(`.github/workflows/ci.yml`), which runs three stages on each change:

1. **Build** — `go build ./...`
2. **Test** — `go test ./...`
3. **Lint** — a `gofmt` formatting check followed by `go vet ./...`

The pinned Go version in CI matches the `go` directive in `go.mod`, so CI and
local builds stay reproducible. If any stage fails, the check fails and acts as
a quality gate.

### Reproduce the checks locally

Run exactly what CI runs before opening a pull request:

```sh
go build ./...
go test ./...
make lint
```

If all three succeed locally, the CI gate will pass. See
[Development environment](development-environment.md) for the Make targets.

## The docs workflow

The docs workflow (`.github/workflows/docs.yml`) builds the documentation site
from `docs/` with **MkDocs Material** and deploys it to **GitHub Pages**. It runs
when `docs/` or `mkdocs.yml` changes on `main`. The build is run with
`mkdocs build --strict`, so a broken internal link or an orphaned page **fails**
the build — keeping the published manual consistent.

### Preview the site locally

Install the pinned tooling (`docs-requirements.txt` pins
`mkdocs-material==9.5.39`) and serve or build the site:

```sh
pip install -r docs-requirements.txt
mkdocs serve            # live preview at http://127.0.0.1:8000
mkdocs build --strict   # exactly what CI runs
```

If `mkdocs build --strict` succeeds locally, the docs gate will pass.

## The release workflow

The release workflow (`.github/workflows/release.yml`) runs **GoReleaser**
(pinned to `v2.16.0`) when you push a tag of the form `vX.Y.Z`. Driven by
`.goreleaser.yaml`, it cross-compiles the six targets — `windows`, `darwin`, and
`linux`, each for `amd64` and `arm64` — packages them as archives with a
checksums file, injects the version through `ldflags`, and publishes everything
as a **GitHub Release**. Those archives (and their checksums file) are what the
install scripts download for end users; see
[Installation](../getting-started/installation.md).

### Verify the release locally

You can exercise the release pipeline without publishing anything:

```sh
goreleaser check                          # validate .goreleaser.yaml
goreleaser release --snapshot --clean     # full local build, no publish
```
