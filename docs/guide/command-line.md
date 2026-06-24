# Command line

[← Back to the manual index](../README.md)

Daedalus is a terminal application. Run it with no arguments to open the
interface, or use a subcommand such as `init`.

## Launching the interface

In an interactive terminal:

```sh
./daedalus
```

At this stage the interface is a minimal skeleton that proves the application
starts and stops cleanly; product screens arrive in later versions. You will see
a short welcome view with a help line at the bottom.

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

| Command         | Purpose                                                |
| --------------- | ------------------------------------------------------ |
| `daedalus init` | Create the `.daedalus/` workspace in a repository.     |

See [Initializing a workspace](initializing-a-workspace.md) for details on
`init`. Each subcommand supports `--help`, for example `daedalus init --help`.
