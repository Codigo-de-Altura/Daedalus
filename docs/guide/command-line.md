# Command line

[← Back to the manual index](../README.md)

Daedalus is a terminal application. Run it with no arguments to open the
interface, or use a subcommand such as `init`.

## Launching the interface

In an interactive terminal:

```sh
./daedalus
```

The interface opens on a **root menu** listing the six areas of your workspace —
**Init**, **Agents**, **Prompts**, **Workflows**, **Backlog**, and **Build**.
Press `enter` to enter an area, `esc` to go back, and `h` to return to the root
menu; a help line at the bottom shows the keys available on the current screen.
The interface is **read-only** — it shows you what is there; you create and edit
artifacts with the `daedalus` commands.

[Navigating the interface](navigating-the-tui.md) is the authoritative chapter on
the TUI: the six areas, moving in and out, the breadcrumb, reading documents,
filtering lists, contextual help, and the full keyboard reference.

### Quitting

Press `q` or `Ctrl+C`. The interface closes cleanly, restores your terminal, and
the process exits with code `0`.

## Non-interactive use

When Daedalus runs without an interactive terminal — piped input, a script, CI,
or a container without a TTY — it does not start the full interface. Instead it
prints a short notice and exits with code `0`:

```sh
echo q | ./daedalus
# daedalus 0.1.0-dev — run in an interactive terminal to launch the TUI.
```

This makes Daedalus safe to invoke from automation without leaving the terminal
in an unexpected state.

## Version and help

```sh
./daedalus --version    # print the version and exit
./daedalus --help       # print usage and exit
```

## Subcommands

Daedalus dispatches subcommands **positionally**. The core subcommands are:

| Command            | Purpose                                                        |
| ------------------ | -------------------------------------------------------------- |
| `daedalus init`    | Create (or non-destructively upgrade) the `.daedalus/` workspace. |
| `daedalus build`   | Compile the canonical definition to your backend (alias `sync`). |
| `daedalus validate`| Check the workspace against the conventions and lint definitions. |
| `daedalus trace`   | Verify or navigate the spec → epic → ticket traceability chain. |

Daedalus also provides definition-management and SDD-backlog subcommands —
`agent`, `prompt`, `workflow`, `spec`, `architecture`, `epic`, and `ticket`.

Each subcommand supports `--help`, for example `daedalus init --help`. For the
full surface — purpose, flags, exit codes, and examples — see the
[Command reference](command-reference.md).
