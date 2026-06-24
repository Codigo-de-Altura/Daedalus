// Command daedalus is the entry point for the Daedalus TUI/CLI.
//
// Daedalus automates the setup and management of a project's AI scaffolding
// (agents, prompts, DAG workflows, SDD backlog) in a backend-agnostic way and
// compiles it to the native format of the chosen tool. With no subcommand it
// launches the Bubble Tea skeleton in an interactive terminal (and exits
// cleanly in non-interactive contexts); the `init` subcommand scaffolds the
// `.daedalus/` workspace in the target repository, and the `agent` subcommand
// lists the built-in catalog and materializes an agent into that workspace.
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
	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
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
		case "agent":
			return runAgent(args[1:], os.Stdout, os.Stderr)
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

// runAgent handles the `daedalus agent` subcommand, a thin CLI surface over the
// built-in catalog (internal/catalog). It dispatches to the operation named by
// the next argument so the verb set can grow without reshaping run(): today
// `list` (enumerate the built-in agents) and `add` (materialize one into the
// workspace). It keeps the same conventions as runInit — own usage, exit code 2
// for usage errors — so the CLI feels uniform.
func runAgent(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprint(stderr, agentUsage)
		return 2
	}

	switch args[0] {
	case "list":
		return runAgentList(args[1:], stdout, stderr)
	case "add":
		return runAgentAdd(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "daedalus: unknown agent operation %q\n\n%s", args[0], agentUsage)
		return 2
	}
}

// agentUsage is the shared help text for the `agent` subcommand, surfaced when no
// operation (or an unknown one) is given.
const agentUsage = "Usage: daedalus agent <operation> [flags]\n\n" +
	"Work with the built-in agent catalog.\n\n" +
	"Operations:\n" +
	"  list           list the built-in catalog agents (id and role)\n" +
	"  add <id>       materialize a catalog agent into .daedalus/agents/\n\n" +
	"Run 'daedalus agent add --help' for the add flags.\n"

// runAgentList handles `daedalus agent list`: it prints the built-in agents
// (id + role) to stdout in deterministic, id-sorted order. It takes no target
// directory because the built-in catalog is embedded in the binary (R1) — there
// is nothing on disk to read.
func runAgentList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus agent list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus agent list\n\n"+
			"List the built-in agents available in the catalog (id and role).\n")
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	entries := catalog.Builtin.List()
	logger := logging.New(stderr)
	logger.Info("catalog listed", "agents", len(entries))

	fmt.Fprintf(stdout, "Built-in agents (%d):\n", len(entries))
	for _, e := range entries {
		fmt.Fprintf(stdout, "  %s\t%s\n", e.ID, e.Role)
	}
	return 0
}

// runAgentAdd handles `daedalus agent add <id>`: it materializes the chosen
// catalog agent into the target workspace's .daedalus/agents/ directory, reusing
// the catalog's Plan/Apply split. It is non-destructive — an agent already
// present is reported as a conflict rather than overwritten (R6/CA4) — and
// --preview reports the files that would be created without writing anything.
func runAgentAdd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus agent add", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/agents/ the agent is added to")
	preview := fs.Bool("preview", false, "show the files that would be created without writing anything (dry run)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus agent add <id> [flags]\n\n"+
			"Materialize a built-in catalog agent into the target workspace's\n"+
			".daedalus/agents/ directory as its canonical definition (yaml + prompt md).\n"+
			"If the agent already exists it is not overwritten.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	// Go's flag parser stops at the first non-flag token, so `add <id> --path x`
	// would otherwise leave the flags unparsed after the positional id. Split the
	// single positional id out of the flag tokens first so the id can appear
	// before or after the flags (the natural `add analyst --path x` ordering).
	id, flags, err := splitAgentID(args)
	if err != nil {
		fmt.Fprintf(stderr, "daedalus: %v\n\n", err)
		fs.Usage()
		return 2
	}
	if err := fs.Parse(flags); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	logger := logging.New(stderr)

	// The agents live under the canonical workspace path so a materialized agent
	// lands exactly where init scaffolds the agents/ directory. We build it here
	// rather than in the catalog package so the catalog stays free of the
	// workspace-location convention.
	agentsRoot := filepath.Join(*dir, workspace.Name, catalog.AgentsDir)

	// Plan first: this validates the id (kebab-case + known agent) and renders the
	// content without touching the filesystem, so an invalid/unknown id fails as a
	// usage error before any write or preview.
	plan, err := catalog.Builtin.MaterializePlanFor(agentsRoot, id)
	if err != nil {
		// ErrAgentNotFound and a malformed id are both user/usage errors (exit 2):
		// the fix is to pick a valid id, not to retry the I/O.
		logger.Error("agent add rejected", "phase", "plan", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		if errors.Is(err, catalog.ErrAgentNotFound) {
			fmt.Fprint(stderr, "run 'daedalus agent list' to see the available agents\n")
		}
		return 2
	}

	logger.Info("agent materialization planned", "id", plan.AgentID, "dir", plan.Dir, "files", len(plan.Files))

	// Preview mode (--preview): report the files that would be created and stop
	// before any write so a validator can inspect the plan safely.
	if *preview {
		writeAgentPreview(stdout, plan)
		logger.Info("agent add preview only", "id", plan.AgentID, "applied", false)
		return 0
	}

	res, err := plan.Apply()
	if err != nil {
		logger.Error("agent add failed", "phase", "apply", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: agent add failed: %v\n", err)
		return 1
	}

	logger.Info("agent materialized",
		"id", res.AgentID,
		"already_existed", res.AlreadyExisted(),
		"created", len(res.Created),
		"skipped", len(res.Skipped))

	writeAgentResult(stdout, res)
	return 0
}

// writeAgentPreview renders, on stdout, the files an add would create. Paths are
// slash-normalized so the preview is byte-identical on Windows and Unix.
func writeAgentPreview(stdout io.Writer, plan *catalog.MaterializePlan) {
	fmt.Fprintf(stdout, "Preview of materializing agent %q into %s:\n",
		plan.AgentID, filepath.ToSlash(plan.Dir))
	for _, f := range plan.Files {
		fmt.Fprintf(stdout, "  + %s (file)\n", filepath.ToSlash(filepath.Join(plan.AgentID, f.Name)))
	}
}

// writeAgentResult reports the outcome of an applied add, choosing wording that
// unambiguously distinguishes a fresh materialization from the non-destructive
// case where the agent (fully or partially) already existed (R6/CA4). When some
// files were skipped, it names them so the user knows exactly what was preserved.
func writeAgentResult(stdout io.Writer, res *catalog.MaterializeResult) {
	dir := filepath.ToSlash(res.Dir)
	switch {
	case res.AlreadyExisted() && len(res.Created) == 0:
		fmt.Fprintf(stdout, "Agent %q already exists at %s — not overwritten (skipped %d file%s).\n",
			res.AgentID, dir, len(res.Skipped), plural(len(res.Skipped), "", "s"))
	case res.AlreadyExisted():
		// Partial conflict: some files were created, others preserved. Surface both
		// so the result is never mistaken for a clean create.
		fmt.Fprintf(stdout, "Agent %q partially materialized at %s (created %d, skipped %d existing file%s).\n",
			res.AgentID, dir, len(res.Created), len(res.Skipped), plural(len(res.Skipped), "", "s"))
	default:
		fmt.Fprintf(stdout, "Materialized agent %q at %s (created %d file%s).\n",
			res.AgentID, dir, len(res.Created), plural(len(res.Created), "", "s"))
	}
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

// valueFlags are the `agent add` flags that take a separate value token (e.g.
// `--path x`). splitAgentID consults this set so it does not mistake a flag's
// value for the positional agent id. Boolean flags (--preview) are absent
// because they never consume a following token.
var valueFlags = map[string]struct{}{
	"-path": {}, "--path": {},
}

// splitAgentID extracts the single positional agent id from an `agent add`
// argument list, returning it plus the remaining (flag) tokens in order. It
// exists because Go's flag parser stops at the first non-flag token: without
// pulling the id out first, `add <id> --path x` would leave --path unparsed.
// Exactly one positional is required; zero or more than one is a usage error so
// a typo like `add analyst architect` is caught instead of silently ignored. A
// `--help`/`-h` token is left in flags so the flag parser can handle it normally.
//
// It is value-flag aware: the token after `--path` is that flag's value, not the
// id, unless the flag already carries its value inline (`--path=x`).
func splitAgentID(args []string) (id string, flags []string, err error) {
	flags = make([]string, 0, len(args))
	var positionals []string
	expectValue := false
	for _, a := range args {
		if expectValue {
			// This token is the value of the preceding value-taking flag.
			flags = append(flags, a)
			expectValue = false
			continue
		}
		if strings.HasPrefix(a, "-") {
			flags = append(flags, a)
			// A `--path x` form (no '=') consumes the next token as its value.
			if _, takesValue := valueFlags[a]; takesValue {
				expectValue = true
			}
			continue
		}
		positionals = append(positionals, a)
	}
	// A help request is valid without an id; let the caller's parser show usage.
	for _, f := range flags {
		if f == "-h" || f == "--help" {
			return "", flags, nil
		}
	}
	switch len(positionals) {
	case 1:
		return positionals[0], flags, nil
	case 0:
		return "", flags, errors.New("agent add requires exactly one agent id")
	default:
		return "", flags, fmt.Errorf("agent add requires exactly one agent id, got %d", len(positionals))
	}
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
