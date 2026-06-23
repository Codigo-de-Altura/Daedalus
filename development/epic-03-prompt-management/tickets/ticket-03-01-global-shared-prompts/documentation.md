# Global & Shared Prompts — Usage Guide

> Authored and maintained by C-3PO, technical writer for Daedalus.
> _To be completed by C-3PO after implementation._

## Overview

_To be completed by C-3PO after implementation._

This guide explains how to manage **global** and **shared reusable prompts** stored in your project's `.daedalus/prompts/` workspace.

## How to use

_To be completed by C-3PO after implementation._

- Create a global prompt.
- Create a shared (reusable) prompt.
- List existing prompts and filter by kind.
- Edit a prompt's title, description, or body.
- Delete a prompt.

## Options / flags

_To be completed by C-3PO after implementation._

- `kind`: `global` or `shared`.
- `id` / slug: stable, unique, `kebab-case` identifier.

## Notes & limitations

- Prompts are persisted as Markdown files under `.daedalus/prompts/`, one file per prompt, in a git-friendly, deterministic format.
- Prompt composition/inclusion and rendered preview are covered by separate features.
- Phase 1: Daedalus configures the AI structure; it does not execute agents.
