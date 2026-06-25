package compile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// TestPlanArtifactsClassifies covers the pure 06-04 seam (PlanArtifacts): it
// classifies each produced artifact as created (absent), unchanged (byte-identical)
// or updated (present and different) WITHOUT writing, and detects orphans.
func TestPlanArtifactsClassifies(t *testing.T) {
	root := t.TempDir()
	arts := Artifacts{
		Backend: workspace.DefaultBackend,
		Files: []Artifact{
			{RelPath: ".claude/agents/new.md", Content: "new\n"},
			{RelPath: ".claude/agents/same.md", Content: "same\n"},
			{RelPath: ".claude/agents/diff.md", Content: "target\n"},
		},
	}

	// Pre-create "same" (identical) and "diff" (different), plus an orphan file in
	// the managed directory the produced set does not include.
	mustWrite(t, filepath.Join(root, ".claude", "agents", "same.md"), "same\n")
	mustWrite(t, filepath.Join(root, ".claude", "agents", "diff.md"), "stale\n")
	mustWrite(t, filepath.Join(root, ".claude", "agents", "orphan.md"), "orphan\n")

	plan, err := PlanArtifacts(root, arts)
	if err != nil {
		t.Fatalf("PlanArtifacts: %v", err)
	}

	wantStatus := map[string]ArtifactStatus{
		".claude/agents/new.md":  StatusCreated,
		".claude/agents/same.md": StatusUnchanged,
		".claude/agents/diff.md": StatusUpdated,
	}
	for _, pa := range plan.Artifacts {
		if want := wantStatus[pa.RelPath]; pa.Status != want {
			t.Errorf("%s status = %s, want %s", pa.RelPath, pa.Status, want)
		}
	}
	if want := []string{".claude/agents/orphan.md"}; !equalStrings(plan.Orphans, want) {
		t.Errorf("orphans = %v, want %v", plan.Orphans, want)
	}

	// PlanArtifacts must not write: the orphan and the others keep their content,
	// and "new" was never created.
	if _, err := os.Stat(filepath.Join(root, ".claude", "agents", "new.md")); !os.IsNotExist(err) {
		t.Errorf("PlanArtifacts created a file (err=%v)", err)
	}
	if got := readFile(t, filepath.Join(root, ".claude", "agents", "diff.md")); got != "stale\n" {
		t.Errorf("PlanArtifacts modified an existing file: %q", got)
	}
}

// TestDeterministicAcrossCleanDirs covers Check-3/REQ-4: building the same
// workspace into two independent clean target dirs yields byte-identical `.claude/`
// trees. We build the same canonical workspace twice by copying agents into two
// roots and comparing outputs.
func TestDeterministicAcrossCleanDirs(t *testing.T) {
	build := func() map[string]string {
		root := initWorkspace(t)
		addAgent(t, root, "analyst")
		addAgent(t, root, "architect")
		if _, err := Build(Options{Root: root}); err != nil {
			t.Fatalf("build: %v", err)
		}
		return snapshotTree(t, claudeRoot(root))
	}
	a := build()
	b := build()
	assertTreesEqual(t, a, b)
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
