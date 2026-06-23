# Area Navigation — Usage Guide

_Authored and maintained by C-3PO, technical writer for Daedalus._

## Overview

_To be completed by C-3PO after implementation._

## How to use

Daedalus organizes its work into six areas. From the root screen you can move into any of them and always find your way back:

- **init** — initialize and manage the `.daedalus/` workspace.
- **agents** — browse and manage agents.
- **prompts** — manage global and shared prompts.
- **workflows** — view and edit declarative workflow DAGs.
- **backlog** — manage spec/PRD, architecture, epics and tickets.
- **build** — compile the canonical definition to your backend.

Launch the TUI, pick an area from the root screen, work inside it, and return to the root at any time. No screen is a dead end: there is always a way back.

## Options / flags (keyboard shortcuts)

| Shortcut | Action |
|---|---|
| _TBD_ | Move selection / navigate the area list |
| _TBD_ | Enter the selected area |
| _TBD_ | Go back to the previous screen |
| _TBD_ | Return to the root screen |
| _TBD_ | Quit the TUI |

_Exact key assignments to be confirmed by C-3PO after implementation (see ticket-07-03 for the keybinding system)._

## Notes & limitations

- Phase 1: Daedalus configures the AI structure; it does not execute agents.
- The navigation shell frames each area; the domain logic behind an area is provided by the core.
