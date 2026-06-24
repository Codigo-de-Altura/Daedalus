package specs_test

// This test is the build-time guard that lets internal/specs stay self-contained
// (it does not import internal/workflows) while guaranteeing its provenance
// constants can never silently drift from the real factory workflow. The package
// duplicates the analyst/sdd-default/spec anchor as constants (spec.go); here — in a
// test, where importing workflows is free — we pin them to the actual phase of
// workflows.DefaultWorkflow. If anyone renames the phase, the agent, or the workflow
// in internal/workflows, this test fails and forces the specs constants to be
// updated in lockstep (R2/CA2, R8/CA7).

import (
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/specs"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

func TestProvenanceConstantsMatchDefaultWorkflow(t *testing.T) {
	if specs.DefaultWorkflowName != workflows.DefaultWorkflowName {
		t.Errorf("specs.DefaultWorkflowName = %q, want workflows.DefaultWorkflowName = %q",
			specs.DefaultWorkflowName, workflows.DefaultWorkflowName)
	}
	if specs.DefaultPhase != workflows.DefaultPhaseSpec {
		t.Errorf("specs.DefaultPhase = %q, want workflows.DefaultPhaseSpec = %q",
			specs.DefaultPhase, workflows.DefaultPhaseSpec)
	}

	// The analyst agent must be the agent of the spec phase in the actual default
	// workflow, so the brief -> spec link the specs package records points at the
	// real step (init.md §6, PRD RF-5.1), not an invented one.
	wf := workflows.DefaultWorkflow()
	idx := wf.PhaseIndex(workflows.DefaultPhaseSpec)
	if idx < 0 {
		t.Fatalf("default workflow has no %q phase", workflows.DefaultPhaseSpec)
	}
	specPhase := wf.Phases[idx]
	if specPhase.Agent != specs.AnalystAgent {
		t.Errorf("default workflow spec phase agent = %q, but specs.AnalystAgent = %q",
			specPhase.Agent, specs.AnalystAgent)
	}

	// The brief must be the spec phase's input, anchoring the brief -> spec wiring.
	hasBriefInput := false
	for _, in := range specPhase.Inputs {
		if in == "brief" {
			hasBriefInput = true
		}
	}
	if !hasBriefInput {
		t.Errorf("default workflow spec phase inputs = %v, expected to include \"brief\"", specPhase.Inputs)
	}
}
