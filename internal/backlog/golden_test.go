package backlog

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden rewrites golden files from current output when set. Mirrors the
// adapter golden harness in internal/compile (RF-8.2 / RNF-5).
var updateGolden = flag.Bool("update", false, "rewrite golden files from current output")

// goldenEpic and goldenTicket are fixed canonical artifacts exercising the
// renderer surface: origin links present (so the planner-step provenance block,
// including the fixed `generated: false` boolean, is emitted), a non-empty
// depends_on list in flow style, and a multi-line body. No volatile data is
// embedded, so the rendered bytes are reproducible.
func goldenEpic() Epic {
	return Epic{
		ID:              "epic-05-sdd-backlog",
		Title:           "SDD Backlog",
		Status:          StatusTodo,
		Priority:        PriorityMedium,
		SpecRef:         "sdd-backlog",
		ArchitectureRef: "sdd-backlog-arch",
		DependsOn:       []string{"epic-04-workflows"},
		Body:            "The epic objective.\nSecond line.",
	}
}

func goldenTicket() Ticket {
	return Ticket{
		ID:              "ticket-05-03-epics-tickets-management",
		EpicID:          "epic-05-sdd-backlog",
		Title:           "Epics & Tickets Management",
		Status:          StatusTodo,
		Priority:        PriorityHigh,
		SpecRef:         "sdd-backlog",
		ArchitectureRef: "sdd-backlog-arch",
		DependsOn:       []string{"ticket-05-02-architecture-docs"},
		Body:            "The ticket feature.\nSecond line.",
	}
}

// TestBacklogGolden renders the canonical epic and ticket and asserts each
// matches its golden file byte-for-byte. Run with -update to regenerate.
func TestBacklogGolden(t *testing.T) {
	assertGolden(t, "epic.md", []byte(RenderEpic(goldenEpic())))
	assertGolden(t, "ticket.md", []byte(RenderTicket(goldenTicket())))
}

// TestBacklogRenderDeterministic covers RF-8.2 / RNF-5: rendering the same epic
// and ticket twice yields identical bytes (no volatile fields).
func TestBacklogRenderDeterministic(t *testing.T) {
	if RenderEpic(goldenEpic()) != RenderEpic(goldenEpic()) {
		t.Error("RenderEpic is not deterministic")
	}
	if RenderTicket(goldenTicket()) != RenderTicket(goldenTicket()) {
		t.Error("RenderTicket is not deterministic")
	}
}

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	goldenPath := filepath.Join("testdata", "golden", filepath.FromSlash(name))

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", goldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to generate)", goldenPath, err)
	}
	if string(want) != string(got) {
		t.Errorf("artifact %s does not match golden:\n--- got ---\n%s\n--- want ---\n%s",
			name, got, want)
	}
}
