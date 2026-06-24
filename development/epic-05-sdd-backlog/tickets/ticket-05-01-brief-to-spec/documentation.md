# Brief to Spec — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated.
>
> **Canonical guide:** the full chapter lives in the user manual at
> [`docs/guide/managing-specs.md`](../../../../docs/guide/managing-specs.md)
> ([manual index](../../../../docs/README.md)). This file is a pointer and summary; the manual is the growing source of truth.

## Overview

Capture a project **brief** and seed the **spec/PRD** it produces, both under
`.daedalus/specs/`. A brief is the human-authored entry point of the SDD
pipeline; the *analyst* agent turns it into a spec. Daedalus manages the
**definition** of this step — it persists the brief, wires it to the *analyst*
step of the `sdd-default` workflow, and seeds the spec at a deterministic path —
but it does **not** run the agent. You generate the spec content by running the
*analyst* on your own backend, then refine it by hand.

A captured pair shares one `kebab-case` slug:
`.daedalus/specs/<slug>.brief.md` (the brief) and `.daedalus/specs/<slug>.md`
(the spec). Both are diff-friendly Markdown with stable, ordered frontmatter.

## How to use

1. **Capture the brief** — `daedalus spec capture <slug> --title <t>` persists
   the brief and seeds the spec destination.
2. **Daedalus wires the link** — the brief's frontmatter records
   `consumed-by: analyst`, `workflow: sdd-default`, `phase: spec`; the seeded
   spec records `brief: <slug>.brief.md`, `agent: analyst`, `generated: false`.
3. **Run the *analyst* on your backend** to generate the spec/PRD from the brief.
4. **Refine the spec** — drop the result in with `daedalus spec edit <slug>` (or
   by hand). The spec is yours; Daedalus never overwrites it.

Inspect and manage the pair with `daedalus spec list`,
`daedalus spec show <slug> [--brief]`, and
`daedalus spec remove <slug> [--brief]`.

## Options / flags

`daedalus spec capture <slug>`:

| Flag | Description |
|---|---|
| `--title <text>` | The brief's title. **Required.** |
| `--body <text>` | Set the brief body inline. |
| `--body-file <path>` | Set the brief body from a file (wins over `--body`). |
| `--path <dir>` | Target repo whose `.daedalus/specs/` is used. Defaults to `.`. |
| `--preview` | Dry run: show the files that would be created without writing. |

`daedalus spec edit <slug>` (at least one edit flag required):

| Flag | Description |
|---|---|
| `--title <text>` | Set the spec's title. |
| `--body <text>` | Set the spec's body inline. |
| `--body-file <path>` | Set the spec's body from a file (wins over `--body`). |
| `--path <dir>` | Target repo whose `.daedalus/specs/` is used. Defaults to `.`. |

`list`, `show`, and `remove` take `--path`; `show` and `remove` also take
`--brief` to target the brief instead of the spec.

## Notes & limitations

- **Phase 1 — Daedalus does not run the agent.** It configures the
  `brief → spec/PRD` step (brief, link, spec destination); generating the spec
  content is done by running the *analyst* on your backend, outside Daedalus.
  The seeded spec says so and carries `generated: false`.
- **Non-destructive.** Re-capturing never overwrites an existing brief or a spec
  you have refined; when only one half of the pair is missing, Daedalus
  re-creates just that file. Use `--preview` to see what a capture would do.
- **Brief vs. spec edits.** `daedalus spec edit` changes only the spec
  (atomic write, validated first, provenance preserved). The brief is your
  authored input and is not edited via the CLI.
- **Deterministic & diff-friendly.** Canonical location `.daedalus/specs/<slug>.md`;
  ordered frontmatter keys, every key present, single trailing newline; bodies
  stored verbatim.

See the full chapter — worked examples, expected output, and error states — in
[`docs/guide/managing-specs.md`](../../../../docs/guide/managing-specs.md).
</content>
