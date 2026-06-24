# Ticket 04-03 — DAG Semantic Validation

> **Pointer:** the user-facing guide for this feature lives in the manual, as a
> section of the workflows chapter:
> [`docs/guide/managing-workflows.md` → Validating a workflow](../../../../docs/guide/managing-workflows.md#validating-a-workflow).
> This file is only a pointer; the chapter is the actual guide.

## Overview

The `daedalus workflow validate <name>` subcommand checks a workflow's **DAG
semantics** — that the graph as a whole is coherent and runnable — and reports
problems in an actionable, one-finding-per-line form. It complements the
structural checks the editing commands already perform.

## How to use

- `daedalus workflow validate <name>` — validate the named workflow's graph.

It detects:

- **Cycles** in the `depends_on` dependencies (names the phases in the loop).
- **Missing artifacts**: a phase consumes an input that is neither the initial
  `brief` nor an output of a transitive predecessor (names phase and artifact).
- **Unknown agents**: a phase references an agent that does not exist in the
  workspace (names phase and agent). A **built-in** agent (`analyst`,
  `architect`, `planner`, `validator`, `documenter`) referenced but not yet
  materialized is valid — not an error.
- **Unknown dependencies**: a `depends_on` entry that names no existing phase
  (typically a typo).

A valid workflow reports "semantically valid" with no findings.

## Options

- `--path <dir>` — target repository directory (defaults to the current one).

## Exit codes

- `0` — the workflow is semantically valid.
- `1` — the workflow is semantically invalid (one or more findings).
- `2` — a usage or load error (for example, the workflow does not exist).

See [`docs/guide/managing-workflows.md` → Validating a workflow](../../../../docs/guide/managing-workflows.md#validating-a-workflow)
for full examples, the exact output of each problem class, and the built-in-agent
note.
