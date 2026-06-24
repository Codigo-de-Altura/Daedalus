// Command daedalus is the entry point for the Daedalus TUI/CLI.
//
// Daedalus automates the setup and management of a project's AI scaffolding
// (agents, prompts, DAG workflows, SDD backlog) in a backend-agnostic way and
// compiles it to the native format of the chosen tool. With no subcommand it
// launches the Bubble Tea skeleton in an interactive terminal (and exits
// cleanly in non-interactive contexts); the `init` subcommand scaffolds the
// `.daedalus/` workspace in the target repository.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Codigo-de-Altura/Daedalus/internal/buildinfo"
	"github.com/Codigo-de-Altura/Daedalus/internal/logging"
	"github.com/Codigo-de-Altura/Daedalus/internal/tui"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

// run dispatches to a subcommand when the first argument names one, otherwise
// it runs the default behavior (print version or launch the TUI). It returns
// the process exit code.
func run(args []string) int {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "init":
			return runInit(args[1:], os.Stdout, os.Stderr)
		default:
			fmt.Fprintf(os.Stderr, "daedalus: unknown command %q\nrun 'daedalus --help' for usage\n", args[0])
			return 2
		}
	}
	return runDefault(args)
}

// runDefault handles invocation without a subcommand: --version prints the
// version, otherwise the TUI is launched (or a notice is printed when there is
// no terminal).
func runDefault(args []string) int {
	fs := flag.NewFlagSet(buildinfo.Name, flag.ContinueOnError)
	showVersion := fs.Bool("version", false, "print version information and exit")
	if err := fs.Parse(args); err != nil {
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

// runInit handles `daedalus init`: it scaffolds the canonical `.daedalus/`
// workspace in the target directory (default: the current directory). When a
// workspace already exists the run becomes a non-destructive upgrade that only
// adds the missing pieces; --preview reports what would change without writing.
func runInit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus init", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory in which to create the .daedalus/ workspace")
	preview := fs.Bool("preview", false, "show the changes that would be made without writing anything (dry run)")
	backend := fs.String("backend", "", "target agent backend(s) to record in the manifest, comma-separated "+
		"(default: claude-code; MVP supports: claude-code)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus init [flags]\n\n"+
			"Create the canonical .daedalus/ workspace in the target repository.\n"+
			"If a workspace already exists, init performs a non-destructive upgrade,\n"+
			"adding only the missing directories and root artifacts.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	logger := logging.New(stderr)

	// Resolve and validate the backend selection BEFORE any filesystem access so
	// an unsupported backend can never leave a partial or invalid workspace
	// behind (R5/CA4). An empty --backend resolves to the deterministic default.
	backends, err := workspace.NormalizeBackends(splitBackends(*backend))
	if err != nil {
		logger.Error("init rejected", "phase", "backend-selection", "requested", *backend, "err", err)
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		return 2
	}
	logger.Info("backend selection resolved", "backends", strings.Join(backends, ","))

	// Detect first so we can decide between create/upgrade and render a preview
	// of the proposed changes before touching the filesystem.
	plan, err := workspace.DetectWithOptions(*dir, workspace.Options{Backends: backends})
	if err != nil {
		logger.Error("init failed", "phase", "detect", "err", err)
		fmt.Fprintf(stderr, "daedalus: init failed: %v\n", err)
		return 1
	}

	logger.Info("workspace detected",
		"path", plan.Path,
		"workspace_existed", plan.WorkspaceExisted,
		"missing_dirs", len(plan.MissingDirs),
		"missing_files", len(plan.MissingFiles))

	// Preview mode (--preview): report the plan and stop before any write so a
	// non-interactive validator can inspect the proposed changes safely.
	if *preview {
		writePreview(stdout, plan)
		logger.Info("init preview only", "applied", false)
		return 0
	}

	// In upgrade mode, always surface what will be added before applying it, so
	// the write is never a surprise (RNF-8 preview/confirm).
	if plan.WorkspaceExisted && !plan.IsEmpty() {
		writePreview(stdout, plan)
	}

	res, err := plan.Apply()
	if err != nil {
		logger.Error("init failed", "phase", "apply", "err", err)
		fmt.Fprintf(stderr, "daedalus: init failed: %v\n", err)
		return 1
	}

	logger.Info("workspace initialized",
		"path", res.Path,
		"already_existed", res.AlreadyExisted,
		"created_dirs", len(res.CreatedDirs),
		"created_files", len(res.CreatedFiles))

	writeResult(stdout, res)
	return 0
}

// writePreview renders, on stdout, the directories and root artifacts a plan
// would create. It lists nothing for an already-complete workspace so an
// idempotent re-run produces an explicit "nothing to update" preview.
func writePreview(stdout io.Writer, plan *workspace.Plan) {
	if plan.IsEmpty() {
		fmt.Fprintf(stdout, "Existing Daedalus workspace at %s is already complete — nothing to update.\n",
			plan.Path)
		return
	}

	fmt.Fprintf(stdout, "Preview of changes to the Daedalus workspace at %s:\n", plan.Path)
	for _, dir := range plan.MissingDirs {
		// filepath.ToSlash keeps the preview identical across Windows and Unix.
		fmt.Fprintf(stdout, "  + %s%s (directory)\n", filepath.ToSlash(dir), "/")
	}
	for _, file := range plan.MissingFiles {
		fmt.Fprintf(stdout, "  + %s (file)\n", filepath.ToSlash(file))
	}
}

// writeResult reports the outcome of an applied init, choosing wording that
// unambiguously distinguishes a from-scratch creation from an upgrade over an
// existing workspace (R7/CA6).
func writeResult(stdout io.Writer, res *workspace.Result) {
	switch {
	case res.AlreadyExisted && len(res.CreatedDirs) == 0 && len(res.CreatedFiles) == 0:
		fmt.Fprintf(stdout, "Existing Daedalus workspace at %s is already complete — nothing to update.\n",
			res.Path)
	case res.AlreadyExisted:
		fmt.Fprintf(stdout, "Upgraded existing Daedalus workspace at %s (added %d director%s, %d file%s).\n",
			res.Path,
			len(res.CreatedDirs), plural(len(res.CreatedDirs), "y", "ies"),
			len(res.CreatedFiles), plural(len(res.CreatedFiles), "", "s"))
	default:
		fmt.Fprintf(stdout, "Created Daedalus workspace at %s from scratch.\n", res.Path)
	}
}

// splitBackends turns the raw --backend flag value into a selection slice for
// workspace.NormalizeBackends. It splits on commas to leave the door open to
// multi-backend selection (--backend a,b) even though the MVP supports a single
// backend, and trims surrounding whitespace so "a, b" and "a,b" behave the same.
// An empty (or whitespace-only) flag yields a nil slice, which NormalizeBackends
// reads as "use the default". Blank entries (e.g. a trailing comma) are dropped
// so they never reach validation as an empty, unsupported backend.
func splitBackends(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// plural picks the singular or plural suffix for n, keeping result messages
// grammatical without pulling in a dependency.
func plural(n int, singular, pluralSuffix string) string {
	if n == 1 {
		return singular
	}
	return pluralSuffix
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
