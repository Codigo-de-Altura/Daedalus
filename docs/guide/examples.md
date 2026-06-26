# Examples

[← Back to the manual index](../README.md)

Realistic, end-to-end scenarios that string the commands together. Every command
and output below matches Daedalus's actual behavior. For the full flags and exit
codes of each command, see the [Command reference](command-reference.md).

## Start a new project

Move into the repository you want to manage and create the workspace:

```sh
cd /path/to/your-repo
daedalus init
```

```
Created Daedalus workspace at .daedalus from scratch.
Seeded factory workflow "sdd-default" at .daedalus/workflows/sdd-default.yaml.
```

You now have a `.daedalus/` workspace with the canonical layout, a manifest, the
project `init.md`, and the seeded `sdd-default` workflow. Commit it with your
project.

## Add an agent

List the built-in catalog, then materialize the one you want:

```sh
daedalus agent list
```

```
Built-in agents (5):
  analyst	Turns a brief into a spec/PRD.
  architect	Defines the architecture from the spec.
  documenter	Produces derived documentation.
  planner	Derives epics and tickets from spec and architecture.
  validator	Verifies artifacts and implementation against gates and criteria.
```

```sh
daedalus agent add analyst
```

```
Materialized agent "analyst" at .daedalus/agents/analyst (created 2 files).
```

The agent's editable source of truth (`agent.yaml` and `prompt.md`) now lives
under `.daedalus/agents/analyst/`. See [Managing agents](managing-agents.md) for
cloning, editing, and importing.

## Add a prompt

Author a reusable prompt — for Claude Code, a prompt becomes a slash command
after you build:

```sh
daedalus prompt create summarize --kind global --title "Summarize a document"
```

```sh
daedalus prompt list
```

See [Managing prompts](managing-prompts.md) for editing, composition, and
rendering.

## Define a workflow

A fresh workspace already has the seeded `sdd-default` workflow:

```sh
daedalus workflow list
```

```
sdd-default	6 phases
```

Create your own and add a phase to it:

```sh
daedalus workflow create release-pipeline
daedalus workflow add-phase release-pipeline --id draft --agent analyst --gate review
```

See [Managing workflows](managing-workflows.md) for the phase schema, editing,
and the DAG validation.

## Validate the workspace

Before compiling — or in CI — check that the workspace follows the conventions
and that the definitions are well-formed:

```sh
daedalus validate
```

```
Conventions: workspace conforms (no violations).
Definitions: all agents, workflows and manifest are valid.
```

Exit `0`. A clean run is your signal that the workspace is ready to compile. If
`validate` reports a finding, see [Troubleshooting](troubleshooting.md).

## Compile to your backend

Preview what the build would write, then write it:

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

In an interactive terminal, run `daedalus build` and confirm at the gate. From a
script or CI, pass `--yes`:

```sh
daedalus build --yes
```

```
Compiled .:
  claude-code: 1 created, 0 updated, 0 unchanged (of 1 artifact)
    + .claude/settings.json
```

As you add agents and prompts, the build produces more artifacts (one
`.claude/agents/<id>.md` per agent, one `.claude/commands/<id>.md` per prompt).
A re-build with nothing changed is an idempotent no-op:

```sh
daedalus build --yes
```

```
Compiled .:
  claude-code: 0 created, 0 updated, 1 unchanged (of 1 artifact)
```

See [Compiling to a backend](compiling-to-a-backend.md) for the interactive
preview and the non-destructive guarantees.

## Read a validation report

When `validate` finds a problem, it prints one finding per line. For example, a
manifest whose `backends` lists an unsupported value:

```sh
daedalus validate
```

```
Conventions: workspace conforms (no violations).
Definitions: 1 error and 0 warnings:
  - [error] .daedalus/daedalus.yaml: backends[nonexistent-backend]: schema: observed unsupported backend "nonexistent-backend"; expected one of the supported backends: claude-code
```

Exit `1`. Read the finding left to right:

- **`[error]`** — the severity (errors fail the check; warnings do not).
- **`.daedalus/daedalus.yaml`** — the file at fault.
- **`backends[nonexistent-backend]`** — the exact spot inside it.
- **`schema`** — the rule that was broken.
- **observed vs. expected** — what was found and what was expected.

Fix the manifest to a supported backend (`claude-code`) and run `daedalus
validate` again to confirm the workspace conforms. See
[Troubleshooting](troubleshooting.md) for more error patterns.

## Verify the backlog's traceability

Once you have specs, epics, and tickets, confirm the chain is consistent:

```sh
daedalus trace verify
```

```
Traceability chain is consistent (no inconsistencies).
```

Navigate it from any artifact with `daedalus trace show <id>`. See
[Tracing the backlog](tracing-the-backlog.md).
