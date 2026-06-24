# Ticket 04-02 — Workflow DAG Visualization

> **Pointer:** the user-facing guide for this feature lives in the manual, as a
> section of the workflows chapter:
> [`docs/guide/managing-workflows.md` → Visualizing a workflow in the TUI](../../../../docs/guide/managing-workflows.md#visualizing-a-workflow-in-the-tui).
> This file is only a pointer; the chapter is the actual guide.

## Overview

Daedalus' interactive terminal interface (run `daedalus` with no subcommand) can
draw a workflow as a read-only **graph (DAG)**, alongside the prompt browser. The
interface opens on the **Prompts** section; `tab` switches to the **Workflows**
section, and opening a workflow there renders its phases as nodes and its
`depends_on` dependencies as edges, laid out top-to-bottom in dependency order.

## How to use

- Run `daedalus` in an interactive terminal — it opens on the **Prompts**
  section.
- Press `tab` to switch to the **Workflows** section (title becomes
  `Daedalus · Workflows`).
- Move the selection with `↑`/`k` and `↓`/`j`; press `enter` (or `l`) to open the
  selected workflow's DAG view.
- In the DAG view, each phase is a bordered node (`<id>  @<agent>` plus compact
  `in:` / `out:` / `gate:` lines), and each dependency is a downward connector
  labelled `after <predecessors>`.
- Scroll with `↑`/`↓`, `pgup`/`pgdn` (also `b`/`f`/`space`), and `g`/`G`; press
  `esc` to return to the list; `q` / `Ctrl+C` to quit; `?` to toggle help.

## Notes & limitations

- The DAG view is strictly **read-only**: it draws the pipeline but never edits a
  phase or runs the workflow.
- It degrades gracefully: an empty workflow shows a clear empty state, an
  unreadable file shows an actionable error, and a dependency cycle is drawn in
  declared order with a warning instead of hanging.
- Phase 1 configures the AI structure; it does not execute workflows.

See [`docs/guide/managing-workflows.md` → Visualizing a workflow in the TUI](../../../../docs/guide/managing-workflows.md#visualizing-a-workflow-in-the-tui)
for the full walkthrough, the node/edge layout, and the keyboard shortcuts.
