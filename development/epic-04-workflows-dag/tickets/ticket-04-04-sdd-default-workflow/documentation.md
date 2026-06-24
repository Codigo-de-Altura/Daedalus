# Ticket 04-04 — Factory `sdd-default` Workflow

> **Pointer:** the user-facing guide for this feature lives in the manual, as a
> section of the workflows chapter:
> [`docs/guide/managing-workflows.md` → The default SDD workflow](../../../../docs/guide/managing-workflows.md#the-default-sdd-workflow).
> This file is only a pointer; the chapter is the actual guide.

## Overview

`daedalus init` seeds a ready-to-use factory workflow, `sdd-default.yaml`, into
`.daedalus/workflows/`, so a new workspace starts with the default SDD pipeline
instead of an empty directory. It is a complete, valid DAG you can use as-is,
adapt, or read as a worked example of the schema.

## How to use

The workflow is created for you by `daedalus init`; you do not author it. It is a
linear chain of six phases, each run by a built-in agent:

```
brief → spec → architecture → epics → tickets → ⟨external implementation⟩ → validation → docs
```

| Phase | Agent | Consumes | Produces | Gate |
|---|---|---|---|---|
| `spec` | `analyst` | `brief` | `spec` | `spec-gate` |
| `architecture` | `architect` | `spec` | `architecture` | `architecture-gate` |
| `epics` | `planner` | `architecture` | `epics` | `epics-gate` |
| `tickets` | `planner` | `epics` | `tickets` | `tickets-gate` |
| `validation` | `validator` | `tickets` | `validation` | `validation-gate` |
| `docs` | `documenter` | `validation` | `docs` | `docs-gate` |

- View it with `daedalus workflow show sdd-default`.
- Validate it with `daedalus workflow validate sdd-default` (it is valid out of
  the box: exit 0).

## Notes & limitations

- **External implementation step.** There is no implementation/developer phase
  between `tickets` and `validation`. The implementation is performed externally
  (a developer or a backend agent), so it is the un-modeled gap on that edge:
  Daedalus hands off the tickets, an external actor implements, and `validation`
  resumes from the same tickets. This is why `validation` consumes `[tickets]`,
  not an `implementation` artifact.
- **Non-destructive seeding.** If you have already created or edited
  `sdd-default.yaml`, re-running `init` leaves your file untouched and reports it
  as already present. In `--preview`, a fresh init lists the factory workflow as
  a change but writes nothing.
- Phase 1 configures the AI structure; it does not execute workflows.

See [`docs/guide/managing-workflows.md` → The default SDD workflow](../../../../docs/guide/managing-workflows.md#the-default-sdd-workflow)
for the full seeded YAML, the external-implementation explanation, and the init
output.
