# Quickstart

[← Back to the manual index](../README.md)

This walkthrough takes you from zero to a **compiled workspace** in a few
commands. Follow it top to bottom in a throwaway directory and you will end with
an initialized `.daedalus/` workspace and a generated `.claude/settings.json`.
Every command and output below matches Daedalus's actual behavior.

## 1. Install and verify

Install Daedalus with the one-line script (see [Installation](installation.md)
for pinning a version, manual downloads, and building from source):

```sh
# Linux and macOS
curl -fsSL https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.sh | sh
```

```powershell
# Windows (PowerShell)
irm https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.ps1 | iex
```

Then confirm it runs:

```sh
daedalus --version
# daedalus 0.1.0-dev
```

## 2. Initialize a workspace

Move to the repository you want to manage and create the `.daedalus/`
workspace:

```sh
cd /path/to/your-repo
daedalus init
```

```
Created Daedalus workspace at .daedalus from scratch.
Seeded factory workflow "sdd-default" at .daedalus/workflows/sdd-default.yaml.
```

This creates the canonical `.daedalus/` structure — the backend-agnostic source
of truth for your project's AI scaffolding — and seeds the default SDD workflow,
`sdd-default`, so you start with a ready-to-use pipeline. It is safe to run in an
existing repository: it never touches files outside `.daedalus/`.

## 3. Inspect the workspace

A freshly initialized workspace contains the canonical layout, the manifest, the
project guideline, and the seeded workflow:

```
.daedalus/
  agents/
  prompts/
  workflows/
    sdd-default.yaml   # the seeded factory workflow
  specs/
  architecture/
  epics/
  tickets/
  docs/
  .state/
  daedalus.yaml        # the workspace manifest
  init.md              # the project guideline
```

See [Concepts](../guide/concepts.md) for what each part is, and
[Initializing a workspace](../guide/initializing-a-workspace.md) for `init`'s
options.

## 4. Validate

Check that the workspace follows the conventions and that its definitions are
well-formed. A fresh workspace passes both axes clean:

```sh
daedalus validate
```

```
Conventions: workspace conforms (no violations).
Definitions: all agents, workflows and manifest are valid.
```

Exit code `0` means the workspace is ready to compile.

## 5. Preview the build

`daedalus build` compiles the canonical definition into your backend's native
format. Preview it first — `--preview` shows the diff and writes nothing. A bare
workspace has just one artifact to compile, `.claude/settings.json`:

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

## 6. Build

In an interactive terminal, run `daedalus build` and confirm at the gate. To
compile non-interactively (or from a script/CI), pass `--yes`:

```sh
daedalus build --yes
```

```
Compiled .:
  claude-code: 1 created, 0 updated, 0 unchanged (of 1 artifact)
    + .claude/settings.json
```

You now have a compiled workspace: `.claude/settings.json` exists. Re-running the
build with nothing changed is an idempotent no-op:

```sh
daedalus build --yes
```

```
Compiled .:
  claude-code: 0 created, 0 updated, 1 unchanged (of 1 artifact)
```

## Where to go next

- [Concepts](../guide/concepts.md) — the workspace, the canonical model, and
  compilation.
- [Core workflow](../guide/core-workflow.md) — the everyday edit → validate →
  build loop.
- [Command reference](../guide/command-reference.md) — every command, its flags,
  and exit codes.
- [Examples](../guide/examples.md) — add an agent, a prompt, and a workflow, then
  compile.
