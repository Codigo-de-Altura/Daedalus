# Ticket 03-01 — Global & Shared Prompts

> **Pointer:** the user-facing guide for this feature lives in the manual:
> [`docs/guide/managing-prompts.md`](../../../../docs/guide/managing-prompts.md).
> This file is only a pointer; the chapter is the actual guide.

## Overview

Daedalus manages reusable **global** and **shared** prompts in your workspace's
`.daedalus/prompts/` directory, one Markdown file per prompt. The
`daedalus prompt` command lets you create, list, show, edit, and remove them.
Each prompt has a stable `kebab-case` id and minimal metadata (id, kind, title,
optional description) in YAML frontmatter, followed by a verbatim Markdown body.

## How to use

- `daedalus prompt list [--kind global|shared]` — list persisted prompts (id,
  kind, title), optionally filtered by kind.
- `daedalus prompt create <id> --kind <global|shared> --title <t> [flags]` —
  create a prompt as `.daedalus/prompts/<id>.md`. Non-destructive: a duplicate id
  is reported, not overwritten.
- `daedalus prompt show <id>` — print the prompt's file content verbatim.
- `daedalus prompt edit <id> [flags]` — edit a prompt's title, description, or
  body in place. At least one edit flag is required.
- `daedalus prompt remove <id>` — delete only that prompt's file.

## Options

- `--path <dir>` — target repository directory (defaults to the current one).
- `--kind <global|shared>` — prompt kind (required on `create`, a filter on `list`).
- `--title`, `--description`, `--body`, `--body-file` — set prompt metadata and body.
- `--preview` — dry run on `create`: show the file that would be written, writing nothing.

See [`docs/guide/managing-prompts.md`](../../../../docs/guide/managing-prompts.md)
for full examples, expected output, and the on-disk format.
