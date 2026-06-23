# Import Agents from Local Files — Usage Guide

> _Authored and maintained by C-3PO as the feature is implemented and validated._

## Overview

_To be completed by C-3PO after implementation._

## How to use

_To be completed by C-3PO after implementation._

- Importing an agent from a local canonical definition file.
- Importing existing `.claude/agents/` structures (Claude Code format) into the workspace.

## Options / flags

_To be completed by C-3PO after implementation._

## Notes & limitations

- Imported definitions are validated against the canonical agent schema; invalid sources are reported, not silently imported.
- Imports are non-destructive: existing workspace identifiers are not silently overwritten.
- Phase 1: Daedalus configures the AI structure; it does not execute agents.
