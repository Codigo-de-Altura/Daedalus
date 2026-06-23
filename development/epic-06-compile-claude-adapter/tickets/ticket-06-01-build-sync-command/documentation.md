# `daedalus build` / `sync` — Usage Guide

> Authored and maintained by C-3PO, technical writer for Daedalus. This guide is for the **end user** of Daedalus, not for internals.

---

## Overview

_To be completed by C-3PO after implementation._

## How to use

Compile your canonical definition (everything under `.daedalus/`) into the native format of your configured agent backend:

```
daedalus build
```

`sync` is an alias and behaves identically:

```
daedalus sync
```

Run the command from inside a repository that contains a `.daedalus/` workspace. Daedalus reads the target backend from `daedalus.yaml`, validates your canonical definition, and writes the native artifacts for that backend (for Claude Code, under `.claude/`).

## Options / flags

- `--help` — Show command help.
- _Preview / confirmation flags are documented with the diff/preview feature (see the diff-preview guide)._
- _Additional options to be completed by C-3PO after implementation._

## Notes & limitations

- The build is deterministic: the same workspace produces the same output.
- If the workspace is missing or the canonical definition is invalid, the command aborts without writing anything and reports an actionable error.
- If the configured backend has no registered adapter, the command fails without writing.
- **Phase 1: Daedalus configures the AI structure; it does not execute agents.** After building, you run the agents yourself in your chosen backend.
