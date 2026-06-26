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
definition, and — **in an interactive terminal** — shows you a
[preview of every change and asks you to confirm](#previewing-and-confirming-changes)
before writing anything.

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
| `--preview` | Show the diff/preview and **exit without writing** anything (a read-only dry run). |
| `--yes` | Write **without** the interactive confirmation, for scripts and CI. See [Writing from a script or CI](#writing-from-a-script-or-ci). |
| `--help` | Show all available options. |

> By default, running `build` in a terminal opens an interactive preview and
> writes only after you confirm — nothing is written silently. The exact behavior
> depends on whether you are in a terminal and which flags you pass; see
> [Previewing and confirming changes](#previewing-and-confirming-changes).

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

## Previewing and confirming changes

`build` never writes silently. In an interactive terminal it first opens a
**preview** of every change and waits for your explicit confirmation before it
touches disk — so you always see exactly what would change and decide whether to
apply it.

### The interactive preview

Run `build` in a terminal:

```sh
daedalus build
```

Daedalus computes the full plan and opens a preview screen. At the top is a
**summary** of the counts per backend; below it, the artifacts are listed and
classified as **new**, **modified**, or **unchanged**; and for the artifact you
select, the **content diff** is shown — added lines prefixed with `+`, removed
lines with `-`. At the bottom is the **confirmation gate**:

```
claude-code: 1 new, 1 modified, 3 unchanged

  [new]        commands/summarize.md
  [modified]   agents/reviewer.md
  [unchanged]  agents/planner.md
  [unchanged]  settings.json
  [unchanged]  commands/release-notes.md

  agents/reviewer.md
  - You are the code reviewer. Report correctness and style issues.
  + You are the code reviewer. Inspect the proposed changes and report any
  + correctness, security, or style issues, worst-first.

Write these changes? y/enter to confirm · n/esc to cancel (nothing is written)
```

Controls:

- **↑ / ↓** — move between artifacts in the list.
- **pgup / pgdn**, and **g / G** — scroll the diff of the selected artifact (top
  and bottom).
- **y** or **enter** — confirm and write the changes.
- **n** or **esc** — cancel; **nothing is written** and your repository is left
  exactly as it was.
- **ctrl+c** — exit at any time.

If there is nothing to write, the preview says so plainly and there is nothing to
confirm:

```
No changes — every artifact is already up to date.
```

#### Orphans in the preview

Any [orphans](#orphans-are-reported-never-deleted) — generated files whose
canonical source you removed — appear in their own read-only section, headed
**“Orphans — left untouched · not selectable.”** They are shown for your
awareness only: you cannot navigate into them, and Daedalus never deletes them.
Remove an orphan yourself if you no longer want it.

### Preview only, never write (`--preview`)

Use `--preview` to inspect the plan **without the option to write**:

```sh
daedalus build --preview
```

In a terminal this opens the same preview screen in **read-only** mode — you can
browse the list and diffs, but there is no confirmation gate, and the footer
reads *“Read-only preview — nothing will be written.”* Outside a terminal it
prints the plan and diff as text. Either way, `--preview` never writes.

### Writing from a script or CI

When `build` runs **without an interactive terminal** — a script, CI, or a
container with no TTY — there is no one to confirm at the gate. To keep the safe
default, a plain `build` in that situation **writes nothing**: it prints the plan
and diff, then tells you how to proceed:

```
Nothing written; pass --yes to write, or run in a terminal to confirm.
```

To actually write from automation, pass `--yes`, which skips the interactive gate
and compiles directly:

```sh
daedalus build --yes
```

`--yes` is the non-interactive write path; it works with or without a terminal.
(If you pass both `--preview` and `--yes`, `--preview` wins — an explicit dry-run
request never writes.)

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
`.daedalus/` always produces the same output, byte for byte. File names are in
**kebab-case**, derived from each item's canonical id, and stay stable from one
build to the next.

You keep editing the clean, backend-agnostic definition in `.daedalus/`, and
Daedalus generates the native files for you — you never hand-edit the generated
output to keep it in sync.

### The Claude Code backend → `.claude/`

When `daedalus.yaml` targets **Claude Code**, `build` writes the `.claude/`
structure that Claude Code reads:

```
.claude/
  agents/
    <agent-id>.md       # one file per canonical agent
  commands/
    <prompt-id>.md      # one file per workspace prompt (your slash commands)
  settings.json         # a minimal, Daedalus-managed settings file
```

#### Agents → `.claude/agents/<id>.md`

Each agent in your workspace becomes one Markdown file under `.claude/agents/`.
The file has a small **frontmatter** block followed by the agent's prompt as its
body. The frontmatter carries:

- `name` — the agent's canonical id;
- `description` — the agent's role;
- `model` — included **only** when the agent defines that parameter.

For an agent whose id is `reviewer`, role is `Reviews code changes`, and prompt
is its body, `build` generates:

```md
---
name: reviewer
description: Reviews code changes
model: opus
---
You are the code reviewer. Inspect the proposed changes and report any
correctness, security, or style issues, worst-first.
```

> Daedalus writes only what your canonical definition actually specifies. In this
> release the agent frontmatter does **not** include `tools` or `color` — they are
> not generated.

#### Prompts → `.claude/commands/<id>.md` (your slash commands)

In Claude Code, the files under `.claude/commands/` are its **slash commands**.
Daedalus builds them from the **prompts** in your workspace: **every prompt in
`.daedalus/prompts/`** — both `global` and `shared` prompts — is compiled into one
command file. This is the key correspondence to keep in mind:

> **A workspace prompt becomes a Claude Code slash command.** Author a prompt
> once in `.daedalus/prompts/`, and after a build you can invoke it as a slash
> command in Claude Code.

Each command file's name derives from the prompt's id. Its frontmatter carries a
single key, `description`, set to the prompt's **title** — and that key is
**omitted entirely** when the prompt has no title. The body is the **resolved**
prompt: any inclusions the prompt references are already expanded inline, so the
command is self-contained.

For a prompt with id `summarize` and title `Summarize a document`, `build`
generates:

```md
---
description: Summarize a document
---
Summarize the document below in five bullet points, then give a one-line
takeaway.
```

#### Settings → `.claude/settings.json`

`build` writes a **minimal, honest** `settings.json`. It contains the official
Claude Code `$schema` and a `daedalus` marker noting that these files are managed
by Daedalus — nothing more:

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "daedalus": {
    "managed": true,
    "generator": "daedalus"
  }
}
```

Daedalus deliberately does **not** fabricate `permissions`, `env`, `hooks`, or a
`model` here — it never writes configuration you did not define yourself. You stay
in control of those settings.

## Re-running build: idempotent and non-destructive

`build` is safe to run as often as you like. Every run classifies each artifact it
manages as **created**, **updated**, or **unchanged**, and prints a short summary
so you can see exactly what happened — with no surprises and no noise.

### Re-running with no changes does nothing

If you run `build` again without touching `.daedalus/`, the output is already
correct, so Daedalus rewrites **nothing**. Every artifact is reported as
`unchanged`, the files stay byte-identical, and there is no churn in your working
tree (or your Git diff):

```
Compiled .:
  claude-code: 0 created, 0 updated, 5 unchanged (of 5 artifacts)
```

This is the **idempotency** guarantee: the same `.daedalus/` always yields the
same `.claude/`, run after run.

### Changing part of the definition updates only what changed

When you edit one part of your canonical definition — say, the prompt of a single
agent — the next build updates **only** the artifact(s) affected and leaves the
rest `unchanged`. The summary lists each created artifact with a `+` and each
updated one with a `~`:

```
Compiled .:
  claude-code: 0 created, 1 updated, 4 unchanged (of 5 artifacts)
    ~ agents/reviewer.md
```

So a small change to your definition produces a correspondingly small, scoped
change to the generated files — never a wholesale rewrite.

### Your own files are preserved

`build` only manages the files it **produces** — its **managed area**. Any file
you place by hand that the build does not generate is **preserved intact**,
whether it sits inside `.claude/` or anywhere else in your repository. `build`
never deletes or overwrites content outside its managed area. This is the safe
default: re-compiling can never destroy your manual work.

### Orphans are reported, never deleted

If you remove the canonical source of an artifact that an earlier build generated
— for example, you delete an agent from your workspace — the previously generated
native file becomes an **orphan**: a file the current build no longer produces.
Daedalus **detects** orphans and **reports** them in the summary, marked with a
`?`, but it does **not** delete them — so nothing is ever removed behind your
back:

```
Compiled .:
  claude-code: 0 created, 0 updated, 4 unchanged (of 4 artifacts)
    ? agents/reviewer.md (orphan: no longer produced; left untouched)
```

An orphan is harmless — it is simply a leftover file Daedalus no longer manages.
If you no longer want it, **delete it yourself**; Daedalus leaves that decision to
you.

To review these classifications artifact by artifact — and the exact content
changes — before anything is written, use the
[interactive preview](#previewing-and-confirming-changes).

## Notes and limitations

- **Deterministic.** The same workspace state always produces the same output,
  byte for byte, with stable, kebab-case file names derived from canonical ids.
- **Idempotent.** Re-running `build` with no canonical changes rewrites nothing —
  every artifact is reported `unchanged` and the files stay byte-identical, so
  there is no churn in your working tree.
- **Non-destructive.** `build` manages only the files it produces; your own
  hand-made files are preserved, and orphaned artifacts (whose canonical source
  you removed) are reported but never deleted.
- **Nothing is written without your say-so.** In a terminal, `build` shows an
  interactive preview and writes only after you confirm; cancel and nothing
  changes. Without a terminal it writes nothing unless you pass `--yes`, and
  `--preview` never writes.
- **Validate-first, all-or-nothing.** If the workspace is missing, the canonical
  definition is invalid, or the configured backend has no adapter, `build` aborts
  and writes nothing — your repository is left untouched.
- **Backend comes from the manifest.** `build` compiles to whatever backend is
  recorded in `daedalus.yaml`; set it with `daedalus init --backend` (see
  [Choosing a backend](initializing-a-workspace.md#choosing-a-backend)). Claude
  Code is the implemented backend in this release.
- **Prompts are your slash commands.** Every prompt in `.daedalus/prompts/`
  (global and shared) is compiled into a Claude Code slash command under
  `.claude/commands/`.
- **Daedalus generates only what you defined.** The Claude Code settings file is
  intentionally minimal — Daedalus never writes `permissions`, `env`, `hooks`, or
  a default `model` you did not specify, and agent frontmatter omits `tools` and
  `color`.
- **Phase 1: Daedalus configures the AI structure; it does not execute agents.**
  After building, you run the agents yourself in your chosen backend (for
  example, Claude Code).
