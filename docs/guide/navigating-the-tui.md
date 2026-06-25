# Navigating the interface

[← Back to the manual index](../README.md)

Daedalus is a terminal application. Launched with no arguments, it opens an
interactive interface organized into **areas** — one for each part of your
project's AI structure. This chapter explains how to move around: how to reach
each area, how to step back, and how to read where you are. The keys are the
same everywhere, so once you learn them in one area you know them in all of them.

> This chapter covers how you move around the interface, read documents in it,
> filter lists, and get help on the available shortcuts. In-interface editing of
> artifacts arrives in a later chapter as that feature ships.

## Launching the interface

In an interactive terminal:

```sh
./daedalus
```

The interface opens on the **root screen** — a menu listing the six areas. It
reads your workspace lazily: nothing is loaded until you enter an area, so
startup is instant.

If you run Daedalus without an interactive terminal (piped input, a script, CI,
or a container with no TTY), it does not start the interface; see
[Command line](command-line.md) for that behavior.

## The six areas

The root screen lists six areas, each one mapping to a part of your project's
AI structure. They always appear in the same order, top to bottom:

| Area | What it shows |
|---|---|
| **Init** | Your workspace status: whether a `.daedalus/` workspace exists, the project name, the configured backends, and what an init would create. |
| **Agents** | The built-in agent catalog, each agent listed by id and role. |
| **Prompts** | The global and shared prompts in your workspace. |
| **Workflows** | Your declarative DAG workflows, each tagged with its phase count. |
| **Backlog** | Your specs, architecture documents, epics, and tickets, in one list. |
| **Build** | A preview of what compiling to your configured backend would change. |

Each row on the menu carries a one-line summary so you can tell the areas apart
at a glance. The areas are **read-only views** in the interface — they show you
what is there. Creating and editing artifacts is done with the matching
`daedalus` commands (for example `daedalus init`, `daedalus prompt`,
`daedalus build`), which the empty-state messages point you to.

## Moving around

Navigation uses one small, consistent set of keys. They behave the same on every
area and every detail screen: the same action always uses the same key, so once
you learn a key it works everywhere it applies.

| Key | Action |
|---|---|
| `↑` / `k` | Move the selection up |
| `↓` / `j` | Move the selection down |
| `enter` / `l` | Enter the selected area, or open the selected item |
| `esc` / `backspace` | Go back one level |
| `h` | Jump straight to the root menu (home) |
| `/` | Filter the current list (shown when an area has items) |
| `r` | Retry loading an area that failed (shown only in the error state) |
| `?` | Toggle the help line between short and expanded |
| `q` / `Ctrl+C` | Quit Daedalus |

A help line at the bottom of every screen always shows the keys available where
you are, so you never have to memorize them. See [Getting help](#getting-help)
below for how to expand it.

### Entering an area

From the root menu, move the selection with `↑`/`↓` (or `k`/`j`) to the area you
want, then press `enter` (or `l`) to go in. The area loads its contents and shows
them as a list.

### Going back, and reaching the root

There are two ways back, and you are never trapped:

- Press `esc` (or `backspace`) to step back **one level** — from a detail screen
  to its area's list, or from an area back to the root menu.
- Press `h` at any time to jump **straight to the root menu**, however deep you
  are.

Every screen is reached by going in, and every screen can be left by stepping
back, so there are no dead ends: a way back always exists.

When you re-enter an area, it is shown exactly as you left it — your place in the
list is remembered.

> Quitting from a detail screen: while you are reading a scrollable detail view,
> `q` is reserved so you do not exit by accident — use `esc` to step back, or
> `Ctrl+C` if you really want to quit. On the root menu and in any area list,
> `q` quits as usual.

## Knowing where you are: the breadcrumb

The top of every screen shows a **breadcrumb** — a trail that names your current
location, starting from the root:

```
Daedalus › Prompts › project-style
```

- `Daedalus` is the root menu.
- The next segment is the area you are in (here, **Prompts**), highlighted so the
  active area is always identifiable.
- A third segment appears when you have opened an item into a detail screen (here,
  the prompt `project-style`).

Read the breadcrumb as the path back: each `›` is one `esc` away, or press `h` to
return to `Daedalus` in a single step.

## Opening an item

Some areas list items you can open into a **detail screen** — a read-only view of
that item:

- **Prompts** opens a prompt's composed text, formatted for reading.
- **Workflows** opens a workflow's DAG.
- **Backlog** opens a spec, architecture document, or epic, formatted for reading.

Select the row and press `enter` (or `l`) to open it. Inside a detail screen the
content scrolls: use `↑`/`↓` to scroll line by line, `pgup`/`pgdn` (or `b`/`f`)
by page, and `g`/`G` to jump to the top or bottom. A hint shows how far through
the content you are. Press `esc` to return to the list.

Some rows are purely informational (for example the **Init** and **Build**
summaries) and do not open a detail screen; pressing `enter` on them does
nothing.

## Reading documents

Most of what Daedalus manages is written as Markdown — prompts, specs,
architecture documents, and epics. When you open one of these in a detail
screen, Daedalus shows it **rendered**, not as raw Markdown source. You see:

- **Headings** set apart from body text.
- **Lists** laid out as bullets and numbers.
- **Tables** drawn with proper rows and columns.
- **Code blocks** in a monospaced block, with syntax highlighting.
- **Emphasis** — bold and italic — shown as styled text.

The document is wrapped to the width of the screen, so lines never run off the
edge — make your terminal wider and the text reflows to use the space. Exact
colors depend on your terminal's color support, but the layout stays readable
everywhere.

(Workflows are the exception: a workflow opens as a diagram of its DAG rather
than as a Markdown document.)

## Filtering a list

When an area lists more rows than you want to scan, you can filter it down to the
ones you care about. Press `/` from any area that has items to open the filter:

1. Type a term to match. The filter is **case-insensitive** and matches anywhere
   in a row's label or its badge (the small tag next to it), so you can filter by
   name, kind, or status.
2. Press `enter` to apply it. The list shrinks to the matching rows.
3. Press `esc` at any time to cancel and leave the list unchanged.

While a filter is active, a banner at the top of the list shows it — for example
`Filter: "spec"  ·  press / to change` — so a short list is never mysterious. To
**change** the filter, press `/` again (it opens pre-filled with your current
term so you can refine it). To **clear** it, press `/`, empty the field, and
press `enter` — an empty term shows everything again.

If a term matches nothing, the list does not go blank or trap you: it tells you
*"No matches"* and reminds you that `/` changes the filter and `esc` goes back.

A blank filter (only spaces) is rejected with a clear message, since it could
never match anything on purpose — clear the field instead to show all rows.

## A consistent look

Across every area and screen, Daedalus uses one visual language: the same accent
color marks the selected row and headings, the same border frames detail panels
and forms, the same badges tag items, and the loading, empty, and error states
all share that styling. Once you recognize how one area looks, every other area
reads the same way.

## Loading, empty, and error states

An area is always in exactly one of these states, and **all of them keep the way
back available** — you can step back or jump home no matter what an area shows:

- **Loading** — while an area fetches its contents you briefly see `Loading…`.
  This is usually instant.
- **Empty** — when an area has nothing to list, it shows a short message that
  tells you why and which command would populate it (for example, *"No
  `.daedalus` workspace here yet. Run `daedalus init` to create one."*). An empty
  area is a hint, not a wall.
- **Error** — if an area cannot load (for example a malformed file), it shows the
  error and a prompt to recover. Press `r` to **retry** the load in place, or
  `esc` to go back. An error never strands you.

## Getting help

You never have to remember the keys. Every screen shows a **help line** at the
bottom listing the shortcuts you can use right now. Two views, one key:

- The **short help line** is always visible — a compact row of the handful of
  shortcuts most worth knowing on the current screen.
- Press `?` to **expand** it into the full list of every shortcut available in
  the current context, grouped for easy scanning. Press `?` again to collapse it.

The help is **contextual** — it shows what applies where you are, and nothing
that does not:

- On an **area list**, it shows move, open, filter, back, and home.
- In a **detail screen**, it shows the scrolling keys (line, page, and
  jump-to-top/bottom) and back.
- In a **form** (such as the list filter), it shows submit, cancel, and how to
  move between fields.
- Even in the **loading**, **empty**, and **error** states the help line stays
  available, so help — and the way out — is always one `?` away.

`?` works in **every** context, including while a form is open. What the help
line announces is exactly what works: the shortcuts you see are the shortcuts the
screen accepts.

## Keyboard shortcuts reference

The same action always uses the same key. Which keys are *available* depends on
the screen, but their meaning never changes.

**Everywhere:**

| Key | Action |
|---|---|
| `?` | Toggle the help line (short ↔ expanded) |
| `q` / `Ctrl+C` | Quit Daedalus (`q` is held back while reading or typing — see below) |

**Root menu and area lists:**

| Key | Action |
|---|---|
| `↑` / `k`, `↓` / `j` | Move the selection |
| `enter` / `l` | Enter an area, or open the selected item |
| `/` | Filter the list (when the area has items) |
| `esc` / `backspace` | Go back one level |
| `h` | Jump to the root menu |
| `r` | Retry a failed load (error state only) |

**Detail screens (reading a document or DAG):**

| Key | Action |
|---|---|
| `↑` / `k`, `↓` / `j` | Scroll line by line |
| `pgup` / `b`, `pgdn` / `f` / `space` | Scroll by page |
| `g`, `G` | Jump to top / bottom |
| `esc` / `backspace` | Back to the list |
| `h` | Jump to the root menu |

**Forms (such as the filter):**

| Key | Action |
|---|---|
| `enter` | Submit |
| `esc` | Cancel |
| `tab`, `shift+tab` | Move to the next / previous field |

While you are reading a detail screen or typing in a form, `q` is *not* a quit —
it scrolls or types normally — so you never exit by accident. Use `esc` to back
out, or `Ctrl+C` if you really want to quit. (The one shortcut that still works
inside a form is `?`, which opens the help rather than typing a literal `?`.)

Shortcuts are fixed in this version of Daedalus; they cannot be remapped.

## Responsiveness

The interface is built to stay out of your way:

- **It responds instantly.** Moving between areas, opening and scrolling
  documents, and using forms react the moment you press a key — the interface
  never freezes while you work.
- **Slow work happens in the background.** When something takes a moment — loading
  an area's contents, or rendering a large document — Daedalus shows `Loading…`
  instead of locking up, and you can always press `esc` to go back or cancel
  while it works.
- **It starts fast.** The interface opens effectively instantly.
- **It is quiet when idle.** While you are not interacting, Daedalus uses no
  noticeable CPU — there are no constant animations or redraws burning cycles in
  the background.
- **It stays stable over a long session.** Navigating in and out of areas
  repeatedly does not slow the interface down or let its memory creep up.

> Known limitation: if you resize your terminal window while a Markdown document
> is open, the document keeps its previous width until you close and reopen it.
> Reopening the document re-wraps it to the new size. This affects only the open
> document view, not navigation.

---

> Phase 1 note: Daedalus configures your project's AI structure; it does not
> execute agents — that stays with your runtime (for example, Claude Code).
