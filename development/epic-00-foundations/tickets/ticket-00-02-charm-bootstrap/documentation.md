# Launching the Daedalus interface

Daedalus runs as a terminal user interface (TUI). At this foundations stage the
interface is a minimal skeleton that proves the application starts and stops
cleanly; product screens arrive in later versions.

## Start the interface

In an interactive terminal:

```sh
./daedalus
```

You will see a minimal Daedalus view with a short welcome message and a help
line at the bottom.

## Quit

Press `q` or `Ctrl+C`. The interface closes cleanly and restores your terminal;
the process exits with code `0`.

## Non-interactive use

When Daedalus is run without an interactive terminal — for example with piped
input, inside a script, or in continuous integration — it does not start the
full interface. Instead it prints a short notice and exits with code `0`:

```sh
echo q | ./daedalus
# daedalus 0.1.0-dev — run in an interactive terminal to launch the TUI.
```

This makes Daedalus safe to invoke from automation without leaving the terminal
in an unexpected state.
