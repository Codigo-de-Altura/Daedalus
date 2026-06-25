package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Codigo-de-Altura/Daedalus/internal/architecture"
	"github.com/Codigo-de-Altura/Daedalus/internal/backlog"
	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/compile"
	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/specs"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// commands.go owns the bridge between the TUI shell and the core. Every call into
// a core package happens inside a tea.Cmd so the UI thread never blocks on
// filesystem I/O, composition or compilation; the result is delivered back to
// Update as a typed Msg. The presentation layer never runs domain logic or reads
// the disk directly — it only triggers these commands and renders their results
// (the hard rule from CLAUDE.md §2).
//
// There are two generic commands, parameterized by the area, so a new area plugs
// in by adding a case rather than a new message type:
//   - loadAreaCmd   → lists an area's rows (the list screen)        → areaLoadedMsg
//   - loadSubCmd    → loads one row's read-only detail (a sub-screen) → subLoadedMsg
//
// Each area's case below names exactly which core interface it consumes (R6).

// areaLoadedMsg reports the result of listing an area's rows. Exactly one of
// items / err is meaningful: a non-nil err means the listing failed (rendered as
// the area's error state), while an empty items slice with a nil err means the
// area is simply empty (a valid empty state, not an error).
type areaLoadedMsg struct {
	id    areaID
	items []areaItem
	err   error
}

// subLoadedMsg reports the result of loading one row's detail for a sub-screen.
// id/key echo which area and row were requested so a late-arriving result for a
// row the user navigated away from can be ignored. content is the FINAL, ready-to-
// display body — any markdown rendering already happened off the UI thread inside
// loadSubCmd (07-04), so Update only stores it into the viewport with no heavy work.
// err holds a load failure.
type subLoadedMsg struct {
	id      areaID
	key     string
	content string
	err     error
}

// loadAreaCmd lists the rows for one area off the UI thread and delivers an
// areaLoadedMsg. It dispatches to the per-area lister; each lister consumes only
// its area's core interface and returns rows (or an error) — never touching the
// disk from the presentation layer itself.
func loadAreaCmd(workdir string, id areaID) tea.Cmd {
	return func() tea.Msg {
		items, err := listArea(workdir, id)
		return areaLoadedMsg{id: id, items: items, err: err}
	}
}

// loadSubCmd loads one row's detail off the UI thread and delivers a subLoadedMsg.
// key is echoed back so Update can discard a stale result. Crucially, the markdown
// render (the only potentially heavy presentation work) happens HERE, in the
// goroutine, not in Update — so even a large document never blocks the Bubble Tea
// loop (07-04 R2). The wrap width is captured by the caller at open time so the
// off-thread render matches the viewport; th provides the themed Glamour style.
func loadSubCmd(workdir string, id areaID, key string, th theme, wrap int) tea.Cmd {
	return func() tea.Msg {
		content, markdown, err := loadSub(workdir, id, key)
		if err == nil && markdown {
			content = th.renderMarkdownWidth(content, wrap)
		}
		return subLoadedMsg{id: id, key: key, content: content, err: err}
	}
}

// listArea routes an area to its core-backed lister. Keeping the routing in one
// place makes the per-area core dependency explicit and uniform.
func listArea(workdir string, id areaID) ([]areaItem, error) {
	switch id {
	case areaInit:
		return listInit(workdir)
	case areaAgents:
		return listAgents()
	case areaPrompts:
		return listPrompts(workdir)
	case areaWorkflows:
		return listWorkflows(workdir)
	case areaBacklog:
		return listBacklog(workdir)
	case areaBuild:
		return listBuild(workdir)
	default:
		return nil, fmt.Errorf("unknown area")
	}
}

// loadSub routes an opened row to its core-backed detail loader. The bool result
// is markdown (Glamour-render) vs. pre-formatted (render verbatim).
func loadSub(workdir string, id areaID, key string) (string, bool, error) {
	switch id {
	case areaPrompts:
		return loadPromptSub(workdir, key)
	case areaWorkflows:
		return loadWorkflowSub(workdir, key)
	case areaBacklog:
		return loadBacklogSub(workdir, key)
	default:
		return "", false, fmt.Errorf("this item has no detail view")
	}
}

// --- init area → internal/workspace (Detect/ReadManifest) -------------------

// listInit consumes internal/workspace to summarize the workspace status: whether
// `.daedalus/` exists, the recorded backends (from the manifest), and what an init
// would create. The rows are informational (no sub-screen), so the area is a clear
// status read rather than an editor — the actual init lives in `daedalus init`.
func listInit(workdir string) ([]areaItem, error) {
	plan, err := workspace.Detect(workdir)
	if err != nil {
		return nil, err
	}

	// A target with no workspace yet is a valid empty state, not an error: the area
	// renders its "run daedalus init" empty message.
	if !plan.WorkspaceExisted {
		return nil, nil
	}

	items := []areaItem{
		{label: "Workspace", badge: filepath.ToSlash(plan.Path)},
	}

	// The manifest is the recorded selection; a workspace that exists but cannot be
	// read back is surfaced as an error rather than a half-status.
	if man, err := workspace.ReadManifest(workdir); err == nil {
		items = append(items,
			areaItem{label: "Project", badge: man.Name},
			areaItem{label: "Backends", badge: strings.Join(man.Backends, ", ")},
		)
	}

	if plan.IsEmpty() {
		items = append(items, areaItem{label: "Status", badge: "up to date"})
	} else {
		items = append(items, areaItem{
			label: "Would create",
			badge: fmt.Sprintf("%d dir(s), %d file(s)", len(plan.MissingDirs), len(plan.MissingFiles)),
		})
	}
	return items, nil
}

// --- agents area → internal/catalog (Builtin.List) --------------------------

// listAgents consumes the built-in catalog (embedded in the binary, so there is
// no disk read) and lists each agent by id and role. Materialize/Clone/Edit/Import
// are write operations that belong to `daedalus agent`; this area is the read-only
// catalog browser the navigation shell needs.
func listAgents() ([]areaItem, error) {
	entries := catalog.Builtin.List()
	items := make([]areaItem, 0, len(entries))
	for _, e := range entries {
		items = append(items, areaItem{key: e.ID, label: e.ID, badge: e.Role})
	}
	return items, nil
}

// --- prompts area → internal/prompts (List/Resolve) -------------------------

// promptsRoot derives the canonical `.daedalus/prompts/` directory under workdir,
// matching where init scaffolds prompts and where the CLI points. Kept here so the
// TUI owns the same workspace-location convention as the CLI without importing it.
func promptsRoot(workdir string) string {
	return filepath.Join(workdir, workspace.Name, prompts.PromptsDir)
}

// listPrompts consumes internal/prompts to list the workspace prompts. A missing
// prompts directory is reported by the core as an empty list (not an error), so an
// uninitialized workspace renders as a clean empty state. Each row opens a preview.
func listPrompts(workdir string) ([]areaItem, error) {
	entries, err := prompts.List(promptsRoot(workdir), "")
	if err != nil {
		return nil, err
	}
	items := make([]areaItem, 0, len(entries))
	for _, e := range entries {
		items = append(items, areaItem{
			key:   e.ID,
			label: e.ID + "  " + e.Title,
			badge: "[" + string(e.Kind) + "]",
			opens: true,
		})
	}
	return items, nil
}

// loadPromptSub composes one prompt via internal/prompts.Resolve (inclusion
// resolution done entirely by the core) and returns its Markdown for the preview.
func loadPromptSub(workdir, id string) (string, bool, error) {
	content, err := prompts.Resolve(promptsRoot(workdir), id)
	if err != nil {
		return "", false, fmt.Errorf("%s", composeErrorMessage(id, err))
	}
	return content, true, nil
}

// --- workflows area → internal/workflows (List/Load) ------------------------

// workflowsRoot derives the canonical `.daedalus/workflows/` directory under
// workdir, alongside promptsRoot.
func workflowsRoot(workdir string) string {
	return filepath.Join(workdir, workspace.Name, workflows.WorkflowsDir)
}

// listWorkflows consumes internal/workflows to list the workspace workflows. As
// with prompts, a missing directory is an empty list, not an error. Each row opens
// the read-only DAG view.
func listWorkflows(workdir string) ([]areaItem, error) {
	entries, err := workflows.List(workflowsRoot(workdir))
	if err != nil {
		return nil, err
	}
	items := make([]areaItem, 0, len(entries))
	for _, e := range entries {
		items = append(items, areaItem{
			key:   e.Name,
			label: e.Name,
			badge: fmt.Sprintf("[%d phases]", e.Phases),
			opens: true,
		})
	}
	return items, nil
}

// loadWorkflowSub loads one workflow via internal/workflows.Load and renders its
// DAG (the layout lives in workflows_view.go). The DAG is pre-formatted (boxes and
// connectors), so it is rendered verbatim, not through Glamour.
func loadWorkflowSub(workdir, name string) (string, bool, error) {
	w, err := workflows.Load(workflowsRoot(workdir), name)
	if err != nil {
		return "", false, fmt.Errorf("%s", workflowLoadErrorMessage(name, err))
	}
	// dagViewportContent is a Model method (pure rendering); a zero Model is enough
	// for the styles, which is how the lister stays off the model's mutable state.
	return Model{theme: defaultTheme()}.dagViewportContent(w), false, nil
}

// --- backlog area → internal/{specs,architecture,backlog} -------------------

// specsRoot/architectureRoot/epicsRoot derive the canonical backlog directories
// under workdir, matching init's scaffolding.
func specsRoot(workdir string) string {
	return filepath.Join(workdir, workspace.Name, specs.SpecsDir)
}

func architectureRoot(workdir string) string {
	return filepath.Join(workdir, workspace.Name, architecture.ArchitectureDir)
}

func epicsRoot(workdir string) string {
	return filepath.Join(workdir, workspace.Name, backlog.EpicsDir)
}

// listBacklog consumes the three backlog cores (specs, architecture, epics) and
// presents them as one flat, navigable list, each row tagged by kind. A failure in
// any one listing fails the area (so a corrupt backlog surfaces an error rather
// than a silently partial list); an all-empty backlog is a clean empty state. Rows
// open a Markdown body via loadBacklogSub. The sub-key encodes the kind so the
// detail loader knows which core to read (e.g. "spec:onboarding").
func listBacklog(workdir string) ([]areaItem, error) {
	var items []areaItem

	specEntries, err := specs.List(specsRoot(workdir))
	if err != nil {
		return nil, fmt.Errorf("listing specs: %w", err)
	}
	for _, e := range specEntries {
		badge := "[brief]"
		if e.HasSpec {
			badge = "[spec]"
		}
		items = append(items, areaItem{
			key:   "spec:" + e.Slug,
			label: e.Slug + "  " + e.Title,
			badge: badge,
			opens: true,
		})
	}

	archEntries, err := architecture.List(architectureRoot(workdir))
	if err != nil {
		return nil, fmt.Errorf("listing architecture: %w", err)
	}
	for _, e := range archEntries {
		items = append(items, areaItem{
			key:   "arch:" + e.Slug,
			label: e.Slug + "  " + e.Title,
			badge: "[arch]",
			opens: true,
		})
	}

	epicEntries, err := backlog.ListEpics(epicsRoot(workdir))
	if err != nil {
		return nil, fmt.Errorf("listing epics: %w", err)
	}
	for _, e := range epicEntries {
		items = append(items, areaItem{
			key:   "epic:" + e.ID,
			label: e.ID + "  " + e.Title,
			badge: "[epic]",
			opens: true,
		})
	}

	return items, nil
}

// loadBacklogSub loads one backlog item's Markdown body by its kind-tagged key.
// Each kind reads only its own core, so the detail view stays a thin read over the
// existing loaders.
func loadBacklogSub(workdir, key string) (string, bool, error) {
	kind, id, ok := strings.Cut(key, ":")
	if !ok {
		return "", false, fmt.Errorf("malformed backlog item %q", key)
	}
	switch kind {
	case "spec":
		// Prefer the spec body when a spec exists; fall back to the brief otherwise so
		// a brief-only entry still opens.
		if sp, err := specs.LoadSpec(specsRoot(workdir), id); err == nil {
			return sp.Body, true, nil
		}
		br, err := specs.LoadBrief(specsRoot(workdir), id)
		if err != nil {
			return "", false, err
		}
		return br.Body, true, nil
	case "arch":
		doc, err := architecture.Load(architectureRoot(workdir), id)
		if err != nil {
			return "", false, err
		}
		return doc.Body, true, nil
	case "epic":
		ep, err := backlog.LoadEpic(epicsRoot(workdir), id)
		if err != nil {
			return "", false, err
		}
		return ep.Body, true, nil
	default:
		return "", false, fmt.Errorf("unknown backlog kind %q", kind)
	}
}

// --- build area → internal/compile (Plan) -----------------------------------

// listBuild consumes internal/compile.Plan (read-only — it never writes) to
// summarize what a build WOULD change, one informational row per backend plus a
// total. The shell is a preview only: the confirm-and-write gate lives in the
// standalone `daedalus build` (RunBuildPreview), so this area never mutates the
// filesystem. A missing workspace / invalid definition surfaces as the area's
// error state with the same actionable wording the CLI uses.
func listBuild(workdir string) ([]areaItem, error) {
	res, err := compile.Plan(compile.Options{Root: workdir})
	if err != nil {
		return nil, fmt.Errorf("%s", planErrorMessage(err))
	}
	if !planHasChanges(res) {
		// All up to date: a valid empty state (nothing to compile).
		return nil, nil
	}

	var items []areaItem
	for _, bp := range res.Backends {
		c, u, n := countStatuses(bp)
		badge := fmt.Sprintf("%d new · %d modified · %d unchanged", c, u, n)
		if len(bp.Orphans) > 0 {
			badge += fmt.Sprintf(" · %d orphan%s", len(bp.Orphans), plural2(len(bp.Orphans)))
		}
		items = append(items, areaItem{label: bp.Backend, badge: badge})
	}
	items = append(items, areaItem{
		label: "Apply",
		badge: "run `daedalus build` to preview the diff and write",
	})
	return items, nil
}
