# Documentation pointer — Ticket 10-01

This ticket's deliverable is the consumer-facing user guide in the product
manual under `docs/`. This file is only a **pointer** to that manual; the manual
itself is the guide.

## Where the guide lives

- **Manual index (adoption journey):** [`docs/README.md`](../../../../docs/README.md)

The index lays out the linear adoption journey a new user follows top to bottom:

1. [Install](../../../../docs/getting-started/installation.md)
2. [Quickstart](../../../../docs/getting-started/quickstart.md)
3. [Concepts](../../../../docs/guide/concepts.md)
4. [Core workflow](../../../../docs/guide/core-workflow.md)
5. [Command reference](../../../../docs/guide/command-reference.md)
6. [Examples](../../../../docs/guide/examples.md)
7. [Troubleshooting](../../../../docs/guide/troubleshooting.md)

## Chapters created or restructured by this ticket

- Restructured: [`docs/README.md`](../../../../docs/README.md) — adoption journey index.
- New: [`docs/guide/concepts.md`](../../../../docs/guide/concepts.md)
- New: [`docs/guide/core-workflow.md`](../../../../docs/guide/core-workflow.md)
- New: [`docs/guide/command-reference.md`](../../../../docs/guide/command-reference.md)
- New: [`docs/guide/examples.md`](../../../../docs/guide/examples.md)
- New: [`docs/guide/troubleshooting.md`](../../../../docs/guide/troubleshooting.md)
- Refreshed: [`docs/getting-started/installation.md`](../../../../docs/getting-started/installation.md) (GitHub Releases + build-from-source, per platform)
- Refreshed: [`docs/getting-started/quickstart.md`](../../../../docs/getting-started/quickstart.md) (end-to-end zero → compiled workspace)
- Refreshed: [`docs/guide/command-line.md`](../../../../docs/guide/command-line.md) (subcommand list)

The existing per-feature chapters under `docs/guide/` (agents, prompts,
workflows, specs, architecture, epics/tickets, trace, validate, build, TUI,
configuration) are reused as-is and linked from the index. The `docs/contributing/`
chapters remain separate (working **on** Daedalus, not **with** it).
