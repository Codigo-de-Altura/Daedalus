package conventions

import (
	"path/filepath"
	"strings"

	"github.com/Codigo-de-Altura/Daedalus/internal/backlog"
	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

// checkNaming verifies the naming conventions across the workspace (R2/CA2): the
// id/file-name shapes for agents, prompts, workflows, epics and tickets.
//
// Crucially it walks the directories RAW (sortedDirEntries) rather than through
// each domain's List function, because those List functions deliberately SKIP
// entries whose names are not well-formed (a misnamed file is "not one of ours").
// That skip is right for listing but wrong for validation: a convention checker
// must SEE the misnamed entry to report it. So we read the directory ourselves
// and apply the shared identity predicates (backlog.IsKebabCase / IsEpicID /
// IsTicketID) to every candidate.
func (w Workspace) checkNaming() ([]Finding, error) {
	var findings []Finding

	agentFindings, err := w.checkAgentNaming()
	if err != nil {
		return nil, err
	}
	promptFindings, err := w.checkPromptNaming()
	if err != nil {
		return nil, err
	}
	workflowFindings, err := w.checkWorkflowNaming()
	if err != nil {
		return nil, err
	}
	backlogFindings, err := w.checkBacklogNaming()
	if err != nil {
		return nil, err
	}

	findings = append(findings, agentFindings...)
	findings = append(findings, promptFindings...)
	findings = append(findings, workflowFindings...)
	findings = append(findings, backlogFindings...)
	return findings, nil
}

// checkAgentNaming verifies each materialized agent directory name is kebab-case
// (an agent's directory name is its id, like a ticket's folder is its id). Files
// directly under agents/ are unexpected (agents are directories) and flagged.
func (w Workspace) checkAgentNaming() ([]Finding, error) {
	entries, err := sortedDirEntries(w.AgentsRoot)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, e := range entries {
		loc := workspaceRel("agents", e.Name())
		if !e.IsDir() {
			findings = append(findings, Finding{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "agent-is-directory",
				Reason:     "an agent must be a directory (agent.yaml + prompt.md); a loose file here is out of place",
			})
			continue
		}
		if !backlog.IsKebabCase(e.Name()) {
			findings = append(findings, Finding{
				Family:     FamilyNaming,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "kebab-case",
				Reason:     "an agent directory name (its id) must be kebab-case: lowercase letters, digits and single dashes",
			})
		}
	}
	return findings, nil
}

// checkPromptNaming verifies each prompt file is a kebab-case `<id>.md`. The id is
// the file's base name; a non-.md file under prompts/ is out of place.
func (w Workspace) checkPromptNaming() ([]Finding, error) {
	entries, err := sortedDirEntries(w.PromptsRoot)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, e := range entries {
		loc := workspaceRel("prompts", e.Name())
		if e.IsDir() {
			findings = append(findings, Finding{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "prompt-is-file",
				Reason:     "a prompt must be a single .md file; an unexpected directory is out of place under prompts/",
			})
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, prompts.FileExt) {
			findings = append(findings, Finding{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "prompt-extension",
				Reason:     "a prompt file must have the .md extension",
			})
			continue
		}
		id := strings.TrimSuffix(name, prompts.FileExt)
		if !backlog.IsKebabCase(id) {
			findings = append(findings, Finding{
				Family:     FamilyNaming,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "kebab-case",
				Reason:     "a prompt id (its file name without .md) must be kebab-case",
			})
		}
	}
	return findings, nil
}

// checkWorkflowNaming verifies each workflow file is a kebab-case `<name>.yaml`.
func (w Workspace) checkWorkflowNaming() ([]Finding, error) {
	entries, err := sortedDirEntries(w.WorkflowsRoot)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, e := range entries {
		loc := workspaceRel("workflows", e.Name())
		if e.IsDir() {
			findings = append(findings, Finding{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "workflow-is-file",
				Reason:     "a workflow must be a single .yaml file; an unexpected directory is out of place under workflows/",
			})
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, workflows.FileExt) {
			findings = append(findings, Finding{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "workflow-extension",
				Reason:     "a workflow file must have the .yaml extension",
			})
			continue
		}
		wfName := strings.TrimSuffix(name, workflows.FileExt)
		if !backlog.IsKebabCase(wfName) {
			findings = append(findings, Finding{
				Family:     FamilyNaming,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "kebab-case",
				Reason:     "a workflow name (its file name without .yaml) must be kebab-case",
			})
		}
	}
	return findings, nil
}

// checkBacklogNaming verifies the epic/ticket id patterns and the nested layout.
// Every directory directly under epics/ must be a well-formed `epic-NN-<slug>`;
// every directory under an epic's tickets/ subdir must be a `ticket-NN-MM-<slug>`.
// The folder name IS the id (CLAUDE.md §6), so a malformed folder is a malformed
// id. We also report a ticket folder whose NN does not match its parent epic's NN,
// since the ticket id encodes the epic it belongs to.
func (w Workspace) checkBacklogNaming() ([]Finding, error) {
	epicDirs, err := sortedDirEntries(w.EpicsRoot)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, ed := range epicDirs {
		loc := workspaceRel("epics", ed.Name())
		if !ed.IsDir() {
			findings = append(findings, Finding{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "epic-is-directory",
				Reason:     "an epic must be a directory (epic-NN-<slug>/ containing epic.md); a loose file here is out of place",
			})
			continue
		}
		if !backlog.IsEpicID(ed.Name()) {
			findings = append(findings, Finding{
				Family:     FamilyNaming,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "epic-id-pattern",
				Reason:     "an epic directory name must match epic-NN-<slug> (NN numeric, slug kebab-case)",
			})
			// A malformed epic id makes its tickets' NN-match check meaningless; still
			// descend to catch malformed ticket ids beneath it.
		}

		ticketFindings, err := w.checkTicketNaming(ed.Name())
		if err != nil {
			return nil, err
		}
		findings = append(findings, ticketFindings...)
	}
	return findings, nil
}

// checkTicketNaming verifies the ticket folders under one epic. Each must be a
// well-formed ticket id, and its epic number (NN) must match the parent epic's.
func (w Workspace) checkTicketNaming(epicDir string) ([]Finding, error) {
	ticketsDir := filepath.Join(w.EpicsRoot, epicDir, backlog.TicketsSubdir)
	ticketDirs, err := sortedDirEntries(ticketsDir)
	if err != nil {
		return nil, err
	}
	var findings []Finding
	for _, td := range ticketDirs {
		loc := workspaceRel("epics", epicDir, backlog.TicketsSubdir, td.Name())
		if !td.IsDir() {
			findings = append(findings, Finding{
				Family:     FamilyStructure,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "ticket-is-directory",
				Reason:     "a ticket must be a directory (ticket-NN-MM-<slug>/ containing ticket.md); a loose file here is out of place",
			})
			continue
		}
		if !backlog.IsTicketID(td.Name()) {
			findings = append(findings, Finding{
				Family:     FamilyNaming,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "ticket-id-pattern",
				Reason:     "a ticket directory name must match ticket-NN-MM-<slug> (NN epic number, MM sequence, slug kebab-case)",
			})
			continue
		}
		// Cross-check the ticket's epic number against its parent epic's, since the
		// ticket id encodes which epic it belongs to (ticket-NN-...). Only meaningful
		// when the parent epic id is itself well-formed.
		if backlog.IsEpicID(epicDir) && !ticketEpicMatches(epicDir, td.Name()) {
			findings = append(findings, Finding{
				Family:     FamilyNaming,
				Severity:   SeverityError,
				Location:   loc,
				Convention: "ticket-epic-number-match",
				Reason:     "the ticket's epic number (NN) does not match its parent epic; a ticket-NN-MM id must carry the NN of the epic it lives under",
			})
		}
	}
	return findings, nil
}

// ticketEpicMatches reports whether the epic number embedded in a ticket id equals
// the epic number embedded in its parent epic id. Both ids are assumed well-formed
// (callers gate on IsEpicID/IsTicketID first). The number is the second
// dash-separated field of each id (epic-NN-... and ticket-NN-MM-...).
func ticketEpicMatches(epicID, ticketID string) bool {
	epicParts := strings.SplitN(epicID, "-", 3)
	ticketParts := strings.SplitN(ticketID, "-", 4)
	if len(epicParts) < 3 || len(ticketParts) < 4 {
		return false
	}
	return epicParts[1] == ticketParts[1]
}
