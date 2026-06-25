# Architecture Documents — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated.
>
> **Canonical guide:** the full chapter lives in the user manual at
> [`docs/guide/managing-architecture.md`](../../../../docs/guide/managing-architecture.md)
> ([manual index](../../../../docs/README.md)). This file is a pointer and summary; the manual is the growing source of truth.

## Overview

Manage your **architecture documents** under `.daedalus/architecture/`. An
architecture document is the blueprint that follows a [spec](../ticket-05-01-brief-to-spec/documentation.md)
in the SDD pipeline — high-level structure and decisions, not implementation
recipes. The *architect* agent derives it from a spec; you then refine it.

Daedalus manages the **definition** of this step: it creates the document at a
deterministic path, optionally wires it to its originating spec (the
`spec → architecture` trace), and seeds a placeholder body. It does **not** run
the agent. You generate the content by running the *architect* on your own
backend, then refine it by hand.

A document lives at `.daedalus/architecture/<slug>.md` (kebab-case slug),
diff-friendly Markdown with stable, ordered frontmatter. The spec link is
**optional**: a linked document records `spec: <slug>.md`, `agent: architect`,
`workflow: sdd-default`, `phase: architecture`, `generated: false`; an unlinked
document carries none of those provenance keys (identity only).

## How to use

1. **Create the document** — `daedalus architecture create <slug> --title <t>`
   creates it at its canonical path with a seeded placeholder.
2. **Link to its spec (optional)** — add `--spec <spec-slug>` to record the
   `spec → architecture` trace; the referenced spec must already exist in
   `.daedalus/specs/`. You can also link/repoint later with `edit`.
3. **Run the *architect* on your backend** to generate the architecture from the
   spec.
4. **Refine the document** — drop the result in with
   `daedalus architecture edit <slug>` (or by hand). It is yours; Daedalus never
   overwrites it.

Inspect and manage documents with `daedalus architecture list`,
`daedalus architecture show <slug>`, and `daedalus architecture remove <slug>`.

## Options / flags

`daedalus architecture create <slug>`:

| Flag | Description |
|---|---|
| `--title <text>` | The document's title. **Required.** |
| `--spec <spec-slug>` | Link to an existing spec by slug (optional; must exist in `.daedalus/specs/`). |
| `--body <text>` | Set the document body inline. |
| `--body-file <path>` | Set the document body from a file (wins over `--body`). |
| `--path <dir>` | Target repo whose `.daedalus/architecture/` is used. Defaults to `.`. |
| `--preview` | Dry run: show the file that would be created without writing. |

`daedalus architecture edit <slug>` (at least one edit flag required):

| Flag | Description |
|---|---|
| `--title <text>` | Set the document's title. |
| `--spec <spec-slug>` | Attach/repoint the spec link by slug; `--spec=` clears it. |
| `--body <text>` | Set the document's body inline. |
| `--body-file <path>` | Set the document's body from a file (wins over `--body`). |
| `--path <dir>` | Target repo whose `.daedalus/architecture/` is used. Defaults to `.`. |

`list`, `show`, and `remove` take `--path`. `list` shows the linked spec (or `-`)
per document.

## Notes & limitations

- **Phase 1 — Daedalus does not run the agent.** It configures the
  `spec → architecture` step (document, optional spec link, destination);
  generating the content is done by running the *architect* on your backend,
  outside Daedalus. The seeded document says so; a linked document carries
  `generated: false`.
- **Optional spec link.** A linked document records the `spec → architecture`
  trace and its *architect*-step provenance (all-or-nothing block); an unlinked
  document carries only its identity. On `edit`, `--spec=` clears the link.
  **PowerShell:** use the single token `--spec=` to clear (`--spec ""` is
  misread by the shell).
- **Non-destructive.** Creating never overwrites an existing document you have
  refined; `edit` is atomic and validated before writing, leaving the file
  intact on an invalid edit.
- **Deterministic & diff-friendly.** Canonical location
  `.daedalus/architecture/<slug>.md`; ordered frontmatter keys, single trailing
  newline; bodies stored verbatim.

See the full chapter — worked examples, expected output, and error states — in
[`docs/guide/managing-architecture.md`](../../../../docs/guide/managing-architecture.md).
</content>
