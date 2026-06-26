# Concepts

[← Back to the manual index](../README.md)

This chapter explains the few ideas you need to use Daedalus with confidence:
what the workspace is, the canonical model it holds, how compilation turns that
model into a backend's native format, and where Phase 1 draws the line. It is
intentionally short — just enough to operate with judgment, not a tour of the
internals.

## The `.daedalus/` workspace

Daedalus keeps everything it manages in a single directory at the root of your
repository: **`.daedalus/`**. This is the **canonical, backend-agnostic source
of truth** for your project's AI structure. You create it once with
[`daedalus init`](initializing-a-workspace.md), version it with Git alongside
your code, and edit it like any other set of project files.

A freshly initialized workspace looks like this:

```
.daedalus/
  agents/           # agent definitions (a role plus a prompt)
  prompts/          # reusable global and shared prompts
  workflows/        # DAG workflows
    sdd-default.yaml # the factory SDD workflow, seeded by init
  specs/            # specifications / PRD briefs
  architecture/     # architecture documents
  epics/            # epics (with their tickets nested inside)
  tickets/          # tickets
  docs/             # derived documentation
  .state/           # progress state (tracked in git)
  daedalus.yaml     # the workspace manifest
  init.md           # the project guideline
```

Everything is plain text — YAML and Markdown — so it diffs cleanly and reads
well in a pull request. Daedalus writes it **deterministically**: the same input
always produces the same bytes, which keeps Git history quiet.

## The canonical model

Inside the workspace, Daedalus manages a small set of artifact kinds. Together
they are the **canonical model** — the description of your project's AI
structure, independent of any particular tool.

| Artifact | What it is | Managed with |
|---|---|---|
| **Agents** | A role plus a prompt — the unit a backend runs. | [`daedalus agent`](managing-agents.md) |
| **Prompts** | Reusable `global` and `shared` text fragments. | [`daedalus prompt`](managing-prompts.md) |
| **Workflows** | Declarative **DAG** pipelines: ordered phases, each with an agent, inputs/outputs, a gate, and dependencies. | [`daedalus workflow`](managing-workflows.md) |
| **Specs** | Briefs and the specifications derived from them. | [`daedalus spec`](managing-specs.md) |
| **Architecture** | Architecture documents, optionally linked to a spec. | [`daedalus architecture`](managing-architecture.md) |
| **Epics & tickets** | The SDD backlog, with traceability links up to specs. | [`daedalus epic`](managing-epics-and-tickets.md) / [`daedalus ticket`](managing-epics-and-tickets.md) |
| **Manifest** | `daedalus.yaml`: project name, schema version, target backend(s), and conventions. | [Configuration](configuration.md) |

The specs, architecture, epics, and tickets together form the **SDD backlog** —
a traceable chain from a spec down to the tickets that implement it. You can
verify and navigate that chain with [`daedalus trace`](tracing-the-backlog.md).

## Backend-agnostic compilation

The canonical model is not what an agent tool reads directly. Daedalus
**compiles** it into the native format of the backend you target. That step is
[`daedalus build`](compiling-to-a-backend.md) (alias `sync`).

- You **edit** the clean, backend-agnostic definition in `.daedalus/`.
- Daedalus **validates** it, then **generates** the backend's native files for
  you.
- For the **Claude Code** backend — the one implemented in this release —
  compilation writes the `.claude/` directory: `agents/`, `commands/` (built
  from your prompts), and a minimal `settings.json`.

Because the canonical model is backend-agnostic, the same workspace can target
different backends in the future without rewriting your definitions. You never
hand-edit the generated `.claude/` output — you re-run `build`.

## Conventions

Daedalus is built for **teams** that share one workspace, so the way files are
named, laid out, and formatted is written down and **machine-checkable**.
[`daedalus validate`](validating-conventions.md) reports any drift — a stray
name, a misplaced file, a broken backlog link — so problems are caught before
they reach the shared repository. The conventions (kebab-case names, the
canonical layout, ordered YAML and structured Markdown) are recorded in the
manifest's `conventions` block.

## The Phase 1 boundary

One boundary matters above all others:

> **Daedalus manages your AI structure's *definitions*; it does not *execute*
> the agents.** Running the agents stays with your runtime — for example, Claude
> Code.

In Phase 1, every Daedalus command creates, edits, validates, traces, or
compiles **definitions**. None of them call an agent, contact a model, or run a
workflow. This is why the management commands say, in their help, that Daedalus
"manages the definition only; it does not run the analyst/architect/planner
agent." After you `build`, you run the agents yourself in your chosen backend.

## Where to go next

- [Core workflow](core-workflow.md) — the everyday loop of edit → validate →
  build.
- [Command reference](command-reference.md) — every core command with its flags
  and exit codes.
