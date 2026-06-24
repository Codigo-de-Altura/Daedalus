package tui

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

// sddDefaultFixture builds a 7-phase SDD-shaped workflow
// (brief→spec→architecture→epics→tickets→validation→docs) chained by depends_on.
// The sdd-default workflow itself does not exist yet (ticket 04-04 creates it), so
// the DAG view tests construct this fixture in-memory to exercise CA4/CA6 against a
// realistic moderate-size graph.
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

// openDAGFor drives the model from startup into the DAG screen showing the given
// workflow, as if the user had toggled to the workflows section, selected it and
// pressed enter, then the async load returned the workflow. It returns the model in
// the dagReady (or dagErrored) state.
func openDAGFor(t *testing.T, w workflows.Workflow) Model {
	t.Helper()
	m := sizedModel(t)
	// Populate the workflows section so the cursor has something to open.
	updated, _ := m.Update(workflowsLoadedMsg{entries: []workflows.Entry{
		{Name: w.Name, Phases: len(w.Phases)},
	}})
	m = updated.(Model)
	// Switch to the workflows section and open the selected workflow.
	updated, _ = m.Update(keyPress("tab"))
	m = updated.(Model)
	updated, _ = m.Update(keyPress("enter"))
	m = updated.(Model)
	// Deliver the async load result.
	updated, _ = m.Update(workflowLoadedMsg{name: w.Name, workflow: w})
	m = updated.(Model)
	return m
}

// TestSectionToggleAndOpenDAG verifies tab switches the list from prompts to
// workflows and enter on a workflow opens the DAG screen (R6, toggle requirement).
func TestSectionToggleAndOpenDAG(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(workflowsLoadedMsg{entries: []workflows.Entry{
		{Name: "sdd-default", Phases: 7},
	}})
	m = updated.(Model)

	if m.section != sectionPrompts {
		t.Fatal("model should start in the prompts section")
	}

	// tab moves to the workflows section; the title reflects it.
	updated, _ = m.Update(keyPress("tab"))
	m = updated.(Model)
	if m.section != sectionWorkflows {
		t.Fatal("tab should switch to the workflows section")
	}
	if !strings.Contains(m.View(), "Daedalus · Workflows") {
		t.Errorf("workflows section title missing, got:\n%s", m.View())
	}

	// enter opens the DAG view and starts the async load.
	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(Model)
	if m.screen != screenWorkflowDAG {
		t.Fatal("enter on a workflow should switch to the DAG screen")
	}
	if m.dagState != dagLoading {
		t.Error("DAG should be loading right after opening")
	}
	if m.dagName != "sdd-default" {
		t.Errorf("dagName = %q, want sdd-default", m.dagName)
	}
	if cmd == nil {
		t.Error("opening a DAG should return a load command")
	}
}

// TestSectionToggleBackToPrompts verifies tab on the list flips back to the prompts
// section, and that each section keeps its own cursor.
func TestSectionToggleBackToPrompts(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(keyPress("tab"))
	m = updated.(Model)
	if m.section != sectionWorkflows {
		t.Fatal("first tab should land on workflows")
	}
	updated, _ = m.Update(keyPress("tab"))
	m = updated.(Model)
	if m.section != sectionPrompts {
		t.Fatal("second tab should return to prompts")
	}
	if !strings.Contains(m.View(), "Daedalus · Prompts") {
		t.Errorf("prompts title missing after toggle back, got:\n%s", m.View())
	}
}

// TestDAGShowsPhaseIDsAndAgents covers CA1/CA2: after a workflow loads, the DAG
// view contains each phase id and its agent.
func TestDAGShowsPhaseIDsAndAgents(t *testing.T) {
	w := sddDefaultFixture()
	m := openDAGFor(t, w)

	if m.dagState != dagReady {
		t.Fatalf("DAG state = %v, want ready", m.dagState)
	}
	// The graph is loaded into a scrollable viewport whose View() only returns the
	// visible window; assert on the full rendered DAG content (the whole graph the
	// user can scroll through).
	content := m.dagViewportContent(w)
	for _, p := range w.Phases {
		if !strings.Contains(content, p.ID) {
			t.Errorf("DAG view missing phase id %q, got:\n%s", p.ID, content)
		}
		if !strings.Contains(content, p.Agent) {
			t.Errorf("DAG view missing agent %q for phase %q, got:\n%s", p.Agent, p.ID, content)
		}
	}
}

// TestDAGShowsEdgesInOrder covers CA3/CA4: dependencies render as connectors and
// the topological order places each predecessor before its successor.
func TestDAGShowsEdgesInOrder(t *testing.T) {
	w := sddDefaultFixture()
	m := openDAGFor(t, w)
	content := m.dagViewportContent(w)

	// An edge connector and direction marker must be present.
	if !strings.Contains(content, "↓") {
		t.Errorf("DAG view should draw directional edge connectors, got:\n%s", content)
	}
	if !strings.Contains(content, "after") {
		t.Errorf("DAG view should label edges with their predecessors, got:\n%s", content)
	}

	// The SDD pipeline is a linear chain, so every phase must appear before the one
	// that depends on it.
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

// TestTopoOrderLinearChain verifies the topological sort returns a linear chain in
// dependency order regardless of the declared order, and reports no cycle.
func TestTopoOrderLinearChain(t *testing.T) {
	// Declare the phases out of order to prove ordering is by dependency, not by
	// declaration.
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

// TestTopoOrderCycleFallsBack covers the R7/CA7 cycle guard: a cyclic graph must
// not loop forever; it falls back to declared order, renders every phase once and
// flags the cycle.
func TestTopoOrderCycleFallsBack(t *testing.T) {
	// a -> b -> a is a cycle; neither has indegree 0, so Kahn drains nothing.
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
	// Declared order preserved in the fallback.
	if order[0].ID != "a" || order[1].ID != "b" {
		t.Errorf("cycle fallback should keep declared order, got %v", []string{order[0].ID, order[1].ID})
	}
}

// TestDAGCycleRendersWithoutHang covers CA6/CA7 end-to-end: a cyclic workflow
// renders (with a warning) instead of hanging or panicking.
func TestDAGCycleRendersWithoutHang(t *testing.T) {
	w := workflows.Workflow{
		Name: "cyclic",
		Phases: []workflows.Phase{
			{ID: "a", Agent: "one", DependsOn: []string{"b"}},
			{ID: "b", Agent: "two", DependsOn: []string{"a"}},
		},
	}
	m := openDAGFor(t, w)
	if m.dagState != dagReady {
		t.Fatalf("DAG state = %v, want ready", m.dagState)
	}
	content := m.dagViewportContent(w)
	if !strings.Contains(content, "cycle") {
		t.Errorf("cyclic DAG should warn about the cycle, got:\n%s", content)
	}
	if !strings.Contains(content, "a") || !strings.Contains(content, "b") {
		t.Errorf("cyclic DAG should still render every phase, got:\n%s", content)
	}
}

// TestDAGReadOnly covers CA5: the DAG view honors no edit/execute binding. We send
// a battery of keys that would mutate or run something in an editor-style view and
// assert the loaded workflow identity is unchanged and the view never leaves
// read-only navigation (it either stays on the DAG screen or goes back to the list
// via esc — never into any edit/exec mode).
func TestDAGReadOnly(t *testing.T) {
	w := sddDefaultFixture()
	m := openDAGFor(t, w)
	before := m.dagName

	// Keys that in many TUIs add/delete/run; here they must be inert (scrolling at
	// most) and must never mutate the workflow identity or screen into an edit mode.
	for _, k := range []string{"a", "d", "x", "e", "r", "n", "i", "enter", "delete", " "} {
		updated, _ := m.Update(keyPress(k))
		m = updated.(Model)
		if m.screen != screenWorkflowDAG {
			t.Fatalf("key %q took the DAG view out of read-only navigation (screen=%v)", k, m.screen)
		}
		if m.dagName != before {
			t.Fatalf("key %q mutated the shown workflow identity", k)
		}
	}

	// q must not quit while reading; esc is the only way back.
	if _, cmd := m.Update(keyPress("q")); cmd != nil {
		t.Error("q inside the DAG view should be a no-op, not quit")
	}
	updated, _ := m.Update(keyPress("esc"))
	m = updated.(Model)
	if m.screen != screenList {
		t.Error("esc should return from the DAG view to the list")
	}
}

// TestDAGEmptyWorkflow covers CA7: a workflow with no phases renders a clear empty
// state, not a blank screen or a panic.
func TestDAGEmptyWorkflow(t *testing.T) {
	w := workflows.Workflow{Name: "empty"} // Phases nil
	m := openDAGFor(t, w)
	if m.dagState != dagReady {
		t.Fatalf("an empty but valid workflow should be ready, got %v", m.dagState)
	}
	content := m.dagViewportContent(w)
	if !strings.Contains(content, "empty") {
		t.Errorf("empty workflow should render an empty-state message, got:\n%s", content)
	}
}

// TestDAGLoadError covers CA7: a load failure drives the DAG view into a readable
// error state (typed messages for not-found and malformed) and never crashes.
func TestDAGLoadError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "not found",
			err:  fmt.Errorf("%w: %q", workflows.ErrWorkflowNotFound, "ghost"),
			want: "no longer exists",
		},
		{
			name: "malformed",
			err:  fmt.Errorf("%w: workflow %q: bad yaml", workflows.ErrMalformedWorkflow, "broken"),
			want: "malformed",
		},
		{
			name: "generic",
			err:  errors.New("permission denied"),
			want: "Cannot load",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := sizedModel(t)
			updated, _ := m.Update(workflowsLoadedMsg{entries: []workflows.Entry{
				{Name: "wf", Phases: 0},
			}})
			m = updated.(Model)
			updated, _ = m.Update(keyPress("tab"))
			m = updated.(Model)
			updated, _ = m.Update(keyPress("enter"))
			m = updated.(Model)

			updated, _ = m.Update(workflowLoadedMsg{name: "wf", err: tc.err})
			m = updated.(Model)

			if m.dagState != dagErrored {
				t.Fatalf("DAG state = %v, want errored", m.dagState)
			}
			if !strings.Contains(m.View(), tc.want) {
				t.Errorf("DAG error view missing %q, got:\n%s", tc.want, m.View())
			}
		})
	}
}

// TestWorkflowsEmptyState covers the workflows section empty state (R7/CA7): no
// workflow files renders a clear, actionable message rather than a blank list.
func TestWorkflowsEmptyState(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(workflowsLoadedMsg{entries: []workflows.Entry{}})
	m = updated.(Model)
	updated, _ = m.Update(keyPress("tab"))
	m = updated.(Model)

	view := m.View()
	if !strings.Contains(view, "No workflows found") {
		t.Errorf("workflows empty state missing, got:\n%s", view)
	}
	if !strings.Contains(view, "daedalus workflow create") {
		t.Errorf("workflows empty state should suggest creating one, got:\n%s", view)
	}
}

// TestWorkflowsListErrorRendered covers R7: a listing failure surfaces as a
// readable error instead of crashing.
func TestWorkflowsListErrorRendered(t *testing.T) {
	m := sizedModel(t)
	updated, _ := m.Update(workflowsLoadedMsg{err: errors.New("permission denied")})
	m = updated.(Model)
	updated, _ = m.Update(keyPress("tab"))
	m = updated.(Model)

	if !strings.Contains(m.View(), "Could not read workflows") {
		t.Errorf("workflows list error message missing, got:\n%s", m.View())
	}
}
