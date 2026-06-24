package tui

import (
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// This file owns the bridge between the TUI and the prompts core
// (internal/prompts). Every call into the core happens inside a tea.Cmd so the
// UI thread never blocks on filesystem I/O or composition work; the result of
// each call is delivered back to Update as a typed Msg. The presentation layer
// stays free of any domain logic — it only triggers these commands and reacts
// to their messages.

// promptsLoadedMsg reports the result of listing the workspace prompts. Exactly
// one of entries / err is meaningful: a non-nil err means the listing failed,
// an empty entries slice with a nil err means the workspace simply has no
// prompts (a valid empty state, not an error).
type promptsLoadedMsg struct {
	entries []prompts.Entry
	err     error
}

// promptResolvedMsg reports the result of resolving (composing) a single prompt
// for the preview. id echoes which prompt was requested so a late-arriving
// message for a prompt the user already navigated away from can be ignored.
// content holds the fully composed text on success; err holds a composition or
// load failure (e.g. *prompts.IncludeCycleError) on failure.
type promptResolvedMsg struct {
	id      string
	content string
	err     error
}

// promptsRoot derives the canonical `.daedalus/prompts/` directory under the
// given working directory, matching exactly where init scaffolds prompts and
// where the CLI's `promptsRootFor` points. Kept here so the TUI owns the same
// workspace-location convention as the CLI without importing it.
func promptsRoot(workdir string) string {
	return filepath.Join(workdir, workspace.Name, prompts.PromptsDir)
}

// loadPromptsCmd lists the workspace prompts off the UI thread and delivers the
// outcome as a promptsLoadedMsg. A missing prompts directory is reported by the
// core as an empty list (not an error), so a freshly initialized or
// not-yet-initialized workspace renders as a clean empty state rather than a
// crash. The empty filter ("") lists prompts of every kind.
func loadPromptsCmd(workdir string) tea.Cmd {
	root := promptsRoot(workdir)
	return func() tea.Msg {
		entries, err := prompts.List(root, "")
		return promptsLoadedMsg{entries: entries, err: err}
	}
}

// resolvePromptCmd composes a single prompt off the UI thread and delivers the
// outcome as a promptResolvedMsg. The composition (inclusion resolution) is done
// entirely by the core (prompts.Resolve); the TUI only renders the result or the
// typed error. id is echoed back so Update can discard a stale result.
func resolvePromptCmd(workdir, id string) tea.Cmd {
	root := promptsRoot(workdir)
	return func() tea.Msg {
		content, err := prompts.Resolve(root, id)
		return promptResolvedMsg{id: id, content: content, err: err}
	}
}

// workflowsLoadedMsg reports the result of listing the workspace workflows
// (ticket 04-02). Like promptsLoadedMsg, exactly one of entries / err is
// meaningful: a non-nil err means the listing failed, while an empty entries
// slice with a nil err means the workspace simply has no workflows (a valid empty
// state, not an error).
type workflowsLoadedMsg struct {
	entries []workflows.Entry
	err     error
}

// workflowLoadedMsg reports the result of loading a single workflow for the DAG
// view. name echoes which workflow was requested so a late-arriving message for a
// workflow the user already navigated away from can be ignored. workflow holds
// the fully parsed model on success; err holds a load/parse failure on failure.
type workflowLoadedMsg struct {
	name     string
	workflow workflows.Workflow
	err      error
}

// workflowsRoot derives the canonical `.daedalus/workflows/` directory under the
// given working directory, matching exactly where init scaffolds workflows and
// where the CLI's `workflowsRootFor` points. Kept alongside promptsRoot so the
// TUI owns the same workspace-location convention as the CLI without importing it.
func workflowsRoot(workdir string) string {
	return filepath.Join(workdir, workspace.Name, workflows.WorkflowsDir)
}

// loadWorkflowsCmd lists the workspace workflows off the UI thread and delivers
// the outcome as a workflowsLoadedMsg. A missing workflows directory is reported
// by the core as an empty list (not an error), so a freshly initialized or
// not-yet-initialized workspace renders as a clean empty state rather than a
// crash (same criterion as prompts).
func loadWorkflowsCmd(workdir string) tea.Cmd {
	root := workflowsRoot(workdir)
	return func() tea.Msg {
		entries, err := workflows.List(root)
		return workflowsLoadedMsg{entries: entries, err: err}
	}
}

// loadWorkflowCmd loads a single workflow off the UI thread and delivers the
// outcome as a workflowLoadedMsg. The parse is done entirely by the core
// (workflows.Load); the TUI only renders the resulting DAG or the load error.
// name is echoed back so Update can discard a stale result.
func loadWorkflowCmd(workdir, name string) tea.Cmd {
	root := workflowsRoot(workdir)
	return func() tea.Msg {
		w, err := workflows.Load(root, name)
		return workflowLoadedMsg{name: name, workflow: w, err: err}
	}
}
