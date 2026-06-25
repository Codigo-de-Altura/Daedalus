# Managing architecture documents

[← Back to the manual index](../README.md)

An **architecture document** is the next artifact after the spec in the SDD
(spec-driven development) pipeline: a blueprint of your project's high-level
structure and decisions — *what* the system is, not *how* to type it. The
*architect* agent derives it from a [spec](managing-specs.md), and the later
stages of the pipeline build on it. You create these documents in your
workspace, version them with Git, and manage them with the
`daedalus architecture` command.

As with specs, Daedalus manages the **definition** of this step: it creates the
document, optionally wires it to its originating spec, and reserves a
deterministic place for it. It does **not** generate the document's content for
you — see
[Phase 1: Daedalus does not run the agent](#phase-1-daedalus-does-not-run-the-agent)
below.

## The task, end to end

Working with an architecture document follows four steps. Daedalus handles the
first two; the last two are yours:

1. **Create the document.** Run `daedalus architecture create` to create the
   document at its canonical path, with a seeded placeholder body.
2. **Link it to its spec (optional).** Pass `--spec <spec-slug>` to record the
   `spec → architecture` trace and wire the document to the *architect* step of
   the `sdd-default` workflow. You can link at create time or later, with `edit`.
3. **Run the architect on your backend.** Generate the architecture content from
   the spec by running the *architect* agent in your own runtime (for example,
   Claude Code) — Daedalus does not do this for you.
4. **Refine the document by hand.** Drop the generated content into the file and
   edit it. The document is yours; Daedalus never overwrites it.

## Where architecture documents live

Every architecture document is persisted as one Markdown file under your
workspace's `.daedalus/architecture/` directory, named after its slug:

```
.daedalus/
  architecture/
    payments-arch.md
```

Each file is a diff-friendly Markdown document: a **YAML frontmatter** block with
the document's metadata, followed by the **body** (your Markdown, stored
verbatim).

A document **with no spec linked** carries only its identity in the frontmatter:

```markdown
---
slug: payments-arch
kind: architecture
title: Payments Architecture
---
# Payments Architecture

> Architecture document placeholder. Daedalus manages this artifact's definition
> but does not run the architect agent (phase 1). Generate the architecture by
> running the "architect" agent (workflow "sdd-default", phase "architecture") on your backend, then replace
> this placeholder with the result and refine it.
```

A document **linked to its originating spec** adds a provenance block recording
the trace:

```markdown
---
slug: payments-arch
kind: architecture
title: Payments Architecture
spec: payments.md
agent: architect
workflow: sdd-default
phase: architecture
generated: false
---
# Payments Architecture

> Architecture document placeholder. Daedalus manages this artifact's definition
> but does not run the architect agent (phase 1). Generate the architecture by
> running the "architect" agent (workflow "sdd-default", phase "architecture") on your backend, using the spec
> below, then replace this placeholder with the result and refine it.

Source spec: payments.md
```

When linked, the frontmatter keeps the **trace** back to its origin: `spec` names
the spec file it derives from, `agent` is the *architect* that produces it, and
`generated: false` records that Daedalus did **not** generate the body — it
seeded a placeholder for you to replace. The provenance block is
**all-or-nothing**: an **unlinked** document carries *none* of those keys, so a
document with no spec never shows misleading *architect* wiring.

The frontmatter keys are always written in the same order, and for a given link
state every key is always present, so the output is **deterministic**: the same
document always produces the same file, byte for byte, which keeps your Git diffs
clean. These files are your **editable source of truth** — you can edit them by
hand, but the commands below are the scriptable alternative.

## The slug

Every architecture document is identified by a **slug**: a stable, unique,
`kebab-case` slug (lowercase letters and digits in dash-separated segments, e.g.
`payments-arch`). The slug is the file name, so it must be unique within the
workspace. A slug that is empty or not valid `kebab-case` is rejected with an
explicit error and nothing is written.

## Creating a document

Use `daedalus architecture create <slug>` to create a new architecture document.
The `--title` flag is required; the slug may appear before or after the flags.

```sh
daedalus architecture create payments-arch --title "Payments Architecture"
```

On success, Daedalus reports the file it created and reminds you of the next
step — running the architect on your backend:

```
Created architecture document "payments-arch" at .daedalus/architecture/payments-arch.md.
Generate the architecture by running the "architect" agent on your backend, then refine it.
```

### Linking to the originating spec

Pass `--spec <spec-slug>` to link the document to an existing spec — this records
the `spec → architecture` trace. Give the spec's **slug** (for example
`payments`); the frontmatter stores the file reference `payments.md`:

```sh
daedalus architecture create payments-arch \
  --title "Payments Architecture" \
  --spec payments
```

When linked, the confirmation names the spec it was wired to:

```
Created architecture document "payments-arch" at .daedalus/architecture/payments-arch.md.
Linked to spec payments.md. Generate the architecture by running the "architect" agent on your backend, then refine it.
```

The referenced spec must already exist in `.daedalus/specs/`. If it does not,
Daedalus rejects the run and tells you to capture it first (or to omit `--spec`),
and nothing is written:

```sh
daedalus architecture create payments-arch --title "Payments Architecture" --spec ghost
```

```
daedalus: spec "ghost" not found in .daedalus/specs; capture it first or omit --spec
```

> Capture the spec with `daedalus spec capture` — see
> [Managing specs](managing-specs.md).

You can also set the document body inline with `--body`, or from a file with
`--body-file`. See all options with:

```sh
daedalus architecture create --help
```

### Options

| Option | Description |
|---|---|
| `--title <text>` | The document's title. **Required.** |
| `--spec <spec-slug>` | Link the document to an existing spec by slug (optional). The spec must exist in `.daedalus/specs/`. |
| `--body <text>` | Set the document body inline. |
| `--body-file <path>` | Set the document body from a file. Takes precedence over `--body` if both are given. |
| `--path <dir>` | Target repository directory whose `.daedalus/architecture/` the document is added to. Defaults to the current directory. |
| `--preview` | Dry run: show the file that would be created without writing anything. |
| `--help` | Show all available options. |

### Previewing without writing

Use `--preview` to perform a **dry run**. Daedalus validates the document and
prints the exact file it would write, but **writes nothing** to disk:

```sh
daedalus architecture create payments-arch --title "Payments Architecture" --preview
```

```
Preview of creating architecture document "payments-arch" at .daedalus/architecture/payments-arch.md:
---
slug: payments-arch
kind: architecture
title: Payments Architecture
---
# Payments Architecture

> Architecture document placeholder. Daedalus manages this artifact's definition
> but does not run the architect agent (phase 1). Generate the architecture by
> running the "architect" agent (workflow "sdd-default", phase "architecture") on your backend, then replace
> this placeholder with the result and refine it.
```

Run the command again without `--preview` to apply it.

### It will not overwrite your work

Creating a document is **non-destructive**. If a document with the same slug
already exists, Daedalus leaves the existing file — including any manual edits —
untouched and reports the conflict instead of overwriting it:

```sh
daedalus architecture create payments-arch --title "Something else"
```

```
daedalus: architecture document already exists: "payments-arch" — not overwritten
```

### Invalid input is rejected before anything is written

A document is checked **before** it touches disk. If the slug or title is
invalid, Daedalus rejects the run, lists **every** problem it found — not just
the first — and writes nothing. Each finding names the field, what was observed,
and what was expected, so you can fix them in one pass:

```sh
daedalus architecture create Bad_Slug --title ""
```

```
daedalus: architecture document "Bad_Slug" is invalid; it was not created:
  - slug: observed "Bad_Slug"; expected kebab-case: lowercase letters/digits in dash-separated segments (e.g. my-architecture)
  - title: observed empty; expected a non-empty title
```

## Listing documents

Use `daedalus architecture list` to see every document with its slug, its linked
spec, and its title:

```sh
daedalus architecture list
```

The documents are listed in slug order; the middle column shows the linked spec
file, or `-` when the document has no spec linked:

```
Architecture documents (2):
  cart-arch	-	Cart Architecture
  payments-arch	payments.md	Payments Architecture
```

When there are no documents yet, the count is zero:

```
Architecture documents (0):
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/architecture/` is listed. Defaults to the current directory. |

## Showing a document

Use `daedalus architecture show <slug>` to print a document's file content
**verbatim** — frontmatter and body, exactly as it is stored:

```sh
daedalus architecture show payments-arch
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/architecture/` holds the document. Defaults to the current directory. |

If the slug does not exist, Daedalus tells you so and writes nothing:

```
daedalus: architecture document "ghost" not found
```

## Editing a document

Once a document exists, refine it with `daedalus architecture edit <slug>`. This
is how you replace the seeded placeholder with the real architecture content —
for example, after running the architect on your backend — change the title, or
manage the spec link. At least one edit flag is required; the slug may appear
before or after the flags. Writes are atomic.

```sh
daedalus architecture edit payments-arch --body-file ./generated/payments-arch.md
```

On success, Daedalus confirms the change:

```
Edited architecture document "payments-arch" at .daedalus/architecture/payments-arch.md.
```

### Managing the spec link

The `--spec` flag attaches or repoints the document's link to its originating
spec; the referenced spec must exist:

```sh
daedalus architecture edit payments-arch --spec payments
```

To **clear** the link — turning the document back into an unlinked one, dropping
its whole provenance block — pass an empty `--spec`:

```sh
daedalus architecture edit payments-arch --spec=
```

> **PowerShell note:** write the clear as a single token, `--spec=`. Passing two
> tokens (`--spec ""`) is misinterpreted by the shell and will not clear the
> link.

### Options

At least one edit flag is required.

| Option | Description |
|---|---|
| `--title <text>` | Set the document's title. |
| `--spec <spec-slug>` | Set the originating spec link by slug. An empty value (`--spec=`) clears the link. A non-empty one must reference an existing spec. |
| `--body <text>` | Set the document's body inline. |
| `--body-file <path>` | Set the document's body from a file. Takes precedence over `--body` if both are given. |
| `--path <dir>` | Target repository directory whose `.daedalus/architecture/` holds the document. Defaults to the current directory. |
| `--help` | Show all available options. |

### Edits are validated before anything is written

An edit is checked **before** it touches disk, and the write is **atomic**. If
the result would be invalid — for example, an empty title — Daedalus rejects the
edit, reports the problem, and leaves your existing file completely intact (never
half-written):

```sh
daedalus architecture edit payments-arch --title ""
```

```
daedalus: architecture document "payments-arch" is invalid; the edit was not applied:
  - title: observed empty; expected a non-empty title
```

A non-empty `--spec` that names a spec which does not exist is rejected the same
way, and the link is left unchanged:

```sh
daedalus architecture edit payments-arch --spec ghost
```

```
daedalus: spec "ghost" not found in .daedalus/specs; capture it first or pass --spec "" to clear the link
```

### Editing requires at least one change

Running `edit` with no edit flag is treated as a usage error, not a silent no-op:

```sh
daedalus architecture edit payments-arch
```

```
daedalus: architecture edit requires at least one edit flag (--title, --spec, --body, --body-file)
```

### Editing a document that does not exist

`edit` only works on a document that already lives in your workspace. If the slug
is not there, Daedalus rejects the run and tells you to create it first:

```sh
daedalus architecture edit ghost --title "anything"
```

```
daedalus: architecture document "ghost" not found
the document must already exist; create it first
```

## Removing a document

Use `daedalus architecture remove <slug>` to delete a document. **Only** that
document's file is removed; no other file in the workspace is touched.

```sh
daedalus architecture remove payments-arch
```

```
Removed architecture document "payments-arch" from .daedalus/architecture/payments-arch.md.
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/architecture/` holds the document. Defaults to the current directory. |

Removing a document that does not exist is reported as an explicit error, not a
silent success:

```
daedalus: architecture document "ghost" not found
```

## Phase 1: Daedalus does not run the agent

Daedalus **configures** the `spec → architecture` step; it does not **execute**
it. When you create a document, Daedalus:

- creates the document at its canonical path with a seeded placeholder body,
- optionally records its link to the originating spec (the `architecture` phase
  of the `sdd-default` workflow), and
- does **not** call a model or launch the *architect*.

Generating the document's real content from the spec is done **by you**, running
the *architect* agent in your own backend (for example, Claude Code). The seeded
document says so in its body, and a linked document's frontmatter carries
`generated: false` to make that explicit. Once you have the generated
architecture, drop it into the file — with `daedalus architecture edit` or by
hand — and refine it.

## Notes & limitations

- Architecture documents are persisted as Markdown files under
  `.daedalus/architecture/`, one file per document, named `<slug>.md`, in a
  deterministic, git-friendly format. The same document always renders the same
  bytes: fixed key order, a single trailing newline, and an all-or-nothing
  provenance block.
- The spec link is **optional**. A linked document records the `spec →
  architecture` trace and its *architect*-step provenance; an unlinked document
  carries only its identity. `--spec=` on `edit` clears the link.
- Creating is **non-destructive**: it never overwrites an existing document you
  have refined. `edit` is **atomic** and validated before writing, so an invalid
  edit leaves your file intact.
- Bodies are stored **verbatim** as arbitrary Markdown — Daedalus does not
  interpret or rewrite them.
- Phase 1 **configures** the `spec → architecture` step; it does not **run** the
  *architect* agent — generating the architecture stays with your runtime (for
  example, Claude Code).
</content>
