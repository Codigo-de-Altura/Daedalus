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

The command checks your workspace along **two axes** and prints both in one
report:

- **Conventions** — how files and ids are named, how the workspace is laid out,
  how artifacts are formatted, and how the backlog is linked. Described next.
- **Definitions** — whether your agents, workflows, and manifest are themselves
  well-formed. See [Validating definitions](#validating-definitions) below.

A fresh [`daedalus init`](initializing-a-workspace.md) workspace passes both axes
clean.

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
against all four convention families **and** lints the definitions (agents,
workflows, manifest), and prints a single report. It writes nothing.

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/` workspace is validated. Defaults to the current directory. |
| `--help` | Show all available options. |

## Reading the report

The report has one line per axis. When the workspace follows every convention
and every definition is well-formed, `validate` says so on both axes and exits
with status `0`:

```
Conventions: workspace conforms (no violations).
Definitions: all agents, workflows and manifest are valid.
```

When an axis finds something, it prints a count followed by one line per
finding. Each finding line has the same shape:

```
[severity] location: spot: rule: reason
```

- **`[severity]`** is `error` or `warning` (see below).
- **`location`** is the file or directory at fault.
- **`spot`** is the exact place inside it (a field, a key, a phase).
- **`rule`** is the short name of the convention or schema rule that was broken.
- **`reason`** explains what was observed and what was expected.

See [Validating definitions](#validating-definitions) below for a worked
`Definitions:` example. Because a single error in either axis exits `1`, fix
every reported finding and run `daedalus validate` again to confirm the
workspace conforms.

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

A workspace whose only findings are warnings still conforms (the `Conventions:`
axis reports no errors) and still exits `0`.

## Validating definitions

Beyond the conventions, the same `daedalus validate` run **lints the definitions
themselves** — your agents, your workflows, and your manifest — and prints them
under a `Definitions:` section of the report. Conventions check that an artifact
is named and placed correctly; the definition linters check that its **content**
is well-formed and coherent. The linters read the **canonical model** in
`.daedalus/`, not any backend's native format, so the result is the same whatever
backend you compile to.

Three families are linted:

- **Agents** — each agent definition is schema-valid: the required fields are
  present, the id is `kebab-case`, the role and prompt are filled in, and any
  parameter types are valid.
- **Workflows (DAG)** — each workflow is a coherent DAG: no **cycles**, no phase
  that consumes an artifact **no phase produces**, no reference to an **unknown
  agent**, no **duplicate phase ids**, and no malformed dependencies.
- **Manifest (`daedalus.yaml`)** — the manifest's required fields are well-formed
  (`name`, `version`, `backends`, `conventions`), every listed backend is a
  **supported** one, and the conventions block is coherent.

Each definition finding is **actionable**: it names the definition at fault, the
exact spot inside it (a field, a phase, or a key), and what was expected versus
what was found. Like the conventions report, the findings are deterministic and
printed in a stable order, so the same workspace always produces the same output.

For example, a manifest that lists a backend Daedalus does not support produces a
`Definitions:` section with a count and one finding per problem. The finding
names the file, the spot (`backends[...]`), the rule (`schema`), and what was
observed versus expected:

```
Conventions: workspace conforms (no violations).
Definitions: 1 error and 0 warnings:
  - [error] .daedalus/daedalus.yaml: backends[nonexistent-backend]: schema: observed unsupported backend "nonexistent-backend"; expected one of the supported backends: claude-code
```

Because there is at least one definition error, the command exits `1`. To fix it
you would set `backends` to a supported value (`claude-code`) and run `daedalus
validate` again to confirm both axes are clean.

The same `Definitions:` section reports the other definition families too — for
example a workflow whose DAG loops back on itself is reported as a `dag` finding
naming the cycle (`cycle detected through phase ...`). Fix the reported spot and
run `daedalus validate` again to confirm both axes are clean.

> A pristine `daedalus init` workspace lints clean: the built-in agents that the
> seeded default workflow references count as **known**, so a freshly initialized
> project reports no definition errors.

## Exit codes

`validate` sets its exit code so you can gate on it from a script or CI:

| Exit code | Meaning |
|---|---|
| `0` | The workspace **conforms** — no violations, or only warnings (an optional gap never fails the check). |
| `1` | The workspace has at least one **error** in **either** axis — a convention violation or a definition lint error. |
| `2` | A usage or I/O error (for example, a path that has no `.daedalus/` workspace). |

A single error in either the conventions or the definitions report is enough to
exit `1`, so gating CI on `daedalus validate` covers both at once.

## Phase 1: read-only, no agents

Like the rest of Daedalus in Phase 1, `validate` configures and checks your AI
structure but does not **execute** any agent. It **reads** your workspace and
**reports** how well it follows the team conventions; it never modifies your
files and never auto-fixes a violation. Bringing a workspace into conformance is
a manual edit on your side — and a deterministic one to verify: the same
workspace state always produces the same report, so a clean run today stays clean
until something actually drifts.
