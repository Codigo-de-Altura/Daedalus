# Ticket 02-03 — Import Agents from Local Files

> **Pointer:** the user-facing guide for this feature lives in the manual,
> in the "Managing agents" chapter:
> [Importing agents](../../../../docs/guide/managing-agents.md#importing-agents).
> This file is only a pointer; the chapter is the actual guide.

## Overview

`daedalus agent import <source>` reads agent definitions from a local file or
directory and converts them into the workspace's canonical format under
`.daedalus/agents/`. It recognizes Claude Code agents (`.claude/agents/*.md`,
frontmatter + body) and already-canonical definitions, so you can adopt existing
agents without rewriting them by hand.

## How to use

- `daedalus agent import <source>` — import one agent (a file) or every valid
  agent in a directory. Ids are normalized to `kebab-case`. Non-destructive: an
  id that already exists is reported and skipped, never overwritten.

When importing a Claude Code agent: `name` becomes the id, `description` the
role, the Markdown body the prompt, and `model` a parameter. The backend-specific
`tools` and `color` fields are dropped (no canonical meaning in Phase 1).

## Options

- `--path <dir>` — target workspace directory (defaults to the current one).
- `--preview` — dry run: show what would be imported, writing nothing.

Exit codes: `0` when every source imported or was skipped; `2` when any source
fails to parse/validate (other valid sources are still imported); `1` when the
source path itself cannot be read.

See [`docs/guide/managing-agents.md`](../../../../docs/guide/managing-agents.md)
for full examples, the Claude Code mapping, and limitations.
