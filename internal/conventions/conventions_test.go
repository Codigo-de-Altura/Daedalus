package conventions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/architecture"
	"github.com/Codigo-de-Altura/Daedalus/internal/backlog"
	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/specs"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// scaffoldConformantWorkspace builds a throwaway repo under t.TempDir() with a
// fully conformant `.daedalus/` workspace: the canonical layout plus one
// well-formed artifact of each kind, all written through their own renderers so
// they are already in canonical format. It returns the repo dir (the parent of
// .daedalus/). Tests then inject violations into this baseline.
func scaffoldConformantWorkspace(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()

	if _, err := workspace.Create(repo); err != nil {
		t.Fatalf("scaffold workspace: %v", err)
	}
	ws := WorkspaceUnder(repo)

	// A materialized agent (directory = id) with canonical files.
	if _, err := catalog.Builtin.Materialize(ws.AgentsRoot, "analyst"); err != nil {
		t.Fatalf("materialize agent: %v", err)
	}

	// A prompt.
	if err := prompts.Create(ws.PromptsRoot, prompts.Prompt{
		ID: "project-style", Kind: prompts.KindGlobal, Title: "Project Style", Body: "Write in English.",
	}); err != nil {
		t.Fatalf("create prompt: %v", err)
	}

	// A workflow.
	if err := workflows.Create(ws.WorkflowsRoot, workflows.Workflow{
		Name: "sdd-default",
		Phases: []workflows.Phase{
			{ID: "spec", Agent: "analyst", Inputs: []string{"brief"}, Outputs: []string{"spec"}, Gate: "spec-gate"},
		},
	}); err != nil {
		t.Fatalf("create workflow: %v", err)
	}

	// A spec (materialized): capture the brief then write a canonical spec body so
	// the spec frontmatter is in canonical form.
	writeFile(t, filepath.Join(ws.SpecsRoot, "payments"+specs.BriefExt),
		specs.RenderBrief(specs.Brief{Slug: "payments", Title: "Payments", Body: "Brief."}))
	writeFile(t, filepath.Join(ws.SpecsRoot, "payments"+specs.FileExt),
		specs.RenderSpec(specs.Spec{Slug: "payments", Title: "Payments", BriefRef: "payments" + specs.BriefExt, Body: "Spec."}))

	// An architecture document linked to the spec.
	writeFile(t, filepath.Join(ws.ArchitectureRoot, "payments-arch"+architecture.FileExt),
		architecture.Render(architecture.Document{Slug: "payments-arch", Title: "Payments Arch", SpecRef: "payments.md", Body: "Arch."}))

	// An epic + a ticket, both linked so traceability is clean.
	epicID := "epic-01-payments"
	ticketID := "ticket-01-01-checkout"
	writeFile(t, filepath.Join(ws.EpicsRoot, epicID, backlog.EpicFile),
		backlog.RenderEpic(backlog.Epic{
			ID: epicID, Title: "Payments", Status: backlog.StatusTodo, Priority: backlog.PriorityMedium,
			SpecRef: "payments", ArchitectureRef: "payments-arch", Body: "Epic.",
		}))
	writeFile(t, filepath.Join(ws.EpicsRoot, epicID, backlog.TicketsSubdir, ticketID, backlog.TicketFile),
		backlog.RenderTicket(backlog.Ticket{
			ID: ticketID, EpicID: epicID, Title: "Checkout", Status: backlog.StatusTodo, Priority: backlog.PriorityHigh,
			SpecRef: "payments", Body: "Ticket.",
		}))

	return repo
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// validate runs the validator over the repo's workspace and returns the report.
func validate(t *testing.T, repo string) *Report {
	t.Helper()
	report, err := WorkspaceUnder(repo).Validate()
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	return report
}

// findingAt reports whether the report carries a finding at loc with the given
// convention id, so a test can assert a specific violation was detected and
// anchored to the right location.
func findingAt(r *Report, loc, convention string) bool {
	for _, f := range r.Findings {
		if f.Location == loc && f.Convention == convention {
			return true
		}
	}
	return false
}

// TestConformantWorkspacePasses covers the happy path: a fully conformant
// workspace yields no error-level findings (Check 2/3/4/5 baseline).
func TestConformantWorkspacePasses(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	report := validate(t, repo)

	if report.HasErrors() {
		t.Errorf("conformant workspace reported errors:\n%s", report.String())
	}
}

// TestKebabCaseViolationDetected covers Check 2: a misnamed prompt (not
// kebab-case) is detected and reported with its location.
func TestKebabCaseViolationDetected(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	ws := WorkspaceUnder(repo)

	// A prompt file whose id is not kebab-case (camelCase).
	writeFile(t, filepath.Join(ws.PromptsRoot, "BadName"+prompts.FileExt),
		prompts.Render(prompts.Prompt{ID: "BadName", Kind: prompts.KindGlobal, Title: "X", Body: "y"}))

	report := validate(t, repo)
	loc := ".daedalus/prompts/BadName.md"
	if !findingAt(report, loc, "kebab-case") {
		t.Errorf("expected a kebab-case finding at %q; got:\n%s", loc, report.String())
	}
	if !report.HasErrors() {
		t.Error("a kebab-case violation must be an error")
	}
}

// TestEpicIDPatternViolationDetected covers Check 2: an epic folder that does not
// match epic-NN-<slug> is detected.
func TestEpicIDPatternViolationDetected(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	ws := WorkspaceUnder(repo)

	// A malformed epic directory (missing the NN number).
	writeFile(t, filepath.Join(ws.EpicsRoot, "epic-payments", backlog.EpicFile), "---\nkind: epic\n---\nx")

	report := validate(t, repo)
	loc := ".daedalus/epics/epic-payments"
	if !findingAt(report, loc, "epic-id-pattern") {
		t.Errorf("expected an epic-id-pattern finding at %q; got:\n%s", loc, report.String())
	}
}

// TestTicketIDPatternViolationDetected covers Check 2: a ticket folder that does
// not match ticket-NN-MM-<slug> is detected under a valid epic.
func TestTicketIDPatternViolationDetected(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	ws := WorkspaceUnder(repo)

	// A malformed ticket directory under the conformant epic.
	writeFile(t, filepath.Join(ws.EpicsRoot, "epic-01-payments", backlog.TicketsSubdir, "ticket-bad", backlog.TicketFile),
		"---\nkind: ticket\n---\nx")

	report := validate(t, repo)
	loc := ".daedalus/epics/epic-01-payments/tickets/ticket-bad"
	if !findingAt(report, loc, "ticket-id-pattern") {
		t.Errorf("expected a ticket-id-pattern finding at %q; got:\n%s", loc, report.String())
	}
}

// TestTicketEpicNumberMismatchDetected covers Check 2: a well-formed ticket id
// whose NN does not match its parent epic is flagged.
func TestTicketEpicNumberMismatchDetected(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	ws := WorkspaceUnder(repo)

	// ticket-09-01 under epic-01 — well-formed id, wrong epic number.
	writeFile(t, filepath.Join(ws.EpicsRoot, "epic-01-payments", backlog.TicketsSubdir, "ticket-09-01-stray", backlog.TicketFile),
		backlog.RenderTicket(backlog.Ticket{
			ID: "ticket-09-01-stray", EpicID: "epic-01-payments", Title: "Stray",
			Status: backlog.StatusTodo, Priority: backlog.PriorityLow, SpecRef: "payments", Body: "x",
		}))

	report := validate(t, repo)
	loc := ".daedalus/epics/epic-01-payments/tickets/ticket-09-01-stray"
	if !findingAt(report, loc, "ticket-epic-number-match") {
		t.Errorf("expected a ticket-epic-number-match finding at %q; got:\n%s", loc, report.String())
	}
}

// TestMissingDirectoryDetected covers Check 3: a required workspace directory that
// is absent is detected and reported with its location.
func TestMissingDirectoryDetected(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	ws := WorkspaceUnder(repo)

	if err := os.RemoveAll(filepath.Join(ws.Root, "architecture")); err != nil {
		t.Fatal(err)
	}

	report := validate(t, repo)
	loc := ".daedalus/architecture"
	if !findingAt(report, loc, "required-directory") {
		t.Errorf("expected a required-directory finding at %q; got:\n%s", loc, report.String())
	}
}

// TestMissingStatePlaceholderDetected covers Check 3 + ticket 08-01: the
// git-tracked .state/ placeholder being absent is detected.
func TestMissingStatePlaceholderDetected(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	ws := WorkspaceUnder(repo)

	if err := os.Remove(filepath.Join(ws.Root, filepath.FromSlash(workspace.StateReadme))); err != nil {
		t.Fatal(err)
	}

	report := validate(t, repo)
	loc := ".daedalus/" + workspace.StateReadme
	if !findingAt(report, loc, "state-tracked") {
		t.Errorf("expected a state-tracked finding at %q; got:\n%s", loc, report.String())
	}
}

// TestFormatViolationDetected covers Check 4: an artifact whose YAML frontmatter
// is out of canonical key order is detected.
func TestFormatViolationDetected(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	ws := WorkspaceUnder(repo)

	// Hand-write a prompt with frontmatter keys in the WRONG order (title before
	// id/kind). It is still loadable, but not in canonical form.
	noncanonical := "---\ntitle: Out Of Order\nkind: global\nid: misordered\n---\nbody\n"
	writeFile(t, filepath.Join(ws.PromptsRoot, "misordered"+prompts.FileExt), noncanonical)

	report := validate(t, repo)
	loc := ".daedalus/prompts/misordered.md"
	if !findingAt(report, loc, "yaml-ordered-keys") {
		t.Errorf("expected a yaml-ordered-keys finding at %q; got:\n%s", loc, report.String())
	}
}

// TestTraceabilityViolationDetected covers Check 5: a ticket whose parent epic
// does not exist (orphan) is detected via the reused traceability checker.
func TestTraceabilityViolationDetected(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	ws := WorkspaceUnder(repo)

	// A well-formed ticket under an epic folder that has NO epic.md => orphan.
	orphanEpicDir := "epic-07-ghost"
	writeFile(t, filepath.Join(ws.EpicsRoot, orphanEpicDir, backlog.TicketsSubdir, "ticket-07-01-lost", backlog.TicketFile),
		backlog.RenderTicket(backlog.Ticket{
			ID: "ticket-07-01-lost", EpicID: orphanEpicDir, Title: "Lost",
			Status: backlog.StatusTodo, Priority: backlog.PriorityLow, SpecRef: "payments", Body: "x",
		}))

	report := validate(t, repo)
	if !findingAt(report, "ticket-07-01-lost", "orphan-ticket") {
		t.Errorf("expected an orphan-ticket finding for the lost ticket; got:\n%s", report.String())
	}
}

// TestReportIsDeterministic covers Check 7: validating the same workspace twice
// yields byte-identical report output.
func TestReportIsDeterministic(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	ws := WorkspaceUnder(repo)

	// Inject several unordered violations so the sort has work to do.
	writeFile(t, filepath.Join(ws.PromptsRoot, "ZName"+prompts.FileExt),
		prompts.Render(prompts.Prompt{ID: "ZName", Kind: prompts.KindGlobal, Title: "X", Body: "y"}))
	writeFile(t, filepath.Join(ws.PromptsRoot, "AName"+prompts.FileExt),
		prompts.Render(prompts.Prompt{ID: "AName", Kind: prompts.KindGlobal, Title: "X", Body: "y"}))
	if err := os.RemoveAll(filepath.Join(ws.Root, "docs")); err != nil {
		t.Fatal(err)
	}

	first := validate(t, repo).String()
	second := validate(t, repo).String()
	if first != second {
		t.Errorf("report not deterministic:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// TestMissingWorkspaceReported covers the no-workspace case: validating a repo
// with no .daedalus/ reports a single actionable structure error rather than
// crashing.
func TestMissingWorkspaceReported(t *testing.T) {
	repo := t.TempDir()
	report := validate(t, repo)
	if !findingAt(report, ".daedalus", "workspace-present") {
		t.Errorf("expected a workspace-present finding; got:\n%s", report.String())
	}
}

// TestFindingsAreActionable covers Check 6: every finding carries a non-empty
// location, convention id and reason, so the report is actionable.
func TestFindingsAreActionable(t *testing.T) {
	repo := scaffoldConformantWorkspace(t)
	ws := WorkspaceUnder(repo)
	writeFile(t, filepath.Join(ws.PromptsRoot, "BadName"+prompts.FileExt),
		prompts.Render(prompts.Prompt{ID: "BadName", Kind: prompts.KindGlobal, Title: "X", Body: "y"}))

	report := validate(t, repo)
	if len(report.Findings) == 0 {
		t.Fatal("expected at least one finding")
	}
	for _, f := range report.Findings {
		if strings.TrimSpace(f.Location) == "" || strings.TrimSpace(f.Convention) == "" || strings.TrimSpace(f.Reason) == "" {
			t.Errorf("finding is not actionable (missing location/convention/reason): %+v", f)
		}
	}
}
