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

## Notes & limitations

- The catalog ships **embedded in the binary**. A remote catalog or marketplace
  is out of scope for Phase 1.
- Adding an agent is **non-destructive**: existing definitions and your manual
  edits are never overwritten.
- Phase 1 **configures** your project's AI structure; it does not **execute**
  agents — that stays with your runtime (for example, Claude Code).
- Validating agent definitions against the canonical schema is covered by a
  later feature; today the built-in agents are already valid by construction.
