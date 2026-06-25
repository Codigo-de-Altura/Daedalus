package compile

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// snapshotTree reads every file under dir and returns a path→content map (paths
// relative to dir, slash form). It is the basis for asserting two builds left the
// tree byte-identical (Check-1/Check-2/Check-3).
func snapshotTree(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(b)
		return nil
	})
	if err != nil {
		t.Fatalf("snapshot %s: %v", dir, err)
	}
	return out
}

// fileModTime returns the modification time of a file, failing the test if it is
// absent. Used to prove an unchanged artifact is not rewritten (no mtime churn).
func fileModTime(t *testing.T, path string) int64 {
	t.Helper()
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	return fi.ModTime().UnixNano()
}

// claudeRoot is the managed `.claude/` directory under a build root.
func claudeRoot(root string) string { return filepath.Join(root, ".claude") }

// TestIdempotentSecondRunNoChanges covers Check-1/Check-2/REQ-2: a second build
// over an unchanged definition reports every artifact as unchanged, writes
// nothing, and leaves the tree byte-identical.
func TestIdempotentSecondRunNoChanges(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")
	addAgent(t, root, "architect")

	if _, err := Build(Options{Root: root}); err != nil {
		t.Fatalf("first build: %v", err)
	}
	before := snapshotTree(t, claudeRoot(root))
	// Capture mtimes of the written artifacts to prove the second run does not
	// rewrite them (no spurious writes / mtime churn).
	mtimes := map[string]int64{}
	for rel := range before {
		mtimes[rel] = fileModTime(t, filepath.Join(claudeRoot(root), filepath.FromSlash(rel)))
	}

	out, err := Build(Options{Root: root})
	if err != nil {
		t.Fatalf("second build: %v", err)
	}

	// Every produced artifact must be classified unchanged; none created/updated.
	bo := out.Backends[0]
	if len(bo.Created) != 0 || len(bo.Updated) != 0 {
		t.Errorf("second run wrote files: created=%v updated=%v", bo.Created, bo.Updated)
	}
	if len(bo.Unchanged) != bo.Planned {
		t.Errorf("unchanged=%d, want all %d artifacts unchanged", len(bo.Unchanged), bo.Planned)
	}

	after := snapshotTree(t, claudeRoot(root))
	assertTreesEqual(t, before, after)
	for rel, m0 := range mtimes {
		if m1 := fileModTime(t, filepath.Join(claudeRoot(root), filepath.FromSlash(rel))); m1 != m0 {
			t.Errorf("artifact %s was rewritten (mtime changed) on an unchanged second run", rel)
		}
	}
}

// TestIdempotentManyRunsIdentical covers Check-2: three consecutive builds leave
// the tree identical each time.
func TestIdempotentManyRunsIdentical(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")

	var prev map[string]string
	for i := 0; i < 3; i++ {
		if _, err := Build(Options{Root: root}); err != nil {
			t.Fatalf("build %d: %v", i, err)
		}
		snap := snapshotTree(t, claudeRoot(root))
		if prev != nil {
			assertTreesEqual(t, prev, snap)
		}
		prev = snap
	}
}

// TestNonDestructionForeignFiles covers Check-6/Check-8/REQ-3: manual files the
// build does not produce — inside the managed `.claude/` tree and outside it —
// survive a re-compilation untouched, and are never deleted.
func TestNonDestructionForeignFiles(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")
	if _, err := Build(Options{Root: root}); err != nil {
		t.Fatalf("first build: %v", err)
	}

	// A user file INSIDE .claude/ that the build does not produce (e.g. a hand-made
	// command, or a settings the user added under a non-managed name).
	insideDir := filepath.Join(claudeRoot(root), "commands")
	if err := os.MkdirAll(insideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	insideFile := filepath.Join(insideDir, "my-manual-command.md")
	if err := os.WriteFile(insideFile, []byte("manual content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A user file OUTSIDE .claude/ entirely.
	outsideFile := filepath.Join(root, "NOTES.md")
	if err := os.WriteFile(outsideFile, []byte("my notes\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Build(Options{Root: root}); err != nil {
		t.Fatalf("second build: %v", err)
	}

	for _, f := range []struct {
		path, want string
	}{
		{insideFile, "manual content\n"},
		{outsideFile, "my notes\n"},
	} {
		got, err := os.ReadFile(f.path)
		if err != nil {
			t.Errorf("foreign file %s was removed: %v", f.path, err)
			continue
		}
		if string(got) != f.want {
			t.Errorf("foreign file %s was modified: got %q, want %q", f.path, string(got), f.want)
		}
	}
}

// TestBoundedRegeneration covers Check-7/REQ-6: changing one agent's canonical
// definition updates ONLY that agent's artifact; every other artifact stays
// unchanged and nothing outside the managed area is touched.
func TestBoundedRegeneration(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")
	addAgent(t, root, "architect")
	if _, err := Build(Options{Root: root}); err != nil {
		t.Fatalf("first build: %v", err)
	}
	before := snapshotTree(t, claudeRoot(root))

	// Edit the analyst's prompt on disk (a change to the canonical definition).
	promptPath := filepath.Join(root, ".daedalus", "agents", "analyst", "prompt.md")
	if err := os.WriteFile(promptPath, []byte("A deliberately changed prompt.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := Build(Options{Root: root})
	if err != nil {
		t.Fatalf("second build: %v", err)
	}

	bo := out.Backends[0]
	if want := []string{".claude/agents/analyst.md"}; !equalStrings(bo.Updated, want) {
		t.Errorf("updated = %v, want exactly %v", bo.Updated, want)
	}
	if len(bo.Created) != 0 {
		t.Errorf("nothing should be created on a bounded regeneration; got %v", bo.Created)
	}

	// Exactly one file differs from the snapshot: the analyst artifact.
	after := snapshotTree(t, claudeRoot(root))
	var changed []string
	for rel, content := range after {
		if before[rel] != content {
			changed = append(changed, rel)
		}
	}
	sort.Strings(changed)
	// snapshotTree is rooted at .claude/, so its keys are relative to that dir.
	if want := []string{"agents/analyst.md"}; !equalStrings(changed, want) {
		t.Errorf("changed files = %v, want exactly %v", changed, want)
	}
}

// TestOrphanDetectedNotDeleted covers the 06-04 seam and Check-8: an artifact the
// build produced before but whose canonical source was removed is detected as an
// orphan and reported, but left on disk (safe default — no auto-delete in 06-03).
func TestOrphanDetectedNotDeleted(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")
	addAgent(t, root, "architect")
	if _, err := Build(Options{Root: root}); err != nil {
		t.Fatalf("first build: %v", err)
	}
	orphanPath := filepath.Join(claudeRoot(root), "agents", "architect.md")
	if _, err := os.Stat(orphanPath); err != nil {
		t.Fatalf("precondition: architect artifact missing: %v", err)
	}

	// Remove the architect's canonical source so the next build no longer produces
	// its artifact; the previously generated file becomes an orphan.
	if err := os.RemoveAll(filepath.Join(root, ".daedalus", "agents", "architect")); err != nil {
		t.Fatal(err)
	}

	out, err := Build(Options{Root: root})
	if err != nil {
		t.Fatalf("rebuild: %v", err)
	}

	bo := out.Backends[0]
	if want := []string{".claude/agents/architect.md"}; !equalStrings(bo.Orphans, want) {
		t.Errorf("orphans = %v, want %v", bo.Orphans, want)
	}
	// The orphan must NOT be deleted (safe default).
	if _, err := os.Stat(orphanPath); err != nil {
		t.Errorf("orphan was deleted despite the safe default: %v", err)
	}
}

// assertTreesEqual fails if two path→content snapshots differ.
func assertTreesEqual(t *testing.T, a, b map[string]string) {
	t.Helper()
	if len(a) != len(b) {
		t.Errorf("tree size differs: %d vs %d files", len(a), len(b))
	}
	for rel, ca := range a {
		cb, ok := b[rel]
		if !ok {
			t.Errorf("file %s present before, missing after", rel)
			continue
		}
		if ca != cb {
			t.Errorf("file %s content changed:\n--- before ---\n%s\n--- after ---\n%s", rel, ca, cb)
		}
	}
	for rel := range b {
		if _, ok := a[rel]; !ok {
			t.Errorf("file %s appeared after but was not present before", rel)
		}
	}
}

// equalStrings reports whether two string slices are equal in order and content.
func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}
