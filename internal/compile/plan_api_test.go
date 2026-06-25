package compile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// findPlanned returns the PlannedArtifact for relPath in a BackendPlan, failing
// the test if it is absent.
func findPlanned(t *testing.T, plan BackendPlan, relPath string) PlannedArtifact {
	t.Helper()
	for _, pa := range plan.Artifacts {
		if pa.RelPath == relPath {
			return pa
		}
	}
	t.Fatalf("planned artifact %q not found in plan for %q", relPath, plan.Backend)
	return PlannedArtifact{}
}

// TestPlanCarriesContentByStatus covers the 06-04 seam: Plan returns, per
// artifact, the Current/Target content the preview diffs — created ⇒ Current empty,
// updated ⇒ Current != Target (both populated), unchanged ⇒ Current == Target.
func TestPlanCarriesContentByStatus(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")   // will be UPDATED after we edit it
	addAgent(t, root, "architect") // will be UNCHANGED

	// First build so analyst/architect artifacts exist on disk.
	if _, err := Build(Options{Root: root}); err != nil {
		t.Fatalf("seed build: %v", err)
	}

	// Add a third agent that has NO artifact yet ⇒ CREATED at plan time.
	addAgent(t, root, "planner")

	// Edit the analyst's prompt so its artifact will differ ⇒ UPDATED.
	promptPath := filepath.Join(root, ".daedalus", "agents", "analyst", "prompt.md")
	if err := os.WriteFile(promptPath, []byte("A changed analyst prompt.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := Plan(Options{Root: root})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if len(plan.Backends) != 1 {
		t.Fatalf("backends = %d, want 1", len(plan.Backends))
	}
	bp := plan.Backends[0]

	// CREATED: planner artifact does not exist yet ⇒ Current empty, Target set,
	// status created.
	created := findPlanned(t, bp, ".claude/agents/planner.md")
	if created.Status != StatusCreated {
		t.Errorf("planner status = %s, want created", created.Status)
	}
	if created.Current != "" {
		t.Errorf("created artifact Current = %q, want empty", created.Current)
	}
	if created.Target == "" {
		t.Error("created artifact Target is empty")
	}

	// UPDATED: analyst differs ⇒ Current != Target, both non-empty.
	updated := findPlanned(t, bp, ".claude/agents/analyst.md")
	if updated.Status != StatusUpdated {
		t.Errorf("analyst status = %s, want updated", updated.Status)
	}
	if updated.Current == "" || updated.Target == "" {
		t.Errorf("updated artifact must carry both contents; current=%q target=%q",
			updated.Current, updated.Target)
	}
	if updated.Current == updated.Target {
		t.Error("updated artifact Current must differ from Target")
	}
	if !strings.Contains(updated.Target, "A changed analyst prompt.") {
		t.Errorf("updated Target does not reflect the edited prompt:\n%s", updated.Target)
	}

	// UNCHANGED: architect is identical ⇒ Current == Target.
	unchanged := findPlanned(t, bp, ".claude/agents/architect.md")
	if unchanged.Status != StatusUnchanged {
		t.Errorf("architect status = %s, want unchanged", unchanged.Status)
	}
	if unchanged.Current != unchanged.Target {
		t.Errorf("unchanged artifact Current must equal Target")
	}
}

// TestPlanWritesNothing covers the read-only guarantee: Plan never touches the
// filesystem, even when artifacts would be created/updated.
func TestPlanWritesNothing(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")

	if _, err := Plan(Options{Root: root}); err != nil {
		t.Fatalf("Plan: %v", err)
	}
	// No .claude/ must have been created by planning a fresh workspace.
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Errorf("Plan wrote to the filesystem (stat err=%v)", err)
	}
}

// TestPlanReportsOrphans covers the orphan seam at the Plan level: a previously
// generated artifact whose canonical source was removed is reported as an orphan
// and never deleted.
func TestPlanReportsOrphans(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")
	addAgent(t, root, "architect")
	if _, err := Build(Options{Root: root}); err != nil {
		t.Fatalf("seed build: %v", err)
	}
	if err := os.RemoveAll(filepath.Join(root, ".daedalus", "agents", "architect")); err != nil {
		t.Fatal(err)
	}

	plan, err := Plan(Options{Root: root})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if want := []string{".claude/agents/architect.md"}; !equalStrings(plan.Backends[0].Orphans, want) {
		t.Errorf("orphans = %v, want %v", plan.Backends[0].Orphans, want)
	}
	// Orphan still on disk (Plan never deletes).
	if _, err := os.Stat(filepath.Join(root, ".claude", "agents", "architect.md")); err != nil {
		t.Errorf("Plan deleted the orphan: %v", err)
	}
}

// TestPlanDeterministic covers determinism: planning the same state twice yields
// identical per-artifact statuses, contents and orphan lists.
func TestPlanDeterministic(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")
	addAgent(t, root, "architect")

	a, err := Plan(Options{Root: root})
	if err != nil {
		t.Fatalf("first plan: %v", err)
	}
	b, err := Plan(Options{Root: root})
	if err != nil {
		t.Fatalf("second plan: %v", err)
	}
	if len(a.Backends) != len(b.Backends) {
		t.Fatalf("backend count differs: %d vs %d", len(a.Backends), len(b.Backends))
	}
	for i := range a.Backends {
		pa, pb := a.Backends[i], b.Backends[i]
		if len(pa.Artifacts) != len(pb.Artifacts) {
			t.Fatalf("artifact count differs for %q", pa.Backend)
		}
		for j := range pa.Artifacts {
			if pa.Artifacts[j] != pb.Artifacts[j] {
				t.Errorf("artifact %d differs between plans:\n%+v\nvs\n%+v",
					j, pa.Artifacts[j], pb.Artifacts[j])
			}
		}
		if !equalStrings(pa.Orphans, pb.Orphans) {
			t.Errorf("orphans differ: %v vs %v", pa.Orphans, pb.Orphans)
		}
	}
}

// TestPlanFailsLikeBuildOnMissingWorkspace covers the shared front half: Plan
// aborts on a missing workspace exactly as Build does (no plan produced).
func TestPlanFailsLikeBuildOnMissingWorkspace(t *testing.T) {
	root := t.TempDir() // no .daedalus/
	_, err := Plan(Options{Root: root})
	if err == nil {
		t.Fatal("Plan over a missing workspace returned nil error")
	}
}

// TestPlanInvalidDefinitionAborts covers validate-before-anything in Plan: an
// invalid canonical definition aborts planning with a *DefinitionError.
func TestPlanInvalidDefinitionAborts(t *testing.T) {
	root := initWorkspace(t)
	agentDir := filepath.Join(root, workspace.Name, "agents", "broken")
	mustWrite(t, filepath.Join(agentDir, "agent.yaml"), "id: broken\n") // missing role
	mustWrite(t, filepath.Join(agentDir, "prompt.md"), "body\n")

	_, err := Plan(Options{Root: root})
	if !IsDefinitionInvalid(err) {
		t.Fatalf("err = %v, want a *DefinitionError", err)
	}
}
