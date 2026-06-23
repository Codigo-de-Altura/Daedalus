# Architecture Documents — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated. This is the initial outline to be filled with real behavior once the ticket passes validation.

## Overview

_To be completed by C-3PO after implementation._

Manage your architecture documents inside `.daedalus/architecture/`. These are blueprints — high-level structure and decisions, not implementation recipes — that follow your spec in the SDD pipeline and feed planning.

## How to use

_Steps the end user follows._

1. Create an architecture document in `.daedalus/architecture/<slug>.md`.
2. Link it to its source spec (the `spec → architecture` step of `sdd-default.yaml`).
3. Refine it by hand — it is yours to edit.

## Options / flags

_If applicable._

## Notes & limitations

- Phase 1: Daedalus configures the AI structure; it does not execute agents. Generating architecture content is done by running the *architect* agent in your backend, outside Daedalus.
- Architecture documents are markdown and editable; Daedalus will not destructively overwrite your manual edits.
