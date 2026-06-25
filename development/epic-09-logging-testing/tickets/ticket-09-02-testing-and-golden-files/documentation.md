# Documentation

The contributor-facing documentation for this feature lives in the Daedalus
manual:

→ [Testing and golden files](../../../../docs/contributing/testing-and-golden-files.md)
— a new chapter in the **Contributing** section.

(Linked from the manual index at [docs/README.md](../../../../docs/README.md), in
the Contributing section, after "Continuous integration".)

This ticket consolidates the project's testing safety net for people working on
Daedalus, so the documentation lives under `docs/contributing/` (CLAUDE.md §6).
The chapter explains how to run the suite (`go test ./...`), what golden files
are and where they live (`internal/compile/testdata/golden/` plus the per-domain
golden tests), the intentional-change workflow with the per-package `-update`
flag (and the caveat that packages without golden tests reject the flag), and the
determinism and git-friendliness guarantee that makes the goldens trustworthy.

This file is intentionally a pointer. Product documentation is maintained as a
single manual under `docs/` (see CLAUDE.md §6).
