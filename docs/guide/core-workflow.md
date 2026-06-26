# Core workflow

[← Back to the manual index](../README.md)

Day to day, using Daedalus is a short, repeatable loop. You **edit** your
canonical definitions, **validate** them, and **build** them into your backend's
native format. This chapter describes that loop and when to reach for each
command. For the full flags and exit codes of each command, see the
[Command reference](command-reference.md).

## The loop

```
edit definitions  →  daedalus validate  →  daedalus build
   (in .daedalus/)     (catch drift early)    (compile to .claude/)
        ▲                                           │
        └───────────────── iterate ─────────────────┘
```

1. **Edit definitions.** Add or change agents, prompts, workflows, specs,
   architecture, epics, and tickets in `.daedalus/`. Use the matching
   `daedalus` commands (for example [`daedalus agent`](managing-agents.md),
   [`daedalus prompt`](managing-prompts.md),
   [`daedalus workflow`](managing-workflows.md)) or edit the files by hand —
   they are plain YAML and Markdown.
2. **Validate.** Run [`daedalus validate`](validating-conventions.md) to check
   the workspace along two axes — the team **conventions** and the
   **definitions** themselves — before the changes reach your shared
   repository. It is read-only and never auto-fixes.
3. **Build.** Run [`daedalus build`](compiling-to-a-backend.md) (alias `sync`)
   to compile the canonical definition into your backend's native format
   (`.claude/` for Claude Code). `build` validates first, then shows you a
   preview and asks you to confirm before writing.

Repeat as your project grows. Because every step is deterministic and
non-destructive, the loop is safe to run as often as you like.

## When to use each command

| You want to… | Command |
|---|---|
| Create the workspace in a repository | [`daedalus init`](initializing-a-workspace.md) |
| Add, clone, edit, or import an agent | [`daedalus agent`](managing-agents.md) |
| Author reusable prompts | [`daedalus prompt`](managing-prompts.md) |
| Define or edit a DAG workflow | [`daedalus workflow`](managing-workflows.md) |
| Build the SDD backlog | [`daedalus spec`](managing-specs.md), [`daedalus architecture`](managing-architecture.md), [`daedalus epic`](managing-epics-and-tickets.md), [`daedalus ticket`](managing-epics-and-tickets.md) |
| Check the workspace follows the conventions | [`daedalus validate`](validating-conventions.md) |
| Verify or navigate the backlog's traceability | [`daedalus trace`](tracing-the-backlog.md) |
| Compile the definition to your backend | [`daedalus build`](compiling-to-a-backend.md) |
| Browse the workspace interactively | [The interface](navigating-the-tui.md) (`daedalus` with no subcommand) |

## The build gate

`build` never writes silently — this is the heart of the safe workflow:

- **In an interactive terminal**, `build` shows a preview of every change and
  asks you to confirm. Press `y`/`enter` to write, or `n`/`esc` to cancel
  (nothing is written).
- **`daedalus build --preview`** shows the diff and **never writes**, in a
  terminal or out of one. Use it to inspect a plan.
- **`daedalus build --yes`** writes **without** the interactive gate. Use it in
  scripts and CI.
- **Without a terminal and without `--yes`**, `build` prints the diff and
  **writes nothing**, telling you to pass `--yes` or run in a terminal.

See [Compiling to a backend](compiling-to-a-backend.md) for the full preview UI
and the idempotent, non-destructive guarantees.

## Gating in CI

Both `validate` and `build` set exit codes you can gate on:

- `daedalus validate` exits `0` (conforms), `1` (errors), or `2` (usage). Run
  it in CI to reject a workspace that drifts from the conventions or has an
  invalid definition.
- `daedalus build --yes` writes the backend output non-interactively and exits
  `0` on success.

## The Phase 1 boundary

This loop **configures** your AI structure; it does not **run** it. After a
successful `build`, you execute the agents yourself in your chosen backend (for
example, Claude Code). See [Concepts](concepts.md#the-phase-1-boundary).
