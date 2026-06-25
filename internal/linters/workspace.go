package linters

import (
	"path/filepath"

	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// Workspace is the set of canonical roots the linters inspect. The CLI builds it
// from the `.daedalus/` location convention and hands it in, so this package stays
// free of where the workspace lives (mirroring conventions.Workspace). Root is the
// repository directory; the manifest and the per-domain roots are resolved from it
// using the same layout constants the rest of the codebase uses.
type Workspace struct {
	// Repo is the target repository directory whose `.daedalus/` is linted.
	Repo string
	// AgentsRoot and WorkflowsRoot are the canonical domain subdirectories the
	// agent and workflow linters walk.
	AgentsRoot    string
	WorkflowsRoot string
}

// WorkspaceUnder builds a Workspace for the canonical `.daedalus/` directory under
// repoDir, resolving every root from the same layout constants the scaffolding and
// the conventions checker use, so the linters can never drift from where artifacts
// are actually written.
func WorkspaceUnder(repoDir string) Workspace {
	root := filepath.Join(repoDir, workspace.Name)
	return Workspace{
		Repo:          repoDir,
		AgentsRoot:    filepath.Join(root, catalog.AgentsDir),
		WorkflowsRoot: filepath.Join(root, workflows.WorkflowsDir),
	}
}

// Lint runs every definition linter over the workspace and returns an ordered,
// deterministic Report (R1/R8). It is read-only: it loads each definition from disk
// and runs the owning package's validator, but never writes. The families run in a
// fixed order (manifest, agents, workflows); the workflow linter is fed a
// knownAgents predicate built from the agents actually loaded, so an unknown-agent
// reference is caught (R3). Findings are stable-sorted at the end so collection
// order never leaks into the output (R8).
//
// A linter never aborts the whole report on one bad definition: a malformed or
// unreadable source becomes a controlled finding (R6), so one broken file never
// hides the rest. A genuine, unexpected I/O failure enumerating a directory is the
// only thing returned as an error.
func (w Workspace) Lint() (*Report, error) {
	report := &Report{}

	manifestFindings, err := w.lintManifest()
	if err != nil {
		return nil, err
	}

	// Load the agents once: the agent linter reports on them, and the loaded set is
	// also the basis for the workflow linter's knownAgents predicate, so a workflow
	// referencing a non-existent agent is detected against the same catalog the user
	// actually has on disk (R3).
	agents, agentFindings, err := w.lintAgents()
	if err != nil {
		return nil, err
	}

	workflowFindings, err := w.lintWorkflows(knownAgentsPredicate(agents))
	if err != nil {
		return nil, err
	}

	report.Findings = append(report.Findings, manifestFindings...)
	report.Findings = append(report.Findings, agentFindings...)
	report.Findings = append(report.Findings, workflowFindings...)

	sortFindings(report.Findings)
	return report, nil
}

// knownAgentsPredicate builds the agent-existence predicate the workflow semantic
// validator needs (workflows.ValidateGraph). An agent id is "known" when it is
// either a built-in catalog agent (embedded in the binary) OR a workspace agent
// that loaded AND validated. Including the built-ins is the established contract:
// the factory-default workflow references the five canonical built-in agents and is
// designed to pass the unknown-agent check against the built-in set even before a
// user materializes any of them (see workflows.DefaultWorkflow's doc-comment, ticket
// 04-04). Restricting "known" to on-disk agents only would make a pristine,
// freshly-initialized workspace report false unknown-agent findings for its own
// seeded workflow.
//
// A workspace agent whose definition is broken is excluded from the on-disk set
// (and already reported by the agent linter), so a workflow leaning on a broken,
// non-built-in agent is still surfaced as unknown.
func knownAgentsPredicate(agentIDs map[string]struct{}) func(string) bool {
	return func(id string) bool {
		if _, ok := agentIDs[id]; ok {
			return true
		}
		// Built-in agents are always available to a workflow: they ship in the binary
		// and a user materializes them on demand. Get returns ErrAgentNotFound for an
		// unknown id, which is exactly the "not known" answer.
		if _, err := catalog.Builtin.Get(id); err == nil {
			return true
		}
		return false
	}
}

// agentIDs is the set of candidate agent ids on disk: the sorted subdirectory names
// under the agents root. It wraps catalogAgentIDs so the agents linter has a single,
// named entry point for "what agents are present" on this workspace.
func (w Workspace) agentIDs() ([]string, error) {
	return catalogAgentIDs(w.AgentsRoot)
}
