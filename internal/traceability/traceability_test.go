package traceability_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/architecture"
	"github.com/Codigo-de-Altura/Daedalus/internal/backlog"
	"github.com/Codigo-de-Altura/Daedalus/internal/specs"
	"github.com/Codigo-de-Altura/Daedalus/internal/traceability"
)

// workspace is a tiny test harness that materializes specs, architecture documents,
// epics and tickets into a temp .daedalus/ layout, then builds the graph over it. It
// exercises the REAL sibling packages (no fakes), proving traceability reuses their
// recorded links as the single source of truth (R4/CA5).
type workspace struct {
	t         *testing.T
	specsRoot string
	archRoot  string
	epicsRoot string
}

func newWorkspace(t *testing.T) *workspace {
	t.Helper()
	root := t.TempDir()
	return &workspace{
		t:         t,
		specsRoot: filepath.Join(root, ".daedalus", specs.SpecsDir),
		archRoot:  filepath.Join(root, ".daedalus", architecture.ArchitectureDir),
		epicsRoot: filepath.Join(root, ".daedalus", backlog.EpicsDir),
	}
}

func (w *workspace) captureSpec(slug, title string) {
	w.t.Helper()
	if _, err := specs.Capture(w.specsRoot, specs.Brief{Slug: slug, Title: title, Body: "b"}); err != nil {
		w.t.Fatalf("capture spec %q: %v", slug, err)
	}
}

func (w *workspace) createArch(slug, title, specRef string) {
	w.t.Helper()
	d := architecture.Document{Slug: slug, Title: title}
	if specRef != "" {
		d.SpecRef = specRef + architecture.FileExt
	}
	if err := architecture.Create(w.archRoot, d); err != nil {
		w.t.Fatalf("create arch %q: %v", slug, err)
	}
}

func (w *workspace) createEpic(id, title, specSlug, archSlug string) {
	w.t.Helper()
	e := backlog.Epic{ID: id, Title: title}
	if specSlug != "" {
		e.SpecRef = specSlug + specs.FileExt
	}
	if archSlug != "" {
		e.ArchitectureRef = archSlug + architecture.FileExt
	}
	if err := backlog.CreateEpic(w.epicsRoot, e); err != nil {
		w.t.Fatalf("create epic %q: %v", id, err)
	}
}

func (w *workspace) createTicket(id, epicID, title string) {
	w.t.Helper()
	if err := backlog.CreateTicket(w.epicsRoot, backlog.Ticket{ID: id, EpicID: epicID, Title: title}); err != nil {
		w.t.Fatalf("create ticket %q: %v", id, err)
	}
}

func (w *workspace) build() *traceability.Graph {
	w.t.Helper()
	g, err := traceability.Build(w.specsRoot, w.archRoot, w.epicsRoot)
	if err != nil {
		w.t.Fatalf("Build: %v", err)
	}
	return g
}

// consistentWorkspace builds a fully-linked, consistent chain: spec -> arch -> epic ->
// ticket. Used by several tests as the happy baseline.
func consistentWorkspace(t *testing.T) *workspace {
	w := newWorkspace(t)
	w.captureSpec("sdd-backlog", "SDD Backlog")
	w.createArch("sdd-arch", "SDD Architecture", "sdd-backlog")
	w.createEpic("epic-05-sdd-backlog", "SDD Backlog", "sdd-backlog", "sdd-arch")
	w.createTicket("ticket-05-04-traceability", "epic-05-sdd-backlog", "Traceability")
	return w
}

// TestDescendFromSpec covers Check 1 (CA1): from a spec, reach its epics and tickets.
func TestDescendFromSpec(t *testing.T) {
	g := consistentWorkspace(t).build()

	chain, ok := g.DescendFromSpec("sdd-backlog")
	if !ok {
		t.Fatal("DescendFromSpec(sdd-backlog) not found")
	}
	if len(chain.Epics) != 1 || chain.Epics[0].Epic.ID != "epic-05-sdd-backlog" {
		t.Fatalf("descend did not reach the epic: %+v", chain.Epics)
	}
	tickets := chain.Epics[0].Tickets
	if len(tickets) != 1 || tickets[0].ID != "ticket-05-04-traceability" {
		t.Fatalf("descend did not reach the ticket: %+v", tickets)
	}
}

// TestAscendFromTicket covers Check 2 (CA2): from a ticket, climb to its epic and origin
// spec/architecture (inherited from the epic).
func TestAscendFromTicket(t *testing.T) {
	g := consistentWorkspace(t).build()

	chain, ok := g.AscendFromTicket("ticket-05-04-traceability")
	if !ok {
		t.Fatal("AscendFromTicket not found")
	}
	if !chain.EpicFound || chain.Epic.ID != "epic-05-sdd-backlog" {
		t.Fatalf("ascend did not reach the epic: %+v", chain)
	}
	// The ticket records no origin of its own; it inherits the epic's spec/arch.
	if !chain.OriginSpecFound || chain.OriginSpec.Slug != "sdd-backlog" {
		t.Errorf("ascend did not resolve the origin spec (inherited): %+v", chain)
	}
	if !chain.OriginArchFound || chain.OriginArch.Slug != "sdd-arch" {
		t.Errorf("ascend did not resolve the origin architecture (inherited): %+v", chain)
	}
}

// TestVerifyConsistent covers Check 3 (CA3): a fully-linked workspace has no errors.
func TestVerifyConsistent(t *testing.T) {
	g := consistentWorkspace(t).build()
	report := g.Verify()
	if report.HasErrors() {
		t.Fatalf("consistent workspace reported errors:\n%s", report.Error())
	}
	if !report.Consistent() {
		t.Fatalf("consistent workspace not reported consistent:\n%s", report.Error())
	}
}

// TestOrphanTicketDetected covers Check 4 (CA4): a ticket whose epic was removed is an
// orphan-ticket error.
func TestOrphanTicketDetected(t *testing.T) {
	w := consistentWorkspace(t)
	// Remove the parent epic's epic.md so the epic no longer loads, but keep the nested
	// ticket on disk (simulating a broken/edited workspace where the epic was deleted).
	if err := removeEpicFileKeepTickets(w); err != nil {
		t.Fatal(err)
	}

	report := w.build().Verify()
	if !report.HasErrors() {
		t.Fatalf("orphan ticket not reported as error:\n%s", report.Error())
	}
	if !hasFinding(report, "ticket-05-04-traceability", traceability.SeverityError, traceability.KindOrphanTicket) {
		t.Errorf("expected orphan-ticket error; got:\n%s", report.Error())
	}
}

// TestBrokenEpicSpecLinkDetected covers Check 4 (CA4): an epic referencing a missing spec
// is a broken-link error.
func TestBrokenEpicSpecLinkDetected(t *testing.T) {
	w := newWorkspace(t)
	// Epic references a spec that was never captured.
	w.createEpic("epic-05-sdd-backlog", "SDD Backlog", "nonexistent-spec", "")

	report := w.build().Verify()
	if !hasFinding(report, "epic-05-sdd-backlog", traceability.SeverityError, traceability.KindBrokenLink) {
		t.Errorf("expected broken-link error for missing spec; got:\n%s", report.Error())
	}
}

// TestBrokenArchSpecLinkDetected covers a broken link from an architecture document.
func TestBrokenArchSpecLinkDetected(t *testing.T) {
	w := newWorkspace(t)
	w.createArch("sdd-arch", "Arch", "nonexistent-spec")

	report := w.build().Verify()
	if !hasFinding(report, "sdd-arch", traceability.SeverityError, traceability.KindBrokenLink) {
		t.Errorf("expected broken-link error from architecture; got:\n%s", report.Error())
	}
}

// TestMissingOriginIsWarningNotError covers the SEVERITY SPLIT (the cross-cutting
// tension): an epic with NO origin link is a warning (gap), not an error — honoring
// 05-03's optional-origin decision. It must NOT make verification fail.
func TestMissingOriginIsWarningNotError(t *testing.T) {
	w := newWorkspace(t)
	// Epic with no spec/architecture, plus a valid nested ticket.
	w.createEpic("epic-05-sdd-backlog", "SDD Backlog", "", "")
	w.createTicket("ticket-05-04-traceability", "epic-05-sdd-backlog", "T")

	report := w.build().Verify()
	if report.HasErrors() {
		t.Fatalf("missing origin should NOT be an error (05-03 made it optional):\n%s", report.Error())
	}
	if !hasFinding(report, "epic-05-sdd-backlog", traceability.SeverityWarning, traceability.KindMissingOrigin) {
		t.Errorf("expected missing-origin warning for the epic; got:\n%s", report.Error())
	}
	// A ticket inheriting its (absent) origin from the epic must NOT itself be flagged —
	// the epic-level warning already covers the gap; double-flagging would be noise.
	if hasFindingForSubject(report, "ticket-05-04-traceability") {
		t.Errorf("ticket should not be flagged for an inherited missing origin; got:\n%s", report.Error())
	}
}

// TestTicketOwnBrokenLinkDetected covers a ticket that records its OWN broken spec link.
func TestTicketOwnBrokenLinkDetected(t *testing.T) {
	w := newWorkspace(t)
	w.captureSpec("sdd-backlog", "Spec")
	w.createEpic("epic-05-sdd-backlog", "Epic", "sdd-backlog", "")
	// Ticket with its own dangling spec link.
	tk := backlog.Ticket{ID: "ticket-05-04-traceability", EpicID: "epic-05-sdd-backlog", Title: "T",
		SpecRef: "ghost-spec" + specs.FileExt}
	if err := backlog.CreateTicket(w.epicsRoot, tk); err != nil {
		t.Fatal(err)
	}

	report := w.build().Verify()
	if !hasFinding(report, "ticket-05-04-traceability", traceability.SeverityError, traceability.KindBrokenLink) {
		t.Errorf("expected broken-link error for the ticket's own spec; got:\n%s", report.Error())
	}
}

// TestDeterminism covers Check 6 (CA6): verifying the same workspace twice yields
// byte-identical reports.
func TestDeterminism(t *testing.T) {
	// A workspace with several findings of mixed severity, to exercise ordering.
	w := newWorkspace(t)
	w.createEpic("epic-02-beta", "Beta", "missing-a", "")     // broken-link error
	w.createEpic("epic-01-alpha", "Alpha", "", "")            // missing-origin warning
	w.createArch("z-arch", "Z", "missing-b")                  // broken-link error
	w.createEpic("epic-03-gamma", "Gamma", "", "missing-arc") // broken-link error

	first := w.build().Verify().Error()
	second := w.build().Verify().Error()
	if first != second {
		t.Errorf("verify is not deterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

// TestFindingsOrderedErrorsFirst covers the deterministic ordering: errors precede
// warnings in the report.
func TestFindingsOrderedErrorsFirst(t *testing.T) {
	w := newWorkspace(t)
	w.createEpic("epic-01-alpha", "Alpha", "", "")      // warning (missing-origin)
	w.createEpic("epic-02-beta", "Beta", "missing", "") // error (broken-link)

	findings := w.build().Verify().Findings
	if len(findings) < 2 {
		t.Fatalf("expected at least 2 findings, got %d", len(findings))
	}
	if findings[0].Severity != traceability.SeverityError {
		t.Errorf("errors should sort first; got first = %+v", findings[0])
	}
}

// TestEmptyWorkspaceIsConsistent covers the degenerate case: an empty workspace has no
// chain and therefore no inconsistencies.
func TestEmptyWorkspaceIsConsistent(t *testing.T) {
	w := newWorkspace(t)
	report := w.build().Verify()
	if report.HasErrors() || len(report.Findings) != 0 {
		t.Errorf("empty workspace should be consistent with no findings; got:\n%s", report.Error())
	}
}

// TestNavigationNotFound covers the not-found path of both navigations.
func TestNavigationNotFound(t *testing.T) {
	g := newWorkspace(t).build()
	if _, ok := g.DescendFromSpec("nope"); ok {
		t.Error("DescendFromSpec of absent spec should be not-found")
	}
	if _, ok := g.AscendFromTicket("ticket-99-99-nope"); ok {
		t.Error("AscendFromTicket of absent ticket should be not-found")
	}
}

// --- test helpers ---

// removeEpicFileKeepTickets deletes only the epic.md of the test epic, leaving its nested
// tickets behind, to simulate an orphaned ticket whose parent epic no longer loads.
func removeEpicFileKeepTickets(w *workspace) error {
	path := filepath.Join(w.epicsRoot, "epic-05-sdd-backlog", backlog.EpicFile)
	return os.Remove(path)
}

func hasFinding(r *traceability.Report, subject string, sev traceability.Severity, kind traceability.FindingKind) bool {
	for _, f := range r.Findings {
		if f.Subject == subject && f.Severity == sev && f.Kind == kind {
			return true
		}
	}
	return false
}

func hasFindingForSubject(r *traceability.Report, subject string) bool {
	for _, f := range r.Findings {
		if f.Subject == subject {
			return true
		}
	}
	return false
}
