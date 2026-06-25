# Diff / preview before writing — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated.
>
> **Canonical guide:** the full chapter lives in the user manual at
> [`docs/guide/compiling-to-a-backend.md`](../../../../docs/guide/compiling-to-a-backend.md)
> — see [Previewing and confirming changes](../../../../docs/guide/compiling-to-a-backend.md#previewing-and-confirming-changes)
> ([manual index](../../../../docs/README.md)). This file is a pointer and summary; the manual is the growing source of truth.

## Overview

`daedalus build` never writes silently. In an interactive terminal it opens a
**preview** of every change and waits for your explicit confirmation before
touching disk. You see, per backend, the counts (new / modified / unchanged) and
any orphans; the artifacts classified as **new / modified / unchanged**; and, for
the selected modified artifact, the content **diff** (`+` added / `-` removed
lines). Confirm to write, cancel and nothing changes.

## Controls

- **↑ / ↓** — move between artifacts.
- **pgup / pgdn**, **g / G** — scroll the selected artifact's diff.
- **y** / **enter** — confirm and write.
- **n** / **esc** — cancel; nothing is written.
- **ctrl+c** — exit any time.

Orphans appear in a separate read-only section (“left untouched · not
selectable”) — reported, never deleted. When there is nothing to write, the
screen says *“No changes — every artifact is already up to date.”*

## Modes (flags)

| Invocation | Behavior |
|---|---|
| `daedalus build` (in a terminal) | Opens the preview and writes **only after you confirm**. |
| `daedalus build --preview` | Shows the diff/preview and **exits without writing** (read-only in a terminal; textual otherwise). |
| `daedalus build --yes` | Writes **without** the gate — for scripts / CI / non-interactive use. |
| `daedalus build` (no terminal, no `--yes`) | Prints the plan/diff and **writes nothing**: *“Nothing written; pass --yes to write, or run in a terminal to confirm.”* |

> Automating a build? A plain `build` without a terminal writes nothing by design;
> pass `--yes` to write from a script or CI. (`--preview` always wins over `--yes`.)

## Notes & limitations

- **Nothing is written without confirmation** (or `--yes`). Cancelling leaves the
  repository exactly as it was.
- The preview covers Daedalus's **managed area**; your manual files outside it are
  preserved and are not shown as changes.
- **Phase 1:** Daedalus configures the AI structure; it does not execute agents —
  the preview shows configuration changes, not agent runs.

See the full chapter — with the preview screen, the confirmation gate, and every
mode — in
[`docs/guide/compiling-to-a-backend.md`](../../../../docs/guide/compiling-to-a-backend.md#previewing-and-confirming-changes).
