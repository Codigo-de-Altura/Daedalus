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

You will see a confirmation with the workspace location:

```
Created Daedalus workspace at .daedalus
```

To target a different directory, use `--path`:

```sh
daedalus init --path ./my-repo
```

See all options with:

```sh
daedalus init --help
```

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

Running `init` again when the workspace already exists leaves your files
untouched and reports that nothing needed to be created.

## Limitations

- Detecting and upgrading a pre-existing workspace is handled separately; this
  command focuses on safe creation.
