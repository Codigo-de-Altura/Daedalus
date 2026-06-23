# Epics & Tickets Management — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated. This is the initial outline to be filled with real behavior once the ticket passes validation.

## Overview

_To be completed by C-3PO after implementation._

Create and manage your SDD backlog: epics (`epic-NN-<slug>`) and tickets (`ticket-NN-MM-<slug>`), each with consistent metadata — status, priority, dependencies, and links to source artifacts (spec, architecture).

## How to use

_Steps the end user follows._

1. Create an epic; it gets a `epic-NN-<slug>` folder.
2. Create tickets under that epic; each gets a `ticket-NN-MM-<slug>` folder.
3. Set metadata: status, priority, dependencies, and links to the source spec/architecture.

## Options / flags

_If applicable._

## Notes & limitations

- Phase 1: Daedalus configures the AI structure; it does not execute agents. Deriving epics/tickets is done by running the *planner* agent in your backend; implementation happens outside Daedalus.
- Epics and tickets are markdown with stable metadata; Daedalus will not destructively overwrite your manual edits.
