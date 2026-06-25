# Validating conventions

[← Back to the manual index](../README.md)

Daedalus is built for **teams**: several people share and version the same
`.daedalus/` workspace. For that to stay consistent, the **conventions** — how
files and ids are named, how the workspace is laid out, how artifacts are
formatted, and how the backlog is linked together — cannot be tribal or
implicit. They are written down once and, just as importantly, they are
**machine-checkable**. The `daedalus validate` command reads your workspace and
reports every place it drifts from those conventions, so a stray name or a
mislaid file is caught early instead of leaking into the shared repository.

`daedalus validate` is **read-only**. It never runs an agent and never writes
anything — it **reports** violations, it does **not** auto-fix them. Fixing a
violation is up to you (rename the file, move it into place, reorder the
frontmatter), and the next run confirms the fix.

## The conventions

The conventions fall into four families. Each one exists so that a workspace
shared across a team reads and diffs the same way no matter who touched it last.

### Naming

Every file and id uses **`kebab-case`** — lowercase words joined by hyphens, no
spaces, underscores, or camelCase. On top of that, epics and tickets follow a
fixed id shape so their place in the backlog is obvious from the name alone:

| Artifact | Pattern | Example |
|---|---|---|
| Epic directory / id | `epic-NN-<slug>` | `epic-08-state-collab` |
| Ticket directory / id | `ticket-NN-MM-<slug>` | `ticket-08-03-team-conventions` |
| Agent (file name without `.md`) | `kebab-case` | `analyst.md` |
| Workflow (file name without `.yaml`) | `kebab-case` | `sdd-default.yaml` |
| Prompt (file name without `.md`) | `kebab-case` | `glossary.md` |

In the patterns, `NN` is the epic number and `MM` is the ticket's sequence
within that epic (both numeric), and `<slug>` is a kebab-case description. **A
ticket's directory name is its id** — there is no separate id to keep in sync.

### Structure

The workspace follows the canonical `.daedalus/` layout that
[`daedalus init`](initializing-a-workspace.md) creates: the expected directories
are present, and the backlog is **nested** — tickets live inside their epic's
`tickets/` folder, not in a flat list. The tracked `.state/` directory is part
of the layout too: it carries a versioned placeholder so the folder exists in
git from the start, ready to hold progress state. The validation flags a
required directory that is missing as well as an artifact that sits where it does
not belong.

### Format

Artifacts are formatted for clean, deterministic diffs:

- **YAML frontmatter** uses a **canonical key order** — the same keys always
  appear in the same sequence, so two people editing the same artifact produce
  the same ordering and the diff shows only what actually changed.
- **Markdown** is **structured**: hierarchical headings, tables for metadata,
  fenced code blocks for schemas and DAGs.

### Traceability

The backlog is linked end to end: **every ticket references its parent epic**,
and **every epic references its origin** (the spec or architecture it derives
from). This is the same chain that [`daedalus trace`](tracing-the-backlog.md)
navigates; `validate` checks that the links the conventions require are in place.

> The canonical sources for these conventions are **`init.md` §7** (the project
> guideline `daedalus init` writes into your workspace) and **`CLAUDE.md` §6**.
> `daedalus validate` is the **machine-checkable expression** of those written
> conventions — when the prose and the command ever seem to disagree, the prose
> is the source of truth and the command is how you enforce it.

## Running the validation

From the root of your repository:

```sh
daedalus validate
```

To target a different directory, use `--path`:

```sh
daedalus validate --path ./my-repo
```

The command inspects the `.daedalus/` workspace in that directory, checks it
against all four convention families, and prints a single report. It writes
nothing.

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/` workspace is validated. Defaults to the current directory. |
| `--help` | Show all available options. |

## Reading the report

When the workspace follows every convention, `validate` says so and exits with
status `0`:

```
Workspace conforms to the conventions (no violations).
```

When it finds something, it prints one line per violation. Each line has the
same shape:

```
[severity] location: convention: reason
```

- **`[severity]`** is `error` or `warning` (see below).
- **`location`** is the file or directory at fault.
- **`convention`** is the short name of the rule that was broken.
- **`reason`** explains what is expected and how to bring it into line.

A run with violations looks like this — and, because there is at least one
error, the command exits `1`:

```
Workspace has 2 convention errors and 0 warnings:
  - [error] .daedalus/epics/epic-bad: epic-id-pattern: an epic directory name must match epic-NN-<slug> (NN numeric, slug kebab-case)
  - [error] .daedalus/prompts/BadName.md: kebab-case: a prompt id (its file name without .md) must be kebab-case
```

To fix these you would rename `epic-bad` to a valid `epic-NN-<slug>` directory
and rename `BadName.md` to `bad-name.md`, then run `daedalus validate` again to
confirm the workspace conforms.

### Errors vs. warnings

Not every finding fails the check:

- **Errors** are genuine convention breaches — a bad name, a missing required
  directory, a misplaced artifact, a broken traceability link. They **fail** the
  validation (exit code `1`).
- **Warnings** flag an **optional origin link that is absent**. Recording the
  spec or architecture an epic derives from is recommended but optional, so a
  missing one is reported for visibility without **failing** the build. This is
  the same severity split that [`daedalus trace`](tracing-the-backlog.md#what-it-checks)
  applies to the traceability chain.

A workspace whose only findings are warnings still conforms and still exits `0`.

## Exit codes

`validate` sets its exit code so you can gate on it from a script or CI:

| Exit code | Meaning |
|---|---|
| `0` | The workspace **conforms** — no violations, or only warnings (an optional gap never fails the check). |
| `1` | The workspace has at least one **convention error**. |
| `2` | A usage or I/O error (for example, a path that has no `.daedalus/` workspace). |

## Phase 1: read-only, no agents

Like the rest of Daedalus in Phase 1, `validate` configures and checks your AI
structure but does not **execute** any agent. It **reads** your workspace and
**reports** how well it follows the team conventions; it never modifies your
files and never auto-fixes a violation. Bringing a workspace into conformance is
a manual edit on your side — and a deterministic one to verify: the same
workspace state always produces the same report, so a clean run today stays clean
until something actually drifts.
