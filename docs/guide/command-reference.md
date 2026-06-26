# Command reference

[← Back to the manual index](../README.md)

This is the reference for the **core** Daedalus commands: the version flag, the
interactive interface, and the subcommands `init`, `build` (alias `sync`),
`validate`, and `trace`. Each entry lists the command's purpose, its
flags/parameters, its exit codes, and at least one example.

Daedalus dispatches subcommands **positionally** — `daedalus init`,
`daedalus build`, `daedalus validate`, `daedalus trace`. Each subcommand has its
own `--help`, for example `daedalus init --help`.

The **definition-management** and **SDD-backlog** commands
(`agent`, `prompt`, `workflow`, `spec`, `architecture`, `epic`, `ticket`) are
documented in their own chapters — see [Managing agents](managing-agents.md),
[Managing prompts](managing-prompts.md),
[Managing workflows](managing-workflows.md), [Managing specs](managing-specs.md),
[Managing architecture documents](managing-architecture.md), and
[Managing epics and tickets](managing-epics-and-tickets.md). Running any of those
with no operation prints its usage and exits `2`.

> **Output convention.** Daedalus prints the **human-readable summary** to
> **standard output (stdout)** and **structured JSON logs** to **standard error
> (stderr)**. The log threshold is controlled by `DAEDALUS_LOG_LEVEL` (default
> `info`); see [Configuration](configuration.md#logging). The examples below show
> the stdout summary.

## `--version`

Print the Daedalus version and exit.

```sh
daedalus --version
# daedalus 0.1.0-dev
```

Exit code: `0`.

## The interface (no subcommand)

Run `daedalus` with **no subcommand** to launch the interactive TUI. It requires
an interactive terminal.

```sh
daedalus
```

In an interactive terminal this opens the interface; see
[Navigating the interface](navigating-the-tui.md) for the areas and keys. When
there is no interactive terminal (piped input, a script, CI, or a container with
no TTY), Daedalus does **not** start the interface — it prints a short notice and
exits `0`. See [Command line](command-line.md#non-interactive-use).

## `daedalus init`

Create the canonical `.daedalus/` workspace. If a workspace already exists,
`init` performs a **non-destructive upgrade** — it adds only the missing
directories and root artifacts and never overwrites your content.

Full chapter: [Initializing a workspace](initializing-a-workspace.md).

### Flags

| Flag | Default | Description |
|---|---|---|
| `-backend <names>` | `claude-code` | Target backend(s) recorded in the manifest, comma-separated. The MVP supports only `claude-code`. |
| `-path <dir>` | `.` | Target repository directory. |
| `-preview` | off | Dry run: show the changes without writing anything. |

### Exit codes

| Exit code | Meaning |
|---|---|
| `0` | The workspace was created or upgraded (or, with `-preview`, the preview was shown). |
| `2` | Usage error — for example, an unsupported `-backend` value. Nothing is written. |

### Examples

First run in an empty directory:

```sh
daedalus init
```

```
Created Daedalus workspace at .daedalus from scratch.
Seeded factory workflow "sdd-default" at .daedalus/workflows/sdd-default.yaml.
```

Exit `0`. This creates the workspace directories plus the root artifacts and
seeds the factory workflow `sdd-default`.

Re-running on a workspace that is already complete:

```sh
daedalus init --preview
```

```
Existing Daedalus workspace at .daedalus is already complete — nothing to update.
```

Exit `0`, nothing written.

## `daedalus build` (alias `daedalus sync`)

Compile the canonical `.daedalus/` definition into the configured backend's
native format (`.claude/` for Claude Code). The definition is **validated
first**; an invalid definition aborts the build with nothing written. `sync` is
an exact alias.

In an interactive terminal, `build` shows a diff/preview and asks you to confirm
before writing. `--preview` shows the diff and never writes. `--yes` writes
without the gate (for CI). Without a TTY and without `--yes`, `build` prints the
diff and writes nothing.

Full chapter: [Compiling to a backend](compiling-to-a-backend.md).

### Flags

| Flag | Default | Description |
|---|---|---|
| `-path <dir>` | `.` | Repository directory to compile. |
| `-preview` | off | Dry run: show the diff and exit without writing. |
| `-yes` | off | Write without the interactive confirmation gate (for scripts and CI). |

### Exit codes

| Exit code | Meaning |
|---|---|
| `0` | Success — the definition was compiled (or previewed). |
| `2` | Usage error — an invalid flag or argument. |
| `3` | Validation error — the canonical definition is invalid; nothing written. |
| `4` | Compilation or write error (for example, no adapter for the configured backend); nothing written. |

### Examples

Preview the plan without writing (a fresh workspace has only `settings.json` to
compile):

```sh
daedalus build --preview
```

```
Preview of compiling . (no files written):
  claude-code: 1 new, 0 modified, 0 unchanged (of 1 artifact)
    [new]      .claude/settings.json
      + {
      +   "$schema": "https://json.schemastore.org/claude-code-settings.json",
      +   "daedalus": {
      +     "managed": true,
      +     "generator": "daedalus"
      +   }
      + }
```

Exit `0`.

Write non-interactively:

```sh
daedalus build --yes
```

```
Compiled .:
  claude-code: 1 created, 0 updated, 0 unchanged (of 1 artifact)
    + .claude/settings.json
```

Exit `0`. Re-running with nothing changed is an idempotent no-op:

```sh
daedalus build --yes
```

```
Compiled .:
  claude-code: 0 created, 0 updated, 1 unchanged (of 1 artifact)
```

Exit `0`.

## `daedalus validate`

Validate the workspace along two axes and **report** — it never fixes anything.

- **Conventions** — kebab-case and id patterns (`epic-NN-<slug>`,
  `ticket-NN-MM-<slug>`), the canonical layout, YAML/Markdown formatting, and
  spec → epic → ticket traceability.
- **Definitions** — the agent, workflow (DAG), and manifest schemas: required
  fields, DAG cycles, missing artifacts, unknown agents, unsupported backends.

Full chapter: [Validating conventions](validating-conventions.md).

### Flags

| Flag | Default | Description |
|---|---|---|
| `-path <dir>` | `.` | Target repository directory whose `.daedalus/` workspace is validated. |

### Exit codes

| Exit code | Meaning |
|---|---|
| `0` | No errors — the workspace conforms. |
| `1` | Errors found in either axis. |
| `2` | Usage error. |

### Examples

A clean workspace:

```sh
daedalus validate
```

```
Conventions: workspace conforms (no violations).
Definitions: all agents, workflows and manifest are valid.
```

Exit `0`.

A workspace with an error — here the manifest lists an unsupported backend:

```sh
daedalus validate
```

```
Conventions: workspace conforms (no violations).
Definitions: 1 error and 0 warnings:
  - [error] .daedalus/daedalus.yaml: backends[nonexistent-backend]: schema: observed unsupported backend "nonexistent-backend"; expected one of the supported backends: claude-code
```

Exit `1`. Each finding names the file, the spot (`backends[...]`), the rule
(`schema`), and observed-vs-expected — the **actionable report** you act on. See
[Troubleshooting](troubleshooting.md) for how to read findings.

## `daedalus trace <operation>`

Read-only navigation and verification of the spec → epic → ticket traceability
chain, using the links already recorded in your artifacts. Running
`daedalus trace` with **no operation** is a usage error and exits `2`.

Full chapter: [Tracing the backlog](tracing-the-backlog.md).

### `daedalus trace verify`

Check that the chain is consistent and report inconsistencies, worst-first.

| Flag | Default | Description |
|---|---|---|
| `-path <dir>` | `.` | Target repository directory whose `.daedalus/` chain is verified. |

Exit codes:

| Exit code | Meaning |
|---|---|
| `0` | The chain is consistent, or has only warnings. |
| `1` | At least one hard error (a `broken-link` or an `orphan-ticket`). |
| `2` | Usage or load error. |

Example on a fresh workspace (no specs/epics/tickets yet):

```sh
daedalus trace verify
```

```
Traceability chain is consistent (no inconsistencies).
```

Exit `0`.

### `daedalus trace show <artifact-id>`

Navigate the chain from an artifact, with the direction inferred from the id
shape:

- a **spec slug** descends — spec → its epics → their tickets;
- an **epic id** (`epic-NN-<slug>`) shows both — the epic's origin and its
  tickets;
- a **ticket id** (`ticket-NN-MM-<slug>`) ascends — ticket → its epic → its
  origin spec/architecture.

| Flag | Default | Description |
|---|---|---|
| `-path <dir>` | `.` | Target repository directory whose `.daedalus/` chain is navigated. |

Example:

```sh
daedalus trace show epic-05-sdd-backlog
```

See [Tracing the backlog](tracing-the-backlog.md#navigating-the-chain) for the
full output shapes. (`trace show` needs an existing artifact id; a freshly
initialized workspace has none yet.)
