package tui

import (
	"strings"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

// workflows_view_test.go covers the read-only DAG renderer (workflows_view.go),
// which the workflows area reconciles as a sub-screen. The renderer is pure (it
// turns a workflows.Workflow into a string), so these tests exercise it directly
// rather than through the navigation shell — the shell's wiring is covered in
// app_test.go / commands_test.go.

// renderModel returns a model usable purely for rendering (theme only); the DAG
// renderer needs no mutable navigation state.
func renderModel() Model {
	return Model{theme: defaultTheme()}
}

// sddDefaultFixture builds a 7-phase SDD-shaped workflow
// (brief→spec→architecture→epics→tickets→validation→docs) chained by depends_on.
func sddDefaultFixture() workflows.Workflow {
	return workflows.Workflow{
		Name: "sdd-default",
		Phases: []workflows.Phase{
			{ID: "brief", Agent: "analyst", Outputs: []string{"brief.md"}, Gate: "brief-approved"},
			{ID: "spec", Agent: "analyst", Inputs: []string{"brief.md"}, Outputs: []string{"spec.md"}, Gate: "spec-approved", DependsOn: []string{"brief"}},
			{ID: "architecture", Agent: "architect", Inputs: []string{"spec.md"}, Outputs: []string{"architecture.md"}, Gate: "arch-approved", DependsOn: []string{"spec"}},
			{ID: "epics", Agent: "planner", Inputs: []string{"architecture.md"}, Outputs: []string{"epics.md"}, Gate: "epics-approved", DependsOn: []string{"architecture"}},
			{ID: "tickets", Agent: "planner", Inputs: []string{"epics.md"}, Outputs: []string{"tickets.md"}, Gate: "tickets-approved", DependsOn: []string{"epics"}},
			{ID: "validation", Agent: "validator", Inputs: []string{"tickets.md"}, Outputs: []string{"report.md"}, Gate: "validated", DependsOn: []string{"tickets"}},
			{ID: "docs", Agent: "documenter", Inputs: []string{"report.md"}, Outputs: []string{"manual.md"}, Gate: "docs-approved", DependsOn: []string{"validation"}},
		},
	}
}

// TestDAGShowsPhaseIDsAndAgents: the rendered DAG contains each phase id and agent.
func TestDAGShowsPhaseIDsAndAgents(t *testing.T) {
	w := sddDefaultFixture()
	content := renderModel().dagViewportContent(w)
	for _, p := range w.Phases {
		if !strings.Contains(content, p.ID) {
			t.Errorf("DAG view missing phase id %q, got:\n%s", p.ID, content)
		}
		if !strings.Contains(content, p.Agent) {
			t.Errorf("DAG view missing agent %q for phase %q, got:\n%s", p.Agent, p.ID, content)
		}
	}
}

// TestDAGShowsEdgesInOrder: dependencies render as connectors and the topological
// order places each predecessor before its successor.
func TestDAGShowsEdgesInOrder(t *testing.T) {
	w := sddDefaultFixture()
	content := renderModel().dagViewportContent(w)

	if !strings.Contains(content, "↓") {
		t.Errorf("DAG view should draw directional edge connectors, got:\n%s", content)
	}
	if !strings.Contains(content, "after") {
		t.Errorf("DAG view should label edges with their predecessors, got:\n%s", content)
	}

	chain := []string{"brief", "spec", "architecture", "epics", "tickets", "validation", "docs"}
	last := -1
	for _, id := range chain {
		idx := strings.Index(content, id)
		if idx < 0 {
			t.Fatalf("phase %q missing from DAG, got:\n%s", id, content)
		}
		if idx <= last {
			t.Errorf("phase %q rendered out of topological order (at %d, previous at %d)", id, idx, last)
		}
		last = idx
	}
}

// TestTopoOrderLinearChain: the topological sort returns a linear chain in
// dependency order regardless of declared order, and reports no cycle.
func TestTopoOrderLinearChain(t *testing.T) {
	w := workflows.Workflow{Phases: []workflows.Phase{
		{ID: "c", DependsOn: []string{"b"}},
		{ID: "a"},
		{ID: "b", DependsOn: []string{"a"}},
	}}
	order, cyclic := topoOrder(w)
	if cyclic {
		t.Fatal("a valid DAG should not be reported cyclic")
	}
	got := []string{order[0].ID, order[1].ID, order[2].ID}
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("topo order = %v, want %v", got, want)
		}
	}
}

// TestTopoOrderCycleFallsBack: a cyclic graph must not loop forever; it falls back
// to declared order, renders every phase once and flags the cycle.
func TestTopoOrderCycleFallsBack(t *testing.T) {
	w := workflows.Workflow{Phases: []workflows.Phase{
		{ID: "a", DependsOn: []string{"b"}},
		{ID: "b", DependsOn: []string{"a"}},
	}}
	order, cyclic := topoOrder(w)
	if !cyclic {
		t.Fatal("a cyclic graph should be reported cyclic")
	}
	if len(order) != 2 {
		t.Fatalf("every phase should still be rendered once, got %d", len(order))
	}
	if order[0].ID != "a" || order[1].ID != "b" {
		t.Errorf("cycle fallback should keep declared order, got %v", []string{order[0].ID, order[1].ID})
	}
}

// TestDAGCycleRendersWithoutHang: a cyclic workflow renders (with a warning)
// instead of hanging or panicking.
func TestDAGCycleRendersWithoutHang(t *testing.T) {
	w := workflows.Workflow{
		Name: "cyclic",
		Phases: []workflows.Phase{
			{ID: "a", Agent: "one", DependsOn: []string{"b"}},
			{ID: "b", Agent: "two", DependsOn: []string{"a"}},
		},
	}
	content := renderModel().dagViewportContent(w)
	if !strings.Contains(content, "cycle") {
		t.Errorf("cyclic DAG should warn about the cycle, got:\n%s", content)
	}
	if !strings.Contains(content, "a") || !strings.Contains(content, "b") {
		t.Errorf("cyclic DAG should still render every phase, got:\n%s", content)
	}
}

// TestDAGEmptyWorkflow: a workflow with no phases renders a clear empty state, not
// a blank screen or a panic.
func TestDAGEmptyWorkflow(t *testing.T) {
	w := workflows.Workflow{Name: "empty"} // Phases nil
	content := renderModel().dagViewportContent(w)
	if !strings.Contains(content, "empty") {
		t.Errorf("empty workflow should render an empty-state message, got:\n%s", content)
	}
}
