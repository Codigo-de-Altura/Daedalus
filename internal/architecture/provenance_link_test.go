package architecture_test

// This test is the build-time guard that lets internal/architecture stay
// self-contained (it does not import internal/workflows) while guaranteeing its
// provenance constants can never silently drift from the real factory workflow. The
// package duplicates the architect/sdd-default/architecture anchor as constants
// (architecture.go); here — in a test, where importing workflows is free — we pin them
// to the actual phase of workflows.DefaultWorkflow. If anyone renames the phase, the
// agent, or the workflow in internal/workflows, this test fails and forces the
// architecture constants to be updated in lockstep (R3/CA3).

import (
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/architecture"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

func TestProvenanceConstantsMatchDefaultWorkflow(t *testing.T) {
	if architecture.DefaultWorkflowName != workflows.DefaultWorkflowName {
		t.Errorf("architecture.DefaultWorkflowName = %q, want workflows.DefaultWorkflowName = %q",
			architecture.DefaultWorkflowName, workflows.DefaultWorkflowName)
	}
	if architecture.DefaultPhase != workflows.DefaultPhaseArchitecture {
		t.Errorf("architecture.DefaultPhase = %q, want workflows.DefaultPhaseArchitecture = %q",
			architecture.DefaultPhase, workflows.DefaultPhaseArchitecture)
	}

	// The architect agent must be the agent of the architecture phase in the actual
	// default workflow, so the spec -> architecture link the package records points at
	// the real step (init.md §6, PRD RF-5.2), not an invented one.
	wf := workflows.DefaultWorkflow()
	idx := wf.PhaseIndex(workflows.DefaultPhaseArchitecture)
	if idx < 0 {
		t.Fatalf("default workflow has no %q phase", workflows.DefaultPhaseArchitecture)
	}
	archPhase := wf.Phases[idx]
	if archPhase.Agent != architecture.ArchitectAgent {
		t.Errorf("default workflow architecture phase agent = %q, but architecture.ArchitectAgent = %q",
			archPhase.Agent, architecture.ArchitectAgent)
	}

	// The spec must be the architecture phase's input, anchoring the spec ->
	// architecture wiring.
	hasSpecInput := false
	for _, in := range archPhase.Inputs {
		if in == "spec" {
			hasSpecInput = true
		}
	}
	if !hasSpecInput {
		t.Errorf("default workflow architecture phase inputs = %v, expected to include \"spec\"", archPhase.Inputs)
	}
}
