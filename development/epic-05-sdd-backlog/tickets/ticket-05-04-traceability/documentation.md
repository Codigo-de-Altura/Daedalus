# Traceability (spec → epic → ticket) — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated.
>
> **Canonical guide:** the full chapter lives in the user manual at
> [`docs/guide/tracing-the-backlog.md`](../../../../docs/guide/tracing-the-backlog.md)
> ([manual index](../../../../docs/README.md)). This file is a pointer and summary; the manual is the growing source of truth.

## Overview

The `daedalus trace` command ties the SDD backlog together: it **verifies** that
the spec → epic → ticket chain is consistent and lets you **navigate** it in both
directions. It is **read-only** — it never runs an agent and never writes
anything, and it keeps **no index of its own**: it reads the origin and parent
links already recorded in each artifact's frontmatter (the single source of
truth), so editing those links by hand is reflected on the next run. Same
workspace state → same report (deterministic, findings worst-first).

## How to use

1. **Verify** the chain — `daedalus trace verify` checks every recorded link
   resolves and every ticket has a parent epic; reports inconsistencies and sets
   its exit code.
2. **Navigate** the chain — `daedalus trace show <artifact-id>` walks the chain,
   inferring direction from the id shape:
   - a **spec slug** descends (spec → epics → tickets);
   - a **ticket id** (`ticket-NN-MM-...`) ascends (ticket → epic → origin
     spec/architecture; a ticket with no own origin inherits the epic's);
   - an **epic id** (`epic-NN-...`) shows **both** directions.

## Inconsistency types

| Kind | Severity | Meaning | Affects exit? |
|---|---|---|---|
| `broken-link` | error | An epic/ticket/architecture references an origin (spec or architecture) that does not exist. | Yes (exit 1) |
| `orphan-ticket` | error | A ticket's parent epic no longer exists. | Yes (exit 1) |
| `missing-origin` | warning | An epic (or architecture doc) records no origin link at all — a legal gap, since the origin link is optional. A ticket with no own origin is not reported (it inherits the epic's). | No (exit 0) |

`verify` exit codes: **0** consistent or warnings-only; **1** at least one hard
error; **2** usage/load error.

## Options / flags

| Command | Flag | Description |
|---|---|---|
| `trace verify` | `--path <dir>` | Repo whose `.daedalus/` chain is verified. Defaults to `.`. |
| `trace show <artifact-id>` | `--path <dir>` | Repo whose `.daedalus/` chain is navigated. Defaults to `.`. |

## Notes & limitations

- **Read-only, no agents, no index.** `trace` only reads artifact frontmatter
  and reports/navigates the chain; it never writes or runs anything, and it does
  not duplicate the links into a separate store.
- **Deterministic.** Same workspace state yields the same report, with findings
  in a stable worst-first order.
- **Severity split is intentional.** A *missing* origin is a warning (optional
  link, never fails `verify`); a *dangling* origin or a missing parent epic is a
  hard error.
- **Fixing links.** Inconsistencies are fixed by editing the source artifact
  (e.g. `daedalus epic edit` / `daedalus ticket edit`, or by hand). To clear a
  link use the single-token `--spec=` / `--architecture=` (PowerShell misreads
  `--spec ""` as two tokens). On Windows, save hand-edited files **without a
  BOM** — a leading BOM breaks frontmatter parsing.

See the full chapter — worked examples, expected tree output, and error states —
in [`docs/guide/tracing-the-backlog.md`](../../../../docs/guide/tracing-the-backlog.md).
</content>
