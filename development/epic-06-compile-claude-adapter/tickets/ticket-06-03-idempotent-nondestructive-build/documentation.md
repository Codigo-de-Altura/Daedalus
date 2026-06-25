# Idempotent & non-destructive build — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated.
>
> **Canonical guide:** the full chapter lives in the user manual at
> [`docs/guide/compiling-to-a-backend.md`](../../../../docs/guide/compiling-to-a-backend.md)
> — see [Re-running build: idempotent and non-destructive](../../../../docs/guide/compiling-to-a-backend.md#re-running-build-idempotent-and-non-destructive)
> ([manual index](../../../../docs/README.md)). This file is a pointer and summary; the manual is the growing source of truth.

## Overview

`daedalus build` is safe to run as often as you like. Each run classifies every
artifact it manages and prints a summary, so re-compiling is predictable and never
destructive.

## Re-build behavior

| Situation | What happens | Reported as |
|---|---|---|
| Re-run, no canonical changes | Nothing is rewritten; files stay byte-identical (no churn) | `unchanged` |
| One part of the definition changed (e.g. an agent's prompt) | Only the affected artifact(s) are rewritten; the rest untouched | `updated` (`~`) / `unchanged` |
| A new canonical item | Its native file is written | `created` (`+`) |
| Canonical source of a previous artifact removed | The leftover native file is detected and reported, **not** deleted | `orphan` (`?`) |

Example summaries (real CLI output):

```
Compiled .:
  claude-code: 0 created, 0 updated, 5 unchanged (of 5 artifacts)
```

```
Compiled .:
  claude-code: 0 created, 1 updated, 4 unchanged (of 5 artifacts)
    ~ agents/reviewer.md
    ? agents/old-helper.md (orphan: no longer produced; left untouched)
```

## Notes & limitations

- **Idempotent.** Same `.daedalus/` → same `.claude/`, byte for byte, run after
  run. A no-change re-run rewrites nothing.
- **Non-destructive (managed area).** `build` manages only the files it produces.
  Files you place by hand that the build does not generate — inside `.claude/` or
  anywhere else — are preserved intact; `build` never deletes or overwrites
  content outside its managed area.
- **Orphans are reported, never deleted.** Remove the canonical source of an
  artifact and the leftover native file is flagged as an orphan in the summary,
  left untouched. Delete it yourself if you no longer want it.
- These classifications and the exact content changes can be reviewed artifact by
  artifact in the
  [interactive preview](../../../../docs/guide/compiling-to-a-backend.md#previewing-and-confirming-changes)
  before anything is written.

See the full chapter — with worked re-build summaries and the orphan example — in
[`docs/guide/compiling-to-a-backend.md`](../../../../docs/guide/compiling-to-a-backend.md#re-running-build-idempotent-and-non-destructive).
