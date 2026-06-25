# Team Conventions — Usage Guide

> Authored and maintained by C-3PO (technical writer). End-user usage guide for Daedalus, not internals.

---

This ticket's end-user documentation lives in the product manual as a dedicated
chapter:

**→ [docs/guide/validating-conventions.md](../../../../docs/guide/validating-conventions.md)** — "Validating conventions"

(Linked from the manual index at [docs/README.md](../../../../docs/README.md), in the User guide section, after "Tracing the backlog".)

## Overview

Daedalus is built for teams, so a shared `.daedalus/` workspace must stay
consistent no matter who edits it. The manual chapter explains the **team
conventions** Daedalus enforces, grouped in four families — **Naming**
(kebab-case plus the `epic-NN-<slug>` / `ticket-NN-MM-<slug>` id patterns for
epics, tickets, agents, workflows, and prompts), **Structure** (the canonical
`.daedalus/` layout with a nested backlog and the tracked `.state/`
placeholder), **Format** (canonical-ordered YAML frontmatter and structured
Markdown), and **Traceability** (each ticket references its epic; each epic
references its origin). It notes that the canonical sources are `init.md` §7 and
`CLAUDE.md` §6, and that `daedalus validate` is the machine-checkable expression
of those.

## How to use

Run `daedalus validate` from the repository root (or `daedalus validate --path
<dir>` for another directory) to check the workspace against the conventions. It
is **read-only**: it reports violations, it does **not** auto-fix them. The
chapter shows how to read the report — each line is `[severity] location:
convention: reason` — with a worked example of both the conforming case and a
run with errors, and explains the error-vs-warning split (warnings are absent
optional origin links and never fail the build).

## Options / flags

| Flag | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/` workspace is validated. Defaults to the current directory. |
| `--help` | Show all available options. |

**Exit codes:** `0` conformant (no violations, or warnings only) · `1`
convention errors · `2` usage/IO error.

## Notes & limitations

- **Phase 1:** Daedalus configures the AI structure; it does not execute agents.
- The conventions validation reports violations; it does not auto-fix them.
- Conventions cover naming (kebab-case, `epic-NN-<slug>`, `ticket-NN-MM-<slug>`), workspace structure, and artifact formatting.
