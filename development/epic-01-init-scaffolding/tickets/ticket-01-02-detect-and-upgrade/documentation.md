# Re-running `init` (Detect & Upgrade) — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated. This is the initial outline to be filled with real behavior once the ticket passes validation.

## Overview

_To be completed by C-3PO after implementation._

When you run `daedalus init` in a repository that already has a `.daedalus/` workspace, Daedalus detects it and performs a non-destructive upgrade: it never overwrites your manual edits, and it shows a preview of any missing pieces it would add before writing anything.

## How to use

_Steps the end user follows._

1. From a repository that already contains `.daedalus/`, run `daedalus init`.
2. Review the preview of proposed additions.
3. Daedalus completes the structure without touching your existing files.

## Options / flags

_If applicable._

## Notes & limitations

- Manual edits inside `.daedalus/` are preserved; the upgrade only adds what is missing.
- Phase 1: Daedalus configures the AI structure; it does not execute agents.
