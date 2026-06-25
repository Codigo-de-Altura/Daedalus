# Claude Code Adapter — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated.
>
> **Canonical guide:** the full chapter lives in the user manual at
> [`docs/guide/compiling-to-a-backend.md`](../../../../docs/guide/compiling-to-a-backend.md)
> — see [The Claude Code backend → `.claude/`](../../../../docs/guide/compiling-to-a-backend.md#the-claude-code-backend--claude)
> ([manual index](../../../../docs/README.md)). This file is a pointer and summary; the manual is the growing source of truth.

## Overview

When `daedalus.yaml` targets **Claude Code**, `daedalus build` compiles your
canonical `.daedalus/` definition into the `.claude/` structure Claude Code reads.
Output is **deterministic** (same `.daedalus/` → same `.claude/`, byte for byte)
and file names are **kebab-case**, derived from each item's canonical id.

## What it generates under `.claude/`

| Canonical source | Generated file | Frontmatter | Body |
|---|---|---|---|
| Agent | `.claude/agents/<agent-id>.md` (one per agent) | `name` (agent id), `description` (role), `model` (only if the agent defines it) | The agent's prompt |
| Prompt — `.daedalus/prompts/` (global **and** shared) | `.claude/commands/<prompt-id>.md` (one per prompt) | `description` (the prompt's title; omitted if none) | The **resolved** prompt (inclusions expanded inline; self-contained) |
| — | `.claude/settings.json` | — | `$schema` + a `daedalus` managed/generator marker |

**Prompts are your slash commands.** Every workspace prompt becomes a Claude Code
slash command under `.claude/commands/` — author a prompt once, invoke it as a
slash command after a build.

## Notes & limitations

- **Daedalus generates only what you defined.** `settings.json` is intentionally
  minimal — no `permissions`, `env`, `hooks`, or default `model` are fabricated.
  Agent frontmatter does **not** include `tools` or `color` in this release.
- **Deterministic, stable names.** Same canonical input → identical output;
  kebab-case file names stay stable across builds.
- **Claude Code is the implemented backend** in this release.
- **Phase 1:** Daedalus configures the AI structure; it does not execute agents.

See the full chapter — with concrete example output for an agent file, a command
file, and `settings.json` — in
[`docs/guide/compiling-to-a-backend.md`](../../../../docs/guide/compiling-to-a-backend.md#the-claude-code-backend--claude).
