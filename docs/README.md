# Daedalus manual

Welcome to the Daedalus user manual. Daedalus is a lightweight TUI/CLI that
automates the setup and management of a project's AI scaffolding — agents,
prompts, DAG workflows, and an SDD backlog — in a backend-agnostic way, and
compiles it to the native format of the tool you use.

This manual grows alongside the product: each feature is documented here as it
ships. Read it top to bottom if you are new, or jump to a section using the
index below.

## Index

### Getting started
- [Installation](getting-started/installation.md) — prerequisites, building, and running Daedalus.
- [Quickstart](getting-started/quickstart.md) — from zero to an initialized workspace in a few commands.

### User guide
- [Command line](guide/command-line.md) — running Daedalus, the interface, version, and help.
- [Initializing a workspace](guide/initializing-a-workspace.md) — `daedalus init`, the `.daedalus/` workspace, and its root artifacts.
- [Managing agents](guide/managing-agents.md) — the built-in agent catalog: `daedalus agent list` and `daedalus agent add`.
- [Managing prompts](guide/managing-prompts.md) — reusable global and shared prompts, composition, and the interactive preview: `daedalus prompt list`, `create`, `edit`, `show`, `render`, and `remove`.
- [Managing workflows](guide/managing-workflows.md) — declarative DAG workflows, the phase schema, the seeded `sdd-default` pipeline, editing, semantic validation, and the interactive DAG view: `daedalus workflow list`, `create`, `show`, `add-phase`, `edit-phase`, `remove-phase`, `validate`, and `remove`.
- [Managing specs](guide/managing-specs.md) — capture a brief, seed its spec at `.daedalus/specs/<slug>.md`, wire it to the *analyst* step of `sdd-default`, then generate and refine the spec yourself: `daedalus spec capture`, `list`, `show`, `edit`, and `remove`.
- [Managing architecture documents](guide/managing-architecture.md) — create an architecture document at `.daedalus/architecture/<slug>.md`, optionally link it to its originating spec (the *architect* step of `sdd-default`), then generate and refine it yourself: `daedalus architecture create`, `list`, `show`, `edit`, and `remove`.
- [Managing epics and tickets](guide/managing-epics-and-tickets.md) — build the SDD backlog under `.daedalus/epics/`: nested epics and tickets with status, priority, dependencies, and origin links to their spec/architecture (the *planner* step of `sdd-default`): `daedalus epic` and `daedalus ticket` (`create`, `list`, `show`, `edit`, `remove`).
- [Tracing the backlog](guide/tracing-the-backlog.md) — verify the spec → epic → ticket chain is consistent (broken links, orphan tickets, missing-origin warnings) and navigate it in both directions: `daedalus trace verify` and `daedalus trace show`.
- [Compiling to a backend](guide/compiling-to-a-backend.md) — compile the canonical `.daedalus/` definition into your configured backend's native format (for Claude Code: `.claude/agents/`, `.claude/commands/` from your prompts, and `settings.json`), with validate-first safety, an idempotent and non-destructive re-build, a dry-run preview, and clear exit codes: `daedalus build` (alias `sync`).
- [Configuration](guide/configuration.md) — the workspace manifest, environment variables, and logging.

### Contributing
- [Development environment](contributing/development-environment.md) — Make targets, Docker, and Compose.
- [Continuous integration](contributing/continuous-integration.md) — what CI runs and how to reproduce it.

## How this manual is organized

- **Getting started** gets you from a clone to a running binary.
- **User guide** covers everyday use of the Daedalus commands and configuration.
- **Contributing** is for working on Daedalus itself (building, tooling, CI).

> Phase 1 note: Daedalus configures your project's AI structure; it does not
> execute agents — that stays with your runtime (for example, Claude Code).
