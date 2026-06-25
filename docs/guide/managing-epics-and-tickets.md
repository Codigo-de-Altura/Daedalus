# Managing epics and tickets

[← Back to the manual index](../README.md)

**Epics** and **tickets** are the planning artifacts of the SDD (spec-driven
development) pipeline — the backlog that follows your
[spec](managing-specs.md) and [architecture](managing-architecture.md). An
**epic** describes an objective and its scope; a **ticket** describes one
feature and always belongs to an epic. The *planner* agent derives them from
your spec and architecture, and you refine them. You manage them in your
workspace, version them with Git, and edit them with the `daedalus epic` and
`daedalus ticket` commands.

As with the earlier artifacts, Daedalus manages the **definition**: it creates
the epic and ticket folders, records their metadata and origin links, and seeds
each with a placeholder body. It does **not** generate their content or run the
implementation — see
[Phase 1: Daedalus does not run the agent](#phase-1-daedalus-does-not-run-the-agent)
below.

## The task, end to end

Working with a backlog follows four steps. Daedalus handles the first two; the
last two are yours:

1. **Create the epic.** Run `daedalus epic create`, optionally linking it to its
   originating spec and architecture, and set its status, priority, and
   dependencies.
2. **Create tickets under it.** Run `daedalus ticket create` for each feature.
   Every ticket is nested under its parent epic and carries its own metadata and
   dependencies.
3. **Run the planner on your backend.** Generate the epic and ticket content from
   the spec and architecture by running the *planner* agent in your own runtime
   (for example, Claude Code) — Daedalus does not do this for you.
4. **Refine by hand.** Drop the generated content into each file and edit it. The
   artifacts are yours; Daedalus never overwrites them.

## Where epics and tickets live

The backlog is a **nested** tree under your workspace's `.daedalus/epics/`
directory. Each epic is a folder named after its id; its tickets live in a
`tickets/` subfolder, each in a folder named after the ticket id:

```
.daedalus/
  epics/
    epic-05-sdd-backlog/
      epic.md
      tickets/
        ticket-05-03-epics-tickets-management/
          ticket.md
```

The **folder name is the id**. An epic id is `epic-NN-<slug>` (a number `NN`
and a `kebab-case` slug); a ticket id is `ticket-NN-MM-<slug>`, where `NN` is the
parent epic's number and `MM` is the ticket's sequence within that epic. The
markdown file inside is named by kind — `epic.md` or `ticket.md` — so renaming a
slug renames only the folder and keeps the file name stable.

Each file is a diff-friendly Markdown document: a **YAML frontmatter** block with
the artifact's metadata, followed by the **body** (your Markdown, stored
verbatim). An `epic.md` linked to its spec and architecture looks like this:

```markdown
---
id: epic-05-sdd-backlog
kind: epic
title: SDD Backlog
status: todo
priority: medium
spec: sdd-backlog.md
architecture: sdd-backlog-arch.md
depends_on: [epic-04-workflows]
agent: planner
workflow: sdd-default
phase: epics
generated: false
---
<epic markdown, verbatim>
```

A `ticket.md` adds the mandatory `epic` link to its parent, right after its
identity:

```markdown
---
id: ticket-05-03-epics-tickets-management
kind: ticket
title: Epics & Tickets Management
epic: epic-05-sdd-backlog
status: todo
priority: high
spec: sdd-backlog.md
architecture: sdd-backlog-arch.md
depends_on: [ticket-05-02-architecture-docs]
agent: planner
workflow: sdd-default
phase: tickets
generated: false
---
<ticket markdown, verbatim>
```

A few things to note about the frontmatter:

- `status`, `priority`, and `depends_on` are **always present** (`depends_on` is
  `[]` when empty), so the metadata shape is stable and a diff never has to tell
  "absent" from "empty".
- The origin links `spec` and `architecture` are **optional** and **omitted**
  when not set — an unlinked artifact carries neither key.
- The *planner*-step provenance block (`agent`, `workflow`, `phase`,
  `generated`) is **all-or-nothing**: it is written only when the artifact
  records at least one origin link (spec or architecture). The `phase` is `epics`
  for an epic and `tickets` for a ticket. `generated: false` records that
  Daedalus did **not** generate the body.

The output is **deterministic** — the same epic or ticket always produces the
same file, byte for byte — which keeps your Git diffs clean. These files are your
**editable source of truth**: you can edit them by hand, but the commands below
are the scriptable alternative.

## Status and priority

`status` and `priority` are **closed sets** — a value outside them is rejected.

| `status` | Meaning |
|---|---|
| `todo` | Defined, not yet started. **The default.** |
| `in-progress` | Work is underway. |
| `blocked` | Work cannot proceed (for example, an unmet dependency). |
| `done` | Completed. |

| `priority` | |
|---|---|
| `low` | |
| `medium` | The default. |
| `high` | |
| `critical` | |

When you omit `--status` or `--priority`, the defaults (`todo` and `medium`)
apply.

## Dependencies and origin links

- **`--depends-on`** records an explicit, comma-separated list of artifact ids
  this epic or ticket depends on (other ticket or epic ids). The list is stored
  in `depends_on`, with duplicates removed and the order you wrote preserved. In
  this phase Daedalus **records** the dependencies; it does not yet verify that
  the referenced ids exist or that the graph is free of cycles.
- **`--spec`** and **`--architecture`** link the artifact to its originating spec
  and architecture document, by slug. These are **optional**, but when given the
  referenced artifact must already exist in `.daedalus/specs/` or
  `.daedalus/architecture/`; otherwise the command is rejected. Together with a
  ticket's mandatory `epic` link, these preserve the trace
  `ticket → epic → spec / architecture`.

# `daedalus epic`

Manage epics with the `epic` subcommands: `create`, `list`, `show`, `edit`, and
`remove`.

## Creating an epic

Use `daedalus epic create <NN> <slug>` to create an epic. You supply the number
and the slug explicitly — they compose the id `epic-NN-<slug>` (no
auto-numbering). The `--title` flag is required.

```sh
daedalus epic create 05 sdd-backlog --title "SDD Backlog"
```

On success, Daedalus reports the folder it created:

```
Created epic "epic-05-sdd-backlog" at .daedalus/epics/epic-05-sdd-backlog.
```

Set metadata and origin links in the same run. A non-existent `--spec` or
`--architecture` is rejected before anything is written:

```sh
daedalus epic create 05 sdd-backlog \
  --title "SDD Backlog" \
  --status in-progress \
  --priority high \
  --spec sdd-backlog \
  --architecture sdd-backlog-arch \
  --depends-on epic-04-workflows
```

See all options with:

```sh
daedalus epic create --help
```

### Options

| Option | Description |
|---|---|
| `--title <text>` | The epic's title. **Required.** |
| `--status <status>` | One of `todo`, `in-progress`, `blocked`, `done`. Defaults to `todo`. |
| `--priority <priority>` | One of `low`, `medium`, `high`, `critical`. Defaults to `medium`. |
| `--spec <spec-slug>` | Link to an existing spec by slug (optional; must exist in `.daedalus/specs/`). |
| `--architecture <arch-slug>` | Link to an existing architecture document by slug (optional; must exist in `.daedalus/architecture/`). |
| `--depends-on <id,...>` | Comma-separated dependency ids (optional). |
| `--body <text>` | Set the epic body inline. |
| `--body-file <path>` | Set the epic body from a file. Takes precedence over `--body` if both are given. |
| `--path <dir>` | Target repository directory whose `.daedalus/epics/` the epic is added to. Defaults to the current directory. |
| `--preview` | Dry run: show the file that would be created without writing anything. |
| `--help` | Show all available options. |

### Previewing without writing

Use `--preview` to perform a **dry run**. Daedalus validates the epic and prints
the file it would write, but **writes nothing** to disk:

```sh
daedalus epic create 05 sdd-backlog --title "SDD Backlog" --preview
```

Run the command again without `--preview` to apply it.

### It will not overwrite your work

Creating an epic is **non-destructive**. If an epic with the same id already
exists, Daedalus leaves the existing folder untouched and reports the conflict:

```sh
daedalus epic create 05 sdd-backlog --title "Something else"
```

```
daedalus: epic already exists: "epic-05-sdd-backlog" — not overwritten
```

### Invalid input is rejected before anything is written

An epic is checked **before** it touches disk. Daedalus lists **every** problem
it found — not just the first — and writes nothing. Each finding names the field,
what was observed, and what was expected:

```sh
daedalus epic create 05 sdd-backlog --title "" --status nope --priority urgent
```

```
daedalus: epic "epic-05-sdd-backlog" is invalid; it was not created:
  - title: observed empty; expected a non-empty title
  - status: observed "nope"; expected one of: todo, in-progress, blocked, done
  - priority: observed "urgent"; expected one of: low, medium, high, critical
```

## Listing epics

Use `daedalus epic list` to see every epic with its id, status, priority, and
title, in id order:

```sh
daedalus epic list
```

```
Epics (1):
  epic-05-sdd-backlog	todo	medium	SDD Backlog
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/epics/` is listed. Defaults to the current directory. |

## Showing an epic

Use `daedalus epic show <epic-id>` to print an epic's `epic.md` content
**verbatim**:

```sh
daedalus epic show epic-05-sdd-backlog
```

If the id does not exist, Daedalus tells you so:

```
daedalus: epic "epic-05-sdd-backlog" not found
```

## Editing an epic

Once an epic exists, change its metadata or body with
`daedalus epic edit <epic-id>`. At least one edit flag is required. The write is
**atomic** and validated first, so an invalid edit leaves your file intact.

```sh
daedalus epic edit epic-05-sdd-backlog --status done --priority critical
```

```
Edited epic "epic-05-sdd-backlog" at .daedalus/epics/epic-05-sdd-backlog/epic.md.
```

The list flags (`--depends-on`) and link flags (`--spec`, `--architecture`)
**replace** the current value. Passing an empty `--spec`, `--architecture`, or
`--depends-on` **clears** it:

```sh
daedalus epic edit epic-05-sdd-backlog --spec=
```

> **PowerShell note:** write a clear as a single token, `--spec=`. Passing two
> tokens (`--spec ""`) is misinterpreted by the shell and will not clear the
> link. The same applies to `--architecture=` and `--depends-on=`.

The epic's **id** is its structural identity and is not editable; to renumber or
rename, remove the epic and create it anew. See all options with
`daedalus epic edit --help`.

### Options

At least one edit flag is required.

| Option | Description |
|---|---|
| `--title <text>` | Set the title. |
| `--status <status>` | Set the status (`todo`, `in-progress`, `blocked`, `done`). |
| `--priority <priority>` | Set the priority (`low`, `medium`, `high`, `critical`). |
| `--spec <spec-slug>` | Set the originating spec link by slug. An empty value (`--spec=`) clears it. A non-empty one must exist. |
| `--architecture <arch-slug>` | Set the originating architecture link by slug. An empty value clears it. A non-empty one must exist. |
| `--depends-on <id,...>` | Set the dependency ids (comma-separated). An empty value clears them. |
| `--body <text>` | Set the body inline. |
| `--body-file <path>` | Set the body from a file. Takes precedence over `--body`. |
| `--path <dir>` | Target repository directory whose `.daedalus/epics/` holds the epic. Defaults to the current directory. |

Running `edit` with no edit flag is a usage error, not a silent no-op:

```
daedalus: epic edit requires at least one edit flag
```

## Removing an epic

Use `daedalus epic remove <epic-id>` to delete an epic. **This removes the epic
folder and all of its nested tickets** — the removal cascades:

```sh
daedalus epic remove epic-05-sdd-backlog
```

```
Removed epic "epic-05-sdd-backlog" (and its tickets) from .daedalus/epics/epic-05-sdd-backlog.
```

Removing an epic that does not exist is reported as an explicit error.

# `daedalus ticket`

Manage tickets with the `ticket` subcommands: `create`, `list`, `show`, `edit`,
and `remove`. Because a ticket is nested under its epic, every operation takes
the parent epic id.

## Creating a ticket

Use `daedalus ticket create <epic-id> <MM> <slug>` to create a ticket under an
existing epic. You supply the sequence `MM` and the slug; the epic number `NN` is
**derived from the parent epic id**, so the ticket id `ticket-NN-MM-<slug>` stays
consistent with its epic. The `--title` flag is required.

```sh
daedalus ticket create epic-05-sdd-backlog 03 epics-tickets-management \
  --title "Epics & Tickets Management"
```

On success, Daedalus reports the folder it created:

```
Created ticket "ticket-05-03-epics-tickets-management" at .daedalus/epics/epic-05-sdd-backlog/tickets/ticket-05-03-epics-tickets-management.
```

The ticket takes the same metadata, link, and dependency flags as an epic:

```sh
daedalus ticket create epic-05-sdd-backlog 03 epics-tickets-management \
  --title "Epics & Tickets Management" \
  --priority high \
  --spec sdd-backlog \
  --architecture sdd-backlog-arch \
  --depends-on ticket-05-02-architecture-docs
```

### The parent epic must exist

A ticket cannot exist without its epic — it lives inside the epic's folder. If
the parent epic is not there, Daedalus rejects the run and tells you to create it
first, writing nothing:

```sh
daedalus ticket create epic-99-ghost 01 something --title "Something"
```

```
daedalus: parent epic does not exist; create the epic first
```

### Options

| Option | Description |
|---|---|
| `--title <text>` | The ticket's title. **Required.** |
| `--status <status>` | One of `todo`, `in-progress`, `blocked`, `done`. Defaults to `todo`. |
| `--priority <priority>` | One of `low`, `medium`, `high`, `critical`. Defaults to `medium`. |
| `--spec <spec-slug>` | Link to an existing spec by slug (optional; must exist). |
| `--architecture <arch-slug>` | Link to an existing architecture document by slug (optional; must exist). |
| `--depends-on <id,...>` | Comma-separated dependency ids (optional). |
| `--body <text>` | Set the ticket body inline. |
| `--body-file <path>` | Set the ticket body from a file. Takes precedence over `--body`. |
| `--path <dir>` | Target repository directory whose `.daedalus/epics/` holds the parent epic. Defaults to the current directory. |
| `--preview` | Dry run: show the file that would be created without writing anything. |
| `--help` | Show all available options. |

Creating a ticket is **non-destructive**: an existing ticket id is preserved, not
overwritten:

```
daedalus: ticket already exists: "ticket-05-03-epics-tickets-management" — not overwritten
```

## Listing tickets

Use `daedalus ticket list <epic-id>` to see an epic's tickets, in id order:

```sh
daedalus ticket list epic-05-sdd-backlog
```

```
Tickets of epic-05-sdd-backlog (1):
  ticket-05-03-epics-tickets-management	todo	high	Epics & Tickets Management
```

## Showing a ticket

Use `daedalus ticket show <epic-id> <ticket-id>` to print a ticket's `ticket.md`
content **verbatim**:

```sh
daedalus ticket show epic-05-sdd-backlog ticket-05-03-epics-tickets-management
```

If the ticket is not found under that epic, Daedalus tells you so:

```
daedalus: ticket "ticket-05-03-epics-tickets-management" not found under "epic-05-sdd-backlog"
```

## Editing a ticket

Change a ticket's metadata or body with
`daedalus ticket edit <epic-id> <ticket-id>`. At least one edit flag is required;
the write is **atomic** and validated first.

```sh
daedalus ticket edit epic-05-sdd-backlog ticket-05-03-epics-tickets-management --status in-progress
```

```
Edited ticket "ticket-05-03-epics-tickets-management" at .daedalus/epics/epic-05-sdd-backlog/tickets/ticket-05-03-epics-tickets-management/ticket.md.
```

The edit flags match `daedalus ticket create` (`--title`, `--status`,
`--priority`, `--spec`, `--architecture`, `--depends-on`, `--body`,
`--body-file`, `--path`); an empty `--spec`, `--architecture`, or `--depends-on`
clears that field (use the single-token form `--spec=` on PowerShell, as above).
A ticket's **id** and its parent **epic** are its structural identity and are not
editable.

## Removing a ticket

Use `daedalus ticket remove <epic-id> <ticket-id>` to delete a single ticket. The
epic and its sibling tickets are left intact:

```sh
daedalus ticket remove epic-05-sdd-backlog ticket-05-03-epics-tickets-management
```

```
Removed ticket "ticket-05-03-epics-tickets-management" from .daedalus/epics/epic-05-sdd-backlog/tickets/ticket-05-03-epics-tickets-management.
```

## Phase 1: Daedalus does not run the agent

Daedalus **configures** the backlog; it does not **execute** the planning or the
implementation. When you create an epic or ticket, Daedalus:

- creates the folder and its `epic.md` / `ticket.md` with a seeded placeholder
  body,
- records the metadata, dependencies, and origin links (and, for a ticket, the
  mandatory parent-epic link), and
- does **not** call a model or launch the *planner*.

Generating the real epic and ticket content from your spec and architecture is
done **by you**, running the *planner* agent in your own backend (for example,
Claude Code); the implementation those tickets describe also happens outside
Daedalus. The seeded body says so, and a linked artifact's frontmatter carries
`generated: false` to make that explicit. Once you have the generated content,
drop it into the file — with `edit` or by hand — and refine it.

## Notes & limitations

- The backlog is a **nested** tree under `.daedalus/epics/`: each epic is a
  folder (`epic-NN-<slug>/epic.md`) and its tickets live in `tickets/`
  (`ticket-NN-MM-<slug>/ticket.md`). The folder name is the id; the markdown file
  is named by kind.
- `status` and `priority` are **closed sets** (`todo`/`in-progress`/`blocked`/`done`
  and `low`/`medium`/`high`/`critical`); an invalid value is rejected with the
  list of valid values. Defaults are `todo` and `medium`.
- **Dependencies are recorded, not yet verified.** `--depends-on` stores an
  explicit, deduplicated list; checking that the ids exist and that the
  dependency graph has no cycles is a later feature.
- **Origin links are optional but checked.** A non-empty `--spec` or
  `--architecture` must reference an artifact that exists; together with a
  ticket's mandatory `epic` link they preserve the
  `ticket → epic → spec / architecture` trace.
- **Non-destructive.** Creating never overwrites an existing artifact; `edit` is
  atomic and validated before writing. Removing an **epic** cascades to its
  tickets; removing a **ticket** leaves the epic and siblings intact.
- Structural identity (an epic's `id`, a ticket's `id` and parent `epic`) is
  **not editable**; status, priority, dependencies, spec/architecture links,
  title, and body are.
- **Editing files by hand on Windows:** save without a BOM (for example,
  `Out-File -Encoding utf8NoBOM`, or an editor that does not inject a byte-order
  mark). A BOM at the start of the file breaks frontmatter parsing.
- Phase 1 **configures** the backlog; it does not **run** the *planner* agent or
  the implementation — that stays with your runtime (for example, Claude Code).
</content>
