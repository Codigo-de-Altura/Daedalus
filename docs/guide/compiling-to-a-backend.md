# Compiling to a backend

[← Back to the manual index](../README.md)

Your `.daedalus/` workspace is the **canonical, backend-agnostic** definition of
your project's AI structure — agents, prompts, workflows, and the SDD backlog.
Before an agent tool such as Claude Code can use it, that definition has to be
**compiled** into the tool's own native format. The `daedalus build` command (and
its alias `daedalus sync`) does exactly that: it reads your canonical
definition, validates it, and writes the native artifacts for the backend you
configured in the manifest.

## The task, end to end

- **You have** a `.daedalus/` workspace (created with
  [`daedalus init`](initializing-a-workspace.md)) and a target backend recorded
  in [`daedalus.yaml`](configuration.md#the-workspace-manifest-daedalusyaml).
- **You run** `daedalus build` from inside the repository.
- **Daedalus** validates the canonical definition, selects the adapter for your
  configured backend, and compiles the definition into that backend's native
  format — or stops with an actionable error and **writes nothing** if anything
  is wrong.

## Usage

From the root of a repository that contains a `.daedalus/` workspace:

```sh
daedalus build
```

`sync` is an exact alias and behaves identically — use whichever name you prefer:

```sh
daedalus sync
```

Daedalus reads the target backend from `daedalus.yaml`, validates your canonical
definition, and compiles it into that backend's native format.

To target a repository in a different directory, use `--path`:

```sh
daedalus build --path ./my-repo
```

See all options with:

```sh
daedalus build --help
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Repository directory to compile. Defaults to the current directory. |
| `--preview` | Dry run: compute the result **without writing** anything to disk. |
| `--help` | Show all available options. |

## Safe by design: validate first, write nothing on error

`build` is built to fail **before** it touches your files, never half-way
through. In each of the cases below it stops and reports an actionable error, and
**no artifacts are written**:

- **No workspace.** If there is no `.daedalus/` workspace in the target
  directory, `build` aborts and points you at
  [`daedalus init`](initializing-a-workspace.md) to create one first.
- **Invalid canonical definition.** Before compiling, `build` validates your
  canonical definition. If it finds problems, it reports them and stops without
  writing — so a broken definition can never produce broken output. Fix the
  reported issues and run the command again.
- **No adapter for the backend.** If the backend configured in `daedalus.yaml`
  has no registered adapter, `build` fails with a clear message and writes
  nothing.

This validate-first behavior means a failed `build` always leaves your repository
exactly as it was.

## Previewing without writing

Use `--preview` to perform a **dry run**: Daedalus computes what the build would
do but **writes nothing** to disk.

```sh
daedalus build --preview
```

This lets you check that the canonical definition is valid and resolves to a
backend before you commit to writing anything. Run the command again without
`--preview` to apply the result.

> An interactive preview-and-confirm step is planned for a later release; this
> chapter will be expanded to cover it when it ships.

## Exit codes

`build` sets a distinct exit code for each outcome, so you can gate on it from a
script or CI:

| Exit code | Meaning |
|---|---|
| `0` | Success — the canonical definition was compiled to the configured backend. |
| `2` | Usage error — an invalid flag or argument. |
| `3` | Validation error — the canonical definition is invalid; nothing was written. |
| `4` | Compilation or write error (for example, no adapter for the configured backend); nothing was written. |

## What it produces

`build` compiles your canonical `.daedalus/` definition into the native format of
the backend recorded in `daedalus.yaml`. The build is **deterministic**: the same
workspace always produces the same result.

> The exact native artifacts generated for Claude Code — the mapping from your
> canonical definition to the files Claude Code reads — are documented together
> with the Claude Code adapter, in a forthcoming section of this chapter. This
> release establishes the `build`/`sync` command itself: its flags, its
> validate-first safety behavior, and its exit codes.

## Notes & limitations

- **Deterministic.** The same workspace state always produces the same result.
- **Validate-first, all-or-nothing.** If the workspace is missing, the canonical
  definition is invalid, or the configured backend has no adapter, `build` aborts
  and writes nothing — your repository is left untouched.
- **Backend comes from the manifest.** `build` compiles to whatever backend is
  recorded in `daedalus.yaml`; set it with `daedalus init --backend` (see
  [Choosing a backend](initializing-a-workspace.md#choosing-a-backend)).
- **Phase 1: Daedalus configures the AI structure; it does not execute agents.**
  After building, you run the agents yourself in your chosen backend (for
  example, Claude Code).
