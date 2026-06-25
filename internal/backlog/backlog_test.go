package backlog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// epicID and ticketID helpers keep the test bodies readable.
const (
	testEpicID   = "epic-05-sdd-backlog"
	testTicketID = "ticket-05-03-epics-tickets-management"
)

func mustCreateEpic(t *testing.T, root string, e Epic) {
	t.Helper()
	if err := CreateEpic(root, e); err != nil {
		t.Fatalf("CreateEpic(%q): %v", e.ID, err)
	}
}

func mustCreateTicket(t *testing.T, root string, tk Ticket) {
	t.Helper()
	if err := CreateTicket(root, tk); err != nil {
		t.Fatalf("CreateTicket(%q): %v", tk.ID, err)
	}
}

// TestEpicCanonicalLocationAndDefaults covers Check 1/3/6 (CA1/CA3/CA6): creating an
// epic persists `.daedalus/epics/epic-NN-<slug>/epic.md` with stable metadata and
// default status/priority.
func TestEpicCanonicalLocationAndDefaults(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "SDD Backlog"})

	path := filepath.Join(root, testEpicID, "epic.md")
	content := readFile(t, path)
	for _, want := range []string{
		"id: epic-05-sdd-backlog",
		"kind: epic",
		"title: SDD Backlog",
		"status: todo",     // default
		"priority: medium", // default
		"depends_on: []",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("epic frontmatter missing %q; got:\n%s", want, content)
		}
	}
}

// TestTicketNestedUnderEpic covers Check 2/5 (CA2/CA5): a ticket is created nested under
// its epic and references its parent epic.
func TestTicketNestedUnderEpic(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "SDD Backlog"})
	mustCreateTicket(t, root, Ticket{ID: testTicketID, EpicID: testEpicID, Title: "Epics & Tickets"})

	path := filepath.Join(root, testEpicID, "tickets", testTicketID, "ticket.md")
	content := readFile(t, path)
	if !strings.Contains(content, "kind: ticket") {
		t.Errorf("missing kind: ticket; got:\n%s", content)
	}
	if !strings.Contains(content, "epic: epic-05-sdd-backlog") {
		t.Errorf("ticket must reference its epic (R5); got:\n%s", content)
	}
}

// TestTicketRequiresExistingParentEpic covers the structural prerequisite: a ticket
// cannot be created under a missing epic.
func TestTicketRequiresExistingParentEpic(t *testing.T) {
	root := t.TempDir()
	err := CreateTicket(root, Ticket{ID: testTicketID, EpicID: testEpicID, Title: "Orphan"})
	if !errors.Is(err, ErrParentEpicMissing) {
		t.Fatalf("create ticket under missing epic error = %v, want ErrParentEpicMissing", err)
	}
}

// TestMetadataStatusPriorityValidated covers Check 6 (CA6): status/priority are closed
// sets; an out-of-set value is rejected.
func TestMetadataStatusPriorityValidated(t *testing.T) {
	root := t.TempDir()
	err := CreateEpic(root, Epic{ID: testEpicID, Title: "T", Status: Status("weird")})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("bad status error = %v, want *ValidationError", err)
	}

	err = CreateEpic(root, Epic{ID: testEpicID, Title: "T", Priority: Priority("urgent")})
	if !errors.As(err, &ve) {
		t.Fatalf("bad priority error = %v, want *ValidationError", err)
	}
}

// TestDependenciesExplicitAndConsistent covers Check 4 (CA4): dependencies render as an
// explicit, deduplicated, order-preserved list.
func TestDependenciesExplicitAndConsistent(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Epic"})
	mustCreateTicket(t, root, Ticket{
		ID:        testTicketID,
		EpicID:    testEpicID,
		Title:     "T",
		DependsOn: []string{"ticket-05-02-architecture-docs", "ticket-05-01-brief-to-spec", "ticket-05-02-architecture-docs"},
	})

	content := readFile(t, filepath.Join(root, testEpicID, "tickets", testTicketID, "ticket.md"))
	// Deduplicated, order preserved (first occurrence position kept).
	want := "depends_on: [ticket-05-02-architecture-docs, ticket-05-01-brief-to-spec]"
	if !strings.Contains(content, want) {
		t.Errorf("dependency list not explicit/deduped; want %q in:\n%s", want, content)
	}
}

// TestOriginLinksAndProvenance covers Check 3/5 (CA3/CA5): origin links (spec/
// architecture) are recorded, and the planner-step provenance is anchored to the real
// phase when linked.
func TestOriginLinksAndProvenance(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Epic", SpecRef: "sdd-backlog", ArchitectureRef: "sdd-backlog-arch"})

	content := readFile(t, filepath.Join(root, testEpicID, "epic.md"))
	for _, want := range []string{
		"spec: sdd-backlog",
		"architecture: sdd-backlog-arch",
		"agent: planner",
		"workflow: sdd-default",
		"phase: epics",
		"generated: false",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("linked epic missing %q; got:\n%s", want, content)
		}
	}
	if strings.Contains(content, `generated: "false"`) {
		t.Errorf("generated should be an unquoted boolean; got:\n%s", content)
	}
}

// TestUnlinkedOmitsOriginAndProvenance covers the omit-empty + all-or-nothing rules: an
// unlinked epic carries no spec/architecture and no provenance group.
func TestUnlinkedOmitsOriginAndProvenance(t *testing.T) {
	content := RenderEpic(Epic{ID: testEpicID, Title: "Epic", Status: StatusTodo, Priority: PriorityMedium})
	for _, key := range []string{"spec:", "architecture:", "agent:", "workflow:", "phase:", "generated:"} {
		if strings.Contains(content, key) {
			t.Errorf("unlinked epic should omit %q; got:\n%s", key, content)
		}
	}
}

// TestTicketPhaseIsTickets covers that a linked ticket records phase: tickets (not
// epics).
func TestTicketPhaseIsTickets(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Epic"})
	mustCreateTicket(t, root, Ticket{ID: testTicketID, EpicID: testEpicID, Title: "T", SpecRef: "sdd-backlog"})

	content := readFile(t, filepath.Join(root, testEpicID, "tickets", testTicketID, "ticket.md"))
	if !strings.Contains(content, "phase: tickets") {
		t.Errorf("linked ticket should record phase: tickets; got:\n%s", content)
	}
}

// TestDuplicateEpicFails covers non-destruction (R8/CA8): a duplicate epic id fails and
// does not overwrite.
func TestDuplicateEpicFails(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Original", Body: "original"})
	original := readFile(t, filepath.Join(root, testEpicID, "epic.md"))

	err := CreateEpic(root, Epic{ID: testEpicID, Title: "Other", Body: "overwrite"})
	if !errors.Is(err, ErrEpicExists) {
		t.Fatalf("duplicate epic error = %v, want ErrEpicExists", err)
	}
	if got := readFile(t, filepath.Join(root, testEpicID, "epic.md")); got != original {
		t.Errorf("duplicate create overwrote the epic")
	}
}

// TestDuplicateTicketFails covers non-destruction for tickets.
func TestDuplicateTicketFails(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Epic"})
	mustCreateTicket(t, root, Ticket{ID: testTicketID, EpicID: testEpicID, Title: "Original", Body: "original"})
	original := readFile(t, filepath.Join(root, testEpicID, "tickets", testTicketID, "ticket.md"))

	err := CreateTicket(root, Ticket{ID: testTicketID, EpicID: testEpicID, Title: "Other", Body: "overwrite"})
	if !errors.Is(err, ErrTicketExists) {
		t.Fatalf("duplicate ticket error = %v, want ErrTicketExists", err)
	}
	if got := readFile(t, filepath.Join(root, testEpicID, "tickets", testTicketID, "ticket.md")); got != original {
		t.Errorf("duplicate create overwrote the ticket")
	}
}

// TestEditDoesNotDestroyOtherFiles covers Check 8 (R8/CA8): editing one ticket changes
// only its file; the epic and sibling artifacts stay byte-identical.
func TestEditDoesNotDestroyOtherFiles(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Epic", Body: "epic body"})
	mustCreateTicket(t, root, Ticket{ID: testTicketID, EpicID: testEpicID, Title: "T", Body: "v1"})

	epicBefore := readFile(t, filepath.Join(root, testEpicID, "epic.md"))

	edited, err := EditTicket(root, testEpicID, testTicketID, TicketEditSpec{SetBody: true, Body: "REFINED BY HUMAN"})
	if err != nil {
		t.Fatalf("EditTicket: %v", err)
	}
	if edited.Body != "REFINED BY HUMAN" {
		t.Errorf("edit not applied; got %q", edited.Body)
	}
	if got := readFile(t, filepath.Join(root, testEpicID, "epic.md")); got != epicBefore {
		t.Errorf("editing a ticket altered the epic file")
	}
	ticketAfter := readFile(t, filepath.Join(root, testEpicID, "tickets", testTicketID, "ticket.md"))
	if !strings.Contains(ticketAfter, "REFINED BY HUMAN") {
		t.Errorf("ticket edit not persisted; got:\n%s", ticketAfter)
	}
}

// TestEditStatusAndPriority covers metadata editing with enum validation.
func TestEditStatusAndPriority(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Epic"})

	edited, err := EditEpic(root, testEpicID, EpicEditSpec{
		SetStatus: true, Status: StatusInProgress, SetPriority: true, Priority: PriorityHigh,
	})
	if err != nil {
		t.Fatalf("EditEpic: %v", err)
	}
	if edited.Status != StatusInProgress || edited.Priority != PriorityHigh {
		t.Errorf("edit not applied; got status=%q priority=%q", edited.Status, edited.Priority)
	}

	// A bad status is rejected and leaves the file intact.
	before := readFile(t, filepath.Join(root, testEpicID, "epic.md"))
	_, err = EditEpic(root, testEpicID, EpicEditSpec{SetStatus: true, Status: Status("nope")})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("bad status edit error = %v, want *ValidationError", err)
	}
	if got := readFile(t, filepath.Join(root, testEpicID, "epic.md")); got != before {
		t.Errorf("rejected edit altered the file")
	}
}

// TestEditClearsOriginLink covers clearing an optional link via SetSpec with empty
// value, which drops the provenance group on the next render.
func TestEditClearsOriginLink(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Epic", SpecRef: "sdd-backlog"})
	if !strings.Contains(readFile(t, filepath.Join(root, testEpicID, "epic.md")), "agent: planner") {
		t.Fatalf("epic should start linked")
	}

	if _, err := EditEpic(root, testEpicID, EpicEditSpec{SetSpec: true, Spec: ""}); err != nil {
		t.Fatalf("EditEpic clear: %v", err)
	}
	// Inspect only the frontmatter region: the seeded placeholder BODY may still mention
	// the old spec (we never rewrite a user's body), which is fine — the structured link
	// lives in the frontmatter, and that is what must be dropped.
	cleared := frontmatterOf(t, readFile(t, filepath.Join(root, testEpicID, "epic.md")))
	for _, key := range []string{"spec:", "agent:", "workflow:", "phase:", "generated:"} {
		if strings.Contains(cleared, key) {
			t.Errorf("clearing the link should drop %q from frontmatter; got:\n%s", key, cleared)
		}
	}
}

// frontmatterOf returns the frontmatter region (between the first two `---` lines) of a
// rendered artifact, so a test can assert on metadata without matching the body.
func frontmatterOf(t *testing.T, content string) string {
	t.Helper()
	parts := strings.SplitN(content, "---\n", 3)
	if len(parts) < 3 {
		t.Fatalf("content has no frontmatter block:\n%s", content)
	}
	return parts[1]
}

// TestDeterminism covers Check 6 (CA6): rendering the same artifact twice is
// byte-identical, and creating into two clean roots yields identical files.
func TestDeterminism(t *testing.T) {
	e := Epic{ID: testEpicID, Title: "Epic", Status: StatusTodo, Priority: PriorityMedium,
		SpecRef: "s", DependsOn: []string{"epic-04-x"}, Body: "a\nb"}
	if RenderEpic(e) != RenderEpic(e) {
		t.Errorf("RenderEpic not deterministic")
	}

	rootA, rootB := t.TempDir(), t.TempDir()
	mustCreateEpic(t, rootA, e)
	mustCreateEpic(t, rootB, e)
	if readFile(t, filepath.Join(rootA, testEpicID, "epic.md")) != readFile(t, filepath.Join(rootB, testEpicID, "epic.md")) {
		t.Errorf("two creates with identical input produced different files")
	}
}

// TestInvalidIDsRejected covers id-format validation (R1/R2).
func TestInvalidIDsRejected(t *testing.T) {
	root := t.TempDir()
	// Bad epic ids.
	for _, id := range []string{"", "epic-sdd", "epic-05-", "Epic-05-x", "epic-05-Bad_Slug", "epic_05_x"} {
		if err := CreateEpic(root, Epic{ID: id, Title: "T"}); err == nil {
			t.Errorf("CreateEpic(%q) succeeded, want rejection", id)
		}
	}
	// Bad ticket ids (with a valid parent epic present).
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Epic"})
	for _, id := range []string{"", "ticket-05-x", "ticket-05-03", "ticket-05-03-", "Ticket-05-03-x"} {
		if err := CreateTicket(root, Ticket{ID: id, EpicID: testEpicID, Title: "T"}); err == nil {
			t.Errorf("CreateTicket(%q) succeeded, want rejection", id)
		}
	}
}

// TestListsSortedById covers listing (R6).
func TestListsSortedById(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: "epic-02-beta", Title: "Beta"})
	mustCreateEpic(t, root, Epic{ID: "epic-01-alpha", Title: "Alpha"})

	epics, err := ListEpics(root)
	if err != nil {
		t.Fatalf("ListEpics: %v", err)
	}
	if len(epics) != 2 || epics[0].ID != "epic-01-alpha" || epics[1].ID != "epic-02-beta" {
		t.Fatalf("ListEpics not sorted: %+v", epics)
	}

	mustCreateTicket(t, root, Ticket{ID: "ticket-01-02-b", EpicID: "epic-01-alpha", Title: "B"})
	mustCreateTicket(t, root, Ticket{ID: "ticket-01-01-a", EpicID: "epic-01-alpha", Title: "A"})
	tickets, err := ListTickets(root, "epic-01-alpha")
	if err != nil {
		t.Fatalf("ListTickets: %v", err)
	}
	if len(tickets) != 2 || tickets[0].ID != "ticket-01-01-a" || tickets[1].ID != "ticket-01-02-b" {
		t.Fatalf("ListTickets not sorted: %+v", tickets)
	}
}

// TestListEmptyWorkspace covers the well-defined empty case.
func TestListEmptyWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "nope")
	epics, err := ListEpics(root)
	if err != nil || len(epics) != 0 {
		t.Errorf("ListEpics empty = (%v, %v), want ([], nil)", epics, err)
	}
}

// TestLoadRoundTrip covers load<->render stability for both artifacts, including the
// dependency list and origin links.
func TestLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	e := Epic{ID: testEpicID, Title: "SDD: Backlog", Status: StatusInProgress, Priority: PriorityHigh,
		SpecRef: "sdd-backlog", ArchitectureRef: "sdd-arch", DependsOn: []string{"epic-04-workflows"}, Body: "line one\nline two"}
	mustCreateEpic(t, root, e)

	loaded, err := LoadEpic(root, testEpicID)
	if err != nil {
		t.Fatalf("LoadEpic: %v", err)
	}
	if loaded.ID != e.ID || loaded.Title != e.Title || loaded.Status != e.Status || loaded.Priority != e.Priority ||
		loaded.SpecRef != e.SpecRef || loaded.ArchitectureRef != e.ArchitectureRef || loaded.Body != e.Body ||
		len(loaded.DependsOn) != 1 || loaded.DependsOn[0] != "epic-04-workflows" {
		t.Errorf("epic round-trip mismatch\nwant: %+v\ngot:  %+v", e, loaded)
	}
	if RenderEpic(loaded) != readFile(t, filepath.Join(root, testEpicID, "epic.md")) {
		t.Errorf("re-render of a loaded epic is not byte-stable")
	}

	tk := Ticket{ID: testTicketID, EpicID: testEpicID, Title: "T", Status: StatusTodo, Priority: PriorityMedium,
		SpecRef: "sdd-backlog", DependsOn: []string{"ticket-05-02-architecture-docs"}, Body: "body"}
	mustCreateTicket(t, root, tk)
	loadedT, err := LoadTicket(root, testEpicID, testTicketID)
	if err != nil {
		t.Fatalf("LoadTicket: %v", err)
	}
	if loadedT.EpicID != testEpicID || len(loadedT.DependsOn) != 1 || loadedT.DependsOn[0] != "ticket-05-02-architecture-docs" {
		t.Errorf("ticket round-trip mismatch: %+v", loadedT)
	}
	if RenderTicket(loadedT) != readFile(t, filepath.Join(root, testEpicID, "tickets", testTicketID, "ticket.md")) {
		t.Errorf("re-render of a loaded ticket is not byte-stable")
	}
}

// TestNotFoundSentinels covers the not-found sentinels.
func TestNotFoundSentinels(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadEpic(root, testEpicID); !errors.Is(err, ErrEpicNotFound) {
		t.Errorf("LoadEpic absent = %v, want ErrEpicNotFound", err)
	}
	if err := RemoveEpic(root, testEpicID); !errors.Is(err, ErrEpicNotFound) {
		t.Errorf("RemoveEpic absent = %v, want ErrEpicNotFound", err)
	}
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Epic"})
	if _, err := LoadTicket(root, testEpicID, testTicketID); !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("LoadTicket absent = %v, want ErrTicketNotFound", err)
	}
	if err := RemoveTicket(root, testEpicID, testTicketID); !errors.Is(err, ErrTicketNotFound) {
		t.Errorf("RemoveTicket absent = %v, want ErrTicketNotFound", err)
	}
}

// TestRemoveEpicCascadesTickets covers that removing an epic removes its nested tickets
// (they live inside it), while RemoveTicket leaves the epic intact.
func TestRemoveEpicCascadesTickets(t *testing.T) {
	root := t.TempDir()
	mustCreateEpic(t, root, Epic{ID: testEpicID, Title: "Epic"})
	mustCreateTicket(t, root, Ticket{ID: testTicketID, EpicID: testEpicID, Title: "T"})

	// RemoveTicket leaves the epic.
	if err := RemoveTicket(root, testEpicID, testTicketID); err != nil {
		t.Fatalf("RemoveTicket: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, testEpicID, "epic.md")); err != nil {
		t.Errorf("RemoveTicket should leave the epic intact: %v", err)
	}

	// RemoveEpic removes the whole folder.
	mustCreateTicket(t, root, Ticket{ID: testTicketID, EpicID: testEpicID, Title: "T"})
	if err := RemoveEpic(root, testEpicID); err != nil {
		t.Fatalf("RemoveEpic: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, testEpicID)); !os.IsNotExist(err) {
		t.Errorf("RemoveEpic should remove the whole epic folder")
	}
}

// TestMalformedArtifact covers the malformed sentinel.
func TestMalformedArtifact(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, testEpicID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "epic.md"), []byte("no frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEpic(root, testEpicID); !errors.Is(err, ErrMalformed) {
		t.Errorf("LoadEpic malformed = %v, want ErrMalformed", err)
	}
}

// TestPlanCreateIsNonWriting covers the plan/apply split (R6/R8).
func TestPlanCreateIsNonWriting(t *testing.T) {
	root := t.TempDir()
	plan, err := PlanCreateEpic(root, Epic{ID: testEpicID, Title: "Epic"})
	if err != nil {
		t.Fatalf("PlanCreateEpic: %v", err)
	}
	if _, err := os.Stat(plan.Path); !os.IsNotExist(err) {
		t.Errorf("PlanCreateEpic wrote a file")
	}
	if err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := readFile(t, plan.Path); got != plan.Content {
		t.Errorf("written content != planned content")
	}
}

// TestIDHelpers covers the id composition/parse helpers.
func TestIDHelpers(t *testing.T) {
	if EpicID("05", "sdd-backlog") != "epic-05-sdd-backlog" {
		t.Errorf("EpicID wrong: %q", EpicID("05", "sdd-backlog"))
	}
	if TicketID("05", "03", "x") != "ticket-05-03-x" {
		t.Errorf("TicketID wrong: %q", TicketID("05", "03", "x"))
	}
	if EpicNumberOf("epic-05-sdd-backlog") != "05" {
		t.Errorf("EpicNumberOf wrong: %q", EpicNumberOf("epic-05-sdd-backlog"))
	}
	if EpicNumberOf("not-an-epic") != "" {
		t.Errorf("EpicNumberOf of malformed should be empty")
	}
}
