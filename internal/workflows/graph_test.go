package workflows

import (
	"reflect"
	"strings"
	"testing"
)

// knownSet builds an agent-existence predicate from a fixed set of ids, the way
// the CLI injects the workspace's known agents into the core validator.
func knownSet(ids ...string) func(string) bool {
	set := make(map[string]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return func(a string) bool { return set[a] }
}

// validPipeline is a correct 3-phase SDD-style pipeline: a clean DAG where every
// input is the initial artifact or produced by a predecessor, and every agent is
// known. Reused as the baseline that must report valid.
func validPipeline() Workflow {
	return Workflow{
		Name: "sdd",
		Phases: []Phase{
			{ID: "spec", Agent: "analyst", Inputs: []string{"brief"}, Outputs: []string{"spec"}, Gate: "spec-gate", DependsOn: []string{"brief"}},
			{ID: "design", Agent: "architect", Inputs: []string{"spec"}, Outputs: []string{"design"}, Gate: "design-gate", DependsOn: []string{"spec"}},
			{ID: "plan", Agent: "planner", Inputs: []string{"spec", "design"}, Outputs: []string{"tickets"}, Gate: "plan-gate", DependsOn: []string{"design"}},
		},
	}
}

// TestValidWorkflowReportsValid covers CA4: a correct workflow is valid with zero
// findings.
func TestValidWorkflowReportsValid(t *testing.T) {
	w := validPipeline()
	rep := w.ValidateGraph(knownSet("analyst", "architect", "planner"))
	if !rep.Valid() {
		t.Fatalf("expected valid, got findings:\n%s", rep.Error())
	}
	if len(rep.Findings) != 0 {
		t.Errorf("valid workflow has %d findings, want 0", len(rep.Findings))
	}
}

// TestTransitiveArtifactAvailability documents the predecessor semantics: an input
// produced by a transitive (not direct) ancestor is available. `plan` depends only
// on `design`, but consumes `spec` produced by `spec` (an ancestor via design).
func TestTransitiveArtifactAvailability(t *testing.T) {
	w := validPipeline()
	rep := w.ValidateGraph(nil)
	if !rep.Valid() {
		t.Errorf("transitive ancestor output should be available; got:\n%s", rep.Error())
	}
}

// TestCycleDetected covers CA1: a dependency cycle is reported as invalid and the
// finding identifies the phases in the loop.
func TestCycleDetected(t *testing.T) {
	w := Workflow{
		Name: "cyclic",
		Phases: []Phase{
			{ID: "a", Agent: "x", Gate: "g", DependsOn: []string{"c"}},
			{ID: "b", Agent: "x", Gate: "g", DependsOn: []string{"a"}},
			{ID: "c", Agent: "x", Gate: "g", DependsOn: []string{"b"}},
		},
	}
	rep := w.ValidateGraph(knownSet("x"))
	if rep.Valid() {
		t.Fatal("expected invalid for a cyclic workflow")
	}
	var cycle *Finding
	for i := range rep.Findings {
		if rep.Findings[i].Kind == KindCycle {
			cycle = &rep.Findings[i]
			break
		}
	}
	if cycle == nil {
		t.Fatalf("no cycle finding; got:\n%s", rep.Error())
	}
	// The observed chain must mention every phase in the loop.
	for _, id := range []string{"a", "b", "c"} {
		if !strings.Contains(cycle.Observed, id) {
			t.Errorf("cycle chain %q missing phase %q", cycle.Observed, id)
		}
	}
}

// TestSelfCycleDetected covers a degenerate cycle (a phase depending on itself):
// it is a loop and must be reported without panicking.
func TestSelfCycleDetected(t *testing.T) {
	w := Workflow{
		Name:   "self",
		Phases: []Phase{{ID: "a", Agent: "x", Gate: "g", DependsOn: []string{"a"}}},
	}
	rep := w.ValidateGraph(knownSet("x"))
	if rep.Valid() {
		t.Fatal("expected invalid for a self-cycle")
	}
	found := false
	for _, f := range rep.Findings {
		if f.Kind == KindCycle {
			found = true
		}
	}
	if !found {
		t.Errorf("self-cycle not reported; got:\n%s", rep.Error())
	}
}

// TestCycleReportedOnce covers determinism of cycle reporting: one loop reachable
// from multiple entry phases is reported exactly once.
func TestCycleReportedOnce(t *testing.T) {
	w := Workflow{
		Name: "diamond-cycle",
		Phases: []Phase{
			// entry1 and entry2 both lead into the a<->b loop.
			{ID: "entry1", Agent: "x", Gate: "g", DependsOn: []string{"a"}},
			{ID: "entry2", Agent: "x", Gate: "g", DependsOn: []string{"a"}},
			{ID: "a", Agent: "x", Gate: "g", DependsOn: []string{"b"}},
			{ID: "b", Agent: "x", Gate: "g", DependsOn: []string{"a"}},
		},
	}
	rep := w.ValidateGraph(knownSet("x"))
	cycles := 0
	for _, f := range rep.Findings {
		if f.Kind == KindCycle {
			cycles++
		}
	}
	if cycles != 1 {
		t.Errorf("expected the a<->b loop reported once, got %d cycle findings:\n%s", cycles, rep.Error())
	}
}

// TestMissingArtifact covers CA2: a phase consuming an artifact no predecessor
// produces (and which is not the initial brief) is reported with phase + artifact.
func TestMissingArtifact(t *testing.T) {
	w := Workflow{
		Name: "missing",
		Phases: []Phase{
			{ID: "spec", Agent: "analyst", Inputs: []string{"brief"}, Outputs: []string{"spec"}, Gate: "g", DependsOn: []string{"brief"}},
			// design consumes "blueprint", which nobody produces.
			{ID: "design", Agent: "architect", Inputs: []string{"blueprint"}, Outputs: []string{"design"}, Gate: "g", DependsOn: []string{"spec"}},
		},
	}
	rep := w.ValidateGraph(knownSet("analyst", "architect"))
	if rep.Valid() {
		t.Fatal("expected invalid for a missing artifact")
	}
	var mf *Finding
	for i := range rep.Findings {
		if rep.Findings[i].Kind == KindMissingArtifact {
			mf = &rep.Findings[i]
			break
		}
	}
	if mf == nil {
		t.Fatalf("no missing-artifact finding; got:\n%s", rep.Error())
	}
	if mf.Phase != "design" || mf.Observed != "blueprint" {
		t.Errorf("missing-artifact finding = phase %q observed %q, want design/blueprint", mf.Phase, mf.Observed)
	}
}

// TestNonPredecessorOutputNotAvailable covers the predecessor semantics' negative
// side: an output produced by a phase that is NOT an ancestor is not available,
// even though it exists somewhere in the workflow.
func TestNonPredecessorOutputNotAvailable(t *testing.T) {
	w := Workflow{
		Name: "siblings",
		Phases: []Phase{
			{ID: "a", Agent: "x", Inputs: []string{"brief"}, Outputs: []string{"art-a"}, Gate: "g", DependsOn: []string{"brief"}},
			// b is a sibling of a (both depend on brief), so a's output is NOT a
			// predecessor output for b.
			{ID: "b", Agent: "x", Inputs: []string{"art-a"}, Outputs: []string{"art-b"}, Gate: "g", DependsOn: []string{"brief"}},
		},
	}
	rep := w.ValidateGraph(knownSet("x"))
	if rep.Valid() {
		t.Fatal("expected invalid: sibling output must not be available")
	}
	found := false
	for _, f := range rep.Findings {
		if f.Kind == KindMissingArtifact && f.Phase == "b" && f.Observed == "art-a" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected missing-artifact for b/art-a; got:\n%s", rep.Error())
	}
}

// TestUnknownAgent covers CA3: a phase referencing an agent absent from the
// injected known set is reported with phase + agent.
func TestUnknownAgent(t *testing.T) {
	w := Workflow{
		Name: "agents",
		Phases: []Phase{
			{ID: "spec", Agent: "analyst", Inputs: []string{"brief"}, Outputs: []string{"spec"}, Gate: "g", DependsOn: []string{"brief"}},
			{ID: "build", Agent: "ghost", Inputs: []string{"spec"}, Outputs: []string{"out"}, Gate: "g", DependsOn: []string{"spec"}},
		},
	}
	// "ghost" is not in the known set.
	rep := w.ValidateGraph(knownSet("analyst", "architect"))
	if rep.Valid() {
		t.Fatal("expected invalid for an unknown agent")
	}
	var uf *Finding
	for i := range rep.Findings {
		if rep.Findings[i].Kind == KindUnknownAgent {
			uf = &rep.Findings[i]
			break
		}
	}
	if uf == nil {
		t.Fatalf("no unknown-agent finding; got:\n%s", rep.Error())
	}
	if uf.Phase != "build" || uf.Observed != "ghost" {
		t.Errorf("unknown-agent finding = phase %q observed %q, want build/ghost", uf.Phase, uf.Observed)
	}
}

// TestNilPredicateSkipsAgentCheck covers the opt-out: a nil knownAgents predicate
// disables agent-existence checking entirely.
func TestNilPredicateSkipsAgentCheck(t *testing.T) {
	w := Workflow{
		Name:   "noagents",
		Phases: []Phase{{ID: "a", Agent: "whoever", Inputs: []string{"brief"}, Outputs: []string{"o"}, Gate: "g", DependsOn: []string{"brief"}}},
	}
	rep := w.ValidateGraph(nil)
	for _, f := range rep.Findings {
		if f.Kind == KindUnknownAgent {
			t.Errorf("nil predicate must not produce unknown-agent findings; got %+v", f)
		}
	}
}

// TestFindingsAreActionable covers CA5: each finding carries phase, kind, observed
// and a non-empty reason.
func TestFindingsAreActionable(t *testing.T) {
	w := Workflow{
		Name: "messy",
		Phases: []Phase{
			{ID: "a", Agent: "ghost", Inputs: []string{"nope"}, Outputs: []string{"o"}, Gate: "g", DependsOn: []string{"dangling"}},
		},
	}
	rep := w.ValidateGraph(knownSet("real"))
	if rep.Valid() {
		t.Fatal("expected invalid")
	}
	for _, f := range rep.Findings {
		if f.Phase == "" {
			t.Errorf("finding missing phase: %+v", f)
		}
		if f.Kind == "" {
			t.Errorf("finding missing kind: %+v", f)
		}
		if f.Observed == "" {
			t.Errorf("finding missing observed: %+v", f)
		}
		if strings.TrimSpace(f.Reason) == "" {
			t.Errorf("finding missing reason: %+v", f)
		}
		// The Error() line must be self-contained.
		if !strings.Contains(f.Error(), string(f.Kind)) {
			t.Errorf("finding Error() %q does not mention kind %q", f.Error(), f.Kind)
		}
	}
}

// TestDeterminism covers CA6: validating the same workflow twice yields identical
// reports and identical finding order.
func TestDeterminismGraph(t *testing.T) {
	w := Workflow{
		Name: "many",
		Phases: []Phase{
			{ID: "a", Agent: "ghost1", Inputs: []string{"missing1"}, Outputs: []string{"o-a"}, Gate: "g", DependsOn: []string{"brief"}},
			{ID: "b", Agent: "ghost2", Inputs: []string{"missing2"}, Outputs: []string{"o-b"}, Gate: "g", DependsOn: []string{"dangling"}},
			{ID: "c", Agent: "known", Inputs: []string{"o-a", "o-b"}, Outputs: []string{"o-c"}, Gate: "g", DependsOn: []string{"a", "b"}},
		},
	}
	known := knownSet("known")
	first := w.ValidateGraph(known)
	second := w.ValidateGraph(known)
	if !reflect.DeepEqual(first.Findings, second.Findings) {
		t.Errorf("validation not deterministic\nfirst:  %+v\nsecond: %+v", first.Findings, second.Findings)
	}
	if first.Error() != second.Error() {
		t.Errorf("rendered report not byte-stable")
	}
	// Findings must be ordered by phase position then kind: a's findings precede
	// b's precede c's.
	lastPos := -1
	posOf := map[string]int{"a": 0, "b": 1, "c": 2}
	for _, f := range first.Findings {
		p := posOf[f.Phase]
		if p < lastPos {
			t.Errorf("findings not ordered by phase position: %+v", first.Findings)
		}
		lastPos = p
	}
}

// TestDanglingDependencyReported covers the additive diagnostic and R8: a
// depends_on naming a non-existent phase is reported as unknown-dependency without
// panicking, and is not confused with the mandatory classes.
func TestDanglingDependencyReported(t *testing.T) {
	w := Workflow{
		Name:   "dangling",
		Phases: []Phase{{ID: "a", Agent: "x", Inputs: []string{"brief"}, Outputs: []string{"o"}, Gate: "g", DependsOn: []string{"ghost-phase"}}},
	}
	rep := w.ValidateGraph(knownSet("x"))
	found := false
	for _, f := range rep.Findings {
		if f.Kind == KindUnknownDependency && f.Observed == "ghost-phase" {
			found = true
		}
	}
	if !found {
		t.Errorf("dangling depends_on not reported; got:\n%s", rep.Error())
	}
}

// TestDegenerateInputsNoPanic covers CA7/R8: empty and trivially-shaped workflows
// are handled as defined cases without panicking, and are valid.
func TestDegenerateInputsNoPanic(t *testing.T) {
	cases := []struct {
		name string
		w    Workflow
	}{
		{"empty workflow", Workflow{Name: "empty"}},
		{"single phase no deps", Workflow{Name: "single", Phases: []Phase{{ID: "a", Agent: "x", Gate: "g"}}}},
		{"empty lists", Workflow{Name: "lists", Phases: []Phase{{ID: "a", Agent: "x", Gate: "g", Inputs: []string{}, Outputs: []string{}, DependsOn: []string{}}}}},
		{"phase consuming only brief", Workflow{Name: "brief", Phases: []Phase{{ID: "a", Agent: "x", Inputs: []string{"brief"}, Gate: "g", DependsOn: []string{"brief"}}}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rep := tc.w.ValidateGraph(knownSet("x")) // must not panic
			if !rep.Valid() {
				t.Errorf("%s should be valid; got:\n%s", tc.name, rep.Error())
			}
		})
	}
}

// TestMultipleClassesTogether covers a workflow exhibiting all three mandatory
// classes at once: every class is present in the report, proving they coexist.
func TestMultipleClassesTogether(t *testing.T) {
	w := Workflow{
		Name: "all",
		Phases: []Phase{
			{ID: "a", Agent: "known", Inputs: []string{"brief"}, Outputs: []string{"o-a"}, Gate: "g", DependsOn: []string{"brief"}},
			{ID: "b", Agent: "ghost", Inputs: []string{"missing"}, Outputs: []string{"o-b"}, Gate: "g", DependsOn: []string{"a"}},
			// c<->d cycle.
			{ID: "c", Agent: "known", Inputs: []string{"o-a"}, Outputs: []string{"o-c"}, Gate: "g", DependsOn: []string{"d", "a"}},
			{ID: "d", Agent: "known", Inputs: []string{"o-c"}, Outputs: []string{"o-d"}, Gate: "g", DependsOn: []string{"c"}},
		},
	}
	rep := w.ValidateGraph(knownSet("known"))
	kinds := map[FindingKind]bool{}
	for _, f := range rep.Findings {
		kinds[f.Kind] = true
	}
	for _, want := range []FindingKind{KindCycle, KindMissingArtifact, KindUnknownAgent} {
		if !kinds[want] {
			t.Errorf("expected a %q finding; got:\n%s", want, rep.Error())
		}
	}
}
