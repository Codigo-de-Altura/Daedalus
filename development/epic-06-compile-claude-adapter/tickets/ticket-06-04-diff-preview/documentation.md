# Diff / Preview Before Writing — Usage Guide

> Authored and maintained by C-3PO, technical writer for Daedalus. This guide is for the **end user** of Daedalus, not for internals.

---

## Overview

_To be completed by C-3PO after implementation._

## How to use

When you compile your project, Daedalus shows you a **preview** of every change before touching disk. For each generated artifact you see whether it would be:

- **new** — the file does not exist yet,
- **modified** — the file exists and its content would change (the change detail is shown),
- **unchanged** — nothing would change.

You then **confirm** to write, or **cancel** to keep everything as is. If you cancel, nothing is written.

You can also run a preview-only pass that shows the diff and exits without writing.

## Options / flags

- _Preview / preview-only flags and confirmation controls to be completed by C-3PO after implementation._
- Navigation and confirm/cancel shortcuts are shown on screen in the TUI.

## Notes & limitations

- Nothing is written until you explicitly confirm — writes are non-destructive by default.
- The preview only covers Daedalus's managed area; manual changes you make outside that area are preserved and are not shown as changes.
- When there is nothing to change, the report says so clearly.
- **Phase 1: Daedalus configures the AI structure; it does not execute agents.** The preview shows configuration changes, not agent runs.
