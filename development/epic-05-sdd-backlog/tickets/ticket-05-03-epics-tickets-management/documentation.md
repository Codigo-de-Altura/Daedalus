# Epics & Tickets Management — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated.
>
> **Canonical guide:** the full chapter lives in the user manual at
> [`docs/guide/managing-epics-and-tickets.md`](../../../../docs/guide/managing-epics-and-tickets.md)
> ([manual index](../../../../docs/README.md)). This file is a pointer and summary; the manual is the growing source of truth.

## Overview

Build the SDD backlog under `.daedalus/epics/`: **epics** (`epic-NN-<slug>`) and
**tickets** (`ticket-NN-MM-<slug>`), each with consistent metadata — status,
priority, dependencies, and origin links to a spec and/or architecture document.
The backlog is a **nested** tree: every ticket lives inside its parent epic's
folder, so a ticket always references its epic, preserving the trace
`ticket → epic → spec / architecture`.

Daedalus manages the **definition** only: it creates the epic/ticket folders,
records their metadata and links, and seeds a placeholder body. It does **not**
run the *planner* or the implementation. You generate the content by running the
*planner* on your own backend, then refine it by hand.

```
.daedalus/epics/
  epic-NN-<slug>/
    epic.md
    tickets/ticket-NN-MM-<slug>/
      ticket.md
```

## How to use

1. **Create the epic** — `daedalus epic create <NN> <slug> --title <t>`
   (optionally `--spec`/`--architecture`, `--status`, `--priority`,
   `--depends-on`). The id `epic-NN-<slug>` is composed from your inputs (no
   auto-numbering).
2. **Create tickets under it** — `daedalus ticket create <epic-id> <MM> <slug> --title <t>`.
   The epic number `NN` is derived from the parent epic id; the parent epic must
   already exist.
3. **Run the *planner* on your backend** to generate epic and ticket content from
   the spec/architecture.
4. **Refine by hand** — drop the result in with `edit` (or by hand). Artifacts are
   yours; Daedalus never overwrites them.

Inspect and manage with `daedalus epic list|show|edit|remove` and
`daedalus ticket list|show|edit|remove` (ticket operations take the parent epic
id).

## Options / flags

`daedalus epic create <NN> <slug>` and `daedalus ticket create <epic-id> <MM> <slug>`
share the same flags:

| Flag | Description |
|---|---|
| `--title <text>` | The artifact's title. **Required.** |
| `--status <status>` | `todo` \| `in-progress` \| `blocked` \| `done`. Default `todo`. |
| `--priority <priority>` | `low` \| `medium` \| `high` \| `critical`. Default `medium`. |
| `--spec <spec-slug>` | Link to an existing spec by slug (optional; must exist). |
| `--architecture <arch-slug>` | Link to an existing architecture doc by slug (optional; must exist). |
| `--depends-on <id,...>` | Comma-separated dependency ids (optional; deduplicated). |
| `--body <text>` / `--body-file <path>` | Set the body inline / from a file (`--body-file` wins). |
| `--path <dir>` | Target repo whose `.daedalus/epics/` is used. Defaults to `.`. |
| `--preview` | Dry run: show the file that would be created without writing. |

`edit` (epic: `<epic-id>`; ticket: `<epic-id> <ticket-id>`) takes the same
metadata flags (at least one required); an empty `--spec`/`--architecture`/`--depends-on`
clears that field.

## Notes & limitations

- **Phase 1 — Daedalus does not run the agent.** It configures the backlog
  (folders, metadata, links, placeholder bodies); generating the content is done
  by running the *planner* on your backend, and the implementation happens
  outside Daedalus. Linked artifacts carry `generated: false`.
- **Closed metadata sets.** `status` and `priority` are validated against their
  closed sets; an invalid value is rejected listing the valid ones.
- **Dependencies recorded, not yet verified.** `--depends-on` stores an explicit,
  deduplicated id list; existence/cycle checks are a later feature.
- **Origin links optional but checked.** A non-empty `--spec`/`--architecture`
  must exist; a ticket's parent `epic` link is mandatory.
- **Non-destructive.** Creating never overwrites an existing artifact; `edit` is
  atomic and validated first. Removing an **epic cascades** to its tickets;
  removing a **ticket** leaves the epic and siblings intact. Structural identity
  (epic id, ticket id, parent epic) is not editable.
- **Windows hand-edits:** save workspace files **without a BOM**
  (`Out-File -Encoding utf8NoBOM` or a BOM-free editor); a leading BOM breaks
  frontmatter parsing.

See the full chapter — worked examples, expected output, and error states — in
[`docs/guide/managing-epics-and-tickets.md`](../../../../docs/guide/managing-epics-and-tickets.md).
</content>
