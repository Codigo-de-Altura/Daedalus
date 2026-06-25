# `daedalus build` / `sync` — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated.
>
> **Canonical guide:** the full chapter lives in the user manual at
> [`docs/guide/compiling-to-a-backend.md`](../../../../docs/guide/compiling-to-a-backend.md)
> ([manual index](../../../../docs/README.md)). This file is a pointer and summary; the manual is the growing source of truth.

## Overview

`daedalus build` (alias `daedalus sync`) compiles your canonical, backend-agnostic
`.daedalus/` definition into the native format of the backend configured in
`daedalus.yaml`. Run it from inside a repository that contains a `.daedalus/`
workspace. The build is **deterministic** — the same workspace always produces the
same result — and **validate-first**: it checks everything before touching disk,
so a failed build leaves your repository untouched.

## How to use

```sh
daedalus build            # compile to the configured backend
daedalus sync             # identical alias
daedalus build --preview  # dry run: compute the result, write nothing
daedalus build --path ./my-repo
```

## Options / flags

| Flag | Description |
|---|---|
| `--path <dir>` | Repository directory to compile. Defaults to `.`. |
| `--preview` | Dry run: compute the result without writing anything. |
| `--help` | Show command help. |

## Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success — compiled to the configured backend. |
| `2` | Usage error (invalid flag or argument). |
| `3` | Validation error — the canonical definition is invalid; nothing written. |
| `4` | Compilation or write error (e.g. no adapter for the configured backend); nothing written. |

## Notes & limitations

- **Validate-first, all-or-nothing.** If the `.daedalus/` workspace is missing,
  the canonical definition is invalid, or the configured backend has no registered
  adapter, `build` aborts with an actionable error and writes nothing.
- **Deterministic.** The same workspace state always produces the same result.
- **Backend comes from the manifest.** `build` compiles to the backend recorded in
  `daedalus.yaml` (set with `daedalus init --backend`).
- The exact native artifacts produced for Claude Code are documented together with
  the Claude Code adapter, in a forthcoming section of the manual chapter.
- **Phase 1:** Daedalus configures the AI structure; it does not execute agents.

See the full chapter — worked usage, safety behavior, preview, and exit codes —
in [`docs/guide/compiling-to-a-backend.md`](../../../../docs/guide/compiling-to-a-backend.md).
