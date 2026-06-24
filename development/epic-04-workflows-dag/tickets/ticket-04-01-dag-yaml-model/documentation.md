# Ticket 04-01 — DAG Workflow YAML Model

> **Pointer:** the user-facing guide for this feature lives in the manual:
> [`docs/guide/managing-workflows.md`](../../../../docs/guide/managing-workflows.md).
> This file is only a pointer; the chapter is the actual guide.

## Overview

Daedalus models a project's pipeline as a declarative **DAG workflow** in your
workspace's `.daedalus/workflows/` directory, one YAML file per workflow. The
`daedalus workflow` command lets you create, list, show, and remove workflows,
and add, edit, and remove their **phases**. Each workflow has a stable
`kebab-case` name (its file name) and an ordered list of phases, each with the
fixed schema `{ id, agent, inputs, outputs, gate, depends_on }` — where
`depends_on` declares the DAG edges.

## How to use

- `daedalus workflow list` — list persisted workflows (name, phase count).
- `daedalus workflow create <name> [--preview]` — create an empty workflow as
  `.daedalus/workflows/<name>.yaml`. Non-destructive: a duplicate name is
  reported, not overwritten.
- `daedalus workflow show <name>` — print the workflow's file content verbatim.
- `daedalus workflow add-phase <name> --id <id> --agent <a> --gate <g> [flags]`
  — append a phase. The list flags take comma-separated values.
- `daedalus workflow edit-phase <name> --id <id> [flags]` — edit a phase in
  place (use `--new-id` to rename).
- `daedalus workflow remove-phase <name> --id <id>` — remove a phase.
- `daedalus workflow remove <name>` — delete only that workflow's file.

## Options

- `--path <dir>` — target repository directory (defaults to the current one).
- `--preview` — dry run on `create`: show the file that would be written, writing nothing.
- `--id`, `--agent`, `--gate`, `--inputs`, `--outputs`, `--depends-on`,
  `--new-id` — set phase fields on the `*-phase` operations (`id`/`agent`/`gate`
  required on `add-phase`).

See [`docs/guide/managing-workflows.md`](../../../../docs/guide/managing-workflows.md)
for full examples, expected output, the phase schema, and the on-disk format.
