# Managing workflows

[← Back to the manual index](../README.md)

A **workflow** is a declarative **DAG** (directed acyclic graph) that describes
your project's pipeline: an ordered list of **phases**, where each phase names
an **agent**, the artifacts it consumes and produces, a validation **gate**, and
the predecessor phases it **depends on**. Those `depends_on` references are the
**edges** of the graph. Instead of keeping the pipeline in your head, you author
it as a workflow in your workspace, version it with Git, and edit it with the
`daedalus workflow` command.

Each workflow is a single YAML file, identified by a unique name. A workflow's
phases capture *what* runs and *in what order* — Daedalus models, edits, and
structurally validates that definition for you.

## The phase schema

Every phase is a mapping with the same six fields, always in the same order:

| Field | Required | What it is |
|---|---|---|
| `id` | yes | The phase's identifier, unique within the workflow, in `kebab-case`. It is the handle other phases reference in their `depends_on`, so it is also the node key of the DAG. |
| `agent` | yes | The agent that runs the phase (for example `analyst`, `architect`). An opaque reference — it is not resolved against the agent catalog here. |
| `inputs` | no | The artifacts the phase consumes. A list; order is preserved. |
| `outputs` | no | The artifacts the phase produces. A list; order is preserved. |
| `gate` | yes | The validation criterion an artifact must satisfy to advance past the phase. An opaque reference. |
| `depends_on` | no | The predecessor phases this phase depends on — the incoming edges of this node. A list; order is preserved. |

The phase order in the file is **significant** and is preserved verbatim: it is
the authored reading order of the pipeline, and Daedalus never reorders it.

## Where workflows live

Every workflow is persisted as one YAML file under your workspace's
`.daedalus/workflows/` directory, named after the workflow:

```
.daedalus/
  workflows/
    release-pipeline.yaml
```

The workflow's name is **not** stored inside the document — it is the file's
base name (`<name>.yaml`), exactly as a prompt's id is its file name. The file
has a single top-level `phases:` key holding the ordered list of phase mappings:

```yaml
phases:
  - id: spec
    agent: analyst
    inputs: [brief]
    outputs: [spec]
    gate: spec-gate
    depends_on: []
  - id: build
    agent: architect
    inputs: [spec]
    outputs: [design]
    gate: design-gate
    depends_on: [spec]
```

Within each phase the keys are always written in the same order — `id`, `agent`,
`inputs`, `outputs`, `gate`, `depends_on` — and every key is always present, so
the shape is stable and a reader never has to tell "absent" from "empty". The
list-valued keys are rendered in compact **flow style** on a single line
(`inputs: [brief]`), and an empty list as `[]`. The file always ends with a
single trailing newline.

This makes the output **deterministic**: the same workflow always produces the
same file, byte for byte, which keeps your Git diffs clean. These files are your
**editable source of truth** — you can edit them by hand, but the commands below
are the scriptable alternative.

## The name

Every workflow is identified by a **name**: a stable, unique, `kebab-case` slug
(lowercase letters and digits in dash-separated segments, e.g.
`release-pipeline`). The name is the file name, so it must be unique within the
workspace. A name that is not valid `kebab-case` is rejected with an explicit
error and nothing is written. Phase ids follow the same `kebab-case` rule.

## The default SDD workflow

You do not start from an empty directory. When you run `daedalus init`, Daedalus
seeds one ready-to-use workflow into `.daedalus/workflows/`: **`sdd-default`**,
the default **SDD** (spec-driven development) pipeline. It is a valid, complete
workflow you can use as-is, adapt to your project, or simply read as a worked
example of the schema.

The pipeline is a straight, linear chain of six phases — each run by one of the
built-in agents, each producing the artifact the next phase consumes:

| Phase | Agent | Consumes | Produces | Gate |
|---|---|---|---|---|
| `spec` | `analyst` | `brief` | `spec` | `spec-gate` |
| `architecture` | `architect` | `spec` | `architecture` | `architecture-gate` |
| `epics` | `planner` | `architecture` | `epics` | `epics-gate` |
| `tickets` | `planner` | `epics` | `tickets` | `tickets-gate` |
| `validation` | `validator` | `tickets` | `validation` | `validation-gate` |
| `docs` | `documenter` | `validation` | `docs` | `docs-gate` |

The first phase, `spec`, is the root: it consumes the external `brief` (the
pipeline's initial artifact) through its `inputs`, so its `depends_on` is empty.
Every other phase depends on the one before it, giving a clean chain with no
cycles. You can see the exact seeded file at any time with
`daedalus workflow show sdd-default`:

```yaml
phases:
  - id: spec
    agent: analyst
    inputs: [brief]
    outputs: [spec]
    gate: spec-gate
    depends_on: []
  - id: architecture
    agent: architect
    inputs: [spec]
    outputs: [architecture]
    gate: architecture-gate
    depends_on: [spec]
  - id: epics
    agent: planner
    inputs: [architecture]
    outputs: [epics]
    gate: epics-gate
    depends_on: [architecture]
  - id: tickets
    agent: planner
    inputs: [epics]
    outputs: [tickets]
    gate: tickets-gate
    depends_on: [epics]
  - id: validation
    agent: validator
    inputs: [tickets]
    outputs: [validation]
    gate: validation-gate
    depends_on: [tickets]
  - id: docs
    agent: documenter
    inputs: [validation]
    outputs: [docs]
    gate: docs-gate
    depends_on: [validation]
```

### Where the implementation step is

You may notice there is no "implementation" or "developer" phase between
`tickets` and `validation`, even though something must turn the tickets into a
working implementation before it can be validated. That is intentional: Daedalus
**configures** your project's AI structure but does not **execute** the pipeline,
and the implementation step is performed **externally** — by a developer, or by an
agent on your backend — not by Daedalus.

So the implementation is the (un-modeled) gap on the edge from `tickets` to
`validation`: Daedalus hands off the `tickets`, an external actor builds the
implementation, and `validation` resumes from those same tickets. This is why the
`validation` phase consumes `[tickets]` rather than an `implementation` artifact —
no phase in this workflow produces an `implementation`, so listing it as an input
would describe an artifact that never exists.

### It is a valid DAG

The seeded workflow passes semantic validation out of the box — no cycles, no
missing artifacts, and every agent is a known built-in:

```sh
daedalus workflow validate sdd-default
```

```
Workflow "sdd-default" is semantically valid.
```

The seeding is **non-destructive**: if you have already created or edited an
`sdd-default.yaml`, re-running `daedalus init` leaves your file untouched and
reports it as already present. It is yours to change — rename the gates, add or
remove phases, or rework the chain entirely with the editing commands below.

## Listing workflows

Use `daedalus workflow list` to see every persisted workflow with its name and
phase count:

```sh
daedalus workflow list
```

The workflows are listed in name order:

```
Workflows (1):
  release-pipeline	2 phases
```

When there are no workflows yet, the count is zero:

```
Workflows (0):
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/workflows/` is listed. Defaults to the current directory. |

## Creating a workflow

Use `daedalus workflow create <name>` to add a new, **empty** workflow. Add its
phases afterwards with `daedalus workflow add-phase`.

```sh
daedalus workflow create release-pipeline
```

On success, Daedalus reports the file it created:

```
Created workflow "release-pipeline" at .daedalus/workflows/release-pipeline.yaml.
```

A brand-new workflow has no phases yet, so its file is just the stable empty
shape:

```yaml
phases: []
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/workflows/` the workflow is added to. Defaults to the current directory. |
| `--preview` | Dry run: show the file that would be created without writing anything. |

### Previewing without writing

Use `--preview` to perform a **dry run**. Daedalus validates the name and prints
the exact file it would write, but **writes nothing** to disk:

```sh
daedalus workflow create release-pipeline --preview
```

```
Preview of creating workflow "release-pipeline" at .daedalus/workflows/release-pipeline.yaml:
phases: []
```

Run the command again without `--preview` to apply it.

### It will not overwrite your work

Creating a workflow is **non-destructive**. If a workflow with the same name
already exists, Daedalus leaves the existing file — including any manual edits —
untouched and reports the conflict instead of overwriting it:

```sh
daedalus workflow create release-pipeline
```

```
daedalus: workflow already exists: "release-pipeline" — not overwritten
```

An invalid name is rejected before anything is written:

```sh
daedalus workflow create Bad_Name
```

```
daedalus: workflow name "Bad_Name" is not valid kebab-case
```

## Showing a workflow

Use `daedalus workflow show <name>` to print a workflow's file content
**verbatim** — exactly the canonical YAML as it is stored:

```sh
daedalus workflow show release-pipeline
```

```yaml
phases:
  - id: spec
    agent: analyst
    inputs: [brief]
    outputs: [spec]
    gate: spec-gate
    depends_on: []
  - id: build
    agent: architect
    inputs: [spec]
    outputs: [design]
    gate: design-gate
    depends_on: [spec]
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/workflows/` holds the workflow. Defaults to the current directory. |

If the name does not exist, Daedalus tells you so and writes nothing:

```
daedalus: workflow "ghost" not found
```

## Adding a phase

Use `daedalus workflow add-phase <name>` to **append** a phase to an existing
workflow. The `--id`, `--agent`, and `--gate` flags are required; the list flags
take comma-separated values.

```sh
daedalus workflow add-phase release-pipeline \
  --id spec \
  --agent analyst \
  --gate spec-gate \
  --inputs brief \
  --outputs spec
```

On success, Daedalus confirms the edit and the resulting phase count:

```
Applied add-phase to workflow "release-pipeline" at .daedalus/workflows/release-pipeline.yaml (1 phase).
```

To wire a phase to its predecessor, pass `--depends-on` — this is how you draw
an edge of the DAG:

```sh
daedalus workflow add-phase release-pipeline \
  --id build \
  --agent architect \
  --gate design-gate \
  --inputs spec \
  --outputs design \
  --depends-on spec
```

```
Applied add-phase to workflow "release-pipeline" at .daedalus/workflows/release-pipeline.yaml (2 phases).
```

The new phase is appended at the end, preserving the order of the existing
phases.

### Options

| Option | Description |
|---|---|
| `--id <id>` | The phase id, `kebab-case`. **Required.** |
| `--agent <agent>` | The agent that runs the phase. **Required.** |
| `--gate <gate>` | The phase's validation gate. **Required.** |
| `--inputs <a,b>` | Comma-separated input artifacts. Optional. |
| `--outputs <a,b>` | Comma-separated output artifacts. Optional. |
| `--depends-on <a,b>` | Comma-separated predecessor references — the DAG edges. Optional. |
| `--path <dir>` | Target repository directory whose `.daedalus/workflows/` holds the workflow. Defaults to the current directory. |

## Editing a phase

Use `daedalus workflow edit-phase <name> --id <id>` to change an existing
phase's fields **in place**, keeping its position in the list. Only the flags
you pass are changed; everything else stays as it was.

```sh
daedalus workflow edit-phase release-pipeline --id build --agent planner
```

```
Applied edit-phase to workflow "release-pipeline" at .daedalus/workflows/release-pipeline.yaml (2 phases).
```

To rename a phase, pass `--new-id`. The list flags (`--inputs`, `--outputs`,
`--depends-on`) **replace** the phase's current list; passing an empty value
clears it.

### Options

| Option | Description |
|---|---|
| `--id <id>` | The id of the phase to edit. **Required.** |
| `--new-id <id>` | Rename the phase to this id. Optional. |
| `--agent <agent>` | Set the phase's agent. |
| `--gate <gate>` | Set the phase's gate. |
| `--inputs <a,b>` | Set the phase's inputs (comma-separated; empty clears). |
| `--outputs <a,b>` | Set the phase's outputs (comma-separated; empty clears). |
| `--depends-on <a,b>` | Set the phase's `depends_on` (comma-separated; empty clears). |
| `--path <dir>` | Target repository directory whose `.daedalus/workflows/` holds the workflow. Defaults to the current directory. |

`edit-phase` requires `--id` so it knows which phase to change:

```sh
daedalus workflow edit-phase release-pipeline
```

```
daedalus: workflow edit-phase requires --id
```

## Removing a phase

Use `daedalus workflow remove-phase <name> --id <id>` to delete a phase,
preserving the order of the rest:

```sh
daedalus workflow remove-phase release-pipeline --id build
```

```
Applied remove-phase to workflow "release-pipeline" at .daedalus/workflows/release-pipeline.yaml (1 phase).
```

Removing a phase touches only that phase; it does **not** rewrite other phases'
`depends_on` lists, so any reference to the removed phase is left exactly as you
wrote it.

### Options

| Option | Description |
|---|---|
| `--id <id>` | The id of the phase to remove. **Required.** |
| `--path <dir>` | Target repository directory whose `.daedalus/workflows/` holds the workflow. Defaults to the current directory. |

## Removing a workflow

Use `daedalus workflow remove <name>` to delete a workflow. **Only** that
workflow's file is removed; no other file in the workspace is touched.

```sh
daedalus workflow remove release-pipeline
```

```
Removed workflow "release-pipeline" from .daedalus/workflows/release-pipeline.yaml.
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/workflows/` holds the workflow. Defaults to the current directory. |

Removing a workflow that does not exist is reported as an explicit error, not a
silent success:

```
daedalus: workflow not found: "ghost"
```

## Edits are validated before anything is written

Every phase edit (`add-phase`, `edit-phase`, `remove-phase`) is checked against
the schema **before** it touches disk, and the write is **atomic**. If the
result would be invalid, Daedalus rejects the edit, lists **every** problem it
found — not just the first — and leaves your existing file completely intact
(never half-written). Each finding names the offending phase, the field, what
was observed, and what was expected, so you can fix them all in one pass:

```sh
daedalus workflow add-phase release-pipeline --id Bad-ID
```

```
daedalus: workflow "release-pipeline" is invalid; the edit was not applied:
  - phase Bad-ID: id: observed "Bad-ID"; expected kebab-case: lowercase letters/digits in dash-separated segments (e.g. write-spec)
  - phase Bad-ID: agent: observed empty; expected a non-empty agent reference (e.g. analyst)
  - phase Bad-ID: gate: observed empty; expected a non-empty gate reference (e.g. spec-gate)
```

A duplicate phase id is rejected the same way, since the id is the DAG node key
and must be unique within the workflow:

```sh
daedalus workflow add-phase release-pipeline --id spec --agent x --gate y
```

```
daedalus: phase already exists: "spec"
```

Editing a phase that is not there names the missing id:

```sh
daedalus workflow edit-phase release-pipeline --id ghost --agent x
```

```
daedalus: phase not found: "ghost"
```

And a phase edit only works on a workflow that already exists:

```sh
daedalus workflow edit-phase ghost --id spec --agent x
```

```
daedalus: workflow not found: "ghost"
the workflow must already exist; create it first
```

## Validating a workflow

The edit checks above are **structural** — they keep each phase well-formed. To
check that the workflow as a whole describes a coherent, runnable graph, use
`daedalus workflow validate <name>`. This is **semantic** validation of the DAG:
it reads a workflow you have already defined and reports problems with the graph
itself.

It checks for:

- **Cycles** — a loop in the `depends_on` dependencies, so the graph is not a DAG.
- **Missing artifacts** — a phase consumes an `input` that is neither the initial
  `brief` artifact nor an `output` of any of its predecessors (any phase reachable
  by following `depends_on` back toward the roots).
- **Unknown agents** — a phase references an `agent` that does not exist in the
  workspace.
- **Unknown dependencies** — a `depends_on` entry that names no existing phase
  (and is not the initial `brief`); this is usually a typo.

```sh
daedalus workflow validate release-pipeline
```

A workflow with no problems reports that it is valid:

```
Workflow "release-pipeline" is semantically valid.
```

When there are problems, each is reported on its own line, naming the affected
phase, the kind of problem, the observed value, and what would make it valid:

```
Workflow "release-pipeline" is semantically invalid (2 findings):
  - phase "build": missing-artifact: observed "blueprint"; no predecessor phase produces this input artifact and it is not the initial artifact "brief"; add a predecessor that outputs it, or correct the input
  - phase "build": unknown-agent: observed "wizard"; no agent with this id exists in the workspace catalog; create or import the agent, or correct the reference
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/workflows/` holds the workflow. Defaults to the current directory. |

### What each problem looks like

A **cycle** names the phases that form the loop, rendered as a chain so it is
self-evident:

```
Workflow "cycle-pipe" is semantically invalid (1 finding):
  - phase "alpha": cycle: observed "alpha -> beta -> alpha"; these phases form a dependency cycle, so the workflow is not a DAG; break the loop by removing one of the depends_on edges
```

A **missing artifact** names the phase and the input that no predecessor produces:

```
Workflow "missing-art" is semantically invalid (1 finding):
  - phase "build": missing-artifact: observed "spec"; no predecessor phase produces this input artifact and it is not the initial artifact "brief"; add a predecessor that outputs it, or correct the input
```

An **unknown agent** names the phase and the agent that could not be found:

```
Workflow "unknown-agent" is semantically invalid (1 finding):
  - phase "spec": unknown-agent: observed "wizard"; no agent with this id exists in the workspace catalog; create or import the agent, or correct the reference
```

An **unknown dependency** names the phase and the dangling reference (here a typo
of `brief`):

```
Workflow "unknown-dep" is semantically invalid (1 finding):
  - phase "spec": unknown-dependency: observed "breif"; depends_on references no existing phase (and is not the initial artifact "brief"); fix the reference or add the missing phase
```

### Built-in agents are always known

The unknown-agent check resolves an agent against the workspace's agent catalog,
which includes **both** the agents you have materialized under `.daedalus/agents/`
**and** the built-in agents (`analyst`, `architect`, `planner`, `validator`,
`documenter`). A phase that references a built-in agent you have **not** added to
the workspace yet is therefore **valid** — the built-in still exists as far as the
project is concerned, so it is not flagged. Only an agent id that is neither
materialized nor a known built-in is reported as unknown.

### Exit codes

`validate` sets its exit code so you can gate on it from a script or CI:

| Exit code | Meaning |
|---|---|
| `0` | The workflow is semantically valid. |
| `1` | The workflow is semantically invalid (one or more findings). |
| `2` | A usage or load error — for example, the workflow does not exist:<br>`daedalus: workflow not found: "ghost"` |

## Visualizing a workflow in the TUI

Besides the commands above, Daedalus has an interactive terminal interface that
can draw a workflow as a **graph (DAG)**, so you can see the pipeline's shape —
its phases and how they depend on each other — without reading the YAML. The view
is **read-only**: it never edits or runs anything.

### Switching to the Workflows section

Run `daedalus` with no subcommand in an interactive terminal:

```sh
daedalus
```

The interface opens on the **Prompts** section. Press `tab` to switch between the
**Prompts** and **Workflows** sections; the title changes to `Daedalus ·
Workflows` and the list shows every workflow in the current directory's
`.daedalus/workflows/`, each with its name and phase count:

```
Daedalus · Workflows

  release-pipeline  [2 phases]
```

> The browser reads the `.daedalus/workflows/` of the directory you launch it
> from, so run it from inside your project.

Each section keeps its own selection, so pressing `tab` to switch back returns
you to where you were.

### Browsing the list

| Key | Action |
|---|---|
| `↑` / `k` | Move the selection up. |
| `↓` / `j` | Move the selection down. |
| `enter` / `l` | Open the DAG view for the selected workflow. |
| `tab` | Switch section (Prompts ↔ Workflows). |
| `?` | Toggle the help footer (short ↔ full). |
| `q` / `Ctrl+C` | Quit. |

### Reading the DAG

Press `enter` (or `l`) on a workflow to open its DAG view. A header names the
workflow, and the graph is laid out **top to bottom in dependency order** —
roots (the phases nothing depends on) at the top, their dependents below — so the
pipeline reads from start to finish.

Each phase is drawn as a bordered **node**. Its headline is the phase id followed
by the agent that runs it, written `@<agent>`; below it, when present, a compact
line each for the phase's inputs, outputs, and gate. A phase with no inputs,
outputs, or gate simply omits those lines. A node looks like this:

```
╭─────────────────────╮
│ spec  @analyst      │
│ in:  brief          │
│ out: spec           │
│ gate: spec-gate     │
╰─────────────────────╯
```

Between nodes, a vertical **connector** shows the dependency edge. When a phase
depends on earlier phases, the connector is labelled `after <predecessors>` (the
ids from its `depends_on`); the arrow points down to the phase the edge leads
into:

```
  │  after spec
  ↓
╭─────────────────────╮
│ build  @architect   │
│ in:  spec           │
│ out: design         │
│ gate: design-gate   │
╰─────────────────────╯
```

### Moving around and leaving the view

The DAG scrolls within the view, and a hint at the bottom shows your position
through a long graph:

| Key | Action |
|---|---|
| `↑` / `↓` | Scroll one line. |
| `pgup` / `pgdn` | Scroll one page (also `b` / `f` / `space`). |
| `g` / `G` | Jump to the top / bottom. |
| `esc` | Return to the list. |
| `?` | Toggle the help footer. |
| `Ctrl+C` | Quit. |

`esc` is the way back to the list; inside the DAG view, `q` is reserved so you do
not leave the app by accident while reading. The view never changes the workflow
— to edit it, use the `daedalus workflow` commands above.

### Empty, unreadable, and cyclic workflows

The view degrades gracefully when there is nothing — or nothing valid — to draw:

- **Empty workflow.** A workflow with no phases is a valid document; the view says
  so and points you to the next step instead of showing a blank graph:

  ```
  This workflow is empty.

  Add phases with `daedalus workflow add-phase`.
  ```

- **Unreadable workflow.** If the file cannot be loaded, the view shows a readable
  error message instead of broken content, and the app keeps running. Return to
  the list with `esc`.

- **Dependency cycle.** If the phases depend on each other in a loop, the view does
  not hang. It draws the phases in their declared order and shows a warning at the
  top:

  ```
  ! dependency cycle detected — phases shown in declared order
  ```

If the Workflows list itself is empty, it tells you so and points you at the CLI:

```
No workflows found.

Create one with `daedalus workflow create`, or run `daedalus init`
if this directory is not a Daedalus workspace yet.
```

## Notes & limitations

- Workflows are persisted as YAML files under `.daedalus/workflows/`, one file
  per workflow, in a deterministic, git-friendly format. The same workflow
  always renders the same bytes: fixed key order, phases never reordered, list
  values in flow style, and a single trailing newline.
- Every write operation is **non-destructive**: creating a workflow never
  overwrites an existing one, and an invalid `add-phase`, `edit-phase`, or
  `remove-phase` is rejected before anything is written, leaving your file
  intact.
- Validation in this phase is **structural** only: it checks each phase against
  the schema (required `id`/`agent`/`gate`, `kebab-case` ids, ids unique within
  the workflow). It does **not** yet check the graph's *meaning* — there is no
  cycle detection and no check that a `depends_on`, `inputs`, or `agent`
  reference resolves to something that exists.
- The interactive DAG view is strictly **read-only**: it draws the pipeline but
  has no binding that edits a phase or runs the workflow.
- Phase 1 **models, edits, and validates** a workflow's definition; it does not
  **execute** workflows — running the pipeline stays with your runtime (for
  example, Claude Code).
