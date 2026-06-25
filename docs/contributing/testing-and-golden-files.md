# Testing and golden files

[← Back to the manual index](../README.md)

This page is for working on Daedalus itself. It covers the test suite, the
golden files that pin the expected compilation output, and the workflow for
updating them when a change is intentional.

## Running the tests

Run the whole suite from the repository root:

```sh
go test ./...
```

If your environment has the Make targets available, `make test` runs the same
suite. See [Development environment](development-environment.md) for the targets,
and [Continuous integration](continuous-integration.md) for what CI runs.

The tests live next to the code they cover, in `_test.go` files across the core
packages under `internal/`. They exercise the domain model, schema validation,
and (de)serialization, and they are **hermetic**: each test uses a temporary
directory (`t.TempDir()`) and avoids the network, the clock, and absolute paths,
so the suite runs the same way on Windows, macOS, and Linux.

## What golden files are

A **golden file** is a checked-in copy of the expected output. The test compiles
(or serializes) a known input and compares the result against the golden,
**byte for byte**. The golden is the reviewed, expected output; if a change makes
the real output differ, the test fails. That failure is the safety net — it
catches unintended changes to compiled artifacts before they reach a pull
request.

The compilation goldens fix the expected `build` output under:

```
internal/compile/testdata/golden/.claude/...
```

`TestClaudeGolden` compares the compiled `.claude/` tree against that golden.
In addition, per-domain golden serialization tests live in eight packages —
`architecture`, `backlog`, `catalog`, `compile`, `prompts`, `specs`,
`workflows`, and `workspace` — each pinning the canonical serialized form of its
domain.

## Updating goldens when a change is intentional

When output changes **on purpose** (for example, you deliberately adjust the
compiled layout or a serialized format), regenerate the affected golden with the
`-update` flag, then review the resulting diff:

```sh
go test ./internal/<pkg> -run Golden -update
```

Regenerate per package, then inspect the golden diff as part of your change so a
reviewer can see exactly what the output became. A golden diff in a pull request
should always be intentional and explained.

> Scope `-update` per package. Packages that have no golden tests (such as
> `internal/tui`) do not register the flag, so running it across every package
> at once fails with `flag provided but not defined: -update`. Target the
> specific package — `go test ./internal/<pkg> -run Golden -update` — rather than
> the whole tree.

## Determinism and git-friendliness

Golden comparison only works because compilation is **deterministic**: the same
input always produces byte-identical artifacts, no matter how many times you run
it. Output is also **ordered and stable**, so it diffs cleanly in Git. This is
what makes the goldens trustworthy — a difference always means the input or the
compiler changed, never incidental noise from ordering, timestamps, the network,
or machine-specific paths.
