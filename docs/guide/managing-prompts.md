# Managing prompts

[← Back to the manual index](../README.md)

**Prompts** are reusable pieces of text that feed your project's AI structure —
the guidelines, conventions, and shared fragments your agents build on. Instead
of rewriting the same style rules, glossary, or role definitions in every
project, you keep them as prompts in your workspace, version them with Git, and
edit them with the `daedalus prompt` command.

Daedalus manages two **kinds** of prompt:

| Kind | What it is for |
|---|---|
| `global` | Project-wide guidelines that apply across the board — for example style, language, or SDD conventions. |
| `shared` | Reusable fragments other prompts and agents can reference — for example a glossary, role definitions, or a commit policy. |

Both kinds live side by side in your workspace and never collide: each prompt is
a single file, identified by a unique id.

## Where prompts live

Every prompt is persisted as one Markdown file under your workspace's
`.daedalus/prompts/` directory, named after its id:

```
.daedalus/
  prompts/
    project-style.md
    glossary.md
```

Each file is a small, diff-friendly Markdown document: a **YAML frontmatter**
block with the prompt's metadata, followed by the **body** (your Markdown,
stored verbatim). A `project-style.md` global prompt looks like this:

```markdown
---
id: project-style
kind: global
title: Project Style
description: House writing and code conventions
---
Write in clear, concise English. Prefer short sentences. Document only
behavior that exists.
```

The frontmatter keys are always written in the same order — `id`, `kind`,
`title`, then `description` — and `description` is omitted entirely when you do
not set one. The body is stored exactly as you wrote it. This makes the output
**deterministic**: the same prompt always produces the same file, byte for byte,
which keeps your Git diffs clean. These files are your **editable source of
truth** — you can edit them by hand, but the commands below are the scriptable
alternative.

## The id

Every prompt is identified by an **id**: a stable, unique, `kebab-case` slug
(lowercase letters and digits in dash-separated segments, e.g. `project-style`).
The id is the file name, so it must be unique within the workspace. An id that
is empty or not valid `kebab-case` is rejected with an explicit error and nothing
is written.

## Listing prompts

Use `daedalus prompt list` to see every persisted prompt with its id, kind, and
title:

```sh
daedalus prompt list
```

The prompts are listed in id order:

```
Prompts (2):
  glossary	shared	Project Glossary
  project-style	global	Project Style
```

Filter by kind with `--kind global` or `--kind shared`:

```sh
daedalus prompt list --kind global
```

```
Prompts (1, kind=global):
  project-style	global	Project Style
```

### Options

| Option | Description |
|---|---|
| `--kind <global\|shared>` | Show only prompts of that kind. Defaults to all. |
| `--path <dir>` | Target repository directory whose `.daedalus/prompts/` is listed. Defaults to the current directory. |

## Creating a prompt

Use `daedalus prompt create <id>` to add a new prompt. The `--kind` and
`--title` flags are required; the id may appear before or after the flags.

```sh
daedalus prompt create project-style --kind global --title "Project Style"
```

On success, Daedalus reports the kind and the file it created:

```
Created prompt "project-style" (global) at .daedalus/prompts/project-style.md.
```

You can set the body inline with `--body`, or from a file with `--body-file`,
and add an optional one-line `--description`:

```sh
daedalus prompt create glossary \
  --kind shared \
  --title "Project Glossary" \
  --description "Shared domain vocabulary" \
  --body-file ./notes/glossary.md
```

See all options with:

```sh
daedalus prompt create --help
```

### Options

| Option | Description |
|---|---|
| `--kind <global\|shared>` | The prompt's kind. **Required.** |
| `--title <text>` | The prompt's title. **Required.** |
| `--description <text>` | An optional one-line description. |
| `--body <text>` | Set the prompt body inline. |
| `--body-file <path>` | Set the prompt body from a file. Takes precedence over `--body` if both are given. |
| `--path <dir>` | Target repository directory whose `.daedalus/prompts/` the prompt is added to. Defaults to the current directory. |
| `--preview` | Dry run: show the file that would be created without writing anything. |
| `--help` | Show all available options. |

### Previewing without writing

Use `--preview` to perform a **dry run**. Daedalus validates the prompt and
prints the exact file it would write, but **writes nothing** to disk:

```sh
daedalus prompt create project-style --kind global --title "Project Style" --preview
```

```
Preview of creating prompt "project-style" (global) at .daedalus/prompts/project-style.md:
---
id: project-style
kind: global
title: Project Style
---
```

Run the command again without `--preview` to apply it.

### It will not overwrite your work

Creating a prompt is **non-destructive**. If a prompt with the same id already
exists, Daedalus leaves the existing file — including any manual edits —
untouched and reports the conflict instead of overwriting it:

```sh
daedalus prompt create project-style --kind global --title "Something else"
```

```
daedalus: prompt already exists: "project-style" — not overwritten
```

### Invalid input is rejected before anything is written

A prompt is checked against its schema **before** it touches disk. If the id,
kind, or title is invalid, Daedalus rejects the run, lists **every** problem it
found — not just the first — and writes nothing. Each finding names the field,
what was observed, and what was expected, so you can fix them in one pass:

```sh
daedalus prompt create Bad_Id --kind nope --title ""
```

```
daedalus: prompt "Bad_Id" is invalid; it was not created:
  - id: observed "Bad_Id"; expected kebab-case: lowercase letters/digits in dash-separated segments (e.g. my-prompt)
  - kind: observed "nope"; expected one of: global, shared
  - title: observed empty; expected a non-empty title
```

## Showing a prompt

Use `daedalus prompt show <id>` to print a prompt's file content **verbatim** —
frontmatter and body, exactly as it is stored:

```sh
daedalus prompt show project-style
```

```
---
id: project-style
kind: global
title: Project Style
description: House writing and code conventions
---
Write in clear, concise English. Prefer short sentences. Document only
behavior that exists.
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/prompts/` holds the prompt. Defaults to the current directory. |

If the id does not exist, Daedalus tells you so and writes nothing:

```
daedalus: prompt "ghost" not found
```

## Editing a prompt

Once a prompt exists, change its title, description, or body with
`daedalus prompt edit <id>`. At least one edit flag is required; the id may
appear before or after the flags. Edits are written directly to the prompt's
file.

```sh
daedalus prompt edit project-style --title "House Style"
```

On success, Daedalus confirms the change:

```
Edited prompt "project-style" at .daedalus/prompts/project-style.md.
```

You can combine flags in one run — for example, set a new description and
replace the body from a file:

```sh
daedalus prompt edit project-style \
  --description "House writing and code conventions" \
  --body-file ./notes/style.md
```

Passing `--description ""` **clears** the description. See all options with:

```sh
daedalus prompt edit --help
```

### Options

At least one edit flag is required.

| Option | Description |
|---|---|
| `--title <text>` | Set the prompt's title. |
| `--description <text>` | Set the prompt's description. An empty value clears it. |
| `--body <text>` | Set the prompt's body inline. |
| `--body-file <path>` | Set the prompt's body from a file. Takes precedence over `--body` if both are given. |
| `--path <dir>` | Target repository directory whose `.daedalus/prompts/` holds the prompt. Defaults to the current directory. |
| `--help` | Show all available options. |

The prompt's `id` and `kind` are part of its identity and are not editable
through `edit`.

### Edits are validated before anything is written

An edit is checked against the schema **before** it touches disk, and the write
is **atomic**. If the result would be invalid — for example, an empty title —
Daedalus rejects the edit, reports the problem, and leaves your existing file
completely intact (never half-written):

```sh
daedalus prompt edit project-style --title ""
```

```
daedalus: prompt "project-style" is invalid; the edit was not applied:
  - title: observed empty; expected a non-empty title
```

### Editing requires at least one change

Running `edit` with no edit flag is treated as a usage error, not a silent
no-op:

```sh
daedalus prompt edit project-style
```

```
daedalus: prompt edit requires at least one edit flag (--title, --description, --body, --body-file)
```

### Editing a prompt that does not exist

`edit` only works on a prompt that already lives in your workspace. If the id is
not there, Daedalus rejects the run and tells you to create it first:

```sh
daedalus prompt edit ghost --title "anything"
```

```
daedalus: prompt not found: "ghost"
the prompt must already exist; create it first
```

## Removing a prompt

Use `daedalus prompt remove <id>` to delete a prompt. **Only** that prompt's
file is removed; no other file in the workspace is touched.

```sh
daedalus prompt remove glossary
```

```
Removed prompt "glossary" from .daedalus/prompts/glossary.md.
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/prompts/` holds the prompt. Defaults to the current directory. |

Removing a prompt that does not exist is reported as an explicit error, not a
silent success:

```
daedalus: prompt not found: "glossary"
```

## Composing prompts (includes)

To keep your prompts **DRY**, one prompt can pull in the content of another
instead of copying it. A shared fragment — a glossary, a style guide, a role
definition — lives in exactly one file and is reused **by reference**. When you
render the prompt, Daedalus expands every reference into a single composed text.
The source files are never modified.

### The include directive

To include another prompt, put an **include directive** on a line of its own in
the body:

```
{{include: <id>}}
```

where `<id>` is the `kebab-case` id of the prompt to pull in. The rules are
deliberately simple:

- The directive must be the **only content on its line** (surrounding whitespace
  and indentation are ignored). A line that has other text alongside the
  `{{include: ...}}` token is **not** a directive — it is left exactly as
  written. This lets you mention the literal token in prose without it being
  expanded.
- `<id>` must name a real prompt; it is matched the same way as a file name.

For example, a `project-style` prompt can reuse a shared `glossary`:

```markdown
---
id: project-style
kind: global
title: Project Style
---
Write in clear, concise English.

Use the project's vocabulary consistently:

{{include: glossary}}
```

The `glossary` fragment lives in its own file and can be reused by any number of
prompts:

```markdown
---
id: glossary
kind: shared
title: Project Glossary
---
- **Workspace**: the `.daedalus/` directory managed by Daedalus.
- **Prompt**: a reusable piece of text under `.daedalus/prompts/`.
```

### Rendering a composed prompt

`daedalus prompt show` always prints the **raw** file, with the directive left
unresolved — it is what is stored on disk:

```sh
daedalus prompt show project-style
```

```
---
id: project-style
kind: global
title: Project Style
---
Write in clear, concise English.

Use the project's vocabulary consistently:

{{include: glossary}}
```

`daedalus prompt render` prints the **composed** prompt, with every include
expanded in place:

```sh
daedalus prompt render project-style
```

```
Write in clear, concise English.

Use the project's vocabulary consistently:

- **Workspace**: the `.daedalus/` directory managed by Daedalus.
- **Prompt**: a reusable piece of text under `.daedalus/prompts/`.
```

Use `show` when you want to read or edit a prompt's own content, and `render`
when you want the final assembled text.

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/prompts/` holds the prompt. Defaults to the current directory. |

### Includes are recursive and deterministic

Composition is **recursive**: an included prompt can itself include others, and
Daedalus resolves the whole chain until nothing is left to expand. It is also
**deterministic** — rendering the same set of prompts always produces the same
text, byte for byte — and **read-only**: `render` never rewrites your prompt
files.

### A missing reference is reported

If a directive references an id that has no prompt file, Daedalus rejects the
render and names both the missing id and the prompt that referenced it, so you
know exactly where to look. Nothing is written:

```sh
daedalus prompt render project-style
```

```
daedalus: included prompt "glossary" not found (referenced by "project-style")
```

### Inclusion cycles are detected

If prompts include each other in a loop — for example `a` includes `b` and `b`
includes `a` — Daedalus detects the cycle instead of looping forever. It rejects
the render and prints the full chain so the loop is self-evident:

```sh
daedalus prompt render a
```

```
daedalus: include cycle detected: a -> b -> a
```

A prompt that includes itself is reported the same way.

## Notes & limitations

- Prompts are persisted as Markdown files under `.daedalus/prompts/`, one file
  per prompt, in a deterministic, git-friendly format. The same prompt always
  renders the same bytes.
- Composition is **DRY** and **non-mutating**: a shared fragment lives in a
  single file, is reused by reference, and `daedalus prompt render` resolves
  includes in memory without ever rewriting the source files.
- Every write operation is **non-destructive**: creating a prompt never
  overwrites an existing one, and an invalid create or edit is rejected before
  anything is written, leaving your files intact.
- The body is stored **verbatim** as arbitrary Markdown — Daedalus does not
  interpret or rewrite it.
- A prompt's `id` and `kind` define its identity and are not editable; to change
  them, remove the prompt and create it anew.
- Phase 1 **configures** your project's AI structure; it does not **execute**
  agents — that stays with your runtime (for example, Claude Code).
