# Claude Code Adapter — Usage Guide

> Authored and maintained by C-3PO, technical writer for Daedalus. This guide is for the **end user** of Daedalus, not for internals.

---

## Overview

_To be completed by C-3PO after implementation._

## How to use

When your `daedalus.yaml` targets the Claude Code backend, running a build compiles your canonical definition into the `.claude/` structure that Claude Code understands:

```
daedalus build
```

This produces:

- `.claude/agents/*.md` — one Markdown file per agent, with frontmatter (metadata) followed by the agent prompt.
- `.claude/commands/*.md` — command files derived from your canonical definition.
- Claude Code settings — the relevant configuration for the backend.

You edit clean canonical definitions in `.daedalus/`; Daedalus generates the native Claude Code files for you. You never hand-edit `.claude/` to keep them in sync.

## Options / flags

- _The adapter runs as part of `daedalus build`; see the build/sync guide for command flags._
- _Adapter-specific notes to be completed by C-3PO after implementation._

## Notes & limitations

- Output is deterministic: the same canonical definition always produces the same `.claude/` files (this is what makes golden-file testing possible).
- File names are derived from the canonical agent/command id in `kebab-case`.
- The adapter interface is extensible: additional backends can be added later without changing the core. In this phase, **Claude Code is the only implemented backend**.
- **Phase 1: Daedalus configures the AI structure; it does not execute agents.** You run the agents in Claude Code yourself after building.
