package workflows

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// readFile reads a file under root, failing the test if it is absent.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// listFiles returns the base names of the regular files directly under dir.
func listFiles(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// snapshotDir captures the content of every file under dir keyed by name, so a
// test can prove byte-for-byte that unrelated files were left untouched.
func snapshotDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	snap := make(map[string]string)
	for _, name := range listFiles(t, dir) {
		snap[name] = readFile(t, filepath.Join(dir, name))
	}
	return snap
}

// sampleWorkflow is a representative 3-phase SDD pipeline with dependencies,
// reused across tests so the cases stay focused on the behavior under test.
func sampleWorkflow() Workflow {
	return Workflow{
		Name: "sdd",
		Phases: []Phase{
			{ID: "spec", Agent: "analyst", Inputs: []string{"brief"}, Outputs: []string{"spec"}, Gate: "spec-gate", DependsOn: []string{"brief"}},
			{ID: "design", Agent: "architect", Inputs: []string{"spec"}, Outputs: []string{"design"}, Gate: "design-gate", DependsOn: []string{"spec"}},
			{ID: "plan", Agent: "planner", Inputs: []string{"spec", "design"}, Outputs: []string{"tickets"}, Gate: "plan-gate", DependsOn: []string{"design"}},
		},
	}
}

// mustCreate creates a workflow and fails the test on error.
func mustCreate(t *testing.T, root string, w Workflow) {
	t.Helper()
	if err := Create(root, w); err != nil {
		t.Fatalf("Create(%q): %v", w.Name, err)
	}
}

// TestLoadValidWorkflow covers CA1: a workflow YAML with phases carrying
// { id, agent, inputs, outputs, gate, depends_on } loads into the model with
// every field recovered, including the list fields and dependencies.
func TestLoadValidWorkflow(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, sampleWorkflow())

	loaded, err := Load(root, "sdd")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Name != "sdd" {
		t.Errorf("Name = %q, want sdd", loaded.Name)
	}
	if len(loaded.Phases) != 3 {
		t.Fatalf("loaded %d phases, want 3", len(loaded.Phases))
	}
	want := sampleWorkflow()
	want.Name = "sdd"
	if !reflect.DeepEqual(loaded, want) {
		t.Errorf("loaded workflow mismatch\nwant: %+v\ngot:  %+v", want, loaded)
	}
}

// TestRoundTripNoLoss covers CA2: serializing a loaded workflow yields a YAML
// equivalent to what produced it, and re-rendering the loaded model is byte-stable.
func TestRoundTripNoLoss(t *testing.T) {
	root := t.TempDir()
	w := sampleWorkflow()
	mustCreate(t, root, w)

	onDisk := readFile(t, filepath.Join(root, "sdd.yaml"))

	loaded, err := Load(root, "sdd")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if Render(loaded) != onDisk {
		t.Errorf("round-trip not lossless\non disk:\n%s\nre-rendered:\n%s", onDisk, Render(loaded))
	}
	// The loaded model (minus the name, which is the file identity) equals the
	// original in-memory model: no field was lost or altered.
	w.Name = "sdd"
	if !reflect.DeepEqual(loaded, w) {
		t.Errorf("model not preserved across round-trip\nwant: %+v\ngot:  %+v", w, loaded)
	}
}

// TestDeterminism covers CA3: rendering the same model twice is byte-identical,
// and creating it into two clean roots yields identical files.
func TestDeterminism(t *testing.T) {
	w := sampleWorkflow()
	first := Render(w)
	second := Render(w)
	if first != second {
		t.Errorf("Render is not deterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	rootA, rootB := t.TempDir(), t.TempDir()
	mustCreate(t, rootA, w)
	mustCreate(t, rootB, w)
	if readFile(t, filepath.Join(rootA, "sdd.yaml")) != readFile(t, filepath.Join(rootB, "sdd.yaml")) {
		t.Errorf("two creates with identical input produced different files")
	}

	// Key order within a phase is fixed and the lists are flow style: assert the
	// exact serialized shape so a regression in ordering is caught.
	wantFirstPhase := "  - id: spec\n" +
		"    agent: analyst\n" +
		"    inputs: [brief]\n" +
		"    outputs: [spec]\n" +
		"    gate: spec-gate\n" +
		"    depends_on: [brief]\n"
	if !strings.HasPrefix(first, "phases:\n"+wantFirstPhase) {
		t.Errorf("unexpected canonical shape:\n%s", first)
	}
	if !strings.HasSuffix(first, "\n") {
		t.Errorf("output must end with a single trailing newline:\n%q", first)
	}
}

// TestEdgesRecoverable covers CA4/R5: the DAG edge set is recoverable from
// depends_on, deterministically and in phase order.
func TestEdgesRecoverable(t *testing.T) {
	w := sampleWorkflow()
	edges := w.Edges()
	want := []Edge{
		{From: "brief", To: "spec"},
		{From: "spec", To: "design"},
		{From: "design", To: "plan"},
	}
	if !reflect.DeepEqual(edges, want) {
		t.Errorf("Edges mismatch\nwant: %+v\ngot:  %+v", want, edges)
	}

	// A phase with multiple dependencies yields one edge per dependency, in order.
	multi := Workflow{Phases: []Phase{
		{ID: "merge", Agent: "a", Gate: "g", DependsOn: []string{"x", "y", "z"}},
	}}
	got := multi.Edges()
	wantMulti := []Edge{{From: "x", To: "merge"}, {From: "y", To: "merge"}, {From: "z", To: "merge"}}
	if !reflect.DeepEqual(got, wantMulti) {
		t.Errorf("multi-dep Edges mismatch\nwant: %+v\ngot:  %+v", wantMulti, got)
	}
}

// TestCreateAndPhaseEdits covers CA5/R6: create a workflow, then add/edit/remove
// phases through a persisted Edit, and confirm the result serializes to valid
// YAML that loads back equal.
func TestCreateAndPhaseEdits(t *testing.T) {
	root := t.TempDir()

	// Create an initially empty workflow, then build it up through edits.
	mustCreate(t, root, Workflow{Name: "build"})

	// Add a phase.
	_, err := Edit(root, "build", func(w *Workflow) error {
		return w.AddPhase(Phase{ID: "spec", Agent: "analyst", Inputs: []string{"brief"}, Outputs: []string{"spec"}, Gate: "spec-gate", DependsOn: []string{"brief"}})
	})
	if err != nil {
		t.Fatalf("AddPhase edit: %v", err)
	}
	// Add a second phase.
	_, err = Edit(root, "build", func(w *Workflow) error {
		return w.AddPhase(Phase{ID: "design", Agent: "architect", Inputs: []string{"spec"}, Outputs: []string{"design"}, Gate: "design-gate", DependsOn: []string{"spec"}})
	})
	if err != nil {
		t.Fatalf("AddPhase edit 2: %v", err)
	}

	// Edit the first phase (change its agent).
	_, err = Edit(root, "build", func(w *Workflow) error {
		return w.EditPhase("spec", Phase{ID: "spec", Agent: "researcher", Inputs: []string{"brief"}, Outputs: []string{"spec"}, Gate: "spec-gate", DependsOn: []string{"brief"}})
	})
	if err != nil {
		t.Fatalf("EditPhase edit: %v", err)
	}

	// Remove the second phase.
	_, err = Edit(root, "build", func(w *Workflow) error {
		return w.RemovePhase("design")
	})
	if err != nil {
		t.Fatalf("RemovePhase edit: %v", err)
	}

	loaded, err := Load(root, "build")
	if err != nil {
		t.Fatalf("Load after edits: %v", err)
	}
	if len(loaded.Phases) != 1 {
		t.Fatalf("after edits want 1 phase, got %d: %+v", len(loaded.Phases), loaded.Phases)
	}
	if loaded.Phases[0].Agent != "researcher" {
		t.Errorf("edited phase agent = %q, want researcher", loaded.Phases[0].Agent)
	}
}

// TestAddPhaseDuplicateRejected covers the in-memory uniqueness guard of R6.
func TestAddPhaseDuplicateRejected(t *testing.T) {
	w := Workflow{Phases: []Phase{{ID: "spec", Agent: "a", Gate: "g"}}}
	if err := w.AddPhase(Phase{ID: "spec", Agent: "b", Gate: "g"}); !errors.Is(err, ErrPhaseExists) {
		t.Errorf("AddPhase duplicate error = %v, want ErrPhaseExists", err)
	}
	if err := w.EditPhase("nope", Phase{ID: "nope", Agent: "a", Gate: "g"}); !errors.Is(err, ErrPhaseNotFound) {
		t.Errorf("EditPhase absent error = %v, want ErrPhaseNotFound", err)
	}
	if err := w.RemovePhase("nope"); !errors.Is(err, ErrPhaseNotFound) {
		t.Errorf("RemovePhase absent error = %v, want ErrPhaseNotFound", err)
	}
}

// TestMalformedYAML covers CA6 (malformed half): a YAML that does not match the
// canonical shape is rejected with ErrMalformedWorkflow and no panic.
func TestMalformedYAML(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{"no phases key", "agents: []\n"},
		{"phase missing gate", "phases:\n  - id: spec\n    agent: analyst\n    inputs: [brief]\n    outputs: [spec]\n    depends_on: [brief]\n"},
		{"unknown phase key", "phases:\n  - id: spec\n    agent: analyst\n    inputs: [brief]\n    outputs: [spec]\n    gate: g\n    depends_on: [brief]\n    extra: x\n"},
		{"list not bracketed", "phases:\n  - id: spec\n    agent: analyst\n    inputs: brief\n    outputs: [spec]\n    gate: g\n    depends_on: [brief]\n"},
		{"continuation without item", "phases:\n    agent: orphan\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, "broken.yaml"), []byte(tc.content), 0o644); err != nil {
				t.Fatal(err)
			}
			_, err := Load(root, "broken")
			if !errors.Is(err, ErrMalformedWorkflow) {
				t.Errorf("Load malformed (%s) error = %v, want ErrMalformedWorkflow", tc.name, err)
			}
		})
	}
}

// TestInvalidPhaseSchema covers CA6 (schema half): a phase that breaks the schema
// is rejected by Validate with an actionable *ValidationError naming the phase and
// field, in a single pass, with no panic. Also exercised through Create.
func TestInvalidPhaseSchema(t *testing.T) {
	w := Workflow{
		Name: "bad",
		Phases: []Phase{
			{ID: "Spec", Agent: "", Gate: "g"}, // bad id (uppercase) + empty agent
			{ID: "ok", Agent: "a", Gate: ""},   // empty gate
			{ID: "ok", Agent: "a", Gate: "g"},  // duplicate id
		},
	}
	err := w.Validate()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("Validate error = %v, want *ValidationError", err)
	}
	// Findings must name the offending field and be self-describing.
	msg := ve.Error()
	for _, want := range []string{"phase Spec", "id", "agent", "phase ok", "gate", "unique"} {
		if !strings.Contains(msg, want) {
			t.Errorf("validation message missing %q; got:\n%s", want, msg)
		}
	}

	// Create rejects it before any write.
	root := t.TempDir()
	if err := Create(root, w); !errors.As(err, &ve) {
		t.Fatalf("Create invalid error = %v, want *ValidationError", err)
	}
	if names := listFiles(t, root); len(names) != 0 {
		t.Errorf("invalid workflow created files: %v", names)
	}
}

// TestInvalidNameRejected covers the name rule for Create.
func TestInvalidNameRejected(t *testing.T) {
	for _, name := range []string{"", "MyFlow", "my flow", "-bad", "my_flow"} {
		root := t.TempDir()
		err := Create(root, Workflow{Name: name})
		if err == nil {
			t.Errorf("Create(name=%q) succeeded, want rejection", name)
		}
		if len(listFiles(t, root)) != 0 {
			t.Errorf("invalid name %q created files", name)
		}
	}
}

// TestDuplicateWorkflowFails covers non-destruction: creating a workflow with an
// existing name fails with ErrWorkflowExists and does not overwrite the original.
func TestDuplicateWorkflowFails(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, sampleWorkflow())
	original := readFile(t, filepath.Join(root, "sdd.yaml"))

	other := sampleWorkflow()
	other.Phases = other.Phases[:1]
	if err := Create(root, other); !errors.Is(err, ErrWorkflowExists) {
		t.Fatalf("duplicate create error = %v, want ErrWorkflowExists", err)
	}
	if got := readFile(t, filepath.Join(root, "sdd.yaml")); got != original {
		t.Errorf("original file overwritten by a duplicate create")
	}
}

// TestEditDoesNotDestroyOtherFiles covers non-destruction on edit: editing one
// workflow changes only its file; every other file stays byte-identical.
func TestEditDoesNotDestroyOtherFiles(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, sampleWorkflow())
	mustCreate(t, root, Workflow{Name: "other", Phases: []Phase{{ID: "x", Agent: "a", Gate: "g"}}})

	before := snapshotDir(t, root)

	_, err := Edit(root, "sdd", func(w *Workflow) error { return w.RemovePhase("plan") })
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}

	after := snapshotDir(t, root)
	if before["other.yaml"] != after["other.yaml"] {
		t.Errorf("editing sdd altered other.yaml")
	}
	if before["sdd.yaml"] == after["sdd.yaml"] {
		t.Errorf("edit did not change sdd.yaml")
	}
}

// TestEditRejectsInvalidResultIntact covers R4/R6: an edit that would leave the
// workflow invalid is rejected and the existing file is left intact.
func TestEditRejectsInvalidResultIntact(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, sampleWorkflow())
	before := readFile(t, filepath.Join(root, "sdd.yaml"))

	_, err := Edit(root, "sdd", func(w *Workflow) error {
		// Rename a phase to an invalid id; the result must fail validation.
		return w.EditPhase("spec", Phase{ID: "Spec!", Agent: "analyst", Gate: "g"})
	})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("Edit invalid result error = %v, want *ValidationError", err)
	}
	if got := readFile(t, filepath.Join(root, "sdd.yaml")); got != before {
		t.Errorf("rejected edit altered the file")
	}
}

// TestEmptyWorkflowRoundTrip covers the empty-phases shape: it serializes as
// `phases: []` and loads back to an empty (non-nil-equivalent) workflow.
func TestEmptyWorkflowRoundTrip(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Workflow{Name: "empty"})
	content := readFile(t, filepath.Join(root, "empty.yaml"))
	if content != "phases: []\n" {
		t.Errorf("empty workflow shape = %q, want %q", content, "phases: []\n")
	}
	loaded, err := Load(root, "empty")
	if err != nil {
		t.Fatalf("Load empty: %v", err)
	}
	if len(loaded.Phases) != 0 {
		t.Errorf("empty workflow loaded %d phases", len(loaded.Phases))
	}
}

// TestListAndRemove covers listing (sorted by name) and non-destructive remove.
func TestListAndRemove(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, sampleWorkflow())
	mustCreate(t, root, Workflow{Name: "alpha", Phases: []Phase{{ID: "x", Agent: "a", Gate: "g"}}})

	entries, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 || entries[0].Name != "alpha" || entries[1].Name != "sdd" {
		t.Fatalf("List not sorted by name: %+v", entries)
	}
	if entries[1].Phases != 3 {
		t.Errorf("sdd phase count = %d, want 3", entries[1].Phases)
	}

	if err := Remove(root, "alpha"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if names := listFiles(t, root); len(names) != 1 || names[0] != "sdd.yaml" {
		t.Errorf("Remove left wrong files: %v", names)
	}
	if err := Remove(root, "alpha"); !errors.Is(err, ErrWorkflowNotFound) {
		t.Errorf("Remove absent error = %v, want ErrWorkflowNotFound", err)
	}
}

// TestListEmptyWorkspace covers the well-defined empty case.
func TestListEmptyWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")
	entries, err := List(root)
	if err != nil {
		t.Fatalf("List on absent dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty list, got %+v", entries)
	}
}

// TestQuotedScalarRoundTrip covers the conservative quoting path: a value that
// needs quoting (contains a comma) round-trips through render and parse intact.
func TestQuotedScalarRoundTrip(t *testing.T) {
	w := Workflow{
		Name: "quoted",
		Phases: []Phase{
			{ID: "p", Agent: "a", Inputs: []string{"a,b", "plain"}, Outputs: []string{}, Gate: "g", DependsOn: []string{}},
		},
	}
	rendered := Render(w)
	if !strings.Contains(rendered, `inputs: ["a,b", plain]`) {
		t.Errorf("quoting wrong; got:\n%s", rendered)
	}
	parsed, err := parse(rendered)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !reflect.DeepEqual(parsed.Phases[0].Inputs, []string{"a,b", "plain"}) {
		t.Errorf("quoted list not recovered: %+v", parsed.Phases[0].Inputs)
	}
}

// TestNoBackendReferences covers CA7: the package source contains no reference to
// a concrete agent backend (Claude Code, etc.). It scans the package's own .go
// files (excluding tests) for forbidden tokens, so a future edit that couples the
// model to a backend is caught.
func TestNoBackendReferences(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read package dir: %v", err)
	}
	forbidden := []string{"claude", "anthropic", ".claude/"}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		content := strings.ToLower(readFile(t, name))
		for _, tok := range forbidden {
			if strings.Contains(content, tok) {
				t.Errorf("%s references a concrete backend token %q; the model must stay backend-agnostic (CA7)", name, tok)
			}
		}
	}
}
