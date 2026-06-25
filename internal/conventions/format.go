package conventions

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Codigo-de-Altura/Daedalus/internal/architecture"
	"github.com/Codigo-de-Altura/Daedalus/internal/backlog"
	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/specs"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

// checkFormat verifies the YAML/Markdown format conventions (R4/CA4): ordered YAML
// keys and structured Markdown. It does so by ROUND-TRIPPING each artifact through
// its own canonical renderer — the very renderers ticket 08-02 pins with golden
// files — and comparing the canonical structural form against what is on disk:
//
//   - For frontmatter Markdown artifacts (prompts, specs, architecture, backlog),
//     it compares only the FRONTMATTER block (between the `---` fences), so a
//     human-authored body never triggers a false positive: the convention being
//     checked is the metadata's key order/shape, not the prose.
//   - For pure-YAML artifacts (workflows), it compares the whole rendered document
//     to the file, since the entire file IS the canonical YAML.
//
// A file whose canonical form differs from its on-disk form has non-canonical key
// ordering or structure and is flagged. Artifacts that fail to load are skipped
// here (their naming/structure problems are reported by the other passes); a
// genuine I/O error is surfaced.
func (w Workspace) checkFormat() ([]Finding, error) {
	var findings []Finding

	promptFindings, err := w.checkPromptFormat()
	if err != nil {
		return nil, err
	}
	workflowFindings, err := w.checkWorkflowFormat()
	if err != nil {
		return nil, err
	}
	specFindings, err := w.checkSpecFormat()
	if err != nil {
		return nil, err
	}
	archFindings, err := w.checkArchitectureFormat()
	if err != nil {
		return nil, err
	}
	backlogFindings, err := w.checkBacklogFormat()
	if err != nil {
		return nil, err
	}

	findings = append(findings, promptFindings...)
	findings = append(findings, workflowFindings...)
	findings = append(findings, specFindings...)
	findings = append(findings, archFindings...)
	findings = append(findings, backlogFindings...)
	return findings, nil
}

// frontmatterFinding is the shared shape of a format violation: the canonical
// frontmatter does not match what is on disk.
func frontmatterFinding(loc string) Finding {
	return Finding{
		Family:     FamilyFormat,
		Severity:   SeverityError,
		Location:   loc,
		Convention: "yaml-ordered-keys",
		Reason:     "the artifact's YAML frontmatter is not in canonical key order/format; re-save it (e.g. via the matching 'daedalus' edit command) to normalize",
	}
}

// wholeDocFinding is the format violation for a pure-YAML artifact (no body).
func wholeDocFinding(loc string) Finding {
	return Finding{
		Family:     FamilyFormat,
		Severity:   SeverityError,
		Location:   loc,
		Convention: "yaml-ordered-keys",
		Reason:     "the YAML document is not in canonical key order/format; re-save it to normalize",
	}
}

// checkPromptFormat round-trips every loadable prompt and compares frontmatter.
func (w Workspace) checkPromptFormat() ([]Finding, error) {
	entries, err := prompts.List(w.PromptsRoot, "")
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, e := range entries {
		p, err := prompts.Load(w.PromptsRoot, e.ID)
		if err != nil {
			continue
		}
		onDisk, ok, err := readFile(filepath.Join(w.PromptsRoot, e.ID+prompts.FileExt))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if !frontmatterMatches(prompts.Render(p), onDisk) {
			findings = append(findings, frontmatterFinding(workspaceRel("prompts", e.ID+prompts.FileExt)))
		}
	}
	return findings, nil
}

// checkWorkflowFormat round-trips every loadable workflow and compares the whole
// document (a workflow file is pure YAML with no body).
func (w Workspace) checkWorkflowFormat() ([]Finding, error) {
	entries, err := workflows.List(w.WorkflowsRoot)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, e := range entries {
		wf, err := workflows.Load(w.WorkflowsRoot, e.Name)
		if err != nil {
			continue
		}
		onDisk, ok, err := readFile(filepath.Join(w.WorkflowsRoot, e.Name+workflows.FileExt))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if normalizeTrailing(workflows.Render(wf)) != normalizeTrailing(onDisk) {
			findings = append(findings, wholeDocFinding(workspaceRel("workflows", e.Name+workflows.FileExt)))
		}
	}
	return findings, nil
}

// checkSpecFormat round-trips every materialized spec and compares frontmatter.
// Briefs are seeded by Daedalus and not re-edited through a render path, so the
// spec (the artifact a user refines) is the one whose frontmatter we pin.
func (w Workspace) checkSpecFormat() ([]Finding, error) {
	entries, err := specs.List(w.SpecsRoot)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, e := range entries {
		if !e.HasSpec {
			continue
		}
		s, err := specs.LoadSpec(w.SpecsRoot, e.Slug)
		if err != nil {
			continue
		}
		onDisk, ok, err := readFile(filepath.Join(w.SpecsRoot, e.Slug+specs.FileExt))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if !frontmatterMatches(specs.RenderSpec(s), onDisk) {
			findings = append(findings, frontmatterFinding(workspaceRel("specs", e.Slug+specs.FileExt)))
		}
	}
	return findings, nil
}

// checkArchitectureFormat round-trips every architecture document and compares
// frontmatter.
func (w Workspace) checkArchitectureFormat() ([]Finding, error) {
	entries, err := architecture.List(w.ArchitectureRoot)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, e := range entries {
		d, err := architecture.Load(w.ArchitectureRoot, e.Slug)
		if err != nil {
			continue
		}
		onDisk, ok, err := readFile(filepath.Join(w.ArchitectureRoot, e.Slug+architecture.FileExt))
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		if !frontmatterMatches(architecture.Render(d), onDisk) {
			findings = append(findings, frontmatterFinding(workspaceRel("architecture", e.Slug+architecture.FileExt)))
		}
	}
	return findings, nil
}

// checkBacklogFormat round-trips every epic and ticket and compares frontmatter.
func (w Workspace) checkBacklogFormat() ([]Finding, error) {
	epicEntries, err := backlog.ListEpics(w.EpicsRoot)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, ee := range epicEntries {
		epic, err := backlog.LoadEpic(w.EpicsRoot, ee.ID)
		if err != nil {
			continue
		}
		onDisk, ok, err := readFile(filepath.Join(w.EpicsRoot, ee.ID, backlog.EpicFile))
		if err != nil {
			return nil, err
		}
		if ok && !frontmatterMatches(backlog.RenderEpic(epic), onDisk) {
			findings = append(findings, frontmatterFinding(workspaceRel("epics", ee.ID, backlog.EpicFile)))
		}

		ticketEntries, err := backlog.ListTickets(w.EpicsRoot, ee.ID)
		if err != nil {
			return nil, err
		}
		for _, te := range ticketEntries {
			ticket, err := backlog.LoadTicket(w.EpicsRoot, ee.ID, te.ID)
			if err != nil {
				continue
			}
			tDisk, ok, err := readFile(filepath.Join(w.EpicsRoot, ee.ID, backlog.TicketsSubdir, te.ID, backlog.TicketFile))
			if err != nil {
				return nil, err
			}
			if ok && !frontmatterMatches(backlog.RenderTicket(ticket), tDisk) {
				findings = append(findings, frontmatterFinding(
					workspaceRel("epics", ee.ID, backlog.TicketsSubdir, te.ID, backlog.TicketFile)))
			}
		}
	}
	return findings, nil
}

// readFile reads path, reporting (content, true, nil) when present, ("", false,
// nil) when absent (not an error — another pass owns missing files), and an error
// only on a real I/O failure.
func readFile(path string) (string, bool, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(b), true, nil
}

// frontmatterMatches reports whether the YAML frontmatter block of the canonical
// render equals the frontmatter block on disk. It compares only the fenced
// frontmatter (between the first two `---` lines), so a human-authored body never
// affects the format verdict — the convention is the metadata's key order/shape.
// If either side has no frontmatter block, the comparison falls back to the whole
// (trailing-normalized) document, so an artifact that should have frontmatter but
// does not is still caught as a mismatch.
func frontmatterMatches(canonical, onDisk string) bool {
	cf, cok := extractFrontmatter(canonical)
	df, dok := extractFrontmatter(onDisk)
	if cok && dok {
		return cf == df
	}
	return normalizeTrailing(canonical) == normalizeTrailing(onDisk)
}

// extractFrontmatter returns the content between the first two `---` fence lines
// (exclusive), and whether a complete block was found. The fences must be the very
// first line and a later `---` line, matching how every renderer emits them.
func extractFrontmatter(doc string) (string, bool) {
	const fence = "---"
	lines := strings.Split(doc, "\n")
	if len(lines) == 0 || lines[0] != fence {
		return "", false
	}
	for i := 1; i < len(lines); i++ {
		if lines[i] == fence {
			return strings.Join(lines[1:i], "\n"), true
		}
	}
	return "", false
}

// normalizeTrailing trims trailing whitespace/newlines so a comparison ignores
// only end-of-file newline differences, never internal structure.
func normalizeTrailing(s string) string {
	return strings.TrimRight(s, "\n ")
}
