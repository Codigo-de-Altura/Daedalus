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
