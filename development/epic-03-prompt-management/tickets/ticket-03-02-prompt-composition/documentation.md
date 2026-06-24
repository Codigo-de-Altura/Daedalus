# Ticket 03-02 — Prompt Composition

> **Pointer:** the user-facing guide for this feature lives in the manual,
> under the "Composing prompts (includes)" section of
> [`docs/guide/managing-prompts.md`](../../../../docs/guide/managing-prompts.md#composing-prompts-includes).
> This file is only a pointer; the chapter is the actual guide.

## Overview

A prompt can include the content of another prompt instead of copying it, so a
shared fragment (glossary, style, role definition) lives in a single file and is
reused **by reference** (DRY). `daedalus prompt render <id>` prints the composed
prompt with every inclusion resolved; the source files are never modified.

## How to use

- Add an include directive on its **own line** in a prompt's body:
  `{{include: <id>}}`, where `<id>` is the `kebab-case` id of another prompt. A
  line with other text alongside the token is left verbatim, not expanded.
- `daedalus prompt render <id>` — print the composed prompt, with all
  `{{include: ...}}` directives resolved recursively.
- `daedalus prompt show <id>` — print the raw file, with the directive
  unresolved (unchanged from ticket 03-01).

## Options

- `--path <dir>` — target repository directory (defaults to the current one).

## Notes

- Resolution is **recursive** (an included prompt can include others) and
  **deterministic** (same prompts → same composed text).
- A missing reference and an inclusion cycle are reported as explicit,
  actionable errors and write nothing.

See the
[Composing prompts (includes)](../../../../docs/guide/managing-prompts.md#composing-prompts-includes)
section for full examples, `render` vs `show`, and the error messages.
