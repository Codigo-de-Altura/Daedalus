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
	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/tui"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
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
		case "prompt":
			return runPrompt(args[1:], os.Stdout, os.Stderr)
		case "workflow":
			return runWorkflow(args[1:], os.Stdout, os.Stderr)
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

	// The TUI browses the current directory's `.daedalus/prompts/`. We resolve the
	// working directory here (falling back to "." if it cannot be determined) so the
	// presentation layer is handed an explicit root rather than reaching for the
	// process state itself.
	workdir, err := os.Getwd()
	if err != nil {
		workdir = "."
	}

	if _, err := tea.NewProgram(tui.New(workdir), tea.WithAltScreen()).Run(); err != nil {
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
	case "clone":
		return runAgentClone(args[1:], stdout, stderr)
	case "edit":
		return runAgentEdit(args[1:], stdout, stderr)
	case "import":
		return runAgentImport(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "daedalus: unknown agent operation %q\n\n%s", args[0], agentUsage)
		return 2
	}
}

// agentUsage is the shared help text for the `agent` subcommand, surfaced when no
// operation (or an unknown one) is given.
const agentUsage = "Usage: daedalus agent <operation> [flags]\n\n" +
	"Work with the agent catalog and workspace agents.\n\n" +
	"Operations:\n" +
	"  list                       list the built-in catalog agents (id and role)\n" +
	"  add <id>                   materialize a catalog agent into .daedalus/agents/\n" +
	"  clone <src> <dest>         copy a built-in agent to a new id you can edit\n" +
	"  edit <id> [flags]          edit a workspace agent's role, prompt or parameters\n" +
	"  import <path>              import agent(s) from a local file or directory\n\n" +
	"Run 'daedalus agent <operation> --help' for an operation's flags.\n"

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

// runAgentClone handles `daedalus agent clone <source-id> <dest-id>`: it copies a
// built-in catalog agent to a new, editable id under .daedalus/agents/, reusing
// the catalog's Plan/Apply split. It is non-destructive — a dest id that already
// exists is reported as a conflict rather than overwritten (R4/CA4) — and
// --preview reports the files that would be created without writing anything.
func runAgentClone(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus agent clone", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/agents/ the clone is written to")
	preview := fs.Bool("preview", false, "show the files that would be created without writing anything (dry run)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus agent clone <source-id> <dest-id> [flags]\n\n"+
			"Copy a built-in catalog agent to a new id in the target workspace's\n"+
			".daedalus/agents/ directory. The clone is an independent canonical\n"+
			"definition you can edit without affecting the built-in. If the dest id\n"+
			"already exists it is not overwritten.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	// Two positional ids (source, dest) sit among the flags; split them out first
	// so they can appear before or after the flags (Go's parser stops at the first
	// non-flag token otherwise).
	ids, flags := splitPositionals(args)
	if err := fs.Parse(flags); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if len(ids) != 2 {
		fmt.Fprint(stderr, "daedalus: agent clone requires exactly a source id and a dest id\n\n")
		fs.Usage()
		return 2
	}
	sourceID, destID := ids[0], ids[1]

	logger := logging.New(stderr)
	agentsRoot := filepath.Join(*dir, workspace.Name, catalog.AgentsDir)

	// Plan first: validates the dest id (kebab-case) and the source (known agent)
	// and renders the clone's content without touching the filesystem, so an
	// invalid/unknown id fails as a usage error before any write or preview.
	plan, err := catalog.Builtin.ClonePlanFor(agentsRoot, sourceID, destID)
	if err != nil {
		logger.Error("agent clone rejected", "phase", "plan", "source", sourceID, "dest", destID, "err", err)
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		if errors.Is(err, catalog.ErrAgentNotFound) {
			fmt.Fprint(stderr, "run 'daedalus agent list' to see the available agents\n")
		}
		return 2
	}

	logger.Info("agent clone planned", "source", sourceID, "dest", plan.AgentID, "dir", plan.Dir, "files", len(plan.Files))

	if *preview {
		writeAgentPreview(stdout, plan)
		logger.Info("agent clone preview only", "dest", plan.AgentID, "applied", false)
		return 0
	}

	res, err := plan.Apply()
	if err != nil {
		logger.Error("agent clone failed", "phase", "apply", "dest", destID, "err", err)
		fmt.Fprintf(stderr, "daedalus: agent clone failed: %v\n", err)
		return 1
	}

	logger.Info("agent cloned",
		"source", sourceID,
		"dest", res.AgentID,
		"already_existed", res.AlreadyExisted(),
		"created", len(res.Created),
		"skipped", len(res.Skipped))

	writeAgentResult(stdout, res)
	return 0
}

// runAgentEdit handles `daedalus agent edit <id>`: it edits a workspace agent's
// canonical definition in place (role, prompt and/or parameters). The edit is
// validated structurally before any write, so an edit that would leave the
// definition invalid (e.g. an empty role) is rejected with an actionable error
// and the existing files are left intact (R5/CA5). Writes are atomic.
func runAgentEdit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus agent edit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/agents/ holds the agent")
	role := fs.String("role", "", "set the agent's role/description")
	prompt := fs.String("prompt", "", "set the agent's prompt inline")
	promptFile := fs.String("prompt-file", "", "set the agent's prompt from a file (takes precedence over --prompt)")
	var setParams multiFlag
	var removeParams multiFlag
	fs.Var(&setParams, "set-param", "add or update a parameter as key=value (repeatable)")
	fs.Var(&removeParams, "remove-param", "remove a parameter by key (repeatable)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus agent edit <id> [flags]\n\n"+
			"Edit a workspace agent's canonical definition (role, prompt, parameters).\n"+
			"At least one edit flag is required. The edit is validated before writing;\n"+
			"an invalid edit is rejected and the existing definition is left intact.\n\n"+
			"If both --prompt-file and --prompt are given, --prompt-file wins.\n\nFlags:\n")
		fs.PrintDefaults()
	}
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

	// Build the edit spec from the flags. We track which flags were actually set
	// (visited) so passing --role "" is a deliberate (and invalid) edit rather
	// than indistinguishable from not passing --role at all.
	spec, err := buildEditSpec(fs, *role, *prompt, *promptFile, setParams, removeParams)
	if err != nil {
		logger.Error("agent edit rejected", "phase", "flags", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		return 2
	}
	if spec.IsEmpty() {
		fmt.Fprint(stderr, "daedalus: agent edit requires at least one edit flag "+
			"(--role, --prompt, --prompt-file, --set-param, --remove-param)\n\n")
		fs.Usage()
		return 2
	}

	agentsRoot := filepath.Join(*dir, workspace.Name, catalog.AgentsDir)

	edited, err := catalog.Builtin.Edit(agentsRoot, id, spec)
	if err != nil {
		// An unknown/absent agent or a malformed id is a usage error (exit 2); an
		// edit that fails validation is also a usage error (the fix is the input).
		// A genuine I/O failure is exit 1.
		switch {
		case errors.Is(err, catalog.ErrAgentNotFound):
			logger.Error("agent edit rejected", "phase", "load", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			fmt.Fprint(stderr, "the agent must already exist in the workspace; clone or add it first\n")
			return 2
		case errors.Is(err, catalog.ErrMalformedDefinition):
			logger.Error("agent edit failed", "phase", "load", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 1
		case isSchemaInvalid(err):
			logger.Error("agent edit rejected", "phase", "validate", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: agent %q is invalid; the edit was not applied:\n", id)
			writeSchemaErrors(stderr, err)
			return 2
		default:
			logger.Error("agent edit failed", "phase", "write", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: agent edit failed: %v\n", err)
			return 1
		}
	}

	logger.Info("agent edited", "id", edited.ID, "params", len(edited.Params))
	fmt.Fprintf(stdout, "Edited agent %q at %s.\n",
		edited.ID, filepath.ToSlash(filepath.Join(agentsRoot, edited.ID)))
	return 0
}

// buildEditSpec assembles a catalog.EditSpec from the parsed edit flags. It uses
// fs.Visit to learn which flags the user actually passed, so an explicit empty
// value (e.g. --role "") is recorded as a (rejectable) change rather than
// confused with the flag's default. --prompt-file takes precedence over --prompt
// when both are given (documented in Usage); the file content becomes the prompt.
func buildEditSpec(fs *flag.FlagSet, role, prompt, promptFile string, setParams, removeParams multiFlag) (catalog.EditSpec, error) {
	var spec catalog.EditSpec

	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	if set["role"] {
		spec.SetRole = true
		spec.Role = role
	}
	// Prompt precedence: --prompt-file wins over --prompt so a caller can keep a
	// long prompt in a file without it being shadowed by an accidental inline one.
	switch {
	case set["prompt-file"]:
		b, err := os.ReadFile(promptFile)
		if err != nil {
			return catalog.EditSpec{}, fmt.Errorf("reading --prompt-file: %w", err)
		}
		spec.SetPrompt = true
		spec.Prompt = string(b)
	case set["prompt"]:
		spec.SetPrompt = true
		spec.Prompt = prompt
	}

	for _, raw := range setParams {
		key, value, ok := strings.Cut(raw, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return catalog.EditSpec{}, fmt.Errorf("--set-param %q must be key=value", raw)
		}
		// The CLI cannot know a user's intended type, so every CLI-set parameter is
		// a string. Typed parameters (numbers/bools) come from the built-in catalog
		// and survive a round-trip untouched; editing them via the CLI converts them
		// to strings, which is the safe, lossless-for-text default.
		spec.SetParams = append(spec.SetParams, catalog.Param{
			Key:   strings.TrimSpace(key),
			Type:  catalog.ParamString,
			Value: value,
		})
	}
	spec.RemoveParams = append(spec.RemoveParams, removeParams...)

	return spec, nil
}

// isSchemaInvalid reports whether err is (or wraps) a canonical-schema validation
// failure — a *catalog.ValidationError. The catalog flows return this rich error
// type from their pre-write gates (ticket-02-04), so the CLI detects it
// structurally with errors.As rather than matching a message prefix, and can then
// render each finding's field/observed/expected.
func isSchemaInvalid(err error) bool {
	var ve *catalog.ValidationError
	return errors.As(err, &ve)
}

// writeSchemaErrors renders a schema validation failure as actionable lines, one
// finding per line (field: observed / expected), so the user can fix every problem
// in one pass (R3/R4). It falls back to the plain error text if err is not a
// *catalog.ValidationError, so it is safe to call on any error.
func writeSchemaErrors(w io.Writer, err error) {
	var ve *catalog.ValidationError
	if !errors.As(err, &ve) {
		fmt.Fprintf(w, "  - %v\n", err)
		return
	}
	for _, se := range ve.Errors {
		fmt.Fprintf(w, "  - %s: observed %s; expected %s\n", se.Field, se.Observed, se.Expected)
	}
}

// runAgentImport handles `daedalus agent import <path>`: it imports agent(s) from
// a local file or directory into the workspace's .daedalus/agents/, converting
// Claude Code (frontmatter) and canonical sources to canonical definitions. It is
// non-destructive — an id that already exists is reported as a conflict, not
// overwritten (R5/CA4) — and --preview reports what would be imported without
// writing. A directory import reports each agent independently and a final
// summary; one invalid source does not abort the valid ones (the failures are
// reported, the rest are imported).
//
// Exit code reflects the worst per-agent outcome: 0 when every source imported (or
// was a non-destructive skip), 2 when any source failed to parse/normalize/
// validate (an actionable user error), 1 on a genuine I/O failure.
func runAgentImport(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus agent import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/agents/ receives the import")
	preview := fs.Bool("preview", false, "show what would be imported without writing anything (dry run)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus agent import <source> [flags]\n\n"+
			"Import agent(s) from a local file or directory into the target workspace's\n"+
			".daedalus/agents/ directory, converting Claude Code (.claude/agents/*.md)\n"+
			"and canonical definitions to the canonical agnostic format. A directory is\n"+
			"scanned shallowly (each file is a candidate). Existing agents are not\n"+
			"overwritten; invalid sources are reported and skipped.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	// One positional source path among the flags; split it out so it can sit before
	// or after the flags (Go's parser stops at the first non-flag token otherwise).
	positionals, flags := splitPositionals(args)
	if err := fs.Parse(flags); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if len(positionals) != 1 {
		fmt.Fprint(stderr, "daedalus: agent import requires exactly one source path\n\n")
		fs.Usage()
		return 2
	}
	source := positionals[0]

	logger := logging.New(stderr)
	agentsRoot := filepath.Join(*dir, workspace.Name, catalog.AgentsDir)

	// Plan first: scans and converts every source without writing, so an invalid
	// source is detected before any filesystem change and --preview is honest.
	plan, err := catalog.ImportPlanFor(agentsRoot, source)
	if err != nil {
		// Stat/read failure on the source path itself (not a per-agent error).
		logger.Error("agent import failed", "phase", "scan", "source", source, "err", err)
		fmt.Fprintf(stderr, "daedalus: agent import failed: %v\n", err)
		return 1
	}

	logger.Info("agent import planned",
		"source", source, "importable", len(plan.Agents), "errors", len(plan.Errors))

	if *preview {
		writeImportPreview(stdout, plan)
		// A preview still surfaces parse/validation errors so the user can fix them
		// before a real run; those make the preview exit non-zero (usage error).
		if len(plan.Errors) > 0 {
			writeImportErrors(stdout, plan.Errors)
			return 2
		}
		return 0
	}

	outcomes, err := plan.Apply()
	if err != nil {
		logger.Error("agent import failed", "phase", "apply", "source", source, "err", err)
		fmt.Fprintf(stderr, "daedalus: agent import failed: %v\n", err)
		return 1
	}

	code := writeImportResult(stdout, stderr, outcomes)
	logger.Info("agent import done", "outcomes", len(outcomes), "exit", code)
	return code
}

// writeImportPreview reports, on stdout, the agents an import would create.
func writeImportPreview(stdout io.Writer, plan *catalog.ImportPlan) {
	if len(plan.Agents) == 0 {
		fmt.Fprintln(stdout, "Preview: no importable agents found.")
		return
	}
	fmt.Fprintf(stdout, "Preview of importing %d agent(s):\n", len(plan.Agents))
	for _, mp := range plan.Agents {
		fmt.Fprintf(stdout, "  + %s -> %s\n", mp.AgentID, filepath.ToSlash(mp.Dir))
	}
}

// writeImportErrors reports parse/normalization/validation failures, one per
// source, with the file that failed and why — the actionable detail (R4/CA3).
func writeImportErrors(w io.Writer, errs []catalog.ImportError) {
	for _, e := range errs {
		fmt.Fprintf(w, "  ! %s: %v\n", filepath.ToSlash(e.SourcePath), e.Err)
	}
}

// writeImportResult prints a per-agent line and a summary, and computes the exit
// code from the outcomes: 2 if any source failed (actionable user error), else 0.
// (A write I/O failure is also reported per-agent and maps to exit 2 here because
// Apply already returned nil for the operation as a whole; a hard operational
// failure is handled by the caller's earlier err check.)
func writeImportResult(stdout, stderr io.Writer, outcomes []catalog.ImportOutcome) int {
	var imported, skipped, failed int
	for _, o := range outcomes {
		switch {
		case o.Err != nil:
			failed++
			src := o.SourcePath
			if src == "" {
				src = o.AgentID
			}
			fmt.Fprintf(stderr, "  ! %s: %v\n", filepath.ToSlash(src), o.Err)
		case o.AlreadyExisted():
			skipped++
			fmt.Fprintf(stdout, "  = %s already exists at %s — not overwritten (skipped %d file%s).\n",
				o.AgentID, filepath.ToSlash(o.Dir), len(o.Skipped), plural(len(o.Skipped), "", "s"))
		default:
			imported++
			fmt.Fprintf(stdout, "  + %s imported to %s (created %d file%s).\n",
				o.AgentID, filepath.ToSlash(o.Dir), len(o.Created), plural(len(o.Created), "", "s"))
		}
	}

	fmt.Fprintf(stdout, "Import summary: %d imported, %d already existed, %d failed.\n",
		imported, skipped, failed)
	if failed > 0 {
		return 2
	}
	return 0
}

// runPrompt handles the `daedalus prompt` subcommand, a thin CLI surface over the
// prompts domain (internal/prompts). It dispatches to the operation named by the
// next argument so the verb set can grow without reshaping run(): list, create,
// edit, show, remove. It keeps the same conventions as runAgent — own usage, exit
// code 2 for usage errors, logging to stderr — so the CLI feels uniform.
func runPrompt(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprint(stderr, promptUsage)
		return 2
	}

	switch args[0] {
	case "list":
		return runPromptList(args[1:], stdout, stderr)
	case "create":
		return runPromptCreate(args[1:], stdout, stderr)
	case "edit":
		return runPromptEdit(args[1:], stdout, stderr)
	case "show":
		return runPromptShow(args[1:], stdout, stderr)
	case "render":
		return runPromptRender(args[1:], stdout, stderr)
	case "remove":
		return runPromptRemove(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "daedalus: unknown prompt operation %q\n\n%s", args[0], promptUsage)
		return 2
	}
}

// promptUsage is the shared help text for the `prompt` subcommand, surfaced when
// no operation (or an unknown one) is given.
const promptUsage = "Usage: daedalus prompt <operation> [flags]\n\n" +
	"Work with reusable global and shared prompts in .daedalus/prompts/.\n\n" +
	"Operations:\n" +
	"  list [--kind global|shared]   list persisted prompts (id, kind, title)\n" +
	"  create <id> --kind <k> --title <t> [flags]   create a new prompt\n" +
	"  edit <id> [flags]             edit a prompt's title, description or body\n" +
	"  show <id>                     print a prompt's file content verbatim (raw)\n" +
	"  render <id>                   print the composed prompt with inclusions resolved\n" +
	"  remove <id>                   delete a prompt's file\n\n" +
	"Run 'daedalus prompt <operation> --help' for an operation's flags.\n"

// promptsRootFor builds the canonical `.daedalus/prompts/` directory under dir so
// a prompt lands exactly where init scaffolds the prompts/ directory. Built here
// rather than in the prompts package so that package stays free of the
// workspace-location convention (mirrors how agentsRoot is derived).
func promptsRootFor(dir string) string {
	return filepath.Join(dir, workspace.Name, prompts.PromptsDir)
}

// runPromptList handles `daedalus prompt list [--kind global|shared]`: it prints
// the persisted prompts (id, kind, title) in deterministic, id-sorted order,
// optionally filtered by kind.
func runPromptList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus prompt list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/prompts/ is listed")
	kind := fs.String("kind", "", "filter by kind: global or shared (default: all)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus prompt list [--kind global|shared] [--path .]\n\n"+
			"List the persisted prompts (id, kind, title), optionally filtered by kind.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	logger := logging.New(stderr)

	filter, err := parseKindFilter(*kind)
	if err != nil {
		logger.Error("prompt list rejected", "phase", "flags", "kind", *kind, "err", err)
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		return 2
	}

	entries, err := prompts.List(promptsRootFor(*dir), filter)
	if err != nil {
		logger.Error("prompt list failed", "phase", "list", "err", err)
		fmt.Fprintf(stderr, "daedalus: prompt list failed: %v\n", err)
		return 1
	}

	logger.Info("prompts listed", "prompts", len(entries), "filter", *kind)

	if *kind != "" {
		fmt.Fprintf(stdout, "Prompts (%d, kind=%s):\n", len(entries), filter)
	} else {
		fmt.Fprintf(stdout, "Prompts (%d):\n", len(entries))
	}
	for _, e := range entries {
		fmt.Fprintf(stdout, "  %s\t%s\t%s\n", e.ID, e.Kind, e.Title)
	}
	return 0
}

// parseKindFilter validates the --kind flag value into a prompts.Kind, treating
// an empty value as "no filter". An unknown value is a usage error so a typo like
// `--kind globl` is caught instead of silently returning everything.
func parseKindFilter(raw string) (prompts.Kind, error) {
	switch raw {
	case "":
		return "", nil
	case string(prompts.KindGlobal):
		return prompts.KindGlobal, nil
	case string(prompts.KindShared):
		return prompts.KindShared, nil
	default:
		return "", fmt.Errorf("invalid --kind %q: expected %s or %s", raw, prompts.KindGlobal, prompts.KindShared)
	}
}

// runPromptCreate handles `daedalus prompt create <id> --kind <k> --title <t>`:
// it creates a new prompt under .daedalus/prompts/. The prompt is validated before
// any write, so an invalid id/kind/title is rejected with an actionable error and
// nothing is written. It is non-destructive — a duplicate id is reported as a
// conflict, not overwritten (R4/R8) — and --preview reports the file that would be
// created without writing anything.
func runPromptCreate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus prompt create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/prompts/ the prompt is added to")
	kind := fs.String("kind", "", "prompt kind: global or shared (required)")
	title := fs.String("title", "", "prompt title (required)")
	description := fs.String("description", "", "optional one-line description")
	body := fs.String("body", "", "prompt body inline")
	bodyFile := fs.String("body-file", "", "prompt body from a file (takes precedence over --body)")
	preview := fs.Bool("preview", false, "show the file that would be created without writing anything (dry run)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus prompt create <id> --kind <global|shared> --title <t> [flags]\n\n"+
			"Create a new reusable prompt in the target workspace's .daedalus/prompts/\n"+
			"directory as <id>.md. If the prompt already exists it is not overwritten.\n"+
			"If both --body-file and --body are given, --body-file wins.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	id, flags, err := splitPromptID(args)
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

	// Resolve the body: --body-file wins over --body so a caller can keep a long
	// prompt in a file without it being shadowed by an accidental inline one.
	bodyText := *body
	if *bodyFile != "" {
		b, err := os.ReadFile(*bodyFile)
		if err != nil {
			logger.Error("prompt create rejected", "phase", "flags", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: reading --body-file: %v\n", err)
			return 2
		}
		bodyText = string(b)
	}

	p := prompts.Prompt{
		ID:          id,
		Kind:        prompts.Kind(*kind),
		Title:       *title,
		Description: *description,
		Body:        bodyText,
	}

	promptsRoot := promptsRootFor(*dir)

	// Plan first: this validates the prompt and renders the content without touching
	// the filesystem, so an invalid prompt fails as a usage error before any write
	// or preview.
	plan, err := prompts.PlanCreate(promptsRoot, p)
	if err != nil {
		logger.Error("prompt create rejected", "phase", "plan", "id", id, "err", err)
		if isPromptInvalid(err) {
			fmt.Fprintf(stderr, "daedalus: prompt %q is invalid; it was not created:\n", id)
			writePromptSchemaErrors(stderr, err)
		} else {
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
		}
		return 2
	}

	logger.Info("prompt create planned", "id", plan.Prompt.ID, "kind", plan.Prompt.Kind, "path", plan.Path)

	if *preview {
		fmt.Fprintf(stdout, "Preview of creating prompt %q (%s) at %s:\n",
			plan.Prompt.ID, plan.Prompt.Kind, filepath.ToSlash(plan.Path))
		fmt.Fprint(stdout, plan.Content)
		logger.Info("prompt create preview only", "id", plan.Prompt.ID, "applied", false)
		return 0
	}

	if err := plan.Apply(); err != nil {
		if errors.Is(err, prompts.ErrPromptExists) {
			logger.Error("prompt create rejected", "phase", "apply", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v — not overwritten\n", err)
			return 2
		}
		logger.Error("prompt create failed", "phase", "apply", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: prompt create failed: %v\n", err)
		return 1
	}

	logger.Info("prompt created", "id", plan.Prompt.ID, "kind", plan.Prompt.Kind)
	fmt.Fprintf(stdout, "Created prompt %q (%s) at %s.\n",
		plan.Prompt.ID, plan.Prompt.Kind, filepath.ToSlash(plan.Path))
	return 0
}

// runPromptEdit handles `daedalus prompt edit <id>`: it edits a prompt's title,
// description and/or body in place. The edit is validated before any write, so an
// edit that would leave the prompt invalid (e.g. an empty title) is rejected with
// an actionable error and the existing file is left intact (R5). Writes are atomic.
func runPromptEdit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus prompt edit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/prompts/ holds the prompt")
	title := fs.String("title", "", "set the prompt's title")
	description := fs.String("description", "", "set the prompt's description (empty clears it)")
	body := fs.String("body", "", "set the prompt's body inline")
	bodyFile := fs.String("body-file", "", "set the prompt's body from a file (takes precedence over --body)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus prompt edit <id> [flags]\n\n"+
			"Edit a prompt's title, description or body. At least one edit flag is\n"+
			"required. The edit is validated before writing; an invalid edit is rejected\n"+
			"and the existing file is left intact. If both --body-file and --body are\n"+
			"given, --body-file wins.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	id, flags, err := splitPromptID(args)
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

	spec, err := buildPromptEditSpec(fs, *title, *description, *body, *bodyFile)
	if err != nil {
		logger.Error("prompt edit rejected", "phase", "flags", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		return 2
	}
	if spec.IsEmpty() {
		fmt.Fprint(stderr, "daedalus: prompt edit requires at least one edit flag "+
			"(--title, --description, --body, --body-file)\n\n")
		fs.Usage()
		return 2
	}

	promptsRoot := promptsRootFor(*dir)

	edited, err := prompts.Edit(promptsRoot, id, spec)
	if err != nil {
		switch {
		case errors.Is(err, prompts.ErrPromptNotFound):
			logger.Error("prompt edit rejected", "phase", "load", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			fmt.Fprint(stderr, "the prompt must already exist; create it first\n")
			return 2
		case errors.Is(err, prompts.ErrMalformedPrompt):
			logger.Error("prompt edit failed", "phase", "load", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 1
		case isPromptInvalid(err):
			logger.Error("prompt edit rejected", "phase", "validate", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: prompt %q is invalid; the edit was not applied:\n", id)
			writePromptSchemaErrors(stderr, err)
			return 2
		default:
			logger.Error("prompt edit failed", "phase", "write", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: prompt edit failed: %v\n", err)
			return 1
		}
	}

	logger.Info("prompt edited", "id", edited.ID, "kind", edited.Kind)
	fmt.Fprintf(stdout, "Edited prompt %q at %s.\n",
		edited.ID, filepath.ToSlash(filepath.Join(promptsRoot, edited.ID+prompts.FileExt)))
	return 0
}

// buildPromptEditSpec assembles a prompts.EditSpec from the parsed edit flags. It
// uses fs.Visit to learn which flags the user actually passed, so an explicit
// empty value (e.g. --title "") is recorded as a (rejectable) change rather than
// confused with the flag's default. --body-file takes precedence over --body.
func buildPromptEditSpec(fs *flag.FlagSet, title, description, body, bodyFile string) (prompts.EditSpec, error) {
	var spec prompts.EditSpec

	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	if set["title"] {
		spec.SetTitle = true
		spec.Title = title
	}
	if set["description"] {
		spec.SetDescription = true
		spec.Description = description
	}
	// Body precedence: --body-file wins over --body so a caller can keep a long
	// body in a file without it being shadowed by an accidental inline one.
	switch {
	case set["body-file"]:
		b, err := os.ReadFile(bodyFile)
		if err != nil {
			return prompts.EditSpec{}, fmt.Errorf("reading --body-file: %w", err)
		}
		spec.SetBody = true
		spec.Body = string(b)
	case set["body"]:
		spec.SetBody = true
		spec.Body = body
	}

	return spec, nil
}

// runPromptShow handles `daedalus prompt show <id>`: it prints the prompt's file
// content verbatim to stdout, so the user sees exactly what is persisted.
func runPromptShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus prompt show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/prompts/ holds the prompt")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus prompt show <id> [--path .]\n\n"+
			"Print the prompt's file content verbatim.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	id, flags, err := splitPromptID(args)
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

	if !prompts.IsKebabCase(id) {
		fmt.Fprintf(stderr, "daedalus: prompt id %q is not valid kebab-case\n", id)
		return 2
	}

	path := filepath.Join(promptsRootFor(*dir), id+prompts.FileExt)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Error("prompt show rejected", "phase", "read", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: prompt %q not found\n", id)
			return 2
		}
		logger.Error("prompt show failed", "phase", "read", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: prompt show failed: %v\n", err)
		return 1
	}

	logger.Info("prompt shown", "id", id)
	fmt.Fprint(stdout, string(content))
	return 0
}

// runPromptRender handles `daedalus prompt render <id>`: it prints the composed
// prompt with all inclusion directives resolved (internal/prompts.Resolve),
// distinct from `show` which prints the raw file. Composition is read-only and
// non-mutating (R8). Composition failures are surfaced as actionable usage errors
// that distinguish a missing reference from a cycle (R5/R6).
func runPromptRender(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus prompt render", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/prompts/ holds the prompt")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus prompt render <id> [--path .]\n\n"+
			"Print the composed prompt with all {{include: <id>}} directives resolved\n"+
			"recursively. The source files are never modified.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	id, flags, err := splitPromptID(args)
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

	if !prompts.IsKebabCase(id) {
		fmt.Fprintf(stderr, "daedalus: prompt id %q is not valid kebab-case\n", id)
		return 2
	}

	composed, err := prompts.Resolve(promptsRootFor(*dir), id)
	if err != nil {
		switch {
		case errors.Is(err, prompts.ErrIncludeCycle):
			logger.Error("prompt render rejected", "phase", "resolve", "id", id, "reason", "cycle", "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		case errors.Is(err, prompts.ErrIncludeNotFound):
			logger.Error("prompt render rejected", "phase", "resolve", "id", id, "reason", "missing-include", "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		case errors.Is(err, prompts.ErrPromptNotFound):
			logger.Error("prompt render rejected", "phase", "resolve", "id", id, "reason", "not-found", "err", err)
			fmt.Fprintf(stderr, "daedalus: prompt %q not found\n", id)
			return 2
		case errors.Is(err, prompts.ErrMalformedPrompt):
			logger.Error("prompt render failed", "phase", "resolve", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 1
		default:
			logger.Error("prompt render failed", "phase", "resolve", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: prompt render failed: %v\n", err)
			return 1
		}
	}

	logger.Info("prompt rendered", "id", id)
	// A single trailing newline so the composed text is a clean line on stdout,
	// matching how the raw `show` output ends.
	fmt.Fprintln(stdout, composed)
	return 0
}

// runPromptRemove handles `daedalus prompt remove <id>`: it deletes the prompt's
// file and nothing else. An absent prompt is reported as an explicit error (R8).
func runPromptRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus prompt remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/prompts/ holds the prompt")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus prompt remove <id> [--path .]\n\n"+
			"Delete a prompt's file from the workspace. Only that file is removed.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	id, flags, err := splitPromptID(args)
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
	promptsRoot := promptsRootFor(*dir)

	if err := prompts.Remove(promptsRoot, id); err != nil {
		if errors.Is(err, prompts.ErrPromptNotFound) {
			logger.Error("prompt remove rejected", "phase", "remove", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		}
		// A malformed (non-kebab-case) id is also a usage error.
		if !prompts.IsKebabCase(id) {
			logger.Error("prompt remove rejected", "phase", "remove", "id", id, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		}
		logger.Error("prompt remove failed", "phase", "remove", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: prompt remove failed: %v\n", err)
		return 1
	}

	logger.Info("prompt removed", "id", id)
	fmt.Fprintf(stdout, "Removed prompt %q from %s.\n",
		id, filepath.ToSlash(filepath.Join(promptsRoot, id+prompts.FileExt)))
	return 0
}

// isPromptInvalid reports whether err is (or wraps) a *prompts.ValidationError —
// the rich error the prompts flows return from their pre-write gates — so the CLI
// detects it structurally with errors.As rather than matching a message prefix.
func isPromptInvalid(err error) bool {
	var ve *prompts.ValidationError
	return errors.As(err, &ve)
}

// writePromptSchemaErrors renders a prompt validation failure as actionable lines,
// one finding per line (field: observed / expected), so the user can fix every
// problem in one pass (R8). It falls back to the plain error text if err is not a
// *prompts.ValidationError, so it is safe to call on any error.
func writePromptSchemaErrors(w io.Writer, err error) {
	var ve *prompts.ValidationError
	if !errors.As(err, &ve) {
		fmt.Fprintf(w, "  - %v\n", err)
		return
	}
	for _, se := range ve.Errors {
		fmt.Fprintf(w, "  - %s: observed %s; expected %s\n", se.Field, se.Observed, se.Expected)
	}
}

// runWorkflow handles the `daedalus workflow` subcommand, a thin CLI surface over
// the workflows domain (internal/workflows). It dispatches to the operation named
// by the next argument so the verb set can grow without reshaping run(): list,
// create, show, remove, plus the phase edit operations add-phase/edit-phase/
// remove-phase. It keeps the same conventions as runPrompt — own usage, exit code
// 2 for usage errors, logging to stderr — so the CLI feels uniform.
func runWorkflow(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprint(stderr, workflowUsage)
		return 2
	}

	switch args[0] {
	case "list":
		return runWorkflowList(args[1:], stdout, stderr)
	case "create":
		return runWorkflowCreate(args[1:], stdout, stderr)
	case "show":
		return runWorkflowShow(args[1:], stdout, stderr)
	case "remove":
		return runWorkflowRemove(args[1:], stdout, stderr)
	case "validate":
		return runWorkflowValidate(args[1:], stdout, stderr)
	case "add-phase":
		return runWorkflowAddPhase(args[1:], stdout, stderr)
	case "edit-phase":
		return runWorkflowEditPhase(args[1:], stdout, stderr)
	case "remove-phase":
		return runWorkflowRemovePhase(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "daedalus: unknown workflow operation %q\n\n%s", args[0], workflowUsage)
		return 2
	}
}

// workflowUsage is the shared help text for the `workflow` subcommand, surfaced
// when no operation (or an unknown one) is given.
const workflowUsage = "Usage: daedalus workflow <operation> [flags]\n\n" +
	"Work with DAG workflows in .daedalus/workflows/.\n\n" +
	"Operations:\n" +
	"  list                          list persisted workflows (name, phase count)\n" +
	"  create <name>                 create a new (empty) workflow as <name>.yaml\n" +
	"  show <name>                   print a workflow's file content verbatim (raw)\n" +
	"  remove <name>                 delete a workflow's file\n" +
	"  validate <name>               check the DAG semantics (cycles, artifacts, agents)\n" +
	"  add-phase <name> --id <id> --agent <a> --gate <g> [flags]   append a phase\n" +
	"  edit-phase <name> --id <id> [flags]                         edit a phase\n" +
	"  remove-phase <name> --id <id>                               remove a phase\n\n" +
	"Run 'daedalus workflow <operation> --help' for an operation's flags.\n"

// workflowsRootFor builds the canonical `.daedalus/workflows/` directory under dir
// so a workflow lands exactly where init scaffolds the workflows/ directory. Built
// here rather than in the workflows package so that package stays free of the
// workspace-location convention (mirrors how promptsRootFor is derived).
func workflowsRootFor(dir string) string {
	return filepath.Join(dir, workspace.Name, workflows.WorkflowsDir)
}

// runWorkflowList handles `daedalus workflow list`: it prints the persisted
// workflows (name, phase count) in deterministic, name-sorted order.
func runWorkflowList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus workflow list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/workflows/ is listed")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus workflow list [--path .]\n\n"+
			"List the persisted workflows (name, phase count).\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	logger := logging.New(stderr)

	entries, err := workflows.List(workflowsRootFor(*dir))
	if err != nil {
		logger.Error("workflow list failed", "phase", "list", "err", err)
		fmt.Fprintf(stderr, "daedalus: workflow list failed: %v\n", err)
		return 1
	}

	logger.Info("workflows listed", "workflows", len(entries))
	fmt.Fprintf(stdout, "Workflows (%d):\n", len(entries))
	for _, e := range entries {
		fmt.Fprintf(stdout, "  %s\t%d phase%s\n", e.Name, e.Phases, plural(e.Phases, "", "s"))
	}
	return 0
}

// runWorkflowCreate handles `daedalus workflow create <name>`: it creates a new,
// empty workflow under .daedalus/workflows/ as <name>.yaml. The name is validated
// before any write. It is non-destructive — a duplicate name is reported as a
// conflict, not overwritten (R4) — and --preview reports the file that would be
// created without writing anything. Phases are added afterwards with add-phase.
func runWorkflowCreate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus workflow create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/workflows/ the workflow is added to")
	preview := fs.Bool("preview", false, "show the file that would be created without writing anything (dry run)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus workflow create <name> [flags]\n\n"+
			"Create a new, empty workflow in the target workspace's .daedalus/workflows/\n"+
			"directory as <name>.yaml. If the workflow already exists it is not\n"+
			"overwritten. Add phases with 'daedalus workflow add-phase'.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	name, flags, err := splitWorkflowName(args)
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
	workflowsRoot := workflowsRootFor(*dir)

	// Plan first: validates the name and renders the content without touching the
	// filesystem, so an invalid name fails as a usage error before any write.
	plan, err := workflows.PlanCreate(workflowsRoot, workflows.Workflow{Name: name})
	if err != nil {
		logger.Error("workflow create rejected", "phase", "plan", "name", name, "err", err)
		if isWorkflowInvalid(err) {
			fmt.Fprintf(stderr, "daedalus: workflow %q is invalid; it was not created:\n", name)
			writeWorkflowSchemaErrors(stderr, err)
		} else {
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
		}
		return 2
	}

	logger.Info("workflow create planned", "name", plan.Workflow.Name, "path", plan.Path)

	if *preview {
		fmt.Fprintf(stdout, "Preview of creating workflow %q at %s:\n",
			plan.Workflow.Name, filepath.ToSlash(plan.Path))
		fmt.Fprint(stdout, plan.Content)
		logger.Info("workflow create preview only", "name", plan.Workflow.Name, "applied", false)
		return 0
	}

	if err := plan.Apply(); err != nil {
		if errors.Is(err, workflows.ErrWorkflowExists) {
			logger.Error("workflow create rejected", "phase", "apply", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v — not overwritten\n", err)
			return 2
		}
		logger.Error("workflow create failed", "phase", "apply", "name", name, "err", err)
		fmt.Fprintf(stderr, "daedalus: workflow create failed: %v\n", err)
		return 1
	}

	logger.Info("workflow created", "name", plan.Workflow.Name)
	fmt.Fprintf(stdout, "Created workflow %q at %s.\n",
		plan.Workflow.Name, filepath.ToSlash(plan.Path))
	return 0
}

// runWorkflowShow handles `daedalus workflow show <name>`: it prints the
// workflow's file content verbatim to stdout, so the user sees exactly what is
// persisted (the canonical YAML).
func runWorkflowShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus workflow show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/workflows/ holds the workflow")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus workflow show <name> [--path .]\n\n"+
			"Print the workflow's file content verbatim.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	name, flags, err := splitWorkflowName(args)
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

	if !workflows.IsKebabCase(name) {
		fmt.Fprintf(stderr, "daedalus: workflow name %q is not valid kebab-case\n", name)
		return 2
	}

	path := filepath.Join(workflowsRootFor(*dir), name+workflows.FileExt)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Error("workflow show rejected", "phase", "read", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: workflow %q not found\n", name)
			return 2
		}
		logger.Error("workflow show failed", "phase", "read", "name", name, "err", err)
		fmt.Fprintf(stderr, "daedalus: workflow show failed: %v\n", err)
		return 1
	}

	logger.Info("workflow shown", "name", name)
	fmt.Fprint(stdout, string(content))
	return 0
}

// runWorkflowRemove handles `daedalus workflow remove <name>`: it deletes the
// workflow's file and nothing else. An absent workflow is an explicit error.
func runWorkflowRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus workflow remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/workflows/ holds the workflow")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus workflow remove <name> [--path .]\n\n"+
			"Delete a workflow's file from the workspace. Only that file is removed.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	name, flags, err := splitWorkflowName(args)
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
	workflowsRoot := workflowsRootFor(*dir)

	if err := workflows.Remove(workflowsRoot, name); err != nil {
		if errors.Is(err, workflows.ErrWorkflowNotFound) || !workflows.IsKebabCase(name) {
			logger.Error("workflow remove rejected", "phase", "remove", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		}
		logger.Error("workflow remove failed", "phase", "remove", "name", name, "err", err)
		fmt.Fprintf(stderr, "daedalus: workflow remove failed: %v\n", err)
		return 1
	}

	logger.Info("workflow removed", "name", name)
	fmt.Fprintf(stdout, "Removed workflow %q from %s.\n",
		name, filepath.ToSlash(filepath.Join(workflowsRoot, name+workflows.FileExt)))
	return 0
}

// runWorkflowValidate handles `daedalus workflow validate <name>`: it loads the
// workflow and runs the core semantic validator (internal/workflows.ValidateGraph)
// against it, reporting cycles, missing artifacts and unknown agents. This CLI
// layer is the one that resolves agent existence: the core stays backend-agnostic
// (it never imports the catalog), so we build the set of known agent ids here and
// inject it as a predicate. Exit code is 0 when valid, 1 when semantically invalid
// (so a CI gate can branch on it), and 2 on a usage/load error.
func runWorkflowValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus workflow validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/workflows/ holds the workflow")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus workflow validate <name> [--path .]\n\n"+
			"Validate a workflow's DAG semantics: dependency cycles, input artifacts not\n"+
			"produced by any predecessor (nor the initial 'brief'), and agents that do not\n"+
			"exist in the workspace. Exit 0 if valid, 1 if invalid, 2 on a usage error.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	name, flags, err := splitWorkflowName(args)
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

	if !workflows.IsKebabCase(name) {
		fmt.Fprintf(stderr, "daedalus: workflow name %q is not valid kebab-case\n", name)
		return 2
	}

	wf, err := workflows.Load(workflowsRootFor(*dir), name)
	if err != nil {
		switch {
		case errors.Is(err, workflows.ErrWorkflowNotFound):
			logger.Error("workflow validate rejected", "phase", "load", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		case errors.Is(err, workflows.ErrMalformedWorkflow):
			logger.Error("workflow validate failed", "phase", "load", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		default:
			logger.Error("workflow validate failed", "phase", "load", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: workflow validate failed: %v\n", err)
			return 1
		}
	}

	known := knownAgentsFor(*dir)
	report := wf.ValidateGraph(func(id string) bool { return known[id] })

	logger.Info("workflow validated",
		"name", name, "valid", report.Valid(), "findings", len(report.Findings), "known_agents", len(known))

	if report.Valid() {
		fmt.Fprintf(stdout, "Workflow %q is semantically valid.\n", name)
		return 0
	}

	fmt.Fprintf(stdout, "Workflow %q is semantically invalid (%d finding%s):\n",
		name, len(report.Findings), plural(len(report.Findings), "", "s"))
	for _, f := range report.Findings {
		fmt.Fprintf(stdout, "  - %s\n", f.Error())
	}
	return 1
}

// knownAgentsFor builds the set of agent ids the workspace recognizes, used by
// `workflow validate` to resolve the unknown-agent check. It is the union of two
// sources:
//
//   - the agents materialized in the workspace (each subdirectory of
//     `.daedalus/agents/` is a materialized agent id, per catalog.Load's layout), and
//   - the built-in catalog ids (catalog.Builtin.List()).
//
// The built-ins are included on purpose: a phase may legitimately reference a
// built-in agent (analyst, architect, ...) that the user has not materialized into
// the workspace yet — it still "exists" as far as the project is concerned, so
// flagging it as unknown would be a false positive. Only an id that is neither
// materialized nor a known built-in is genuinely unknown. This resolution lives in
// the CLI, not in internal/workflows, so the core stays backend/catalog-agnostic.
func knownAgentsFor(dir string) map[string]bool {
	known := make(map[string]bool)

	// Built-in catalog ids are always considered known.
	for _, e := range catalog.Builtin.List() {
		known[e.ID] = true
	}

	// Materialized workspace agents: each subdirectory of .daedalus/agents/ whose
	// name is a valid agent id. A missing agents directory is simply "no extra
	// agents" — not an error — so validating a workspace without materialized agents
	// still works (only built-ins are known).
	agentsRoot := filepath.Join(dir, workspace.Name, catalog.AgentsDir)
	entries, err := os.ReadDir(agentsRoot)
	if err != nil {
		return known
	}
	for _, de := range entries {
		if !de.IsDir() {
			continue
		}
		if catalog.IsKebabCase(de.Name()) {
			known[de.Name()] = true
		}
	}
	return known
}

// runWorkflowAddPhase handles `daedalus workflow add-phase <name> --id <id> ...`:
// it appends a phase to an existing workflow. The edit is validated before any
// write (the whole resulting workflow must stay structurally valid), so a bad
// phase id or a missing required field is rejected with an actionable error and
// the existing file is left intact (R4/R6). Writes are atomic.
func runWorkflowAddPhase(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus workflow add-phase", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/workflows/ holds the workflow")
	id := fs.String("id", "", "phase id (required, kebab-case)")
	agent := fs.String("agent", "", "agent that runs the phase (required)")
	gate := fs.String("gate", "", "validation gate for the phase (required)")
	inputs := fs.String("inputs", "", "comma-separated input artifacts (optional)")
	outputs := fs.String("outputs", "", "comma-separated output artifacts (optional)")
	dependsOn := fs.String("depends-on", "", "comma-separated predecessor references / DAG edges (optional)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus workflow add-phase <name> --id <id> --agent <a> --gate <g> [flags]\n\n"+
			"Append a phase to an existing workflow. id, agent and gate are required.\n"+
			"List flags take comma-separated values (e.g. --inputs brief,spec). The edit\n"+
			"is validated before writing; an invalid result is rejected and the existing\n"+
			"file is left intact.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	name, flags, err := splitWorkflowName(args)
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

	phase := workflows.Phase{
		ID:        *id,
		Agent:     *agent,
		Gate:      *gate,
		Inputs:    splitList(*inputs),
		Outputs:   splitList(*outputs),
		DependsOn: splitList(*dependsOn),
	}
	return applyWorkflowEdit(stdout, stderr, *dir, name, "add-phase",
		func(w *workflows.Workflow) error { return w.AddPhase(phase) })
}

// runWorkflowEditPhase handles `daedalus workflow edit-phase <name> --id <id> ...`:
// it replaces the named phase's fields. Only the flags the user passes are
// changed; the phase keeps its position. The whole resulting workflow is validated
// before any write (R4/R6); writes are atomic. To rename a phase, pass --new-id.
func runWorkflowEditPhase(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus workflow edit-phase", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/workflows/ holds the workflow")
	id := fs.String("id", "", "id of the phase to edit (required)")
	newID := fs.String("new-id", "", "rename the phase to this id (optional)")
	agent := fs.String("agent", "", "set the phase's agent")
	gate := fs.String("gate", "", "set the phase's gate")
	inputs := fs.String("inputs", "", "set the phase's inputs (comma-separated; empty clears)")
	outputs := fs.String("outputs", "", "set the phase's outputs (comma-separated; empty clears)")
	dependsOn := fs.String("depends-on", "", "set the phase's depends_on (comma-separated; empty clears)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus workflow edit-phase <name> --id <id> [flags]\n\n"+
			"Edit an existing phase in place, keeping its position. Only the flags you\n"+
			"pass are changed; --new-id renames the phase. The edit is validated before\n"+
			"writing; an invalid result is rejected and the existing file is left intact.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	name, flags, err := splitWorkflowName(args)
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
	if strings.TrimSpace(*id) == "" {
		fmt.Fprint(stderr, "daedalus: workflow edit-phase requires --id\n\n")
		fs.Usage()
		return 2
	}

	// Learn which flags were actually set so an explicit empty value (e.g.
	// --inputs "") is a deliberate clear rather than indistinguishable from absence.
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	mutate := func(w *workflows.Workflow) error {
		idx := w.PhaseIndex(*id)
		if idx < 0 {
			return fmt.Errorf("%w: %q", workflows.ErrPhaseNotFound, *id)
		}
		// Start from the current phase and apply only the named changes, preserving
		// the rest, so an edit is a focused field update rather than a full rewrite.
		p := w.Phases[idx]
		if set["new-id"] {
			p.ID = *newID
		}
		if set["agent"] {
			p.Agent = *agent
		}
		if set["gate"] {
			p.Gate = *gate
		}
		if set["inputs"] {
			p.Inputs = splitList(*inputs)
		}
		if set["outputs"] {
			p.Outputs = splitList(*outputs)
		}
		if set["depends-on"] {
			p.DependsOn = splitList(*dependsOn)
		}
		return w.EditPhase(*id, p)
	}
	return applyWorkflowEdit(stdout, stderr, *dir, name, "edit-phase", mutate)
}

// runWorkflowRemovePhase handles `daedalus workflow remove-phase <name> --id <id>`:
// it deletes the named phase, preserving the order of the rest. The resulting
// workflow is validated before any write; writes are atomic.
func runWorkflowRemovePhase(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus workflow remove-phase", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/workflows/ holds the workflow")
	id := fs.String("id", "", "id of the phase to remove (required)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus workflow remove-phase <name> --id <id> [--path .]\n\n"+
			"Remove a phase from a workflow, preserving the order of the rest.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	name, flags, err := splitWorkflowName(args)
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
	if strings.TrimSpace(*id) == "" {
		fmt.Fprint(stderr, "daedalus: workflow remove-phase requires --id\n\n")
		fs.Usage()
		return 2
	}

	return applyWorkflowEdit(stdout, stderr, *dir, name, "remove-phase",
		func(w *workflows.Workflow) error { return w.RemovePhase(*id) })
}

// applyWorkflowEdit is the shared tail of the phase edit operations: it runs the
// persisted workflows.Edit cycle (load → mutate → validate → atomic write) and
// maps the possible outcomes to consistent messages and exit codes. A not-found
// workflow or phase, a malformed file, and a schema-invalid result are all usage
// errors (exit 2); a genuine I/O failure is exit 1. This keeps the per-operation
// handlers focused on building their mutation.
func applyWorkflowEdit(stdout, stderr io.Writer, dir, name, op string, mutate workflows.EditFunc) int {
	logger := logging.New(stderr)

	if !workflows.IsKebabCase(name) {
		fmt.Fprintf(stderr, "daedalus: workflow name %q is not valid kebab-case\n", name)
		return 2
	}

	workflowsRoot := workflowsRootFor(dir)
	edited, err := workflows.Edit(workflowsRoot, name, mutate)
	if err != nil {
		switch {
		case errors.Is(err, workflows.ErrWorkflowNotFound):
			logger.Error("workflow "+op+" rejected", "phase", "load", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			fmt.Fprint(stderr, "the workflow must already exist; create it first\n")
			return 2
		case errors.Is(err, workflows.ErrPhaseNotFound), errors.Is(err, workflows.ErrPhaseExists):
			logger.Error("workflow "+op+" rejected", "phase", "mutate", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		case errors.Is(err, workflows.ErrMalformedWorkflow):
			logger.Error("workflow "+op+" failed", "phase", "load", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 1
		case isWorkflowInvalid(err):
			logger.Error("workflow "+op+" rejected", "phase", "validate", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: workflow %q is invalid; the edit was not applied:\n", name)
			writeWorkflowSchemaErrors(stderr, err)
			return 2
		default:
			logger.Error("workflow "+op+" failed", "phase", "write", "name", name, "err", err)
			fmt.Fprintf(stderr, "daedalus: workflow %s failed: %v\n", op, err)
			return 1
		}
	}

	logger.Info("workflow "+op+" applied", "name", edited.Name, "phases", len(edited.Phases))
	fmt.Fprintf(stdout, "Applied %s to workflow %q at %s (%d phase%s).\n",
		op, edited.Name, filepath.ToSlash(filepath.Join(workflowsRoot, name+workflows.FileExt)),
		len(edited.Phases), plural(len(edited.Phases), "", "s"))
	return 0
}

// splitWorkflowName extracts the single positional workflow name from an argument
// list, returning it plus the remaining (flag) tokens. It mirrors splitPromptID
// but names a "workflow name" in its errors so the `workflow` subcommand's usage
// messages read correctly. Exactly one positional is required; a `--help`/`-h`
// token is allowed without a name so the caller's parser can show usage.
func splitWorkflowName(args []string) (name string, flags []string, err error) {
	positionals, flags := splitPositionals(args)
	for _, f := range flags {
		if f == "-h" || f == "--help" {
			return "", flags, nil
		}
	}
	switch len(positionals) {
	case 1:
		return positionals[0], flags, nil
	case 0:
		return "", flags, errors.New("this operation requires exactly one workflow name")
	default:
		return "", flags, fmt.Errorf("this operation requires exactly one workflow name, got %d", len(positionals))
	}
}

// splitList turns a comma-separated flag value into a slice of trimmed,
// non-empty entries, mirroring splitBackends. An empty value yields a nil slice,
// which the renderer serializes as an empty list `[]`. Blank entries (e.g. a
// trailing comma) are dropped so they never reach the model as an empty artifact
// reference.
func splitList(raw string) []string {
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

// isWorkflowInvalid reports whether err is (or wraps) a *workflows.ValidationError
// — the rich error the workflows flows return from their pre-write gates — so the
// CLI detects it structurally with errors.As rather than matching a message prefix.
func isWorkflowInvalid(err error) bool {
	var ve *workflows.ValidationError
	return errors.As(err, &ve)
}

// writeWorkflowSchemaErrors renders a workflow validation failure as actionable
// lines, one finding per line, so the user can fix every problem in one pass (R7).
// It falls back to the plain error text if err is not a *workflows.ValidationError,
// so it is safe to call on any error.
func writeWorkflowSchemaErrors(w io.Writer, err error) {
	var ve *workflows.ValidationError
	if !errors.As(err, &ve) {
		fmt.Fprintf(w, "  - %v\n", err)
		return
	}
	for _, se := range ve.Errors {
		fmt.Fprintf(w, "  - %s\n", se.Error())
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

// valueFlags are the agent-subcommand flags that take a separate value token
// (e.g. `--path x`, `--role x`). The positional splitter consults this set so it
// never mistakes a flag's value for a positional agent id. It is the union across
// the agent operations — harmless to over-list, since a flag a given operation
// does not define simply never appears in its args. Boolean flags (--preview) are
// absent because they never consume a following token. Both the single-dash and
// double-dash spellings the Go flag parser accepts are listed.
var valueFlags = map[string]struct{}{
	"-path": {}, "--path": {},
	"-role": {}, "--role": {},
	"-prompt": {}, "--prompt": {},
	"-prompt-file": {}, "--prompt-file": {},
	"-set-param": {}, "--set-param": {},
	"-remove-param": {}, "--remove-param": {},
	// prompt-subcommand value flags. Listed here too because the `prompt` and
	// `agent` subcommands share splitPositionals; over-listing is harmless since a
	// flag a given operation does not define simply never appears in its args.
	"-kind": {}, "--kind": {},
	"-title": {}, "--title": {},
	"-description": {}, "--description": {},
	"-body": {}, "--body": {},
	"-body-file": {}, "--body-file": {},
	// workflow-subcommand value flags (phase edit operations). Listed here too
	// because the workflow subcommand shares splitPositionals; over-listing is
	// harmless since a flag a given operation does not define simply never appears
	// in its args.
	"-id": {}, "--id": {},
	"-new-id": {}, "--new-id": {},
	"-agent": {}, "--agent": {},
	"-inputs": {}, "--inputs": {},
	"-outputs": {}, "--outputs": {},
	"-gate": {}, "--gate": {},
	"-depends-on": {}, "--depends-on": {},
}

// splitPositionals separates the positional tokens from the flag tokens in an
// agent-subcommand argument list, preserving order within each group. It exists
// because Go's flag parser stops at the first non-flag token: without pulling the
// positionals (the agent ids) out first, `clone <src> <dest> --path x` or
// `edit <id> --role x` would leave the flags unparsed.
//
// It is value-flag aware: the token after a value-taking flag (e.g. `--role`) is
// that flag's value — not a positional — unless the flag carries its value inline
// (`--role=x`). The split is purely syntactic; the caller enforces how many
// positionals an operation expects.
func splitPositionals(args []string) (positionals, flags []string) {
	flags = make([]string, 0, len(args))
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
			// A `--flag x` form (no '=') consumes the next token as its value.
			if _, takesValue := valueFlags[a]; takesValue {
				expectValue = true
			}
			continue
		}
		positionals = append(positionals, a)
	}
	return positionals, flags
}

// splitAgentID extracts the single positional agent id from an argument list,
// returning it plus the remaining (flag) tokens. Exactly one positional is
// required; zero or more than one is a usage error so a typo like
// `add analyst architect` is caught instead of silently ignored. A `--help`/`-h`
// token is allowed without an id so the caller's parser can show usage.
func splitAgentID(args []string) (id string, flags []string, err error) {
	positionals, flags := splitPositionals(args)
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
		return "", flags, errors.New("this operation requires exactly one agent id")
	default:
		return "", flags, fmt.Errorf("this operation requires exactly one agent id, got %d", len(positionals))
	}
}

// splitPromptID extracts the single positional prompt id from an argument list,
// returning it plus the remaining (flag) tokens. It mirrors splitAgentID but
// names a "prompt id" in its errors so the `prompt` subcommand's usage messages
// read correctly. Exactly one positional is required; a `--help`/`-h` token is
// allowed without an id so the caller's parser can show usage.
func splitPromptID(args []string) (id string, flags []string, err error) {
	positionals, flags := splitPositionals(args)
	for _, f := range flags {
		if f == "-h" || f == "--help" {
			return "", flags, nil
		}
	}
	switch len(positionals) {
	case 1:
		return positionals[0], flags, nil
	case 0:
		return "", flags, errors.New("this operation requires exactly one prompt id")
	default:
		return "", flags, fmt.Errorf("this operation requires exactly one prompt id, got %d", len(positionals))
	}
}

// multiFlag is a repeatable string flag: each occurrence appends its value, so
// `--set-param a=1 --set-param b=2` collects both. It implements flag.Value so it
// plugs into the standard flag parser without a dependency.
type multiFlag []string

func (m *multiFlag) String() string { return strings.Join(*m, ",") }

func (m *multiFlag) Set(v string) error {
	*m = append(*m, v)
	return nil
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
