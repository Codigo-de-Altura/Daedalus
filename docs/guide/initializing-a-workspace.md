# Initializing a workspace

[← Back to the manual index](../README.md)

`daedalus init` creates the canonical `.daedalus/` workspace inside your
repository. This workspace is the single, backend-agnostic source of truth for
your project's AI structure (agents, prompts, workflows, and the SDD backlog).

## Usage

From the root of your repository:

```sh
daedalus init
```

When the workspace does not exist yet, you will see a confirmation that it was
created from scratch:

```
Created Daedalus workspace at .daedalus from scratch.
```

To target a different directory, use `--path`:

```sh
daedalus init --path ./my-repo
```

See all options with:

```sh
daedalus init --help
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Directory to initialize. Defaults to the current directory. |
| `--preview` | Dry run: show what would be created or added without writing anything. |
| `--help` | Show all available options. |

## What it creates

```
.daedalus/
  daedalus.yaml     # project manifest
  init.md           # project guideline
  agents/           # agent definitions
  prompts/          # shared prompts
  workflows/        # DAG workflows
  specs/            # specifications / PRD
  architecture/     # architecture documents
  epics/            # epics
  tickets/          # tickets
  docs/             # derived documentation
  .state/           # progress state (tracked in git)
```

> The manifest (`daedalus.yaml`) and `init.md` are created as part of the
> structure; their content is filled in by later steps.

## Safe to run on existing repositories

`init` is **non-destructive**: it never modifies or deletes anything outside the
`.daedalus/` directory it creates, so it is safe to run in a repository that
already contains your code. It is also **deterministic** — the same repository
always produces the same structure, which keeps Git diffs clean.

## Re-running `init`: detect & upgrade

`init` is safe to run as many times as you like. When a `.daedalus/` workspace
already exists, Daedalus detects it and performs a **non-destructive upgrade**
instead of creating a new one. It never overwrites or deletes files and folders
that contain your own content — it only **adds whatever is missing** so the
workspace structure stays complete.

There are two possible outcomes:

**The workspace is already complete.** Nothing needs to change, so Daedalus
leaves every file untouched and reports that the workspace is up to date:

```
Existing Daedalus workspace at .daedalus is already complete — nothing to update.
```

**The workspace is missing some pieces.** Daedalus first prints a preview of
exactly what it will add, then completes the structure and reports how many
directories and files it created:

```
Preview of changes to the Daedalus workspace at .daedalus:
  + docs/ (directory)
  + init.md (file)
Upgraded existing Daedalus workspace at .daedalus (added 1 directories, 1 files).
```

Because the upgrade only fills in what is absent, any edits you have made inside
`.daedalus/` — for example, content you added to `init.md` or files you created
under `agents/` — are always preserved.

## Previewing changes without writing

Use the `--preview` flag to perform a **dry run**. Daedalus inspects the target
directory and prints the same preview of what it would create or add, but
**writes nothing** to disk:

```sh
daedalus init --path ./my-repo --preview
```

```
Preview of changes to the Daedalus workspace at ./my-repo:
  + docs/ (directory)
  + init.md (file)
```

This lets you see what an `init` run would do before committing to it. Run the
command again without `--preview` to apply the changes.
