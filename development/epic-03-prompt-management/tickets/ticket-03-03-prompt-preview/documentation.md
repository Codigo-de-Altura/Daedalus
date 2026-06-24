# Ticket 03-03 — Prompt Preview

> **Pointer:** the user-facing guide for this feature lives in the manual,
> under the "Previewing prompts in the interface" section of
> [`docs/guide/managing-prompts.md`](../../../../docs/guide/managing-prompts.md#previewing-prompts-in-the-interface).
> This file is only a pointer; the chapter is the actual guide.

## Overview

Running `daedalus` with no subcommand in an interactive terminal opens the
prompt browser: a list of the current directory's prompts (id, kind, title) that
opens a **read-only preview** of any prompt's fully composed text, rendered as
Markdown. The preview shows the prompt with every `{{include: ...}}` already
resolved, so the user sees the final result before it is compiled to a backend.

## How to use

- Launch the interface: `daedalus` (in an interactive terminal).
- **List:** `↑`/`k` and `↓`/`j` move the selection; `enter` (or `l`) opens the
  preview; `?` toggles help; `q`/`Ctrl+C` quit.
- **Preview:** `↑`/`↓` scroll a line; `pgup`/`pgdn` (`b`/`f`/`space`) page;
  `g`/`G` jump to top/bottom; `esc` returns to the list; `?` toggles help.

## Notes

- The preview is **read-only**; it never edits or saves the prompt.
- An empty workspace shows a "No prompts found" guide; a prompt with a
  composition error (cycle or missing reference) shows a readable error message
  instead of broken content, without crashing.

See the
[Previewing prompts in the interface](../../../../docs/guide/managing-prompts.md#previewing-prompts-in-the-interface)
section for the full walkthrough, the keyboard shortcuts, and the empty/error
states. The interface launch is also cross-linked from
[`docs/guide/command-line.md`](../../../../docs/guide/command-line.md).
