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
# Generated from the built-in catalog. Keys are ordered and stable for clean diffs.
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

An edit is checked **before** it touches disk, and the write is **atomic**. If
the result would be invalid — for example, an empty role — Daedalus rejects the
edit with status code `2` and leaves your existing definition completely intact
(never half-written):

```sh
daedalus agent edit analyst-custom --role ""
```

```
daedalus: invalid edit to agent "analyst-custom": agent "analyst-custom" has an empty role
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

## Notes & limitations

- The catalog ships **embedded in the binary**. A remote catalog or marketplace
  is out of scope for Phase 1.
- Adding and cloning an agent are **non-destructive**: existing definitions and
  your manual edits are never overwritten.
- A clone is **independent** of the built-in agent it came from — editing the
  clone never changes the original.
- Phase 1 **configures** your project's AI structure; it does not **execute**
  agents — that stays with your runtime (for example, Claude Code).
- Edits are checked structurally before writing, but full validation against the
  canonical schema is covered by a later feature; until then, an edit that keeps
  the role, prompt, and parameters well-formed is accepted.
