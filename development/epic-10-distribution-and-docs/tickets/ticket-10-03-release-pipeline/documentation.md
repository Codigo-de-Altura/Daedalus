# Documentation pointer — Ticket 10-03

This ticket publishes the Daedalus binaries as downloadable releases. This file
is only a **pointer**; the user-facing instructions live in the manual.

## What this ticket delivers

Release binaries are published to **GitHub Releases** with **GoReleaser**:

- `.goreleaser.yaml` — the release configuration (build targets, archives,
  checksums, version injected through `ldflags`).
- `.github/workflows/release.yml` — runs GoReleaser when a tag of the form
  `vX.Y.Z` is pushed, cross-compiling the six targets (`windows`, `darwin`,
  `linux` × `amd64`, `arm64`) and publishing the archives and checksums as a
  GitHub Release.

## Where to read more

- For the **end user** — how to download and install a prebuilt binary:
  [`docs/getting-started/installation.md`](../../../../docs/getting-started/installation.md),
  section "Option A: Download a prebuilt binary".
- For the **CI detail** of the release pipeline (and how to verify it locally
  with `goreleaser check` / `goreleaser release --snapshot --clean`):
  [`docs/contributing/continuous-integration.md`](../../../../docs/contributing/continuous-integration.md).
