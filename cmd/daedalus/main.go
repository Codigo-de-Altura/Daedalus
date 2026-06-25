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
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Codigo-de-Altura/Daedalus/internal/architecture"
	"github.com/Codigo-de-Altura/Daedalus/internal/backlog"
	"github.com/Codigo-de-Altura/Daedalus/internal/buildinfo"
	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/compile"
	"github.com/Codigo-de-Altura/Daedalus/internal/conventions"
	"github.com/Codigo-de-Altura/Daedalus/internal/logging"
	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/specs"
	"github.com/Codigo-de-Altura/Daedalus/internal/traceability"
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
		case "spec":
			return runSpec(args[1:], os.Stdout, os.Stderr)
		case "architecture":
			return runArchitecture(args[1:], os.Stdout, os.Stderr)
		case "epic":
			return runEpic(args[1:], os.Stdout, os.Stderr)
		case "ticket":
			return runTicket(args[1:], os.Stdout, os.Stderr)
		case "trace":
			return runTrace(args[1:], os.Stdout, os.Stderr)
		case "validate":
			return runValidate(args[1:], os.Stdout, os.Stderr)
		case "build", "sync":
			// `sync` is a documented alias of `build` (REQ-1): both compile the
			// canonical .daedalus/ definition to the configured backend's native format.
			return runBuild(args[1:], os.Stdout, os.Stderr)
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
	// non-interactive validator can inspect the proposed changes safely. The
	// factory workflow seeding is part of init, so the preview mentions it too —
	// but only when it would actually be written (it is non-destructive: an
	// existing sdd-default.yaml is never reported as a change).
	if *preview {
		writePreview(stdout, plan)
		writeSeedPreview(stdout, *dir)
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

	// Seed the factory-default SDD workflow into .daedalus/workflows/. This is
	// orchestrated here, in the CLI, rather than inside internal/workspace so that
	// package stays free of a workspace->workflows dependency (the seeding is a
	// composition of two already-imported packages, not a new coupling). The seed
	// is non-destructive: an existing sdd-default.yaml (e.g. one the user has
	// edited) is never overwritten — Create uses O_CREATE|O_EXCL and reports
	// ErrWorkflowExists, which we treat as "already present, left intact". A seed
	// I/O failure does not fail the whole init: the workspace itself is already in
	// place, so we log it and report it but still return success.
	seedDefaultWorkflow(stdout, stderr, *dir)
	return 0
}

// seedDefaultWorkflow writes the factory sdd-default.yaml into the target's
// .daedalus/workflows/ directory non-destructively and reports the outcome on
// stdout. The content is generated deterministically from workflows.DefaultWorkflow
// via the store's Create (PlanCreate+Apply with O_EXCL), so re-running init never
// clobbers a user-edited workflow and a clean run is byte-stable.
func seedDefaultWorkflow(stdout, stderr io.Writer, dir string) {
	logger := logging.New(stderr)
	workflowsRoot := workflowsRootFor(dir)
	path := filepath.Join(workflowsRoot, workflows.DefaultWorkflowName+workflows.FileExt)

	err := workflows.Create(workflowsRoot, workflows.DefaultWorkflow())
	switch {
	case err == nil:
		logger.Info("seeded default workflow", "name", workflows.DefaultWorkflowName, "path", path)
		fmt.Fprintf(stdout, "Seeded factory workflow %q at %s.\n",
			workflows.DefaultWorkflowName, filepath.ToSlash(path))
	case errors.Is(err, workflows.ErrWorkflowExists):
		// Already present — left intact (non-destructive re-run or user-edited file).
		logger.Info("default workflow already present", "name", workflows.DefaultWorkflowName, "path", path)
		fmt.Fprintf(stdout, "Factory workflow %q already present at %s — left intact.\n",
			workflows.DefaultWorkflowName, filepath.ToSlash(path))
	default:
		// A seed failure is non-fatal: the workspace is already initialized.
		logger.Error("seeding default workflow failed", "name", workflows.DefaultWorkflowName, "err", err)
		fmt.Fprintf(stderr, "daedalus: warning: could not seed factory workflow %q: %v\n",
			workflows.DefaultWorkflowName, err)
	}
}

// writeSeedPreview reports, during a --preview init, whether the factory workflow
// would be seeded. It mirrors the non-destructive seed semantics: if an
// sdd-default.yaml already exists it is not listed as a change. It performs only a
// read (a stat) and never writes.
func writeSeedPreview(stdout io.Writer, dir string) {
	path := filepath.Join(workflowsRootFor(dir), workflows.DefaultWorkflowName+workflows.FileExt)
	if _, err := os.Stat(path); err == nil {
		// Already present; a preview of an upgrade should not claim it as new.
		return
	}
	fmt.Fprintf(stdout, "  + %s (factory workflow)\n",
		filepath.ToSlash(filepath.Join(workspace.Name, workflows.WorkflowsDir,
			workflows.DefaultWorkflowName+workflows.FileExt)))
}

// Build exit codes. They are differentiated so a caller (a script, a CI gate, a
// validator) can tell the failure modes apart from the process status alone
// (REQ-8): a definition that fails validation is a different problem — fixed by
// editing the canonical sources — than a backend that cannot be compiled or
// written. Usage errors keep the project-wide exit code 2; the two build-specific
// failure classes get their own codes above it so they never collide with it.
const (
	exitBuildOK         = 0 // success (REQ-8)
	exitBuildUsage      = 2 // flag/usage error, matching the rest of the CLI
	exitBuildValidation = 3 // canonical definition failed validation (REQ-3/REQ-8)
	exitBuildCompile    = 4 // workspace missing, no adapter, compile or write failure (REQ-8)
)

// runBuild handles `daedalus build` (and its alias `sync`): it compiles the
// canonical .daedalus/ definition to the configured backend(s)' native format
// (RF-6.1). It is the thin CLI surface over internal/compile: it parses flags,
// delegates the whole pipeline to compile.Build (which locates the workspace,
// validates the definition before writing, routes each backend through the
// adapter registry and writes the artifacts), renders the summary, and maps the
// typed errors to differentiated exit codes. --preview computes the plan without
// writing anything; the deep diff/preview rendering is ticket-06-04's job.
func runBuild(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus build", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/ workspace is compiled")
	preview := fs.Bool("preview", false, "show what would be compiled without writing anything (dry run)")
	yes := fs.Bool("yes", false, "write without the interactive confirmation gate (for CI/non-interactive use)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus build [flags]   (alias: daedalus sync)\n\n"+
			"Compile the canonical .daedalus/ definition to the native format of the\n"+
			"backend(s) configured in daedalus.yaml. The definition is validated before\n"+
			"anything is written; an invalid definition aborts the build with nothing\n"+
			"changed.\n\n"+
			"In an interactive terminal, build shows a diff/preview and asks you to\n"+
			"confirm before writing (RF-6.4). --preview shows the diff and never writes.\n"+
			"--yes writes without the gate (CI). Without a terminal and without --yes,\n"+
			"build prints the diff and writes nothing.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitBuildOK
		}
		return exitBuildUsage
	}

	logger := logging.New(stderr)

	// --yes is the non-interactive write path (CI), valid with or without a TTY: it
	// reuses the original compile.Build + summary flow with no gate. --preview takes
	// precedence over --yes if both are passed (preview never writes), so an
	// explicit dry-run request is always honored.
	if *yes && !*preview {
		out, err := compile.Build(compile.Options{Root: *dir, Preview: false, Logger: logger})
		if err != nil {
			return writeBuildError(stderr, err)
		}
		writeBuildSummary(stdout, out)
		return exitBuildOK
	}

	// Interactive terminal: launch the diff/preview TUI. In --preview it is
	// read-only (view + quit, never writes); otherwise it gates the write behind an
	// explicit confirmation (RF-6.4/RNF-8).
	if isInteractive() {
		return runBuildInteractive(*dir, *preview, stdout, stderr, logger)
	}

	// No terminal and no --yes (or --preview without a terminal): print the textual
	// diff/plan and write nothing. The plan runs the same validate-before-anything
	// pipeline, so a missing workspace or invalid definition still fails with the
	// right exit code — just with nothing written.
	return runBuildTextualPreview(*dir, *preview, stdout, stderr, logger)
}

// runBuildInteractive launches the Bubble Tea build preview and maps its outcome
// to the build exit codes. readOnly is --preview (no confirm/write). On confirm
// the model invokes compile.Build itself (through the tui package), recompiling at
// confirm time (TOCTOU-safe); here we only render the result and pick the code.
func runBuildInteractive(dir string, readOnly bool, stdout, stderr io.Writer, logger *slog.Logger) int {
	res, err := tui.RunBuildPreview(dir, readOnly)
	if err != nil {
		logger.Error("build preview failed", "err", err)
		fmt.Fprintf(stderr, "daedalus: build preview failed: %v\n", err)
		return exitBuildCompile
	}

	switch {
	case res.PlanErr != nil:
		return writeBuildError(stderr, res.PlanErr)
	case res.WriteErr != nil:
		fmt.Fprintf(stderr, "daedalus: build failed: %v\n", res.WriteErr)
		return exitBuildCompile
	case res.Wrote:
		writeBuildSummary(stdout, res.Outcome)
		return exitBuildOK
	case res.NoChanges:
		fmt.Fprintln(stdout, "Nothing to write — everything is already up to date.")
		return exitBuildOK
	default:
		// Cancelled, or quit a read-only preview: nothing was written.
		fmt.Fprintln(stdout, "Cancelled — nothing was written.")
		return exitBuildOK
	}
}

// runBuildTextualPreview computes the plan and prints it as text WITHOUT writing,
// for the no-TTY paths: `--preview` piped/in CI, and a plain `build` with neither
// a terminal nor --yes. The second case adds a clear notice that nothing was
// written and how to write (RF-6.4 decision: never write without a confirmation or
// --yes). Exit 0 on success; plan failures map to the usual build exit codes.
func runBuildTextualPreview(dir string, explicitPreview bool, stdout, stderr io.Writer, logger *slog.Logger) int {
	res, err := compile.Plan(compile.Options{Root: dir, Logger: logger})
	if err != nil {
		return writeBuildError(stderr, err)
	}

	tui.RenderPlanText(stdout, res)

	if !explicitPreview {
		// A plain `build` with no terminal and no --yes is a safe dry run: say so and
		// tell the user how to actually write.
		fmt.Fprintln(stdout, "\nNothing written; pass --yes to write, or run in a terminal to confirm.")
	}
	return exitBuildOK
}

// writeBuildError renders a build failure on stderr and returns the matching
// exit code, distinguishing the failure classes REQ-8 requires. A validation
// error is rendered finding-by-finding (the canonical sources to fix); the
// workspace-missing, no-adapter, compile and write failures are rendered as a
// single actionable line and share the compile/write exit code.
func writeBuildError(stderr io.Writer, err error) int {
	switch {
	case compile.IsDefinitionInvalid(err):
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		return exitBuildValidation
	case errors.Is(err, compile.ErrWorkspaceNotFound):
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		fmt.Fprint(stderr, "run 'daedalus init' to create the workspace first\n")
		return exitBuildCompile
	case errors.Is(err, compile.ErrNoAdapter):
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		return exitBuildCompile
	default:
		// A malformed manifest, a compile failure (incl. the not-yet-implemented
		// adapter) or an I/O write failure: all compilation/write errors (REQ-8).
		fmt.Fprintf(stderr, "daedalus: build failed: %v\n", err)
		return exitBuildCompile
	}
}

// writeBuildSummary reports the outcome of a build on stdout: per backend, what
// was (or would be) created/updated/left unchanged, plus any detected orphans
// (REQ-7). The core classifies every artifact in both modes — a preview reports
// exactly what a real run would do — so the wording differs only in tense; the
// numbers come from the same plan. The deep per-file diff and the confirmation
// gate are ticket-06-04's job; this is the plain, deterministic summary it builds
// on. Paths are already slash-normalized by the core (RNF-5).
func writeBuildSummary(stdout io.Writer, out *compile.Outcome) {
	if out.Preview {
		fmt.Fprintf(stdout, "Preview of compiling %s (no files written):\n", filepath.ToSlash(out.Root))
	} else {
		fmt.Fprintf(stdout, "Compiled %s:\n", filepath.ToSlash(out.Root))
	}
	for _, b := range out.Backends {
		verbCreate, verbUpdate := "created", "updated"
		if out.Preview {
			verbCreate, verbUpdate = "to create", "to update"
		}
		fmt.Fprintf(stdout, "  %s: %d %s, %d %s, %d unchanged (of %d artifact%s)\n",
			b.Backend, len(b.Created), verbCreate, len(b.Updated), verbUpdate,
			len(b.Unchanged), b.Planned, plural(b.Planned, "", "s"))
		for _, f := range b.Created {
			fmt.Fprintf(stdout, "    + %s\n", f)
		}
		for _, f := range b.Updated {
			fmt.Fprintf(stdout, "    ~ %s\n", f)
		}
		// Orphans are surfaced but never deleted (safe default, RF-6.3). The note
		// makes the managed-area boundary visible without acting on it; RF-6.4's
		// preview is where the user decides what to do with them.
		for _, f := range b.Orphans {
			fmt.Fprintf(stdout, "    ? %s (orphan: no longer produced; left untouched)\n", f)
		}
	}
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

// runSpec handles the `daedalus spec` subcommand, a thin CLI surface over the SDD
// backlog's brief/spec domain (internal/specs). It dispatches to the operation
// named by the next argument so the verb set can grow without reshaping run():
// capture (persist a brief and seed its spec), list, show, edit, remove. It keeps
// the same conventions as runPrompt/runWorkflow — own usage, exit code 2 for usage
// errors, logging to stderr — so the CLI feels uniform.
//
// Phase 1: none of these operations run the analyst agent (R5/CA5). `capture` only
// manages the definition — it persists the human's brief and seeds the canonical
// spec destination, wired (in frontmatter) to the analyst step of the default SDD
// workflow; the user generates the spec body by running the agent on their backend.
func runSpec(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprint(stderr, specUsage)
		return 2
	}

	switch args[0] {
	case "capture":
		return runSpecCapture(args[1:], stdout, stderr)
	case "list":
		return runSpecList(args[1:], stdout, stderr)
	case "show":
		return runSpecShow(args[1:], stdout, stderr)
	case "edit":
		return runSpecEdit(args[1:], stdout, stderr)
	case "remove":
		return runSpecRemove(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "daedalus: unknown spec operation %q\n\n%s", args[0], specUsage)
		return 2
	}
}

// specUsage is the shared help text for the `spec` subcommand, surfaced when no
// operation (or an unknown one) is given.
const specUsage = "Usage: daedalus spec <operation> [flags]\n\n" +
	"Work with the SDD backlog's briefs and specs in .daedalus/specs/.\n" +
	"Daedalus manages the definition only; it does not run the analyst agent.\n\n" +
	"Operations:\n" +
	"  capture <slug> --title <t> [flags]   capture a brief and seed its spec/PRD\n" +
	"  list                                 list captured briefs (slug, title, has-spec)\n" +
	"  show <slug> [--brief]                print the spec (or the brief) verbatim\n" +
	"  edit <slug> [flags]                  edit the materialized spec's title or body\n" +
	"  remove <slug> [--brief]              delete the spec (or the brief) file\n\n" +
	"Run 'daedalus spec <operation> --help' for an operation's flags.\n"

// specsRootFor builds the canonical `.daedalus/specs/` directory under dir so a
// brief/spec lands exactly where init scaffolds the specs/ directory. Built here
// rather than in the specs package so that package stays free of the
// workspace-location convention (mirrors promptsRootFor/workflowsRootFor).
func specsRootFor(dir string) string {
	return filepath.Join(dir, workspace.Name, specs.SpecsDir)
}

// runSpecCapture handles `daedalus spec capture <slug> --title <t>`: it captures a
// brief and seeds its canonical spec destination under .daedalus/specs/. The brief
// is validated before any write, so an invalid slug/title is rejected with an
// actionable error and nothing is written. It is non-destructive — an existing brief
// or spec (e.g. one the user has refined) is preserved, not overwritten (R4/R7) —
// and --preview reports the files that would be created without writing anything.
func runSpecCapture(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus spec capture", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/specs/ the brief is captured into")
	title := fs.String("title", "", "brief title (required)")
	body := fs.String("body", "", "brief body inline")
	bodyFile := fs.String("body-file", "", "brief body from a file (takes precedence over --body)")
	preview := fs.Bool("preview", false, "show the files that would be created without writing anything (dry run)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus spec capture <slug> --title <t> [flags]\n\n"+
			"Capture a brief as <slug>.brief.md and seed its canonical spec destination\n"+
			"<slug>.md in the target workspace's .daedalus/specs/ directory. The spec is\n"+
			"wired to the analyst step of the sdd-default workflow and references the brief,\n"+
			"but Daedalus does NOT run the agent: generate the spec on your backend and\n"+
			"refine it. Existing files are not overwritten. If both --body-file and --body\n"+
			"are given, --body-file wins.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	slug, flags, err := splitSpecSlug(args)
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
	// brief in a file without it being shadowed by an accidental inline one.
	bodyText := *body
	if *bodyFile != "" {
		b, err := os.ReadFile(*bodyFile)
		if err != nil {
			logger.Error("spec capture rejected", "phase", "flags", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: reading --body-file: %v\n", err)
			return 2
		}
		bodyText = string(b)
	}

	brief := specs.Brief{Slug: slug, Title: *title, Body: bodyText}
	specsRoot := specsRootFor(*dir)

	// Plan first: validates the brief and renders both files' content without touching
	// the filesystem, so an invalid brief fails as a usage error before any write or
	// preview.
	plan, err := specs.PlanCapture(specsRoot, brief)
	if err != nil {
		logger.Error("spec capture rejected", "phase", "plan", "slug", slug, "err", err)
		if isSpecInvalid(err) {
			fmt.Fprintf(stderr, "daedalus: brief %q is invalid; it was not captured:\n", slug)
			writeSpecSchemaErrors(stderr, err)
		} else {
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
		}
		return 2
	}

	logger.Info("spec capture planned", "slug", plan.Brief.Slug, "brief", plan.BriefPath, "spec", plan.SpecPath)

	if *preview {
		fmt.Fprintf(stdout, "Preview of capturing brief %q into %s:\n", plan.Brief.Slug, filepath.ToSlash(specsRoot))
		fmt.Fprintf(stdout, "  + %s (brief)\n", filepath.ToSlash(filepath.Base(plan.BriefPath)))
		fmt.Fprintf(stdout, "  + %s (spec, seeded; generate with the analyst on your backend)\n",
			filepath.ToSlash(filepath.Base(plan.SpecPath)))
		logger.Info("spec capture preview only", "slug", plan.Brief.Slug, "applied", false)
		return 0
	}

	res, err := plan.Apply()
	if err != nil {
		logger.Error("spec capture failed", "phase", "apply", "slug", slug, "err", err)
		fmt.Fprintf(stderr, "daedalus: spec capture failed: %v\n", err)
		return 1
	}

	logger.Info("spec captured",
		"slug", res.Slug, "brief_created", res.BriefCreated, "spec_created", res.SpecCreated)
	writeSpecCaptureResult(stdout, res)
	return 0
}

// writeSpecCaptureResult reports the outcome of a capture, choosing wording that
// unambiguously distinguishes a fresh capture from the non-destructive case where
// the brief and/or its spec already existed and were preserved (R4/R7).
func writeSpecCaptureResult(stdout io.Writer, res *specs.CaptureResult) {
	brief := filepath.ToSlash(res.BriefPath)
	spec := filepath.ToSlash(res.SpecPath)
	switch {
	case res.BriefCreated && res.SpecCreated:
		fmt.Fprintf(stdout, "Captured brief %q at %s and seeded its spec at %s.\n", res.Slug, brief, spec)
		fmt.Fprintf(stdout, "Generate the spec by running the %q agent on your backend, then refine %s.\n",
			specs.AnalystAgent, spec)
	case !res.BriefCreated && !res.SpecCreated:
		fmt.Fprintf(stdout, "Brief %q and its spec already exist — left intact (nothing overwritten).\n", res.Slug)
	case res.SpecCreated:
		// Brief existed, spec was missing and got re-seeded (e.g. it was deleted).
		fmt.Fprintf(stdout, "Brief %q already existed; re-seeded the missing spec at %s.\n", res.Slug, spec)
	default:
		// Spec existed (likely user-refined), brief was missing and got re-created.
		fmt.Fprintf(stdout, "Spec for %q already existed and was preserved; re-created the missing brief at %s.\n",
			res.Slug, brief)
	}
}

// runSpecList handles `daedalus spec list`: it prints the captured briefs (slug,
// title, whether a spec is materialized) in deterministic, slug-sorted order.
func runSpecList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus spec list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/specs/ is listed")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus spec list [--path .]\n\n"+
			"List the captured briefs (slug, title, and whether a spec is materialized).\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	logger := logging.New(stderr)

	entries, err := specs.List(specsRootFor(*dir))
	if err != nil {
		logger.Error("spec list failed", "phase", "list", "err", err)
		fmt.Fprintf(stderr, "daedalus: spec list failed: %v\n", err)
		return 1
	}

	logger.Info("specs listed", "briefs", len(entries))
	fmt.Fprintf(stdout, "Briefs (%d):\n", len(entries))
	for _, e := range entries {
		specState := "no-spec"
		if e.HasSpec {
			specState = "spec"
		}
		fmt.Fprintf(stdout, "  %s\t%s\t%s\n", e.Slug, specState, e.Title)
	}
	return 0
}

// runSpecShow handles `daedalus spec show <slug> [--brief]`: it prints the spec's
// file content verbatim (or the brief's, with --brief) so the user sees exactly what
// is persisted.
func runSpecShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus spec show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/specs/ holds the artifact")
	brief := fs.Bool("brief", false, "show the brief (<slug>.brief.md) instead of the spec (<slug>.md)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus spec show <slug> [--brief] [--path .]\n\n"+
			"Print the spec's file content verbatim, or the brief's with --brief.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	slug, flags, err := splitSpecSlug(args)
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

	if !specs.IsKebabCase(slug) {
		fmt.Fprintf(stderr, "daedalus: spec slug %q is not valid kebab-case\n", slug)
		return 2
	}

	name := slug + specs.FileExt
	if *brief {
		name = slug + specs.BriefExt
	}
	path := filepath.Join(specsRootFor(*dir), name)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Error("spec show rejected", "phase", "read", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: %s for %q not found\n", artifactWord(*brief), slug)
			return 2
		}
		logger.Error("spec show failed", "phase", "read", "slug", slug, "err", err)
		fmt.Fprintf(stderr, "daedalus: spec show failed: %v\n", err)
		return 1
	}

	logger.Info("spec shown", "slug", slug, "brief", *brief)
	fmt.Fprint(stdout, string(content))
	return 0
}

// artifactWord names the artifact a flag selects, for human-readable messages.
func artifactWord(brief bool) string {
	if brief {
		return "brief"
	}
	return "spec"
}

// runSpecEdit handles `daedalus spec edit <slug>`: it edits the materialized spec's
// title and/or body in place. The edit is validated before any write, so an edit
// that would leave the spec invalid (e.g. an empty title) is rejected with an
// actionable error and the existing file is left intact (R4). The brief reference
// and provenance are not editable. Writes are atomic. The brief itself is not
// editable here: it is the human's authored input, refined outside Daedalus.
func runSpecEdit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus spec edit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/specs/ holds the spec")
	title := fs.String("title", "", "set the spec's title")
	body := fs.String("body", "", "set the spec's body inline")
	bodyFile := fs.String("body-file", "", "set the spec's body from a file (takes precedence over --body)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus spec edit <slug> [flags]\n\n"+
			"Edit the materialized spec's title or body. At least one edit flag is required.\n"+
			"The brief reference and provenance are preserved. The edit is validated before\n"+
			"writing; an invalid edit is rejected and the existing file is left intact. If\n"+
			"both --body-file and --body are given, --body-file wins.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	slug, flags, err := splitSpecSlug(args)
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

	spec, err := buildSpecEditSpec(fs, *title, *body, *bodyFile)
	if err != nil {
		logger.Error("spec edit rejected", "phase", "flags", "slug", slug, "err", err)
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		return 2
	}
	if spec.IsEmpty() {
		fmt.Fprint(stderr, "daedalus: spec edit requires at least one edit flag (--title, --body, --body-file)\n\n")
		fs.Usage()
		return 2
	}

	specsRoot := specsRootFor(*dir)

	edited, err := specs.EditSpecArtifact(specsRoot, slug, spec)
	if err != nil {
		switch {
		case errors.Is(err, specs.ErrSpecNotFound):
			logger.Error("spec edit rejected", "phase", "load", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			fmt.Fprint(stderr, "the spec must already exist; capture its brief first\n")
			return 2
		case errors.Is(err, specs.ErrMalformed):
			logger.Error("spec edit failed", "phase", "load", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 1
		case isSpecInvalid(err):
			logger.Error("spec edit rejected", "phase", "validate", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: spec %q is invalid; the edit was not applied:\n", slug)
			writeSpecSchemaErrors(stderr, err)
			return 2
		default:
			logger.Error("spec edit failed", "phase", "write", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: spec edit failed: %v\n", err)
			return 1
		}
	}

	logger.Info("spec edited", "slug", edited.Slug)
	fmt.Fprintf(stdout, "Edited spec %q at %s.\n",
		edited.Slug, filepath.ToSlash(filepath.Join(specsRoot, edited.Slug+specs.FileExt)))
	return 0
}

// buildSpecEditSpec assembles a specs.SpecEditSpec from the parsed edit flags. It
// uses fs.Visit to learn which flags the user actually passed, so an explicit empty
// value (e.g. --title "") is recorded as a (rejectable) change rather than confused
// with the flag's default. --body-file takes precedence over --body.
func buildSpecEditSpec(fs *flag.FlagSet, title, body, bodyFile string) (specs.SpecEditSpec, error) {
	var spec specs.SpecEditSpec

	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	if set["title"] {
		spec.SetTitle = true
		spec.Title = title
	}
	switch {
	case set["body-file"]:
		b, err := os.ReadFile(bodyFile)
		if err != nil {
			return specs.SpecEditSpec{}, fmt.Errorf("reading --body-file: %w", err)
		}
		spec.SetBody = true
		spec.Body = string(b)
	case set["body"]:
		spec.SetBody = true
		spec.Body = body
	}

	return spec, nil
}

// runSpecRemove handles `daedalus spec remove <slug> [--brief]`: it deletes the
// spec's file (or the brief's, with --brief) and nothing else. An absent artifact is
// reported as an explicit error. Removing the spec leaves the brief intact and vice
// versa, so the user can drop and re-seed one half of the pair deliberately.
func runSpecRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus spec remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/specs/ holds the artifact")
	brief := fs.Bool("brief", false, "remove the brief (<slug>.brief.md) instead of the spec (<slug>.md)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus spec remove <slug> [--brief] [--path .]\n\n"+
			"Delete the spec's file from the workspace, or the brief's with --brief. Only\n"+
			"that one file is removed; the other half of the pair is left intact.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	slug, flags, err := splitSpecSlug(args)
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

	if !specs.IsKebabCase(slug) {
		fmt.Fprintf(stderr, "daedalus: spec slug %q is not valid kebab-case\n", slug)
		return 2
	}

	specsRoot := specsRootFor(*dir)
	name := slug + specs.FileExt
	if *brief {
		name = slug + specs.BriefExt
	}
	path := filepath.Join(specsRoot, name)
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Error("spec remove rejected", "phase", "remove", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: %s for %q not found\n", artifactWord(*brief), slug)
			return 2
		}
		logger.Error("spec remove failed", "phase", "remove", "slug", slug, "err", err)
		fmt.Fprintf(stderr, "daedalus: spec remove failed: %v\n", err)
		return 1
	}

	logger.Info("spec artifact removed", "slug", slug, "brief", *brief)
	fmt.Fprintf(stdout, "Removed %s %q from %s.\n", artifactWord(*brief), slug, filepath.ToSlash(path))
	return 0
}

// splitSpecSlug extracts the single positional spec slug from an argument list,
// returning it plus the remaining (flag) tokens. It mirrors splitPromptID but names
// a "spec slug" in its errors so the `spec` subcommand's usage messages read
// correctly. Exactly one positional is required; a `--help`/`-h` token is allowed
// without a slug so the caller's parser can show usage.
func splitSpecSlug(args []string) (slug string, flags []string, err error) {
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
		return "", flags, errors.New("this operation requires exactly one spec slug")
	default:
		return "", flags, fmt.Errorf("this operation requires exactly one spec slug, got %d", len(positionals))
	}
}

// isSpecInvalid reports whether err is (or wraps) a *specs.ValidationError — the
// rich error the specs flows return from their pre-write gates — so the CLI detects
// it structurally with errors.As rather than matching a message prefix.
func isSpecInvalid(err error) bool {
	var ve *specs.ValidationError
	return errors.As(err, &ve)
}

// writeSpecSchemaErrors renders a specs validation failure as actionable lines, one
// finding per line (field: observed / expected), so the user can fix every problem
// in one pass. It falls back to the plain error text if err is not a
// *specs.ValidationError, so it is safe to call on any error.
func writeSpecSchemaErrors(w io.Writer, err error) {
	var ve *specs.ValidationError
	if !errors.As(err, &ve) {
		fmt.Fprintf(w, "  - %v\n", err)
		return
	}
	for _, se := range ve.Errors {
		fmt.Fprintf(w, "  - %s: observed %s; expected %s\n", se.Field, se.Observed, se.Expected)
	}
}

// runArchitecture handles the `daedalus architecture` subcommand, a thin CLI surface
// over the SDD backlog's architecture-document domain (internal/architecture). It
// dispatches to the operation named by the next argument so the verb set can grow
// without reshaping run(): create, list, show, edit, remove. It keeps the same
// conventions as runSpec/runWorkflow — own usage, exit code 2 for usage errors,
// logging to stderr — so the CLI feels uniform.
//
// Phase 1: none of these operations run the architect agent (R5/CA5). `create` only
// manages the definition — it persists a document and, when linked, wires it (in
// frontmatter) to the architect step of the default SDD workflow; the user generates
// the document body by running the agent on their backend.
func runArchitecture(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprint(stderr, architectureUsage)
		return 2
	}

	switch args[0] {
	case "create":
		return runArchitectureCreate(args[1:], stdout, stderr)
	case "list":
		return runArchitectureList(args[1:], stdout, stderr)
	case "show":
		return runArchitectureShow(args[1:], stdout, stderr)
	case "edit":
		return runArchitectureEdit(args[1:], stdout, stderr)
	case "remove":
		return runArchitectureRemove(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "daedalus: unknown architecture operation %q\n\n%s", args[0], architectureUsage)
		return 2
	}
}

// architectureUsage is the shared help text for the `architecture` subcommand,
// surfaced when no operation (or an unknown one) is given.
const architectureUsage = "Usage: daedalus architecture <operation> [flags]\n\n" +
	"Work with the SDD backlog's architecture documents in .daedalus/architecture/.\n" +
	"Daedalus manages the definition only; it does not run the architect agent.\n\n" +
	"Operations:\n" +
	"  create <slug> --title <t> [--spec <s>] [flags]   create an architecture document\n" +
	"  list                                             list documents (slug, spec, title)\n" +
	"  show <slug>                                      print a document verbatim (raw)\n" +
	"  edit <slug> [flags]                              edit a document's title, spec or body\n" +
	"  remove <slug>                                    delete a document's file\n\n" +
	"Run 'daedalus architecture <operation> --help' for an operation's flags.\n"

// architectureRootFor builds the canonical `.daedalus/architecture/` directory under
// dir so a document lands exactly where init scaffolds the architecture/ directory.
// Built here rather than in the architecture package so that package stays free of the
// workspace-location convention (mirrors promptsRootFor/specsRootFor).
func architectureRootFor(dir string) string {
	return filepath.Join(dir, workspace.Name, architecture.ArchitectureDir)
}

// specExistsFor reports whether the referenced spec slug has a materialized
// `<slug>.md` spec under the workspace's specs directory. It is the friendly,
// filesystem-aware existence check the architecture store deliberately does NOT do
// (the store stays a pure model->bytes transform and must not couple to internal/
// specs): the CLI knows where specs live, so it resolves the link here. This mirrors
// how runWorkflowValidate resolves agent existence at the CLI layer rather than in the
// core. A missing spec is rejected at create time as an actionable usage error,
// coherent with the validation's precondition that a spec exists to link; full
// spec->architecture->... traceability is ticket 05-04.
func specExistsFor(dir, specSlug string) bool {
	path := filepath.Join(dir, workspace.Name, specs.SpecsDir, specSlug+specs.FileExt)
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// runArchitectureCreate handles `daedalus architecture create <slug> --title <t>
// [--spec <spec-slug>]`: it creates an architecture document under
// .daedalus/architecture/. The document is validated before any write, so an invalid
// slug/title is rejected with an actionable error and nothing is written. The --spec
// link is OPTIONAL (R3); when given, the referenced spec must already exist (a
// friendly check) and is recorded as the spec -> architecture trace. It is
// non-destructive — an existing document (e.g. one the user refined) is preserved, not
// overwritten (R4/R7) — and --preview reports the file that would be created without
// writing anything.
func runArchitectureCreate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus architecture create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/architecture/ the document is added to")
	title := fs.String("title", "", "document title (required)")
	specSlug := fs.String("spec", "", "slug of the originating spec to link (optional; must exist in .daedalus/specs/)")
	body := fs.String("body", "", "document body inline")
	bodyFile := fs.String("body-file", "", "document body from a file (takes precedence over --body)")
	preview := fs.Bool("preview", false, "show the file that would be created without writing anything (dry run)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus architecture create <slug> --title <t> [--spec <spec-slug>] [flags]\n\n"+
			"Create an architecture document as <slug>.md in the target workspace's\n"+
			".daedalus/architecture/ directory. With --spec, link it to an existing spec as\n"+
			"its origin (the spec -> architecture trace), wired to the architect step of the\n"+
			"sdd-default workflow; Daedalus does NOT run the agent. Existing documents are not\n"+
			"overwritten. If both --body-file and --body are given, --body-file wins.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	slug, flags, err := splitArchitectureSlug(args)
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
	// document in a file without it being shadowed by an accidental inline one.
	bodyText := *body
	if *bodyFile != "" {
		b, err := os.ReadFile(*bodyFile)
		if err != nil {
			logger.Error("architecture create rejected", "phase", "flags", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: reading --body-file: %v\n", err)
			return 2
		}
		bodyText = string(b)
	}

	// Friendly spec-link resolution: a non-empty --spec must reference an existing
	// spec. We translate the spec slug to its file reference (<spec-slug>.md), which is
	// exactly what the frontmatter records as the trace.
	var specRef string
	if strings.TrimSpace(*specSlug) != "" {
		if !specs.IsKebabCase(*specSlug) {
			fmt.Fprintf(stderr, "daedalus: --spec %q is not valid kebab-case\n", *specSlug)
			return 2
		}
		if !specExistsFor(*dir, *specSlug) {
			logger.Error("architecture create rejected", "phase", "spec-link", "spec", *specSlug)
			fmt.Fprintf(stderr, "daedalus: spec %q not found in %s; capture it first or omit --spec\n",
				*specSlug, filepath.ToSlash(filepath.Join(workspace.Name, specs.SpecsDir)))
			return 2
		}
		specRef = *specSlug + specs.FileExt
	}

	doc := architecture.Document{Slug: slug, Title: *title, SpecRef: specRef, Body: bodyText}
	archRoot := architectureRootFor(*dir)

	// Plan first: validates the document and renders the content without touching the
	// filesystem, so an invalid document fails as a usage error before any write or
	// preview.
	plan, err := architecture.PlanCreate(archRoot, doc)
	if err != nil {
		logger.Error("architecture create rejected", "phase", "plan", "slug", slug, "err", err)
		if isArchitectureInvalid(err) {
			fmt.Fprintf(stderr, "daedalus: architecture document %q is invalid; it was not created:\n", slug)
			writeArchitectureSchemaErrors(stderr, err)
		} else {
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
		}
		return 2
	}

	logger.Info("architecture create planned", "slug", plan.Document.Slug, "spec", plan.Document.SpecRef, "path", plan.Path)

	if *preview {
		fmt.Fprintf(stdout, "Preview of creating architecture document %q at %s:\n",
			plan.Document.Slug, filepath.ToSlash(plan.Path))
		fmt.Fprint(stdout, plan.Content)
		logger.Info("architecture create preview only", "slug", plan.Document.Slug, "applied", false)
		return 0
	}

	if err := plan.Apply(); err != nil {
		if errors.Is(err, architecture.ErrDocumentExists) {
			logger.Error("architecture create rejected", "phase", "apply", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v — not overwritten\n", err)
			return 2
		}
		logger.Error("architecture create failed", "phase", "apply", "slug", slug, "err", err)
		fmt.Fprintf(stderr, "daedalus: architecture create failed: %v\n", err)
		return 1
	}

	logger.Info("architecture document created", "slug", plan.Document.Slug, "spec", plan.Document.SpecRef)
	fmt.Fprintf(stdout, "Created architecture document %q at %s.\n",
		plan.Document.Slug, filepath.ToSlash(plan.Path))
	if plan.Document.SpecRef != "" {
		fmt.Fprintf(stdout, "Linked to spec %s. Generate the architecture by running the %q agent on your backend, then refine it.\n",
			plan.Document.SpecRef, architecture.ArchitectAgent)
	} else {
		fmt.Fprintf(stdout, "Generate the architecture by running the %q agent on your backend, then refine it.\n",
			architecture.ArchitectAgent)
	}
	return 0
}

// runArchitectureList handles `daedalus architecture list`: it prints the architecture
// documents (slug, spec link, title) in deterministic, slug-sorted order.
func runArchitectureList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus architecture list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/architecture/ is listed")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus architecture list [--path .]\n\n"+
			"List the architecture documents (slug, linked spec or '-', title).\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	logger := logging.New(stderr)

	entries, err := architecture.List(architectureRootFor(*dir))
	if err != nil {
		logger.Error("architecture list failed", "phase", "list", "err", err)
		fmt.Fprintf(stderr, "daedalus: architecture list failed: %v\n", err)
		return 1
	}

	logger.Info("architecture documents listed", "documents", len(entries))
	fmt.Fprintf(stdout, "Architecture documents (%d):\n", len(entries))
	for _, e := range entries {
		specRef := e.SpecRef
		if specRef == "" {
			specRef = "-"
		}
		fmt.Fprintf(stdout, "  %s\t%s\t%s\n", e.Slug, specRef, e.Title)
	}
	return 0
}

// runArchitectureShow handles `daedalus architecture show <slug>`: it prints the
// document's file content verbatim to stdout, so the user sees exactly what is
// persisted.
func runArchitectureShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus architecture show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/architecture/ holds the document")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus architecture show <slug> [--path .]\n\n"+
			"Print the architecture document's file content verbatim.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	slug, flags, err := splitArchitectureSlug(args)
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

	if !architecture.IsKebabCase(slug) {
		fmt.Fprintf(stderr, "daedalus: architecture slug %q is not valid kebab-case\n", slug)
		return 2
	}

	path := filepath.Join(architectureRootFor(*dir), slug+architecture.FileExt)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			logger.Error("architecture show rejected", "phase", "read", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: architecture document %q not found\n", slug)
			return 2
		}
		logger.Error("architecture show failed", "phase", "read", "slug", slug, "err", err)
		fmt.Fprintf(stderr, "daedalus: architecture show failed: %v\n", err)
		return 1
	}

	logger.Info("architecture document shown", "slug", slug)
	fmt.Fprint(stdout, string(content))
	return 0
}

// runArchitectureEdit handles `daedalus architecture edit <slug>`: it edits a
// document's title, spec link and/or body in place. The edit is validated before any
// write, so an edit that would leave the document invalid (e.g. an empty title) is
// rejected with an actionable error and the existing file is left intact (R4). A
// non-empty --spec must reference an existing spec (friendly check); --spec "" clears
// the link. Writes are atomic.
func runArchitectureEdit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus architecture edit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/architecture/ holds the document")
	title := fs.String("title", "", "set the document's title")
	specSlug := fs.String("spec", "", "set the originating spec link by slug (empty clears the link)")
	body := fs.String("body", "", "set the document's body inline")
	bodyFile := fs.String("body-file", "", "set the document's body from a file (takes precedence over --body)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus architecture edit <slug> [flags]\n\n"+
			"Edit an architecture document's title, spec link or body. At least one edit flag\n"+
			"is required. A non-empty --spec must reference an existing spec; --spec \"\" clears\n"+
			"the link. The edit is validated before writing; an invalid edit is rejected and\n"+
			"the existing file is left intact. If both --body-file and --body are given,\n"+
			"--body-file wins.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	slug, flags, err := splitArchitectureSlug(args)
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

	spec, err := buildArchitectureEditSpec(fs, *dir, *title, *specSlug, *body, *bodyFile, stderr)
	if err != nil {
		// buildArchitectureEditSpec already reported the specific reason to stderr.
		logger.Error("architecture edit rejected", "phase", "flags", "slug", slug, "err", err)
		return 2
	}
	if spec.IsEmpty() {
		fmt.Fprint(stderr, "daedalus: architecture edit requires at least one edit flag "+
			"(--title, --spec, --body, --body-file)\n\n")
		fs.Usage()
		return 2
	}

	archRoot := architectureRootFor(*dir)

	edited, err := architecture.Edit(archRoot, slug, spec)
	if err != nil {
		switch {
		case errors.Is(err, architecture.ErrDocumentNotFound):
			logger.Error("architecture edit rejected", "phase", "load", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			fmt.Fprint(stderr, "the document must already exist; create it first\n")
			return 2
		case errors.Is(err, architecture.ErrMalformed):
			logger.Error("architecture edit failed", "phase", "load", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 1
		case isArchitectureInvalid(err):
			logger.Error("architecture edit rejected", "phase", "validate", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: architecture document %q is invalid; the edit was not applied:\n", slug)
			writeArchitectureSchemaErrors(stderr, err)
			return 2
		default:
			logger.Error("architecture edit failed", "phase", "write", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: architecture edit failed: %v\n", err)
			return 1
		}
	}

	logger.Info("architecture document edited", "slug", edited.Slug, "spec", edited.SpecRef)
	fmt.Fprintf(stdout, "Edited architecture document %q at %s.\n",
		edited.Slug, filepath.ToSlash(filepath.Join(archRoot, edited.Slug+architecture.FileExt)))
	return 0
}

// buildArchitectureEditSpec assembles an architecture.EditSpec from the parsed edit
// flags. It uses fs.Visit to learn which flags the user actually passed, so an
// explicit empty value (e.g. --title "" or --spec "") is recorded as a deliberate
// change rather than confused with the flag's default. --body-file takes precedence
// over --body. A non-empty --spec is resolved to its file reference and verified to
// exist (friendly check); on any flag-level error it reports to stderr and returns an
// error so the caller exits 2.
func buildArchitectureEditSpec(fs *flag.FlagSet, dir, title, specSlug, body, bodyFile string, stderr io.Writer) (architecture.EditSpec, error) {
	var spec architecture.EditSpec

	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })

	if set["title"] {
		spec.SetTitle = true
		spec.Title = title
	}
	if set["spec"] {
		spec.SetSpec = true
		// An empty --spec clears the link; a non-empty one must reference an existing
		// spec and is stored as its file reference (<slug>.md).
		if strings.TrimSpace(specSlug) == "" {
			spec.Spec = ""
		} else {
			if !specs.IsKebabCase(specSlug) {
				fmt.Fprintf(stderr, "daedalus: --spec %q is not valid kebab-case\n", specSlug)
				return architecture.EditSpec{}, fmt.Errorf("invalid spec slug")
			}
			if !specExistsFor(dir, specSlug) {
				fmt.Fprintf(stderr, "daedalus: spec %q not found in %s; capture it first or pass --spec \"\" to clear the link\n",
					specSlug, filepath.ToSlash(filepath.Join(workspace.Name, specs.SpecsDir)))
				return architecture.EditSpec{}, fmt.Errorf("spec not found")
			}
			spec.Spec = specSlug + specs.FileExt
		}
	}
	switch {
	case set["body-file"]:
		b, err := os.ReadFile(bodyFile)
		if err != nil {
			fmt.Fprintf(stderr, "daedalus: reading --body-file: %v\n", err)
			return architecture.EditSpec{}, err
		}
		spec.SetBody = true
		spec.Body = string(b)
	case set["body"]:
		spec.SetBody = true
		spec.Body = body
	}

	return spec, nil
}

// runArchitectureRemove handles `daedalus architecture remove <slug>`: it deletes the
// document's file and nothing else. An absent document is reported as an explicit
// error.
func runArchitectureRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus architecture remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/architecture/ holds the document")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus architecture remove <slug> [--path .]\n\n"+
			"Delete an architecture document's file from the workspace. Only that file is\n"+
			"removed.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	slug, flags, err := splitArchitectureSlug(args)
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
	archRoot := architectureRootFor(*dir)

	if err := architecture.Remove(archRoot, slug); err != nil {
		if errors.Is(err, architecture.ErrDocumentNotFound) || !architecture.IsKebabCase(slug) {
			logger.Error("architecture remove rejected", "phase", "remove", "slug", slug, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		}
		logger.Error("architecture remove failed", "phase", "remove", "slug", slug, "err", err)
		fmt.Fprintf(stderr, "daedalus: architecture remove failed: %v\n", err)
		return 1
	}

	logger.Info("architecture document removed", "slug", slug)
	fmt.Fprintf(stdout, "Removed architecture document %q from %s.\n",
		slug, filepath.ToSlash(filepath.Join(archRoot, slug+architecture.FileExt)))
	return 0
}

// splitArchitectureSlug extracts the single positional architecture slug from an
// argument list, returning it plus the remaining (flag) tokens. It mirrors
// splitSpecSlug but names an "architecture slug" in its errors so the subcommand's
// usage messages read correctly. Exactly one positional is required; a `--help`/`-h`
// token is allowed without a slug so the caller's parser can show usage.
func splitArchitectureSlug(args []string) (slug string, flags []string, err error) {
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
		return "", flags, errors.New("this operation requires exactly one architecture slug")
	default:
		return "", flags, fmt.Errorf("this operation requires exactly one architecture slug, got %d", len(positionals))
	}
}

// isArchitectureInvalid reports whether err is (or wraps) a
// *architecture.ValidationError — the rich error the architecture flows return from
// their pre-write gates — so the CLI detects it structurally with errors.As rather
// than matching a message prefix.
func isArchitectureInvalid(err error) bool {
	var ve *architecture.ValidationError
	return errors.As(err, &ve)
}

// writeArchitectureSchemaErrors renders an architecture validation failure as
// actionable lines, one finding per line (field: observed / expected), so the user can
// fix every problem in one pass. It falls back to the plain error text if err is not a
// *architecture.ValidationError, so it is safe to call on any error.
func writeArchitectureSchemaErrors(w io.Writer, err error) {
	var ve *architecture.ValidationError
	if !errors.As(err, &ve) {
		fmt.Fprintf(w, "  - %v\n", err)
		return
	}
	for _, se := range ve.Errors {
		fmt.Fprintf(w, "  - %s: observed %s; expected %s\n", se.Field, se.Observed, se.Expected)
	}
}

// epicsRootFor builds the canonical `.daedalus/epics/` directory under dir, which roots
// the nested epic/ticket tree. Built here rather than in the backlog package so that
// package stays free of the workspace-location convention (mirrors the sibling roots).
func epicsRootFor(dir string) string {
	return filepath.Join(dir, workspace.Name, backlog.EpicsDir)
}

// --- epic subcommand ---

// runEpic handles the `daedalus epic` subcommand, a thin CLI surface over the backlog's
// epic operations (internal/backlog). It dispatches to create/list/show/edit/remove,
// keeping the same conventions as the sibling subcommands (own usage, exit code 2 for
// usage errors, logging to stderr). Phase 1: none of these run the planner (R7/CA7).
func runEpic(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprint(stderr, epicUsage)
		return 2
	}
	switch args[0] {
	case "create":
		return runEpicCreate(args[1:], stdout, stderr)
	case "list":
		return runEpicList(args[1:], stdout, stderr)
	case "show":
		return runEpicShow(args[1:], stdout, stderr)
	case "edit":
		return runEpicEdit(args[1:], stdout, stderr)
	case "remove":
		return runEpicRemove(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "daedalus: unknown epic operation %q\n\n%s", args[0], epicUsage)
		return 2
	}
}

const epicUsage = "Usage: daedalus epic <operation> [flags]\n\n" +
	"Work with the SDD backlog's epics in .daedalus/epics/.\n" +
	"Daedalus manages the definition only; it does not run the planner agent.\n\n" +
	"Operations:\n" +
	"  create <NN> <slug> --title <t> [flags]   create an epic folder epic-NN-<slug>\n" +
	"  list                                     list epics (id, status, priority, title)\n" +
	"  show <epic-id>                           print an epic's epic.md verbatim\n" +
	"  edit <epic-id> [flags]                   edit an epic's metadata or body\n" +
	"  remove <epic-id>                         delete an epic folder (and its tickets)\n\n" +
	"Run 'daedalus epic <operation> --help' for an operation's flags.\n"

// runEpicCreate handles `daedalus epic create <NN> <slug> --title <t>`. Numbering is
// explicit and deterministic: the user supplies NN and the slug, which compose the
// canonical id `epic-NN-<slug>` (no auto-increment, no hidden state). Metadata flags set
// status/priority/links/dependencies. Optional --spec/--architecture links are verified
// to exist (friendly check, CLI-layer) before recording. Non-destructive; --preview
// shows the file without writing.
func runEpicCreate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus epic create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/epics/ the epic is added to")
	title := fs.String("title", "", "epic title (required)")
	status := fs.String("status", "", "status: "+joinBacklogStatuses()+" (default: "+string(backlog.DefaultStatus)+")")
	priority := fs.String("priority", "", "priority: "+joinBacklogPriorities()+" (default: "+string(backlog.DefaultPriority)+")")
	specSlug := fs.String("spec", "", "originating spec slug to link (optional; must exist)")
	archSlug := fs.String("architecture", "", "originating architecture slug to link (optional; must exist)")
	dependsOn := fs.String("depends-on", "", "comma-separated dependency ids (optional)")
	body := fs.String("body", "", "epic body inline")
	bodyFile := fs.String("body-file", "", "epic body from a file (takes precedence over --body)")
	preview := fs.Bool("preview", false, "show the file that would be created without writing anything (dry run)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus epic create <NN> <slug> --title <t> [flags]\n\n"+
			"Create an epic as the folder epic-NN-<slug>/ with epic.md in the target\n"+
			"workspace's .daedalus/epics/ directory. NN is the epic number and <slug> is\n"+
			"kebab-case. Daedalus does NOT run the planner. Existing epics are not\n"+
			"overwritten.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	positionals, flags := splitPositionals(args)
	if hasHelp(flags) {
		fs.Usage()
		return 0
	}
	if err := fs.Parse(flags); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if len(positionals) != 2 {
		fmt.Fprint(stderr, "daedalus: epic create requires exactly a number and a slug\n\n")
		fs.Usage()
		return 2
	}
	number, slug := positionals[0], positionals[1]

	logger := logging.New(stderr)

	bodyText, code := resolveBody(*body, *bodyFile, stderr, logger, "epic create")
	if code != 0 {
		return code
	}

	// Resolve optional origin links by slug, verifying existence (friendly CLI check).
	specRef, code := resolveSpecLink(*dir, *specSlug, stderr)
	if code != 0 {
		return code
	}
	archRef, code := resolveArchitectureLink(*dir, *archSlug, stderr)
	if code != 0 {
		return code
	}

	epic := backlog.Epic{
		ID:              backlog.EpicID(number, slug),
		Title:           *title,
		Status:          backlog.Status(*status),
		Priority:        backlog.Priority(*priority),
		SpecRef:         specRef,
		ArchitectureRef: archRef,
		DependsOn:       splitList(*dependsOn),
		Body:            bodyText,
	}
	epicsRoot := epicsRootFor(*dir)

	plan, err := backlog.PlanCreateEpic(epicsRoot, epic)
	if err != nil {
		logger.Error("epic create rejected", "phase", "plan", "id", epic.ID, "err", err)
		if isBacklogInvalid(err) {
			fmt.Fprintf(stderr, "daedalus: epic %q is invalid; it was not created:\n", epic.ID)
			writeBacklogSchemaErrors(stderr, err)
		} else {
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
		}
		return 2
	}

	logger.Info("epic create planned", "id", plan.Epic.ID, "path", plan.Path)

	if *preview {
		fmt.Fprintf(stdout, "Preview of creating epic %q at %s:\n", plan.Epic.ID, filepath.ToSlash(plan.Path))
		fmt.Fprint(stdout, plan.Content)
		return 0
	}

	if err := plan.Apply(); err != nil {
		if errors.Is(err, backlog.ErrEpicExists) {
			logger.Error("epic create rejected", "phase", "apply", "id", epic.ID, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v — not overwritten\n", err)
			return 2
		}
		logger.Error("epic create failed", "phase", "apply", "id", epic.ID, "err", err)
		fmt.Fprintf(stderr, "daedalus: epic create failed: %v\n", err)
		return 1
	}

	logger.Info("epic created", "id", plan.Epic.ID)
	fmt.Fprintf(stdout, "Created epic %q at %s.\n", plan.Epic.ID, filepath.ToSlash(plan.Path))
	return 0
}

// runEpicList handles `daedalus epic list`.
func runEpicList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus epic list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/epics/ is listed")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus epic list [--path .]\n\n"+
			"List the epics (id, status, priority, title).\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	logger := logging.New(stderr)
	entries, err := backlog.ListEpics(epicsRootFor(*dir))
	if err != nil {
		logger.Error("epic list failed", "phase", "list", "err", err)
		fmt.Fprintf(stderr, "daedalus: epic list failed: %v\n", err)
		return 1
	}

	logger.Info("epics listed", "epics", len(entries))
	fmt.Fprintf(stdout, "Epics (%d):\n", len(entries))
	for _, e := range entries {
		fmt.Fprintf(stdout, "  %s\t%s\t%s\t%s\n", e.ID, e.Status, e.Priority, e.Title)
	}
	return 0
}

// runEpicShow handles `daedalus epic show <epic-id>`.
func runEpicShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus epic show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/epics/ holds the epic")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus epic show <epic-id> [--path .]\n\n"+
			"Print the epic's epic.md content verbatim.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	id, flags, err := splitSinglePositional(args, "epic id")
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
	if !backlog.IsEpicID(id) {
		fmt.Fprintf(stderr, "daedalus: epic id %q is not a valid epic-NN-<slug> id\n", id)
		return 2
	}

	path := filepath.Join(epicsRootFor(*dir), id, backlog.EpicFile)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "daedalus: epic %q not found\n", id)
			return 2
		}
		logger.Error("epic show failed", "phase", "read", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: epic show failed: %v\n", err)
		return 1
	}
	logger.Info("epic shown", "id", id)
	fmt.Fprint(stdout, string(content))
	return 0
}

// runEpicEdit handles `daedalus epic edit <epic-id>`: edits metadata/body in place,
// validated before an atomic write; an invalid edit leaves the file intact (R8/CA8).
func runEpicEdit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus epic edit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/epics/ holds the epic")
	title := fs.String("title", "", "set the title")
	status := fs.String("status", "", "set the status: "+joinBacklogStatuses())
	priority := fs.String("priority", "", "set the priority: "+joinBacklogPriorities())
	specSlug := fs.String("spec", "", "set the originating spec link by slug (empty clears it)")
	archSlug := fs.String("architecture", "", "set the originating architecture link by slug (empty clears it)")
	dependsOn := fs.String("depends-on", "", "set the dependency ids (comma-separated; empty clears)")
	body := fs.String("body", "", "set the body inline")
	bodyFile := fs.String("body-file", "", "set the body from a file (takes precedence over --body)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus epic edit <epic-id> [flags]\n\n"+
			"Edit an epic's metadata (title, status, priority, links, dependencies) or body.\n"+
			"At least one edit flag is required. A non-empty --spec/--architecture must exist;\n"+
			"passing them empty clears the link. The edit is validated before writing; an\n"+
			"invalid edit is rejected and the existing file is left intact.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	id, flags, err := splitSinglePositional(args, "epic id")
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

	spec, code := buildEpicEditSpec(fs, *dir, *title, *status, *priority, *specSlug, *archSlug, *dependsOn, *body, *bodyFile, stderr)
	if code != 0 {
		return code
	}
	if spec.IsEmpty() {
		fmt.Fprint(stderr, "daedalus: epic edit requires at least one edit flag\n\n")
		fs.Usage()
		return 2
	}

	epicsRoot := epicsRootFor(*dir)
	edited, err := backlog.EditEpic(epicsRoot, id, spec)
	if err != nil {
		return reportBacklogEditError(stderr, logger, "epic", id, err)
	}
	logger.Info("epic edited", "id", edited.ID)
	fmt.Fprintf(stdout, "Edited epic %q at %s.\n", edited.ID, filepath.ToSlash(filepath.Join(epicsRoot, edited.ID, backlog.EpicFile)))
	return 0
}

// runEpicRemove handles `daedalus epic remove <epic-id>`. Removing an epic removes its
// nested tickets too (they live inside it); the result message says so.
func runEpicRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus epic remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/epics/ holds the epic")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus epic remove <epic-id> [--path .]\n\n"+
			"Delete an epic folder, including its nested tickets.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	id, flags, err := splitSinglePositional(args, "epic id")
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
	epicsRoot := epicsRootFor(*dir)
	if err := backlog.RemoveEpic(epicsRoot, id); err != nil {
		if errors.Is(err, backlog.ErrEpicNotFound) || !backlog.IsEpicID(id) {
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		}
		logger.Error("epic remove failed", "phase", "remove", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: epic remove failed: %v\n", err)
		return 1
	}
	logger.Info("epic removed", "id", id)
	fmt.Fprintf(stdout, "Removed epic %q (and its tickets) from %s.\n",
		id, filepath.ToSlash(filepath.Join(epicsRoot, id)))
	return 0
}

// --- ticket subcommand ---

// runTicket handles the `daedalus ticket` subcommand. Tickets are nested under their
// epic, so every operation takes the epic id plus the ticket id/sequence.
func runTicket(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprint(stderr, ticketUsage)
		return 2
	}
	switch args[0] {
	case "create":
		return runTicketCreate(args[1:], stdout, stderr)
	case "list":
		return runTicketList(args[1:], stdout, stderr)
	case "show":
		return runTicketShow(args[1:], stdout, stderr)
	case "edit":
		return runTicketEdit(args[1:], stdout, stderr)
	case "remove":
		return runTicketRemove(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "daedalus: unknown ticket operation %q\n\n%s", args[0], ticketUsage)
		return 2
	}
}

const ticketUsage = "Usage: daedalus ticket <operation> [flags]\n\n" +
	"Work with the SDD backlog's tickets, nested under their epic in .daedalus/epics/.\n" +
	"Daedalus manages the definition only; it does not run the planner agent.\n\n" +
	"Operations:\n" +
	"  create <epic-id> <MM> <slug> --title <t> [flags]   create a ticket under the epic\n" +
	"  list <epic-id>                                     list an epic's tickets\n" +
	"  show <epic-id> <ticket-id>                         print a ticket's ticket.md verbatim\n" +
	"  edit <epic-id> <ticket-id> [flags]                 edit a ticket's metadata or body\n" +
	"  remove <epic-id> <ticket-id>                       delete a ticket folder\n\n" +
	"Run 'daedalus ticket <operation> --help' for an operation's flags.\n"

// runTicketCreate handles `daedalus ticket create <epic-id> <MM> <slug> --title <t>`.
// The epic number NN is derived from the parent epic id so the ticket id stays
// consistent (ticket-NN-MM-<slug>); the user supplies only MM and the slug. The parent
// epic must already exist (structural prerequisite). Non-destructive; --preview supported.
func runTicketCreate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus ticket create", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/epics/ holds the parent epic")
	title := fs.String("title", "", "ticket title (required)")
	status := fs.String("status", "", "status: "+joinBacklogStatuses()+" (default: "+string(backlog.DefaultStatus)+")")
	priority := fs.String("priority", "", "priority: "+joinBacklogPriorities()+" (default: "+string(backlog.DefaultPriority)+")")
	specSlug := fs.String("spec", "", "originating spec slug to link (optional; must exist)")
	archSlug := fs.String("architecture", "", "originating architecture slug to link (optional; must exist)")
	dependsOn := fs.String("depends-on", "", "comma-separated dependency ids (optional)")
	body := fs.String("body", "", "ticket body inline")
	bodyFile := fs.String("body-file", "", "ticket body from a file (takes precedence over --body)")
	preview := fs.Bool("preview", false, "show the file that would be created without writing anything (dry run)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus ticket create <epic-id> <MM> <slug> --title <t> [flags]\n\n"+
			"Create a ticket as the folder ticket-NN-MM-<slug>/ with ticket.md nested under\n"+
			"its epic (NN is taken from the epic id). The parent epic must already exist.\n"+
			"Daedalus does NOT run the planner. Existing tickets are not overwritten.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	positionals, flags := splitPositionals(args)
	if hasHelp(flags) {
		fs.Usage()
		return 0
	}
	if err := fs.Parse(flags); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if len(positionals) != 3 {
		fmt.Fprint(stderr, "daedalus: ticket create requires exactly an epic id, a sequence and a slug\n\n")
		fs.Usage()
		return 2
	}
	epicID, sequence, slug := positionals[0], positionals[1], positionals[2]

	logger := logging.New(stderr)

	if !backlog.IsEpicID(epicID) {
		fmt.Fprintf(stderr, "daedalus: epic id %q is not a valid epic-NN-<slug> id\n", epicID)
		return 2
	}
	// Derive the ticket's NN from its parent epic so the two are always consistent.
	number := backlog.EpicNumberOf(epicID)

	bodyText, code := resolveBody(*body, *bodyFile, stderr, logger, "ticket create")
	if code != 0 {
		return code
	}
	specRef, code := resolveSpecLink(*dir, *specSlug, stderr)
	if code != 0 {
		return code
	}
	archRef, code := resolveArchitectureLink(*dir, *archSlug, stderr)
	if code != 0 {
		return code
	}

	ticket := backlog.Ticket{
		ID:              backlog.TicketID(number, sequence, slug),
		EpicID:          epicID,
		Title:           *title,
		Status:          backlog.Status(*status),
		Priority:        backlog.Priority(*priority),
		SpecRef:         specRef,
		ArchitectureRef: archRef,
		DependsOn:       splitList(*dependsOn),
		Body:            bodyText,
	}
	epicsRoot := epicsRootFor(*dir)

	plan, err := backlog.PlanCreateTicket(epicsRoot, ticket)
	if err != nil {
		if errors.Is(err, backlog.ErrParentEpicMissing) {
			logger.Error("ticket create rejected", "phase", "plan", "id", ticket.ID, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v; create the epic first\n", err)
			return 2
		}
		logger.Error("ticket create rejected", "phase", "plan", "id", ticket.ID, "err", err)
		if isBacklogInvalid(err) {
			fmt.Fprintf(stderr, "daedalus: ticket %q is invalid; it was not created:\n", ticket.ID)
			writeBacklogSchemaErrors(stderr, err)
		} else {
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
		}
		return 2
	}

	logger.Info("ticket create planned", "id", plan.Ticket.ID, "epic", plan.Ticket.EpicID, "path", plan.Path)

	if *preview {
		fmt.Fprintf(stdout, "Preview of creating ticket %q at %s:\n", plan.Ticket.ID, filepath.ToSlash(plan.Path))
		fmt.Fprint(stdout, plan.Content)
		return 0
	}

	if err := plan.Apply(); err != nil {
		if errors.Is(err, backlog.ErrTicketExists) {
			logger.Error("ticket create rejected", "phase", "apply", "id", ticket.ID, "err", err)
			fmt.Fprintf(stderr, "daedalus: %v — not overwritten\n", err)
			return 2
		}
		logger.Error("ticket create failed", "phase", "apply", "id", ticket.ID, "err", err)
		fmt.Fprintf(stderr, "daedalus: ticket create failed: %v\n", err)
		return 1
	}

	logger.Info("ticket created", "id", plan.Ticket.ID, "epic", plan.Ticket.EpicID)
	fmt.Fprintf(stdout, "Created ticket %q at %s.\n", plan.Ticket.ID, filepath.ToSlash(plan.Path))
	return 0
}

// runTicketList handles `daedalus ticket list <epic-id>`.
func runTicketList(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus ticket list", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/epics/ holds the epic")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus ticket list <epic-id> [--path .]\n\n"+
			"List an epic's tickets (id, status, priority, title).\n\nFlags:\n")
		fs.PrintDefaults()
	}
	epicID, flags, err := splitSinglePositional(args, "epic id")
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
	if !backlog.IsEpicID(epicID) {
		fmt.Fprintf(stderr, "daedalus: epic id %q is not a valid epic-NN-<slug> id\n", epicID)
		return 2
	}

	entries, err := backlog.ListTickets(epicsRootFor(*dir), epicID)
	if err != nil {
		logger.Error("ticket list failed", "phase", "list", "epic", epicID, "err", err)
		fmt.Fprintf(stderr, "daedalus: ticket list failed: %v\n", err)
		return 1
	}
	logger.Info("tickets listed", "epic", epicID, "tickets", len(entries))
	fmt.Fprintf(stdout, "Tickets of %s (%d):\n", epicID, len(entries))
	for _, tk := range entries {
		fmt.Fprintf(stdout, "  %s\t%s\t%s\t%s\n", tk.ID, tk.Status, tk.Priority, tk.Title)
	}
	return 0
}

// runTicketShow handles `daedalus ticket show <epic-id> <ticket-id>`.
func runTicketShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus ticket show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/epics/ holds the ticket")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus ticket show <epic-id> <ticket-id> [--path .]\n\n"+
			"Print the ticket's ticket.md content verbatim.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	ids, flags, err := splitTwoPositionals(args, "epic id", "ticket id")
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
	epicID, ticketID := ids[0], ids[1]

	logger := logging.New(stderr)
	if !backlog.IsEpicID(epicID) || !backlog.IsTicketID(ticketID) {
		fmt.Fprintf(stderr, "daedalus: invalid epic or ticket id (%q / %q)\n", epicID, ticketID)
		return 2
	}

	path := filepath.Join(epicsRootFor(*dir), epicID, backlog.TicketsSubdir, ticketID, backlog.TicketFile)
	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(stderr, "daedalus: ticket %q not found under %q\n", ticketID, epicID)
			return 2
		}
		logger.Error("ticket show failed", "phase", "read", "id", ticketID, "err", err)
		fmt.Fprintf(stderr, "daedalus: ticket show failed: %v\n", err)
		return 1
	}
	logger.Info("ticket shown", "id", ticketID)
	fmt.Fprint(stdout, string(content))
	return 0
}

// runTicketEdit handles `daedalus ticket edit <epic-id> <ticket-id>`.
func runTicketEdit(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus ticket edit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/epics/ holds the ticket")
	title := fs.String("title", "", "set the title")
	status := fs.String("status", "", "set the status: "+joinBacklogStatuses())
	priority := fs.String("priority", "", "set the priority: "+joinBacklogPriorities())
	specSlug := fs.String("spec", "", "set the originating spec link by slug (empty clears it)")
	archSlug := fs.String("architecture", "", "set the originating architecture link by slug (empty clears it)")
	dependsOn := fs.String("depends-on", "", "set the dependency ids (comma-separated; empty clears)")
	body := fs.String("body", "", "set the body inline")
	bodyFile := fs.String("body-file", "", "set the body from a file (takes precedence over --body)")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus ticket edit <epic-id> <ticket-id> [flags]\n\n"+
			"Edit a ticket's metadata (title, status, priority, links, dependencies) or body.\n"+
			"At least one edit flag is required. A non-empty --spec/--architecture must exist;\n"+
			"passing them empty clears the link. The edit is validated before writing; an\n"+
			"invalid edit is rejected and the existing file is left intact.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	ids, flags, err := splitTwoPositionals(args, "epic id", "ticket id")
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
	epicID, ticketID := ids[0], ids[1]

	logger := logging.New(stderr)

	spec, code := buildTicketEditSpec(fs, *dir, *title, *status, *priority, *specSlug, *archSlug, *dependsOn, *body, *bodyFile, stderr)
	if code != 0 {
		return code
	}
	if spec.IsEmpty() {
		fmt.Fprint(stderr, "daedalus: ticket edit requires at least one edit flag\n\n")
		fs.Usage()
		return 2
	}

	epicsRoot := epicsRootFor(*dir)
	edited, err := backlog.EditTicket(epicsRoot, epicID, ticketID, spec)
	if err != nil {
		return reportBacklogEditError(stderr, logger, "ticket", ticketID, err)
	}
	logger.Info("ticket edited", "id", edited.ID, "epic", edited.EpicID)
	fmt.Fprintf(stdout, "Edited ticket %q at %s.\n", edited.ID,
		filepath.ToSlash(filepath.Join(epicsRoot, epicID, backlog.TicketsSubdir, ticketID, backlog.TicketFile)))
	return 0
}

// runTicketRemove handles `daedalus ticket remove <epic-id> <ticket-id>`.
func runTicketRemove(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus ticket remove", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/epics/ holds the ticket")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus ticket remove <epic-id> <ticket-id> [--path .]\n\n"+
			"Delete a ticket folder. The epic and sibling tickets are left intact.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	ids, flags, err := splitTwoPositionals(args, "epic id", "ticket id")
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
	epicID, ticketID := ids[0], ids[1]

	logger := logging.New(stderr)
	epicsRoot := epicsRootFor(*dir)
	if err := backlog.RemoveTicket(epicsRoot, epicID, ticketID); err != nil {
		if errors.Is(err, backlog.ErrTicketNotFound) || !backlog.IsEpicID(epicID) || !backlog.IsTicketID(ticketID) {
			fmt.Fprintf(stderr, "daedalus: %v\n", err)
			return 2
		}
		logger.Error("ticket remove failed", "phase", "remove", "id", ticketID, "err", err)
		fmt.Fprintf(stderr, "daedalus: ticket remove failed: %v\n", err)
		return 1
	}
	logger.Info("ticket removed", "id", ticketID, "epic", epicID)
	fmt.Fprintf(stdout, "Removed ticket %q from %s.\n", ticketID,
		filepath.ToSlash(filepath.Join(epicsRoot, epicID, backlog.TicketsSubdir, ticketID)))
	return 0
}

// --- backlog CLI helpers ---

// buildEpicEditSpec assembles a backlog.EpicEditSpec from the parsed flags, using
// fs.Visit so an explicit empty value is a deliberate change. A non-empty --spec/
// --architecture is resolved and verified to exist; on a flag-level error it reports to
// stderr and returns exit code 2. Returns (spec, 0) on success.
func buildEpicEditSpec(fs *flag.FlagSet, dir, title, status, priority, specSlug, archSlug, dependsOn, body, bodyFile string, stderr io.Writer) (backlog.EpicEditSpec, int) {
	var spec backlog.EpicEditSpec
	set := visitedFlags(fs)

	if set["title"] {
		spec.SetTitle = true
		spec.Title = title
	}
	if set["status"] {
		spec.SetStatus = true
		spec.Status = backlog.Status(status)
	}
	if set["priority"] {
		spec.SetPriority = true
		spec.Priority = backlog.Priority(priority)
	}
	if set["spec"] {
		ref, code := resolveSpecLink(dir, specSlug, stderr)
		if code != 0 {
			return backlog.EpicEditSpec{}, code
		}
		spec.SetSpec = true
		spec.Spec = ref
	}
	if set["architecture"] {
		ref, code := resolveArchitectureLink(dir, archSlug, stderr)
		if code != 0 {
			return backlog.EpicEditSpec{}, code
		}
		spec.SetArchitecture = true
		spec.Architecture = ref
	}
	if set["depends-on"] {
		spec.SetDependsOn = true
		spec.DependsOn = splitList(dependsOn)
	}
	if code := applyBodyEdit(set, body, bodyFile, stderr, &spec.SetBody, &spec.Body); code != 0 {
		return backlog.EpicEditSpec{}, code
	}
	return spec, 0
}

// buildTicketEditSpec mirrors buildEpicEditSpec for tickets.
func buildTicketEditSpec(fs *flag.FlagSet, dir, title, status, priority, specSlug, archSlug, dependsOn, body, bodyFile string, stderr io.Writer) (backlog.TicketEditSpec, int) {
	var spec backlog.TicketEditSpec
	set := visitedFlags(fs)

	if set["title"] {
		spec.SetTitle = true
		spec.Title = title
	}
	if set["status"] {
		spec.SetStatus = true
		spec.Status = backlog.Status(status)
	}
	if set["priority"] {
		spec.SetPriority = true
		spec.Priority = backlog.Priority(priority)
	}
	if set["spec"] {
		ref, code := resolveSpecLink(dir, specSlug, stderr)
		if code != 0 {
			return backlog.TicketEditSpec{}, code
		}
		spec.SetSpec = true
		spec.Spec = ref
	}
	if set["architecture"] {
		ref, code := resolveArchitectureLink(dir, archSlug, stderr)
		if code != 0 {
			return backlog.TicketEditSpec{}, code
		}
		spec.SetArchitecture = true
		spec.Architecture = ref
	}
	if set["depends-on"] {
		spec.SetDependsOn = true
		spec.DependsOn = splitList(dependsOn)
	}
	if code := applyBodyEdit(set, body, bodyFile, stderr, &spec.SetBody, &spec.Body); code != 0 {
		return backlog.TicketEditSpec{}, code
	}
	return spec, 0
}

// applyBodyEdit resolves the --body/--body-file pair into the SetBody/Body targets,
// honoring the --body-file-wins precedence. Returns exit code 2 on a read failure.
func applyBodyEdit(set map[string]bool, body, bodyFile string, stderr io.Writer, setBody *bool, bodyOut *string) int {
	switch {
	case set["body-file"]:
		b, err := os.ReadFile(bodyFile)
		if err != nil {
			fmt.Fprintf(stderr, "daedalus: reading --body-file: %v\n", err)
			return 2
		}
		*setBody = true
		*bodyOut = string(b)
	case set["body"]:
		*setBody = true
		*bodyOut = body
	}
	return 0
}

// resolveBody resolves the create-time --body/--body-file pair (body-file wins). Returns
// (text, 0) on success or ("", 2) on a read failure (already reported).
func resolveBody(body, bodyFile string, stderr io.Writer, logger *slog.Logger, op string) (string, int) {
	if bodyFile != "" {
		b, err := os.ReadFile(bodyFile)
		if err != nil {
			logger.Error(op+" rejected", "phase", "flags", "err", err)
			fmt.Fprintf(stderr, "daedalus: reading --body-file: %v\n", err)
			return "", 2
		}
		return string(b), 0
	}
	return body, 0
}

// resolveSpecLink translates an optional --spec slug into the stored reference
// (<slug>.md), verifying the spec exists (friendly CLI check, as in 05-02). An empty slug
// yields an empty reference (no link). Returns (ref, 0) on success or ("", 2) on error.
func resolveSpecLink(dir, specSlug string, stderr io.Writer) (string, int) {
	if strings.TrimSpace(specSlug) == "" {
		return "", 0
	}
	if !specs.IsKebabCase(specSlug) {
		fmt.Fprintf(stderr, "daedalus: --spec %q is not valid kebab-case\n", specSlug)
		return "", 2
	}
	if !specExistsFor(dir, specSlug) {
		fmt.Fprintf(stderr, "daedalus: spec %q not found in %s; capture it first or omit --spec\n",
			specSlug, filepath.ToSlash(filepath.Join(workspace.Name, specs.SpecsDir)))
		return "", 2
	}
	return specSlug + specs.FileExt, 0
}

// resolveArchitectureLink translates an optional --architecture slug into the stored
// reference (<slug>.md), verifying the document exists. Mirrors resolveSpecLink.
func resolveArchitectureLink(dir, archSlug string, stderr io.Writer) (string, int) {
	if strings.TrimSpace(archSlug) == "" {
		return "", 0
	}
	if !architecture.IsKebabCase(archSlug) {
		fmt.Fprintf(stderr, "daedalus: --architecture %q is not valid kebab-case\n", archSlug)
		return "", 2
	}
	path := filepath.Join(dir, workspace.Name, architecture.ArchitectureDir, archSlug+architecture.FileExt)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		fmt.Fprintf(stderr, "daedalus: architecture document %q not found in %s; create it first or omit --architecture\n",
			archSlug, filepath.ToSlash(filepath.Join(workspace.Name, architecture.ArchitectureDir)))
		return "", 2
	}
	return archSlug + architecture.FileExt, 0
}

// reportBacklogEditError maps an epic/ticket edit error to a consistent message and exit
// code: not-found and schema-invalid are usage errors (2); malformed and I/O are 1.
func reportBacklogEditError(stderr io.Writer, logger *slog.Logger, kind, id string, err error) int {
	switch {
	case errors.Is(err, backlog.ErrEpicNotFound), errors.Is(err, backlog.ErrTicketNotFound):
		logger.Error(kind+" edit rejected", "phase", "load", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		fmt.Fprintf(stderr, "the %s must already exist; create it first\n", kind)
		return 2
	case errors.Is(err, backlog.ErrMalformed):
		logger.Error(kind+" edit failed", "phase", "load", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: %v\n", err)
		return 1
	case isBacklogInvalid(err):
		logger.Error(kind+" edit rejected", "phase", "validate", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: %s %q is invalid; the edit was not applied:\n", kind, id)
		writeBacklogSchemaErrors(stderr, err)
		return 2
	default:
		logger.Error(kind+" edit failed", "phase", "write", "id", id, "err", err)
		fmt.Fprintf(stderr, "daedalus: %s edit failed: %v\n", kind, err)
		return 1
	}
}

// visitedFlags returns the set of flag names the user actually passed, so an explicit
// empty value is distinguishable from an unset flag.
func visitedFlags(fs *flag.FlagSet) map[string]bool {
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	return set
}

// hasHelp reports whether the flag tokens include a help request, so a multi-positional
// operation can show usage without first failing the positional-count check.
func hasHelp(flags []string) bool {
	for _, f := range flags {
		if f == "-h" || f == "--help" {
			return true
		}
	}
	return false
}

// splitSinglePositional extracts exactly one positional (named noun for errors) plus the
// flag tokens, allowing a help token without a positional.
func splitSinglePositional(args []string, noun string) (string, []string, error) {
	positionals, flags := splitPositionals(args)
	if hasHelp(flags) {
		return "", flags, nil
	}
	switch len(positionals) {
	case 1:
		return positionals[0], flags, nil
	case 0:
		return "", flags, fmt.Errorf("this operation requires exactly one %s", noun)
	default:
		return "", flags, fmt.Errorf("this operation requires exactly one %s, got %d", noun, len(positionals))
	}
}

// splitTwoPositionals extracts exactly two positionals (named for errors) plus the flag
// tokens, allowing a help token without positionals.
func splitTwoPositionals(args []string, first, second string) ([2]string, []string, error) {
	positionals, flags := splitPositionals(args)
	if hasHelp(flags) {
		return [2]string{}, flags, nil
	}
	if len(positionals) != 2 {
		return [2]string{}, flags, fmt.Errorf("this operation requires exactly a %s and a %s", first, second)
	}
	return [2]string{positionals[0], positionals[1]}, flags, nil
}

// joinBacklogStatuses / joinBacklogPriorities render the closed metadata sets for CLI
// help, so the values a user sees in help match the validator's set exactly.
func joinBacklogStatuses() string {
	parts := make([]string, 0)
	for _, s := range backlog.Statuses() {
		parts = append(parts, string(s))
	}
	return strings.Join(parts, "|")
}

func joinBacklogPriorities() string {
	parts := make([]string, 0)
	for _, p := range backlog.Priorities() {
		parts = append(parts, string(p))
	}
	return strings.Join(parts, "|")
}

// isBacklogInvalid reports whether err is (or wraps) a *backlog.ValidationError.
func isBacklogInvalid(err error) bool {
	var ve *backlog.ValidationError
	return errors.As(err, &ve)
}

// writeBacklogSchemaErrors renders a backlog validation failure as actionable lines, one
// finding per line, so the user can fix every problem in one pass.
func writeBacklogSchemaErrors(w io.Writer, err error) {
	var ve *backlog.ValidationError
	if !errors.As(err, &ve) {
		fmt.Fprintf(w, "  - %v\n", err)
		return
	}
	for _, se := range ve.Errors {
		fmt.Fprintf(w, "  - %s: observed %s; expected %s\n", se.Field, se.Observed, se.Expected)
	}
}

// --- trace subcommand ---

// runTrace handles the `daedalus trace` subcommand, a thin CLI surface over the
// traceability aggregator (internal/traceability). It dispatches to verify (check the
// whole chain) and show (navigate from one artifact). It is read-only and runs no agents
// (R5/CA6). It keeps the same conventions as the sibling subcommands (own usage, logging
// to stderr).
func runTrace(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprint(stderr, traceUsage)
		return 2
	}
	switch args[0] {
	case "verify":
		return runTraceVerify(args[1:], stdout, stderr)
	case "show":
		return runTraceShow(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "daedalus: unknown trace operation %q\n\n%s", args[0], traceUsage)
		return 2
	}
}

const traceUsage = "Usage: daedalus trace <operation> [flags]\n\n" +
	"Consolidate and verify the SDD spec -> epic -> ticket traceability chain.\n" +
	"Read-only: it reuses the links already recorded in specs/architecture/epics/tickets.\n\n" +
	"Operations:\n" +
	"  verify                    check the chain; report inconsistencies (exit 1 on hard errors)\n" +
	"  show <artifact-id>        navigate the chain from a spec slug, epic id or ticket id\n\n" +
	"Run 'daedalus trace <operation> --help' for an operation's flags.\n"

// buildTraceGraph assembles the traceability graph over the three canonical workspace
// roots under dir. Centralized so verify and show build the graph identically.
func buildTraceGraph(dir string) (*traceability.Graph, error) {
	return traceability.Build(specsRootFor(dir), architectureRootFor(dir), epicsRootFor(dir))
}

// runTraceVerify handles `daedalus trace verify`: it builds the chain and reports every
// inconsistency in deterministic order, with a summary. Exit code mirrors workflow
// validate: 0 when the chain has no hard errors (warnings/gaps do NOT fail), 1 when there
// is at least one error-level inconsistency, 2 on a usage/IO error. Soft traceability
// gaps (missing optional origin links) are reported but never affect the exit code,
// honoring 05-03's optional-origin decision.
func runTraceVerify(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus trace verify", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/ chain is verified")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus trace verify [--path .]\n\n"+
			"Verify the spec -> epic -> ticket traceability chain. Reports broken links,\n"+
			"orphan tickets and (as warnings) missing optional origin links. Exit 0 if there\n"+
			"are no hard errors, 1 if there are, 2 on a usage error.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	logger := logging.New(stderr)

	graph, err := buildTraceGraph(*dir)
	if err != nil {
		logger.Error("trace verify failed", "phase", "build", "err", err)
		fmt.Fprintf(stderr, "daedalus: trace verify failed: %v\n", err)
		return 1
	}

	report := graph.Verify()
	errs, warns := report.Counts()
	logger.Info("trace verified", "errors", errs, "warnings", warns, "consistent", report.Consistent())

	if report.Consistent() && warns == 0 {
		fmt.Fprintln(stdout, "Traceability chain is consistent (no inconsistencies).")
		return 0
	}

	if report.Consistent() {
		fmt.Fprintf(stdout, "Traceability chain is consistent with %d warning%s (no hard errors):\n",
			warns, plural(warns, "", "s"))
	} else {
		fmt.Fprintf(stdout, "Traceability chain has %d error%s and %d warning%s:\n",
			errs, plural(errs, "", "s"), warns, plural(warns, "", "s"))
	}
	for _, f := range report.Findings {
		fmt.Fprintf(stdout, "  - %s\n", f.Error())
	}

	// Only hard errors affect the exit code; warnings are informational.
	if report.HasErrors() {
		return 1
	}
	return 0
}

// runTraceShow handles `daedalus trace show <artifact-id>`: it navigates the chain from
// the given artifact. The artifact kind is inferred from its id shape — a ticket id
// (ticket-NN-MM-<slug>) climbs ascending; an epic id (epic-NN-<slug>) shows its origin
// and its tickets; anything else is treated as a spec slug and descends. This single
// entry point keeps the CLI small while covering both directions (R1/CA1/CA2).
func runTraceShow(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus trace show", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/ chain is navigated")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus trace show <artifact-id> [--path .]\n\n"+
			"Navigate the traceability chain from an artifact. A spec slug descends to its\n"+
			"epics and tickets; a ticket id (ticket-NN-MM-<slug>) ascends to its epic and\n"+
			"origin spec/architecture; an epic id (epic-NN-<slug>) shows both.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	id, flags, err := splitSinglePositional(args, "artifact id")
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

	graph, err := buildTraceGraph(*dir)
	if err != nil {
		logger.Error("trace show failed", "phase", "build", "err", err)
		fmt.Fprintf(stderr, "daedalus: trace show failed: %v\n", err)
		return 1
	}

	switch {
	case backlog.IsTicketID(id):
		return traceShowTicket(stdout, stderr, graph, id)
	case backlog.IsEpicID(id):
		return traceShowEpic(stdout, stderr, graph, id)
	default:
		return traceShowSpec(stdout, stderr, graph, id)
	}
}

// traceShowSpec renders the descending chain from a spec slug (R1/CA1).
func traceShowSpec(stdout, stderr io.Writer, graph *traceability.Graph, slug string) int {
	chain, ok := graph.DescendFromSpec(slug)
	if !ok {
		fmt.Fprintf(stderr, "daedalus: no spec %q in the traceability chain "+
			"(a spec must be materialized; check 'daedalus spec list')\n", slug)
		return 2
	}
	fmt.Fprintf(stdout, "spec %s — %s\n", chain.Spec.Slug, chain.Spec.Title)
	if len(chain.Epics) == 0 {
		fmt.Fprintln(stdout, "  (no epics link to this spec)")
		return 0
	}
	for _, ewt := range chain.Epics {
		fmt.Fprintf(stdout, "  └─ epic %s — %s\n", ewt.Epic.ID, ewt.Epic.Title)
		if len(ewt.Tickets) == 0 {
			fmt.Fprintln(stdout, "       (no tickets)")
			continue
		}
		for _, tk := range ewt.Tickets {
			fmt.Fprintf(stdout, "       └─ ticket %s — %s\n", tk.ID, tk.Title)
		}
	}
	return 0
}

// traceShowEpic renders an epic's origin links (ascending half) and its tickets
// (descending half), covering both directions from an epic (R1/CA1/CA2).
func traceShowEpic(stdout, stderr io.Writer, graph *traceability.Graph, epicID string) int {
	epic, ok := graph.Epics[epicID]
	if !ok {
		fmt.Fprintf(stderr, "daedalus: no epic %q in the traceability chain "+
			"(check 'daedalus epic list')\n", epicID)
		return 2
	}
	fmt.Fprintf(stdout, "epic %s — %s\n", epic.ID, epic.Title)
	fmt.Fprintf(stdout, "  origin spec:         %s\n", traceRefOrNone(epic.SpecRef))
	fmt.Fprintf(stdout, "  origin architecture: %s\n", traceRefOrNone(epic.ArchRef))

	tickets := graph.TicketsOfEpic(epicID)
	if len(tickets) == 0 {
		fmt.Fprintln(stdout, "  tickets: (none)")
	} else {
		fmt.Fprintln(stdout, "  tickets:")
		for _, tk := range tickets {
			fmt.Fprintf(stdout, "    └─ ticket %s — %s\n", tk.ID, tk.Title)
		}
	}
	return 0
}

// traceShowTicket renders the ascending chain from a ticket (R1/CA2): its epic and origin
// spec/architecture, flagging a missing epic (orphan) explicitly.
func traceShowTicket(stdout, stderr io.Writer, graph *traceability.Graph, ticketID string) int {
	chain, ok := graph.AscendFromTicket(ticketID)
	if !ok {
		fmt.Fprintf(stderr, "daedalus: no ticket %q in the traceability chain "+
			"(check 'daedalus ticket list <epic-id>')\n", ticketID)
		return 2
	}
	fmt.Fprintf(stdout, "ticket %s — %s\n", chain.Ticket.ID, chain.Ticket.Title)
	if chain.EpicFound {
		fmt.Fprintf(stdout, "  └─ epic %s — %s\n", chain.Epic.ID, chain.Epic.Title)
	} else {
		fmt.Fprintf(stdout, "  └─ epic %s — MISSING (orphan ticket)\n", chain.Ticket.EpicID)
	}
	if chain.OriginSpecFound {
		fmt.Fprintf(stdout, "       └─ origin spec %s — %s\n", chain.OriginSpec.Slug, chain.OriginSpec.Title)
	} else {
		fmt.Fprintln(stdout, "       └─ origin spec: (none resolved)")
	}
	if chain.OriginArchFound {
		fmt.Fprintf(stdout, "       └─ origin architecture %s — %s\n", chain.OriginArch.Slug, chain.OriginArch.Title)
	} else {
		fmt.Fprintln(stdout, "       └─ origin architecture: (none resolved)")
	}
	return 0
}

// traceRefOrNone renders an origin reference or a placeholder when it is empty.
func traceRefOrNone(ref string) string {
	if ref == "" {
		return "(none)"
	}
	return ref
}

// runValidate handles `daedalus validate`: it checks the `.daedalus/` workspace
// against the team conventions (naming, structure, format, traceability — RF-8.3)
// and reports every violation in deterministic order. It REPORTS, never fixes.
//
// Exit code mirrors `trace verify` (the established convention-checking command in
// this CLI): 0 when there are no hard errors (warnings are advisory and do NOT
// fail), 1 when there is at least one error-level violation, 2 on a usage/IO
// error. Keeping the same codes as trace verify means CI scripts treat both
// checkers uniformly.
func runValidate(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("daedalus validate", flag.ContinueOnError)
	fs.SetOutput(stderr)
	dir := fs.String("path", ".", "target repository directory whose .daedalus/ workspace is validated")
	fs.Usage = func() {
		fmt.Fprint(stderr, "Usage: daedalus validate [--path .]\n\n"+
			"Validate the .daedalus/ workspace against the team conventions: kebab-case and\n"+
			"id patterns (epic-NN-<slug>, ticket-NN-MM-<slug>), the canonical directory\n"+
			"layout, YAML/Markdown format, and spec -> epic -> ticket traceability. It reports\n"+
			"violations with their location; it never edits or auto-fixes. Exit 0 if there are\n"+
			"no errors, 1 if there are, 2 on a usage error.\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	logger := logging.New(stderr)

	report, err := conventions.WorkspaceUnder(*dir).Validate()
	if err != nil {
		logger.Error("validate failed", "phase", "scan", "err", err)
		fmt.Fprintf(stderr, "daedalus: validate failed: %v\n", err)
		return 2
	}

	errs, warns := report.Counts()
	logger.Info("workspace validated", "errors", errs, "warnings", warns, "conformant", report.Conformant())

	if report.Conformant() && warns == 0 {
		fmt.Fprintln(stdout, "Workspace conforms to the conventions (no violations).")
		return 0
	}

	if report.Conformant() {
		fmt.Fprintf(stdout, "Workspace conforms with %d warning%s (no errors):\n",
			warns, plural(warns, "", "s"))
	} else {
		fmt.Fprintf(stdout, "Workspace has %d convention error%s and %d warning%s:\n",
			errs, plural(errs, "", "s"), warns, plural(warns, "", "s"))
	}
	for _, f := range report.Findings {
		fmt.Fprintf(stdout, "  - %s\n", f.Error())
	}

	// Only hard errors affect the exit code; warnings are informational.
	if report.HasErrors() {
		return 1
	}
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
	for _, file := range plan.MissingTrackedFiles {
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
	// architecture-subcommand value flag (the optional spec link). Listed here too
	// because the architecture subcommand shares splitPositionals; over-listing is
	// harmless since a flag a given operation does not define simply never appears in
	// its args.
	"-spec": {}, "--spec": {},
	// backlog (epic/ticket) value flags. Listed here too because the epic/ticket
	// subcommands share splitPositionals; over-listing is harmless.
	"-status": {}, "--status": {},
	"-priority": {}, "--priority": {},
	"-architecture": {}, "--architecture": {},
	"-epic": {}, "--epic": {},
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
