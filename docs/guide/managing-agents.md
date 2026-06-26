# Managing agents

[← Back to the manual index](../README.md)

Daedalus ships with a **built-in agent catalog**: a small, fixed set of canonical
agents that cover the Spec-Driven Development (SDD) pipeline out of the box. The
catalog lets you start from proven roles instead of writing agents from scratch.
You can **list** the agents the catalog offers and **add** (materialize) any of
them into your workspace as an editable definition.

The catalog is embedded in the Daedalus binary, so it works offline and needs no
external files. The agents are the units (a role plus a prompt) that, in later
phases, compile to the native format of your backend.

## The built-in agents

The catalog provides five canonical agents, one for each stage of the SDD
pipeline:

| Id | Role |
|---|---|
| `analyst` | Turns a brief into a spec/PRD. |
| `architect` | Defines the architecture from the spec. |
| `planner` | Derives epics and tickets from spec and architecture. |
| `validator` | Verifies artifacts and implementation against gates and criteria. |
| `documenter` | Produces derived documentation. |

## Listing the catalog

Use `daedalus agent list` to see every built-in agent with its id and role:

```sh
daedalus agent list
```

The agents are listed in id order:

```
Built-in agents (5):
  analyst	Turns a brief into a spec/PRD.
  architect	Defines the architecture from the spec.
  documenter	Produces derived documentation.
  planner	Derives epics and tickets from spec and architecture.
  validator	Verifies artifacts and implementation against gates and criteria.
```

## Adding an agent to your workspace

Use `daedalus agent add <id>` to materialize a catalog agent into your
workspace. The id may appear before or after the flags, so both
`daedalus agent add analyst --path ./my-repo` and
`daedalus agent add --path ./my-repo analyst` work.

```sh
daedalus agent add analyst
```

On success, Daedalus reports the directory and how many files it created:

```
Materialized agent "analyst" at .daedalus/agents/analyst (created 2 files).
```

See all options with:

```sh
daedalus agent add --help
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/agents/` the agent is added to. Defaults to the current directory. |
| `--preview` | Dry run: show the files that would be created without writing anything. |
| `--help` | Show all available options. |

### It will not overwrite your work

Adding an agent is **non-destructive**. If an agent with the same id already
exists in the workspace, Daedalus leaves the existing files — including any
manual edits — untouched and tells you nothing was overwritten:

```
Agent "analyst" already exists at .daedalus/agents/analyst — not overwritten (skipped 2 files).
```

If only some of the agent's files are present (for example, you deleted one),
Daedalus fills in the missing file and preserves the rest, reporting both:

```
Agent "analyst" partially materialized at .daedalus/agents/analyst (created 1, skipped 1 existing file).
```

### Previewing without writing

Use the `--preview` flag to perform a **dry run**. Daedalus prints the files it
would create but **writes nothing** to disk:

```sh
daedalus agent add planner --preview
```

```
Preview of materializing agent "planner" into .daedalus/agents/planner:
  + planner/agent.yaml (file)
  + planner/prompt.md (file)
```

Run the command again without `--preview` to apply the changes.

### Unknown agents

If you ask for an id that is not in the catalog, Daedalus rejects the run, exits
with status code `2`, and writes nothing. It also points you to `agent list`:

```sh
daedalus agent add bogus
```

```
daedalus: agent not found in catalog: "bogus"
run 'daedalus agent list' to see the available agents
```

## What gets created on disk

Each agent is materialized into its own directory under your workspace's
`.daedalus/agents/`, as two files:

```
.daedalus/
  agents/
    analyst/
      agent.yaml     # canonical agent definition (editable source of truth)
      prompt.md      # the agent's prompt, as Markdown
```

`agent.yaml` is the **canonical definition**: a small, diff-friendly metadata
file. The prompt itself lives alongside it in `prompt.md`, so you can edit it as
Markdown without touching the definition. A materialized `analyst/agent.yaml`
looks like this:

```yaml
# Daedalus canonical agent definition.
# Managed by Daedalus. Keys are ordered and stable for clean diffs.
# This file is the editable source of truth; the prompt lives in prompt.md.
id: analyst
version: "1"
role: Turns a brief into a spec/PRD.
prompt: prompt.md
parameters:
  model: default
```

These files are your **editable source of truth**: change the role, the prompt,
or the parameters as your project needs. The output is **deterministic** — adding
the same agent into a clean workspace always produces the same files, byte for
byte, which keeps Git diffs clean.

## Cloning an agent

`daedalus agent add` materializes a catalog agent under its own id. When you want
to start from a catalog agent but keep your own customized copy under a different
name, use `daedalus agent clone <source-id> <dest-id>`. The clone is an
**independent** copy: editing it never changes the original built-in agent.

```sh
daedalus agent clone analyst analyst-custom
```

The destination id must be `kebab-case`. On success, Daedalus reports the new
directory and how many files it created:

```
Materialized agent "analyst-custom" at .daedalus/agents/analyst-custom (created 2 files).
```

The ids may appear before or after the flags. See all options with:

```sh
daedalus agent clone --help
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/agents/` the clone is written to. Defaults to the current directory. |
| `--preview` | Dry run: show the files that would be created without writing anything. |
| `--help` | Show all available options. |

### It will not overwrite an existing clone

Cloning is **non-destructive**. If the destination id already exists in the
workspace, Daedalus leaves the existing files untouched and tells you nothing was
overwritten:

```
Agent "analyst-custom" already exists at .daedalus/agents/analyst-custom — not overwritten (skipped 2 files).
```

### Previewing a clone without writing

Use `--preview` to perform a **dry run** that prints the files it would create
but **writes nothing**:

```sh
daedalus agent clone analyst analyst-custom --preview
```

```
Preview of materializing agent "analyst-custom" into .daedalus/agents/analyst-custom:
  + analyst-custom/agent.yaml (file)
  + analyst-custom/prompt.md (file)
```

### Unknown source or invalid destination

If the source id is not in the catalog, Daedalus rejects the run, exits with
status code `2`, and points you to `agent list`:

```sh
daedalus agent clone bogus my-agent
```

```
daedalus: agent not found in catalog: "bogus"
run 'daedalus agent list' to see the available agents
```

If the destination id is not valid `kebab-case`, the run is rejected with status
code `2` and writes nothing:

```sh
daedalus agent clone analyst Bad_Id
```

```
daedalus: destination agent id "Bad_Id" is not valid kebab-case
```

## Editing an agent

Once an agent exists in your workspace — whether you added it or cloned it — you
can change its role, prompt, and parameters with `daedalus agent edit <id>`.
Edits are written directly to the agent's `agent.yaml` and `prompt.md`. You can,
of course, also edit those files by hand; this command is the scriptable
alternative.

```sh
daedalus agent edit analyst-custom --role "Drafts product specs for the mobile team"
```

On success, Daedalus confirms the change:

```
Edited agent "analyst-custom" at .daedalus/agents/analyst-custom.
```

The id may appear before or after the flags. See all options with:

```sh
daedalus agent edit --help
```

### Options

At least one edit flag is required.

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/agents/` holds the agent. Defaults to the current directory. |
| `--role <text>` | Set the agent's role/description. |
| `--prompt <text>` | Set the agent's prompt inline. |
| `--prompt-file <path>` | Set the agent's prompt from a file. Takes precedence over `--prompt` if both are given. |
| `--set-param key=value` | Add or update a parameter. **Repeatable.** |
| `--remove-param key` | Remove a parameter by key. **Repeatable.** |
| `--help` | Show all available options. |

You can combine several flags in one run. For example, set the prompt from a
file, add a parameter, and drop another:

```sh
daedalus agent edit analyst-custom \
  --prompt-file ./prompts/analyst.md \
  --set-param temperature=0.2 \
  --remove-param model
```

Parameters set through the CLI are stored as **strings** — Daedalus does not
infer number or boolean types from the value you type. (Typed parameters that
came from the built-in catalog keep their type until you edit them via the CLI.)

### Edits are validated before anything is written

An edit is checked against the [canonical schema](#agent-validation) **before** it
touches disk, and the write is **atomic**. If the result would be invalid — for
example, an empty role — Daedalus rejects the edit with status code `2`, lists
every problem it found, and leaves your existing definition completely intact
(never half-written):

```sh
daedalus agent edit analyst-custom --role ""
```

```
daedalus: agent "analyst-custom" is invalid; the edit was not applied:
  - role: observed empty; expected a non-empty role/description
```

### Editing requires at least one change

Running `edit` with no edit flag is treated as a usage error (status code `2`),
not as a silent no-op:

```sh
daedalus agent edit analyst-custom
```

```
daedalus: agent edit requires at least one edit flag (--role, --prompt, --prompt-file, --set-param, --remove-param)
```

### Editing an agent that does not exist

`edit` only works on an agent that already lives in your workspace. If the id is
not there, Daedalus rejects the run with status code `2` and tells you to create
it first:

```sh
daedalus agent edit ghost --role "anything"
```

```
daedalus: agent not found in catalog: "ghost"
the agent must already exist in the workspace; clone or add it first
```

## Importing agents

If your project already has agents defined outside Daedalus — for example a
Claude Code `.claude/agents/` directory — you do not have to rewrite them by
hand. `daedalus agent import <source>` reads those definitions and converts them
into the workspace's canonical format under `.daedalus/agents/`.

The source may be a single **file** or a **directory**, and may appear before or
after the flags:

```sh
daedalus agent import .claude/agents/reviewer.md
```

On success, Daedalus reports each imported agent and a summary:

```
  + reviewer imported to .daedalus/agents/reviewer (created 2 files).
Import summary: 1 imported, 0 already existed, 0 failed.
```

See all options with:

```sh
daedalus agent import --help
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/agents/` receives the import. Defaults to the current directory. |
| `--preview` | Dry run: show what would be imported without writing anything. |
| `--help` | Show all available options. |

### What gets recognized and how it is converted

Import understands two source formats:

- **Claude Code agents** (`.claude/agents/*.md`): a Markdown file with a YAML
  frontmatter block followed by the prompt body.
- **Canonical definitions**: an agent already in Daedalus's own format.

When importing a Claude Code agent, Daedalus maps its fields to the canonical
definition like this:

| Claude Code field | Becomes | Notes |
|---|---|---|
| `name` | the agent **id** | Normalized to `kebab-case`. If `name` is missing, the id is derived from the file name. |
| `description` | the **role** | |
| the Markdown body | the **prompt** | Everything after the closing `---`. |
| `model` | a `model` **parameter** | Only when present. |
| `tools` | *dropped* | Backend-specific to Claude Code; not part of the canonical model in Phase 1. |
| `color` | *dropped* | A Claude Code UI affordance with no canonical meaning. |

`tools` and `color` are intentionally **not** carried over: they are specific to
Claude Code and have no backend-agnostic meaning yet. When Daedalus later
compiles your canonical agents back to `.claude/`, those concerns are resolved at
that point.

### Importing a whole directory

Point `import` at a directory and Daedalus imports every valid agent it finds,
reporting each one and a final summary:

```sh
daedalus agent import .claude/agents
```

```
  + reviewer imported to .daedalus/agents/reviewer (created 2 files).
  + planner imported to .daedalus/agents/planner (created 2 files).
Import summary: 2 imported, 0 already existed, 0 failed.
```

If one source in the directory is invalid, it is reported and skipped — it does
**not** abort the valid ones (see [Invalid sources](#invalid-sources)).

### Previewing without writing

Use `--preview` to perform a **dry run** that lists what would be imported but
**writes nothing**:

```sh
daedalus agent import .claude/agents --preview
```

```
Preview of importing 2 agent(s):
  + reviewer -> .daedalus/agents/reviewer
  + planner -> .daedalus/agents/planner
```

If nothing in the source is importable, the preview says so:

```
Preview: no importable agents found.
```

### It will not overwrite existing agents

Import is **non-destructive**. If an agent id already exists in the workspace,
Daedalus leaves the existing files untouched and marks it as skipped:

```
  = reviewer already exists at .daedalus/agents/reviewer — not overwritten (skipped 2 files).
Import summary: 0 imported, 1 already existed, 0 failed.
```

### Invalid sources

A source that cannot be converted — for example, one whose role ends up empty —
fails the [canonical schema](#agent-validation) and is reported with the
offending file and every problem it found, prefixed with `!`. Other valid agents
are still imported, but the run exits with status code `2` so you notice the
failure:

```
  ! .claude/agents/broken.md: agent "broken" is invalid (1 issue):
  - role: observed empty; expected a non-empty role/description
Import summary: 0 imported, 0 already existed, 1 failed.
```

If the source **path** itself cannot be read at all (for example, it does not
exist), the run fails with status code `1` and writes nothing.

## Agent validation

Every agent definition Daedalus writes must satisfy a single **canonical
schema**. This schema is the quality gate behind all four operations above — it
runs on `add`, `clone`, `edit`, and `import` — so a definition is only ever
written when it is valid. The check looks at the definition itself; it never
executes the agent or contacts a backend.

### What the schema requires

| Field | Required | Rule |
|---|---|---|
| `id` | Yes | Non-empty, `kebab-case` (lowercase letters/digits in dash-separated segments, e.g. `my-agent`). |
| `role` | Yes | Non-empty. |
| `prompt` | Yes | Non-empty. |
| `parameters` | No | Optional. Each key must be non-empty and unique, and each value must have a known type (string, number, or bool). |
| `version` | — | Stamped by Daedalus; not something you author. |

### Actionable validation errors

When a definition is invalid, Daedalus does not just say "invalid". It reports
**every** problem at once — not only the first — so you can fix them in a single
pass. Each finding names the **field**, what was **observed**, and what was
**expected**:

```
daedalus: agent "analyst-custom" is invalid; the edit was not applied:
  - role: observed empty; expected a non-empty role/description
  - prompt: observed empty; expected a non-empty prompt
```

The findings are listed in a stable order (`id`, `role`, `prompt`, then
`parameters`), so the same definition always produces the same report.

## Notes and limitations

- The catalog ships **embedded in the binary**. A remote catalog or marketplace
  is out of scope for Phase 1.
- Adding, cloning, and importing agents are **non-destructive**: existing
  definitions and your manual edits are never overwritten.
- A clone is **independent** of the built-in agent it came from — editing the
  clone never changes the original.
- Import converts definitions only; it does **not** carry over backend-specific
  Claude Code fields (`tools`, `color`), which have no canonical meaning yet.
- Phase 1 **configures** your project's AI structure; it does not **execute**
  agents — that stays with your runtime (for example, Claude Code).
- Every operation that writes an agent validates it against the
  [canonical schema](#agent-validation) first; an invalid definition is reported
  with actionable findings and never written.
