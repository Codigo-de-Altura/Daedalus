package workflows

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden rewrites golden files from current output when set. Mirrors the
// adapter golden harness in internal/compile (RF-8.2 / RNF-5).
var updateGolden = flag.Bool("update", false, "rewrite golden files from current output")

// goldenWorkflow is a fixed canonical workflow exercising the renderer surface:
// multiple phases in authored order, non-empty and empty list-valued keys, and
// the flow-style list rendering. Its rendered bytes are reproducible.
func goldenWorkflow() Workflow {
	return Workflow{
		Phases: []Phase{
			{ID: "spec", Agent: "analyst", Inputs: []string{"brief"}, Outputs: []string{"spec"}, Gate: "spec-gate", DependsOn: []string{"brief"}},
			{ID: "architecture", Agent: "architect", Inputs: []string{"spec"}, Outputs: []string{"architecture"}, Gate: "architecture-gate", DependsOn: []string{"spec"}},
		},
	}
}

// TestWorkflowsGolden renders the canonical workflow and asserts it matches its
// golden file byte-for-byte. Run with -update to regenerate.
func TestWorkflowsGolden(t *testing.T) {
	assertGolden(t, "sdd-default.yaml", []byte(Render(goldenWorkflow())))
}

// TestWorkflowsRenderDeterministic covers RF-8.2 / RNF-5: rendering the same
// workflow twice yields identical bytes.
func TestWorkflowsRenderDeterministic(t *testing.T) {
	if Render(goldenWorkflow()) != Render(goldenWorkflow()) {
		t.Error("Render is not deterministic")
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
