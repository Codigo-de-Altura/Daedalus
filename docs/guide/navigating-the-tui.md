# Navigating the interface

[← Back to the manual index](../README.md)

Daedalus is a terminal application. Launched with no arguments, it opens an
interactive interface organized into **areas** — one for each part of your
project's AI structure. This chapter explains how to move around: how to reach
each area, how to step back, and how to read where you are. The keys are the
same everywhere, so once you learn them in one area you know them in all of them.

> This chapter covers the navigation shell only. The visual styling, Markdown
> rendering, forms, the full keyboard-shortcut help system, and editing actions
> arrive in later chapters as those features ship.

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
area and every detail screen.

| Key | Action |
|---|---|
| `↑` / `k` | Move the selection up |
| `↓` / `j` | Move the selection down |
| `enter` / `l` | Enter the selected area, or open the selected item |
| `esc` / `backspace` | Go back one level |
| `h` | Jump straight to the root menu (home) |
| `r` | Retry loading an area that failed (shown only in the error state) |
| `?` | Toggle the help line between short and expanded |
| `q` / `Ctrl+C` | Quit Daedalus |

A help line at the bottom of every screen always shows the keys available where
you are, so you never have to memorize them. Press `?` to expand it for the full
list on the current screen.

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

- **Prompts** opens a prompt's composed text.
- **Workflows** opens a workflow's DAG.
- **Backlog** opens a spec, architecture document, or epic.

Select the row and press `enter` (or `l`) to open it. Inside a detail screen the
content scrolls: use `↑`/`↓` to scroll line by line, `pgup`/`pgdn` (or `b`/`f`)
by page, and `g`/`G` to jump to the top or bottom. A hint shows how far through
the content you are. Press `esc` to return to the list.

Some rows are purely informational (for example the **Init** and **Build**
summaries) and do not open a detail screen; pressing `enter` on them does
nothing.

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

---

> Phase 1 note: Daedalus configures your project's AI structure; it does not
> execute agents — that stays with your runtime (for example, Claude Code).
