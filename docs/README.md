# Daedalus manual

Welcome to the Daedalus user manual. Daedalus is a lightweight TUI/CLI that
automates the setup and management of a project's AI scaffolding — agents,
prompts, DAG workflows, and an SDD backlog — in a backend-agnostic way, and
compiles it to the native format of the tool you use.

New here? Follow the **adoption journey** below from top to bottom: it takes you
from installing Daedalus to compiling your first workspace, then on to the
concepts, the everyday workflow, the full command reference, worked examples, and
troubleshooting. Already comfortable? Jump straight to the
[command reference](guide/command-reference.md) or any per-feature chapter.

## Adoption journey

Follow these in order the first time through.

1. **[Install](getting-started/installation.md)** — run the one-line install
   script (or download a prebuilt binary manually / build from source), per
   platform; verify with `daedalus --version`.
2. **[Quickstart](getting-started/quickstart.md)** — go from zero to a compiled
   workspace: `init` → `validate` → `build`. A reproducible end-to-end run.
3. **[Concepts](guide/concepts.md)** — the `.daedalus/` workspace, the canonical
   model, backend-agnostic compilation, and the Phase 1 boundary. Just enough to
   operate with judgment.
4. **[Core workflow](guide/core-workflow.md)** — the everyday loop: edit
   definitions → `validate` → `build`, and when to use each command.
5. **[Command reference](guide/command-reference.md)** — every core command
   (`--version`, the interface, `init`, `build`/`sync`, `validate`,
   `trace verify`/`show`) with purpose, flags, exit codes, and examples.
6. **[Examples](guide/examples.md)** — realistic scenarios: start a project, add
   an agent/prompt/workflow, validate, compile, and read a report.
7. **[Troubleshooting](guide/troubleshooting.md)** — common errors and how to
   read the actionable reports and logs.

## User guide

Using Daedalus day to day. The chapters above thread through these; here they
are grouped by topic for reference.

### Reference

- [Command reference](guide/command-reference.md) — the core commands with flags, exit codes, and examples.
- [Command line](guide/command-line.md) — running Daedalus, the interface, version, and help.
- [Navigating the interface](guide/navigating-the-tui.md) — the areas, moving in and out, the breadcrumb, reading documents, filtering, contextual help, and the keyboard reference.
- [Configuration](guide/configuration.md) — the workspace manifest (`daedalus.yaml`), environment variables (`DAEDALUS_LOG_LEVEL`), and logging.

### Working with the workspace

- [Initializing a workspace](guide/initializing-a-workspace.md) — `daedalus init`, the `.daedalus/` workspace, and its root artifacts.
- [Managing agents](guide/managing-agents.md) — the built-in agent catalog: `daedalus agent list`, `add`, `clone`, `edit`, and `import`.
- [Managing prompts](guide/managing-prompts.md) — reusable global and shared prompts, composition, and the interactive preview: `daedalus prompt list`, `create`, `edit`, `show`, `render`, and `remove`.
- [Managing workflows](guide/managing-workflows.md) — declarative DAG workflows, the phase schema, the seeded `sdd-default` pipeline, editing, semantic validation, and the interactive DAG view: `daedalus workflow list`, `create`, `show`, `add-phase`, `edit-phase`, `remove-phase`, `validate`, and `remove`.
- [Managing specs](guide/managing-specs.md) — capture a brief, seed its spec, and refine it: `daedalus spec capture`, `list`, `show`, `edit`, and `remove`.
- [Managing architecture documents](guide/managing-architecture.md) — create an architecture document, optionally linked to its spec: `daedalus architecture create`, `list`, `show`, `edit`, and `remove`.
- [Managing epics and tickets](guide/managing-epics-and-tickets.md) — build the SDD backlog under `.daedalus/epics/`: `daedalus epic` and `daedalus ticket` (`create`, `list`, `show`, `edit`, `remove`).
- [Tracing the backlog](guide/tracing-the-backlog.md) — verify and navigate the spec → epic → ticket chain: `daedalus trace verify` and `daedalus trace show`.
- [Validating conventions](guide/validating-conventions.md) — check a shared workspace along two axes (conventions and definitions), read the report, and gate on its exit codes: `daedalus validate`.
- [Compiling to a backend](guide/compiling-to-a-backend.md) — compile the canonical definition to your backend's native format with validate-first safety, an interactive diff/preview, idempotent re-builds, and clear exit codes: `daedalus build` (alias `sync`).

## Contributing

For working **on** Daedalus itself (building, tooling, CI) — separate from using
the product.

- [Development environment](contributing/development-environment.md) — Make targets, Docker, and Compose.
- [Continuous integration](contributing/continuous-integration.md) — what CI runs and how to reproduce it.
- [Testing and golden files](contributing/testing-and-golden-files.md) — running the suite, what golden files are, updating them with `-update`, and the determinism guarantee.

## How this manual is organized

- **Adoption journey** is the linear path for a new user: install → quickstart →
  concepts → workflow → reference → examples → troubleshooting.
- **User guide** covers everyday use of Daedalus — the commands, the interface,
  and configuration.
- **Contributing** is for working on Daedalus itself (building, tooling, CI), and
  is kept separate from using the product.

> Phase 1 note: Daedalus configures your project's AI structure; it does not
> execute agents — that stays with your runtime (for example, Claude Code).
