# Ticket 02-02 — Clone & Edit an Agent

> **Pointer:** the user-facing guide for this feature lives in the manual,
> in the "Managing agents" chapter:
> [Cloning an agent](../../../../docs/guide/managing-agents.md#cloning-an-agent)
> and [Editing an agent](../../../../docs/guide/managing-agents.md#editing-an-agent).
> This file is only a pointer; the chapter is the actual guide.

## Overview

Building on the built-in catalog (ticket-02-01), you can clone a catalog agent to
a new id and then edit its canonical definition (role, prompt, parameters). A
clone is an independent copy: editing it never changes the original built-in
agent.

## How to use

- `daedalus agent clone <source-id> <dest-id>` — copy a built-in agent to a new
  `kebab-case` id under `.daedalus/agents/<dest-id>/`. Non-destructive: an
  existing dest id is never overwritten.
- `daedalus agent edit <id>` — edit a workspace agent's `agent.yaml` / `prompt.md`
  in place. The edit is validated before writing and the write is atomic, so an
  invalid edit leaves the existing definition intact.

## Options

- clone: `--path <dir>` (target directory, default `.`), `--preview` (dry run).
- edit: `--path <dir>`, `--role <text>`, `--prompt <text>`,
  `--prompt-file <path>` (takes precedence over `--prompt`),
  `--set-param key=value` (repeatable), `--remove-param key` (repeatable). At
  least one edit flag is required.

See [`docs/guide/managing-agents.md`](../../../../docs/guide/managing-agents.md)
for full examples, expected output, and limitations.
