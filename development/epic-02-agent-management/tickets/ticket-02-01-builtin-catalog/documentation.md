# Ticket 02-01 — Built-in Agent Catalog

> **Pointer:** the user-facing guide for this feature lives in the manual:
> [`docs/guide/managing-agents.md`](../../../../docs/guide/managing-agents.md).
> This file is only a pointer; the chapter is the actual guide.

## Overview

Daedalus ships with a built-in catalog of five canonical SDD agents (`analyst`,
`architect`, `planner`, `validator`, `documenter`), embedded in the binary. The
`daedalus agent` command lets you list those agents and materialize any of them
into your workspace as an editable, canonical definition.

## How to use

- `daedalus agent list` — list the built-in agents (id and role), ordered by id.
- `daedalus agent add <id>` — materialize an agent into
  `.daedalus/agents/<id>/` as `agent.yaml` + `prompt.md`. The add is
  non-destructive: an existing agent is never overwritten.

## Options

- `--path <dir>` — target repository directory (defaults to the current one).
- `--preview` — dry run: show the files that would be created, writing nothing.

See [`docs/guide/managing-agents.md`](../../../../docs/guide/managing-agents.md)
for full examples, expected output, and the on-disk layout.
