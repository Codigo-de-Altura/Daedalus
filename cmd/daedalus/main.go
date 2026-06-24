// Command daedalus is the entry point for the Daedalus TUI/CLI.
//
// Daedalus automates the setup and management of a project's AI scaffolding
// (agents, prompts, DAG workflows, SDD backlog) in a backend-agnostic way and
// compiles it to the native format of the chosen tool. This binary is the
// minimal bootstrap established by epic-00; product commands live in later
// epics. It launches the Bubble Tea skeleton in an interactive terminal and
// exits cleanly in non-interactive contexts (piped input, CI, validation).
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Codigo-de-Altura/Daedalus/internal/buildinfo"
	"github.com/Codigo-de-Altura/Daedalus/internal/logging"
	"github.com/Codigo-de-Altura/Daedalus/internal/tui"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run wires up the binary and returns its process exit code. Keeping it
// separate from main keeps the exit handling testable.
func run(args []string) int {
	fs := flag.NewFlagSet(buildinfo.Name, flag.ContinueOnError)
	showVersion := fs.Bool("version", false, "print version information and exit")
	if err := fs.Parse(args); err != nil {
		// flag.ErrHelp is returned for -h/--help: a clean, successful exit.
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if *showVersion {
		fmt.Fprintf(os.Stdout, "%s %s\n", buildinfo.Name, buildinfo.Version)
		return 0
	}

	// Logs go to stderr so they never corrupt the Bubble Tea render on stdout.
	logger := logging.New(os.Stderr)
	interactive := isInteractive()
	logger.Info("daedalus starting", "version", buildinfo.Version, "interactive", interactive)

	if !interactive {
		// No TTY on stdin (piped input, CI, automated validation): render a
		// short notice instead of starting the full event loop, which would
		// otherwise take over a non-terminal session.
		fmt.Fprintf(os.Stdout, "%s %s — run in an interactive terminal to launch the TUI.\n",
			buildinfo.Name, buildinfo.Version)
		logger.Info("daedalus exiting", "reason", "non-interactive")
		return 0
	}

	if _, err := tea.NewProgram(tui.New()).Run(); err != nil {
		logger.Error("tui exited with error", "err", err)
		return 1
	}

	logger.Info("daedalus exiting", "reason", "user-quit")
	return 0
}

// isInteractive reports whether both stdin and stdout are connected to a
// terminal. The Bubble Tea event loop needs a real TTY for input and render;
// when either end is redirected (piped input, CI, a container without -t,
// automated validation) the program stays headless and exits cleanly instead
// of trying — and failing — to open /dev/tty.
func isInteractive() bool {
	return isTerminal(os.Stdin) && isTerminal(os.Stdout)
}

// isTerminal reports whether f is backed by a character device (a terminal).
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
