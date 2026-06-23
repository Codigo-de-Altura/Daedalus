# Backend Selection — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated. This is the initial outline to be filled with real behavior once the ticket passes validation.

## Overview

_To be completed by C-3PO after implementation._

When you initialize a workspace, Daedalus lets you choose the target agent backend(s) your canonical definitions will later compile to. In the MVP the only supported backend is Claude Code, which is also the default. Your choice is recorded in the `backends` field of `daedalus.yaml`.

## How to use

_Steps the end user follows._

1. Run `daedalus init`.
2. Choose your target backend when prompted (or accept the default, Claude Code).
3. Your selection is saved to `.daedalus/daedalus.yaml`.

## Options / flags

_If applicable._

## Notes & limitations

- MVP supports a single backend (Claude Code); the manifest format is ready for more.
- Phase 1: Daedalus configures the AI structure; it does not execute agents.
