# Managing specs

[← Back to the manual index](../README.md)

A **brief** is the human-authored entry point of the SDD (spec-driven
development) pipeline: a short Markdown statement of intent. The *analyst* agent
turns that brief into a **spec** (a spec/PRD) — a plan that says *what* you are
building and *why*, which the later stages of the pipeline build on. Instead of
keeping these artifacts loose in your repository, you capture them into your
workspace, version them with Git, and manage them with the `daedalus spec`
command.

Daedalus manages the **definition** of this step: it captures the brief, wires
it to the *analyst* agent, and reserves a deterministic place for the spec to
land. It does **not** generate the spec's content for you — see
[Phase 1: Daedalus does not run the agent](#phase-1-daedalus-does-not-run-the-agent)
below.

## The task, end to end

Working with a spec follows four steps. Daedalus handles the first two; the last
two are yours:

1. **Capture the brief.** Run `daedalus spec capture` to persist your brief and
   seed the spec's destination.
2. **Get the spec destination.** Daedalus wires the brief to the *analyst* step
   of the `sdd-default` workflow and creates a seeded spec file at the canonical
   path, ready to receive the generated content.
3. **Run the analyst on your backend.** Generate the spec/PRD from the brief by
   running the *analyst* agent in your own runtime (for example, Claude Code) —
   Daedalus does not do this for you.
4. **Refine the spec by hand.** Drop the generated content into the spec file
   and edit it. The spec is yours; Daedalus never overwrites it.

## Where briefs and specs live

A brief and its spec are a **pair**, sharing one `kebab-case` **slug**. Both live
side by side under your workspace's `.daedalus/specs/` directory:

```
.daedalus/
  specs/
    my-feature.brief.md   # the brief you captured
    my-feature.md         # the spec it seeded
```

The slug is the file name. The brief is `<slug>.brief.md`; the spec is
`<slug>.md`. Each is a single, diff-friendly Markdown document: a **YAML
frontmatter** block with the artifact's metadata, followed by the **body** (your
Markdown, stored verbatim).

A captured `my-feature.brief.md` looks like this:

```markdown
---
slug: my-feature
kind: brief
title: My Feature
consumed-by: analyst
workflow: sdd-default
phase: spec
---
<your brief markdown, verbatim>
```

The brief's frontmatter records the **link** Daedalus manages: it is
`consumed-by` the `analyst` agent, at the `spec` phase of the `sdd-default`
workflow — the `brief → spec/PRD` step of the
[default SDD pipeline](managing-workflows.md#the-default-sdd-workflow).

Its seeded spec, `my-feature.md`, looks like this:

```markdown
---
slug: my-feature
kind: spec
title: My Feature
brief: my-feature.brief.md
agent: analyst
workflow: sdd-default
phase: spec
generated: false
---
# My Feature

> Spec/PRD placeholder. Daedalus manages this artifact's definition but does
> not run the analyst agent (phase 1). Generate the spec by running the
> "analyst" agent (workflow "sdd-default", phase "spec") on your backend, using the brief below,
> then replace this placeholder with the result and refine it.

Source brief: my-feature.brief.md
```

The spec's frontmatter keeps the **trace** back to its origin: `brief` names the
file it was seeded from, `agent` is the *analyst* that produces it, and
`generated: false` records that Daedalus did **not** generate the body — it
seeded a placeholder for you to replace. The frontmatter keys are always written
in the same order and every key is always present, so the output is
**deterministic**: the same brief always produces the same files, byte for byte,
which keeps your Git diffs clean. These files are your **editable source of
truth** — you can edit them by hand, but the commands below are the scriptable
alternative.

## The slug

Every brief/spec pair is identified by a **slug**: a stable, unique,
`kebab-case` slug (lowercase letters and digits in dash-separated segments, e.g.
`my-feature`). The slug is the file name, so it must be unique within the
workspace. A slug that is empty or not valid `kebab-case` is rejected with an
explicit error and nothing is written.

## Capturing a brief

Use `daedalus spec capture <slug>` to persist a brief and seed its spec
destination. The `--title` flag is required; the slug may appear before or after
the flags.

```sh
daedalus spec capture my-feature --title "My Feature"
```

On success, Daedalus reports both files it created and reminds you of the next
step — running the analyst on your backend:

```
Captured brief "my-feature" at .daedalus/specs/my-feature.brief.md and seeded its spec at .daedalus/specs/my-feature.md.
Generate the spec by running the "analyst" agent on your backend, then refine .daedalus/specs/my-feature.md.
```

You can set the brief body inline with `--body`, or from a file with
`--body-file`:

```sh
daedalus spec capture my-feature \
  --title "My Feature" \
  --body-file ./notes/my-feature-brief.md
```

See all options with:

```sh
daedalus spec capture --help
```

### Options

| Option | Description |
|---|---|
| `--title <text>` | The brief's title. **Required.** |
| `--body <text>` | Set the brief body inline. |
| `--body-file <path>` | Set the brief body from a file. Takes precedence over `--body` if both are given. |
| `--path <dir>` | Target repository directory whose `.daedalus/specs/` the brief is captured into. Defaults to the current directory. |
| `--preview` | Dry run: show the files that would be created without writing anything. |
| `--help` | Show all available options. |

### Previewing without writing

Use `--preview` to perform a **dry run**. Daedalus validates the brief and lists
the files it would create, but **writes nothing** to disk:

```sh
daedalus spec capture my-feature --title "My Feature" --preview
```

```
Preview of capturing brief "my-feature" into .daedalus/specs:
  + my-feature.brief.md (brief)
  + my-feature.md (spec, seeded; generate with the analyst on your backend)
```

Run the command again without `--preview` to apply it.

### It will not overwrite your work

Capturing is **non-destructive**. If you have already captured a brief and
refined its spec, re-capturing the same slug leaves both files exactly as they
are and tells you nothing was overwritten:

```sh
daedalus spec capture my-feature --title "My Feature"
```

```
Brief "my-feature" and its spec already exist — left intact (nothing overwritten).
```

If only one half of the pair is present — for example you deleted the seeded
spec, or the spec exists but the brief was removed — Daedalus re-creates only the
missing file and preserves the one that is there:

```
Brief "my-feature" already existed; re-seeded the missing spec at .daedalus/specs/my-feature.md.
```

```
Spec for "my-feature" already existed and was preserved; re-created the missing brief at .daedalus/specs/my-feature.brief.md.
```

This is what protects a spec you have edited: re-capturing never clobbers it.

### Invalid input is rejected before anything is written

A brief is checked **before** it touches disk. If the slug or title is invalid,
Daedalus rejects the run, lists **every** problem it found — not just the first —
and writes nothing. Each finding names the field, what was observed, and what was
expected, so you can fix them in one pass:

```sh
daedalus spec capture Bad_Slug --title ""
```

```
daedalus: brief "Bad_Slug" is invalid; it was not captured:
  - slug: observed "Bad_Slug"; expected kebab-case: lowercase letters/digits in dash-separated segments (e.g. my-feature)
  - title: observed empty; expected a non-empty title
```

## Listing briefs

Use `daedalus spec list` to see every captured brief with its slug, whether a
spec has been materialized for it, and its title:

```sh
daedalus spec list
```

The briefs are listed in slug order; the middle column shows `spec` when a spec
exists alongside the brief and `no-spec` when it does not:

```
Briefs (2):
  my-feature	spec	My Feature
  other-idea	no-spec	Other Idea
```

When there are no briefs yet, the count is zero:

```
Briefs (0):
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/specs/` is listed. Defaults to the current directory. |

## Showing a brief or spec

Use `daedalus spec show <slug>` to print the **spec's** file content
**verbatim** — frontmatter and body, exactly as it is stored:

```sh
daedalus spec show my-feature
```

Add `--brief` to print the **brief** instead:

```sh
daedalus spec show my-feature --brief
```

### Options

| Option | Description |
|---|---|
| `--brief` | Show the brief (`<slug>.brief.md`) instead of the spec (`<slug>.md`). |
| `--path <dir>` | Target repository directory whose `.daedalus/specs/` holds the artifact. Defaults to the current directory. |

If the artifact does not exist, Daedalus tells you so and writes nothing:

```
daedalus: spec for "ghost" not found
```

```
daedalus: brief for "ghost" not found
```

## Editing the spec

Once a spec exists, refine it with `daedalus spec edit <slug>`. This is how you
replace the seeded placeholder with the real spec/PRD content — for example,
after running the analyst on your backend — or tweak it later. At least one edit
flag is required; the slug may appear before or after the flags.

```sh
daedalus spec edit my-feature --body-file ./generated/my-feature-spec.md
```

On success, Daedalus confirms the change:

```
Edited spec "my-feature" at .daedalus/specs/my-feature.md.
```

You can set the spec's title and body in the same run; `--body-file` reads the
body from a file, and `--body` sets it inline:

```sh
daedalus spec edit my-feature --title "My Feature (revised)" --body "..."
```

See all options with:

```sh
daedalus spec edit --help
```

### Options

At least one edit flag is required.

| Option | Description |
|---|---|
| `--title <text>` | Set the spec's title. |
| `--body <text>` | Set the spec's body inline. |
| `--body-file <path>` | Set the spec's body from a file. Takes precedence over `--body` if both are given. |
| `--path <dir>` | Target repository directory whose `.daedalus/specs/` holds the spec. Defaults to the current directory. |
| `--help` | Show all available options. |

`edit` changes **only the spec**. The brief is your authored input — refined
outside Daedalus — and is not editable through this command; edit the brief file
by hand if you need to. The spec's `brief` reference and the rest of its
provenance are preserved across an edit.

### Edits are validated before anything is written

An edit is checked **before** it touches disk, and the write is **atomic**. If
the result would be invalid — for example, an empty title — Daedalus rejects the
edit, reports the problem, and leaves your existing file completely intact (never
half-written):

```sh
daedalus spec edit my-feature --title ""
```

```
daedalus: spec "my-feature" is invalid; the edit was not applied:
  - title: observed empty; expected a non-empty title
```

### Editing requires at least one change

Running `edit` with no edit flag is treated as a usage error, not a silent no-op:

```sh
daedalus spec edit my-feature
```

```
daedalus: spec edit requires at least one edit flag (--title, --body, --body-file)
```

### Editing a spec that does not exist

`edit` only works on a spec that already lives in your workspace. If it is not
there, Daedalus rejects the run and points you to capture the brief first:

```sh
daedalus spec edit ghost --title "anything"
```

```
daedalus: spec "ghost" not found
the spec must already exist; capture its brief first
```

## Removing a brief or spec

Use `daedalus spec remove <slug>` to delete the **spec** file, or
`--brief` to delete the **brief** file. **Only** the one file you name is
removed; the other half of the pair is left intact, so you can drop and re-seed
one half deliberately.

```sh
daedalus spec remove my-feature
```

```
Removed spec "my-feature" from .daedalus/specs/my-feature.md.
```

```sh
daedalus spec remove my-feature --brief
```

```
Removed brief "my-feature" from .daedalus/specs/my-feature.brief.md.
```

### Options

| Option | Description |
|---|---|
| `--brief` | Remove the brief (`<slug>.brief.md`) instead of the spec (`<slug>.md`). |
| `--path <dir>` | Target repository directory whose `.daedalus/specs/` holds the artifact. Defaults to the current directory. |

Removing an artifact that does not exist is reported as an explicit error, not a
silent success:

```
daedalus: spec for "my-feature" not found
```

## Phase 1: Daedalus does not run the agent

Daedalus **configures** the `brief → spec/PRD` step; it does not **execute** it.
When you capture a brief, Daedalus:

- persists the brief,
- records its link to the *analyst* agent (the `spec` phase of the `sdd-default`
  workflow), and
- seeds the spec file at its canonical path, with a placeholder body.

It does **not** call a model or launch the *analyst*. Generating the spec's real
content from the brief is done **by you**, running the *analyst* agent in your
own backend (for example, Claude Code). The seeded spec says so in its body, and
its frontmatter carries `generated: false` to make that explicit. Once you have
the generated spec/PRD, drop it into the spec file — with `daedalus spec edit` or
by hand — and refine it.

## Notes and limitations

- A brief and its spec are persisted as Markdown files under `.daedalus/specs/`,
  paired by a shared `kebab-case` slug (`<slug>.brief.md` and `<slug>.md`), in a
  deterministic, git-friendly format. The same brief always renders the same
  bytes: fixed key order, every key present, and a single trailing newline.
- Capturing is **non-destructive**: it never overwrites an existing brief or a
  spec you have refined. When only one half of the pair is missing, Daedalus
  re-creates just that file and preserves the other.
- The brief is **input you author**; `daedalus spec` does not edit it. `edit`
  changes only the spec, preserving its `brief` reference and provenance.
- Bodies are stored **verbatim** as arbitrary Markdown — Daedalus does not
  interpret or rewrite them.
- Every write is validated **before** anything is written: an invalid capture or
  edit is rejected with actionable findings and leaves your files intact.
- Phase 1 **configures** the `brief → spec/PRD` step; it does not **run** the
  *analyst* agent — generating the spec stays with your runtime (for example,
  Claude Code).
</content>
</invoke>
