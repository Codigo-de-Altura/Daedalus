# Traceability (spec → epic → ticket) — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated. This is the initial outline to be filled with real behavior once the ticket passes validation.

## Overview

_To be completed by C-3PO after implementation._

Navigate and verify the SDD backlog chain: from a spec down to its epics and tickets, and from a ticket back up to its epic and source spec/architecture. The chain is checkable for broken links and orphans.

## How to use

_Steps the end user follows._

1. From a spec, navigate to its epics and tickets.
2. From a ticket, trace back to its epic and source spec/architecture.
3. Run the traceability check to confirm every link resolves and report any broken links or orphans.

## Options / flags

_If applicable._

## Notes & limitations

- Phase 1: Daedalus configures the AI structure; it does not execute agents. Traceability operates on the workspace artifacts.
- The check is deterministic: the same workspace yields the same result, and it reuses existing links rather than duplicating the source of truth.
