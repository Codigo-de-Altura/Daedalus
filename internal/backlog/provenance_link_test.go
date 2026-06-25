package backlog_test

// This test is the build-time guard that lets internal/backlog stay self-contained (it
// does not import internal/workflows) while guaranteeing its provenance constants can
// never silently drift from the real factory workflow. The package duplicates the
// planner/sdd-default/epics|tickets anchor as constants (backlog.go); here — in a test,
// where importing workflows is free — we pin them to the actual phases of
// workflows.DefaultWorkflow. If anyone renames a phase, the agent, or the workflow in
// internal/workflows, this test fails and forces the backlog constants to be updated in
// lockstep (R5/CA5, R7/CA7).

import (
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/backlog"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

func TestProvenanceConstantsMatchDefaultWorkflow(t *testing.T) {
	if backlog.DefaultWorkflowName != workflows.DefaultWorkflowName {
		t.Errorf("backlog.DefaultWorkflowName = %q, want workflows.DefaultWorkflowName = %q",
			backlog.DefaultWorkflowName, workflows.DefaultWorkflowName)
	}
	if backlog.PhaseEpics != workflows.DefaultPhaseEpics {
		t.Errorf("backlog.PhaseEpics = %q, want workflows.DefaultPhaseEpics = %q",
			backlog.PhaseEpics, workflows.DefaultPhaseEpics)
	}
	if backlog.PhaseTickets != workflows.DefaultPhaseTickets {
		t.Errorf("backlog.PhaseTickets = %q, want workflows.DefaultPhaseTickets = %q",
			backlog.PhaseTickets, workflows.DefaultPhaseTickets)
	}

	wf := workflows.DefaultWorkflow()
	// Both planner steps must name the planner agent, so the provenance backlog records
	// points at the real steps (init.md §6, PRD RF-5.3), not invented ones.
	for _, phaseID := range []string{workflows.DefaultPhaseEpics, workflows.DefaultPhaseTickets} {
		idx := wf.PhaseIndex(phaseID)
		if idx < 0 {
			t.Fatalf("default workflow has no %q phase", phaseID)
		}
		if got := wf.Phases[idx].Agent; got != backlog.PlannerAgent {
			t.Errorf("default workflow %q phase agent = %q, but backlog.PlannerAgent = %q",
				phaseID, got, backlog.PlannerAgent)
		}
	}
}
