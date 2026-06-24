package workflows

import (
	"strings"
	"testing"
)

// builtinAgentIDs is the set of built-in catalog agent ids the factory workflow
// may reference. It is duplicated here (not imported from internal/catalog,
// which this package must not depend on) so the test injects the same known set
// the CLI would build from the built-ins. Keep in sync with catalog/builtin.go.
var builtinAgentIDs = map[string]bool{
	"analyst":    true,
	"architect":  true,
	"planner":    true,
	"validator":  true,
	"documenter": true,
}

func knownBuiltins(id string) bool { return builtinAgentIDs[id] }

// TestDefaultWorkflowLoadsRoundTrip covers CA2: the default model renders to YAML
// that loads back into an equivalent model (a clean round-trip), proving the
// factory content is valid canonical YAML for the 04-01 model.
func TestDefaultWorkflowLoadsRoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := Create(root, DefaultWorkflow()); err != nil {
		t.Fatalf("Create default: %v", err)
	}

	loaded, err := Load(root, DefaultWorkflowName)
	if err != nil {
		t.Fatalf("Load default: %v", err)
	}
	if Render(loaded) != Render(DefaultWorkflow()) {
		t.Errorf("default workflow does not round-trip\nloaded:\n%s\nmodel:\n%s",
			Render(loaded), Render(DefaultWorkflow()))
	}
}

// TestDefaultWorkflowPhasesInOrder covers CA3: the workflow contains the six
// pipeline phases chained by dependencies in pipeline order.
func TestDefaultWorkflowPhasesInOrder(t *testing.T) {
	w := DefaultWorkflow()
	wantOrder := []string{"spec", "architecture", "epics", "tickets", "validation", "docs"}
	if len(w.Phases) != len(wantOrder) {
		t.Fatalf("default has %d phases, want %d", len(w.Phases), len(wantOrder))
	}
	for i, want := range wantOrder {
		if w.Phases[i].ID != want {
			t.Errorf("phase[%d] = %q, want %q", i, w.Phases[i].ID, want)
		}
	}

	// depends_on must chain each phase to the one before it; spec is the root.
	wantDeps := map[string][]string{
		"spec":         {},
		"architecture": {"spec"},
		"epics":        {"architecture"},
		"tickets":      {"epics"},
		"validation":   {"tickets"},
		"docs":         {"validation"},
	}
	for _, p := range w.Phases {
		got := p.DependsOn
		want := wantDeps[p.ID]
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Errorf("phase %q depends_on = %v, want %v", p.ID, got, want)
		}
	}
}

// TestDefaultWorkflowAgents covers CA4: each phase references the correct built-in
// agent (planner appears twice).
func TestDefaultWorkflowAgents(t *testing.T) {
	wantAgent := map[string]string{
		"spec":         "analyst",
		"architecture": "architect",
		"epics":        "planner",
		"tickets":      "planner",
		"validation":   "validator",
		"docs":         "documenter",
	}
	for _, p := range DefaultWorkflow().Phases {
		if wantAgent[p.ID] != p.Agent {
			t.Errorf("phase %q agent = %q, want %q", p.ID, p.Agent, wantAgent[p.ID])
		}
		// Every referenced agent must be a built-in (the known set), or the
		// unknown-agent check would fail.
		if !builtinAgentIDs[p.Agent] {
			t.Errorf("phase %q references non-built-in agent %q", p.ID, p.Agent)
		}
	}
}

// TestDefaultWorkflowInputsOutputsGates covers CA5: every phase declares inputs,
// outputs and a gate, with the canonical artifact wiring.
func TestDefaultWorkflowInputsOutputsGates(t *testing.T) {
	type io struct{ in, out, gate string }
	want := map[string]io{
		"spec":         {"brief", "spec", "spec-gate"},
		"architecture": {"spec", "architecture", "architecture-gate"},
		"epics":        {"architecture", "epics", "epics-gate"},
		"tickets":      {"epics", "tickets", "tickets-gate"},
		"validation":   {"tickets", "validation", "validation-gate"},
		"docs":         {"validation", "docs", "docs-gate"},
	}
	for _, p := range DefaultWorkflow().Phases {
		w := want[p.ID]
		if len(p.Inputs) != 1 || p.Inputs[0] != w.in {
			t.Errorf("phase %q inputs = %v, want [%s]", p.ID, p.Inputs, w.in)
		}
		if len(p.Outputs) != 1 || p.Outputs[0] != w.out {
			t.Errorf("phase %q outputs = %v, want [%s]", p.ID, p.Outputs, w.out)
		}
		if p.Gate != w.gate {
			t.Errorf("phase %q gate = %q, want %q", p.ID, p.Gate, w.gate)
		}
	}
}

// TestDefaultWorkflowPassesSemanticValidation is the central test (CA6): the
// factory workflow validates clean against the 04-03 semantic validator with the
// built-in agents as the known set — no cycles, no missing artifacts, no unknown
// agents. This is the hard constraint the whole design hangs on.
func TestDefaultWorkflowPassesSemanticValidation(t *testing.T) {
	report := DefaultWorkflow().ValidateGraph(knownBuiltins)
	if !report.Valid() {
		t.Fatalf("default workflow must be semantically valid, got findings:\n%s", report.Error())
	}
	if len(report.Findings) != 0 {
		t.Errorf("default workflow has %d findings, want 0:\n%s", len(report.Findings), report.Error())
	}
}

// TestDefaultWorkflowAlsoPassesSchemaValidation covers structural validity too: the
// default must satisfy the 04-01 schema validator (unique kebab ids, non-empty
// agent/gate), not just the semantic one.
func TestDefaultWorkflowAlsoPassesSchemaValidation(t *testing.T) {
	if err := DefaultWorkflow().Validate(); err != nil {
		t.Errorf("default workflow fails schema validation: %v", err)
	}
}

// TestDefaultWorkflowDeterministicBytes covers CA7: rendering the default twice
// yields byte-identical output (golden-stable). The exact canonical bytes are
// asserted so a regression in shape/order is caught.
func TestDefaultWorkflowDeterministicBytes(t *testing.T) {
	first := Render(DefaultWorkflow())
	second := Render(DefaultWorkflow())
	if first != second {
		t.Errorf("default render is not deterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	const golden = "phases:\n" +
		"  - id: spec\n" +
		"    agent: analyst\n" +
		"    inputs: [brief]\n" +
		"    outputs: [spec]\n" +
		"    gate: spec-gate\n" +
		"    depends_on: []\n" +
		"  - id: architecture\n" +
		"    agent: architect\n" +
		"    inputs: [spec]\n" +
		"    outputs: [architecture]\n" +
		"    gate: architecture-gate\n" +
		"    depends_on: [spec]\n" +
		"  - id: epics\n" +
		"    agent: planner\n" +
		"    inputs: [architecture]\n" +
		"    outputs: [epics]\n" +
		"    gate: epics-gate\n" +
		"    depends_on: [architecture]\n" +
		"  - id: tickets\n" +
		"    agent: planner\n" +
		"    inputs: [epics]\n" +
		"    outputs: [tickets]\n" +
		"    gate: tickets-gate\n" +
		"    depends_on: [epics]\n" +
		"  - id: validation\n" +
		"    agent: validator\n" +
		"    inputs: [tickets]\n" +
		"    outputs: [validation]\n" +
		"    gate: validation-gate\n" +
		"    depends_on: [tickets]\n" +
		"  - id: docs\n" +
		"    agent: documenter\n" +
		"    inputs: [validation]\n" +
		"    outputs: [docs]\n" +
		"    gate: docs-gate\n" +
		"    depends_on: [validation]\n"
	if first != golden {
		t.Errorf("default render does not match golden bytes\ngot:\n%s\nwant:\n%s", first, golden)
	}
}
