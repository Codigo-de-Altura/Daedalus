# Ticket 02-04 — Agent Schema Validation

> **Pointer:** the user-facing guide for this feature lives in the manual,
> in the "Managing agents" chapter:
> [Agent validation](../../../../docs/guide/managing-agents.md#agent-validation).
> This file is only a pointer; the chapter is the actual guide.

## Overview

Daedalus defines a single **canonical agent schema** and validates every agent
definition against it. The validator runs as the quality gate behind all agent
operations (`add`, `clone`, `edit`, `import`), so a definition is only written
when it is valid. It checks the definition only; it never executes the agent.

## How to use

There is no separate command: validation happens automatically whenever an
operation would write an agent. When a definition is invalid, the operation is
rejected (exit code `2`) and nothing is written.

The schema requires:

- `id` — non-empty, `kebab-case`.
- `role` — non-empty.
- `prompt` — non-empty.
- `parameters` — optional; keys non-empty and unique, values of a known type
  (string, number, or bool).
- `version` — stamped by Daedalus, not user-authored.

## Errors

Validation failures are **actionable**: Daedalus reports every problem at once
(not just the first), each naming the field, what was observed, and what was
expected, in a stable order. For example:

```
daedalus: agent "analyst-custom" is invalid; the edit was not applied:
  - role: observed empty; expected a non-empty role/description
  - prompt: observed empty; expected a non-empty prompt
```

See [`docs/guide/managing-agents.md`](../../../../docs/guide/managing-agents.md)
for the full schema table and error examples.
