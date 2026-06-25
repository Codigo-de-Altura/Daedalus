package conventions

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/Codigo-de-Altura/Daedalus/internal/architecture"
	"github.com/Codigo-de-Altura/Daedalus/internal/backlog"
	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/specs"
	"github.com/Codigo-de-Altura/Daedalus/internal/traceability"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// Workspace is the set of canonical roots the validator inspects. The CLI builds
// it from the `.daedalus/` location convention and hands it in, so this package
// stays free of where the workspace lives (mirroring how traceability.Build takes
// explicit roots). Root is the workspace directory itself (`<repo>/.daedalus`),
// used for the structure pass; the per-domain roots are its canonical subdirs.
type Workspace struct {
	// Root is the `.daedalus/` directory, the anchor for the structure check.
	Root string
	// AgentsRoot, PromptsRoot, WorkflowsRoot, SpecsRoot, ArchitectureRoot and
	// EpicsRoot are the canonical domain subdirectories under Root.
	AgentsRoot       string
	PromptsRoot      string
	WorkflowsRoot    string
	SpecsRoot        string
	ArchitectureRoot string
	EpicsRoot        string
}

// WorkspaceUnder builds a Workspace for the canonical `.daedalus/` directory under
// repoDir, resolving every domain root from the same layout constants the rest of
// the codebase uses, so the validator can never drift from where artifacts are
// actually written.
func WorkspaceUnder(repoDir string) Workspace {
	root := filepath.Join(repoDir, workspace.Name)
	return Workspace{
		Root:             root,
		AgentsRoot:       filepath.Join(root, catalog.AgentsDir),
		PromptsRoot:      filepath.Join(root, prompts.PromptsDir),
		WorkflowsRoot:    filepath.Join(root, workflows.WorkflowsDir),
		SpecsRoot:        filepath.Join(root, specs.SpecsDir),
		ArchitectureRoot: filepath.Join(root, architecture.ArchitectureDir),
		EpicsRoot:        filepath.Join(root, backlog.EpicsDir),
	}
}

// Validate runs every convention pass over the workspace and returns an ordered,
// deterministic Report (R6/R7). It is read-only: it inspects the filesystem and
// re-renders artifacts in memory but never writes. The passes run in a fixed
// order (naming, structure, format, traceability); findings are stable-sorted at
// the end so collection order never leaks into the output (R7/CA7).
//
// A genuine I/O failure reading a directory aborts with an error; a single
// malformed or missing artifact does not — it is reported as a finding, so one
// broken file never hides the rest of the report.
func (w Workspace) Validate() (*Report, error) {
	report := &Report{}

	structureFindings, err := w.checkStructure()
	if err != nil {
		return nil, err
	}
	namingFindings, err := w.checkNaming()
	if err != nil {
		return nil, err
	}
	formatFindings, err := w.checkFormat()
	if err != nil {
		return nil, err
	}
	traceFindings, err := w.checkTraceability()
	if err != nil {
		return nil, err
	}

	report.Findings = append(report.Findings, namingFindings...)
	report.Findings = append(report.Findings, structureFindings...)
	report.Findings = append(report.Findings, formatFindings...)
	report.Findings = append(report.Findings, traceFindings...)

	sortFindings(report.Findings)
	return report, nil
}

// checkStructure verifies the canonical `.daedalus/` layout: every required
// subdirectory of workspace.Subdirs is present (R3/CA3). A missing required
// directory is a hard error anchored to its workspace-relative path. The check
// never fails on I/O for an individual entry — it stats each expected path
// independently — so a partially-scaffolded workspace produces one finding per
// missing piece rather than a single opaque failure.
func (w Workspace) checkStructure() ([]Finding, error) {
	var findings []Finding

	// The workspace root itself must exist; without it there is no workspace to
	// validate and every subdir check would be redundant noise.
	info, err := os.Stat(w.Root)
	if err != nil {
		if os.IsNotExist(err) {
			return []Finding{{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   filepath.ToSlash(workspace.Name),
				Convention: "workspace-present",
				Reason:     "no .daedalus/ workspace found; run 'daedalus init' to scaffold it",
			}}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return []Finding{{
			Family:     FamilyStructure,
			Severity:   SeverityError,
			Location:   filepath.ToSlash(workspace.Name),
			Convention: "workspace-present",
			Reason:     ".daedalus exists but is not a directory; the workspace must be a directory",
		}}, nil
	}

	for _, sub := range workspace.Subdirs {
		path := filepath.Join(w.Root, sub)
		di, err := os.Stat(path)
		switch {
		case err == nil && di.IsDir():
			// Present and correct.
		case err == nil && !di.IsDir():
			findings = append(findings, Finding{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   workspaceRel(sub),
				Convention: "required-directory",
				Reason:     "expected a directory but found a file; the canonical layout requires this to be a directory",
			})
		case os.IsNotExist(err):
			findings = append(findings, Finding{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   workspaceRel(sub),
				Convention: "required-directory",
				Reason:     "required workspace directory is missing; run 'daedalus init' to restore the canonical layout",
			})
		default:
			return nil, err
		}
	}

	// The progress-state placeholder anchors .state/ in git (ticket 08-01); its
	// absence means the state directory would vanish from the repo, so flag it.
	statePath := filepath.Join(w.Root, filepath.FromSlash(workspace.StateReadme))
	if _, err := os.Stat(statePath); err != nil {
		if os.IsNotExist(err) {
			findings = append(findings, Finding{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   workspaceRel(filepath.FromSlash(workspace.StateReadme)),
				Convention: "state-tracked",
				Reason:     "the git-tracked .state/ placeholder is missing; run 'daedalus init' so .state/ persists in git",
			})
		} else {
			return nil, err
		}
	}

	return findings, nil
}

// checkTraceability delegates to internal/traceability, the existing owner of the
// spec -> epic -> ticket chain, and re-expresses its findings as convention
// findings (R5/CA5). Reusing it means ticket->epic and epic->origin checks are
// never re-implemented here: a recorded-but-dangling reference is a hard error,
// an entirely missing OPTIONAL origin link is a warning (honoring 05-03's
// optional-origin decision), exactly as `trace verify` already reports.
func (w Workspace) checkTraceability() ([]Finding, error) {
	graph, err := traceability.Build(w.SpecsRoot, w.ArchitectureRoot, w.EpicsRoot)
	if err != nil {
		return nil, err
	}
	report := graph.Verify()

	findings := make([]Finding, 0, len(report.Findings))
	for _, f := range report.Findings {
		findings = append(findings, Finding{
			Family:     FamilyTraceability,
			Severity:   mapTraceSeverity(f.Severity),
			Location:   f.Subject,
			Convention: string(f.Kind),
			Reason:     f.Reason,
		})
	}
	return findings, nil
}

// mapTraceSeverity maps a traceability severity onto a conventions severity. The
// two share the same error/warning split, so the mapping is direct; keeping it
// explicit means a future divergence in either package is a deliberate edit here,
// not a silent mismatch.
func mapTraceSeverity(s traceability.Severity) Severity {
	if s == traceability.SeverityError {
		return SeverityError
	}
	return SeverityWarning
}

// workspaceRel renders a workspace-subdir path as a workspace-relative location
// with forward slashes (e.g. ".daedalus/agents"), the stable form used in every
// finding's Location so the report is identical across operating systems.
func workspaceRel(parts ...string) string {
	all := append([]string{workspace.Name}, parts...)
	return filepath.ToSlash(filepath.Join(all...))
}

// sortedDirEntries reads dir and returns its entries sorted by name, or an empty
// slice when the directory does not exist (a not-yet-populated domain is not an
// error). Sorting makes every walk deterministic regardless of filesystem order
// (R7/CA7). A real I/O error (other than not-exist) is surfaced.
func sortedDirEntries(dir string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	return entries, nil
}
