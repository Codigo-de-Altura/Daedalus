package workspace

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestCreateScaffoldsFullTree verifies every canonical subdirectory and root
// artifact is created on a fresh target.
func TestCreateScaffoldsFullTree(t *testing.T) {
	root := t.TempDir()

	res, err := Create(root)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.AlreadyExisted {
		t.Errorf("AlreadyExisted = true, want false on a fresh root")
	}

	for _, sub := range Subdirs {
		path := filepath.Join(root, Name, sub)
		if info, err := os.Stat(path); err != nil || !info.IsDir() {
			t.Errorf("expected subdirectory %q (stat err=%v)", sub, err)
		}
	}
	for _, name := range RootArtifacts {
		if _, err := os.Stat(filepath.Join(root, Name, name)); err != nil {
			t.Errorf("expected root artifact %q: %v", name, err)
		}
	}
}

// TestCreateIsNonDestructive verifies files outside the workspace are untouched
// and existing artifacts are not truncated.
func TestCreateIsNonDestructive(t *testing.T) {
	root := t.TempDir()

	outside := filepath.Join(root, "README.md")
	if err := os.WriteFile(outside, []byte("keep me"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(root); err != nil {
		t.Fatal(err)
	}

	// Give an artifact content, then re-run: it must survive verbatim.
	artifact := filepath.Join(root, Name, "daedalus.yaml")
	if err := os.WriteFile(artifact, []byte("name: demo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(root); err != nil {
		t.Fatal(err)
	}

	if b, _ := os.ReadFile(outside); string(b) != "keep me" {
		t.Errorf("file outside the workspace was modified: %q", b)
	}
	if b, _ := os.ReadFile(artifact); string(b) != "name: demo" {
		t.Errorf("existing artifact was overwritten: %q", b)
	}
}

// TestCreateIsDeterministic verifies two runs over identical fresh inputs yield
// identical recursive listings.
func TestCreateIsDeterministic(t *testing.T) {
	listing := func() []string {
		root := t.TempDir()
		if _, err := Create(root); err != nil {
			t.Fatal(err)
		}
		base := filepath.Join(root, Name)
		var paths []string
		err := filepath.WalkDir(base, func(path string, _ fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			paths = append(paths, filepath.ToSlash(rel))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		sort.Strings(paths)
		return paths
	}

	a, b := listing(), listing()
	if len(a) != len(b) {
		t.Fatalf("listings differ in length: %v vs %v", a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("listing mismatch at %d: %q vs %q", i, a[i], b[i])
		}
	}
}

// TestCreateReportsAlreadyExisted verifies a second run flags the pre-existing
// workspace instead of failing.
func TestCreateReportsAlreadyExisted(t *testing.T) {
	root := t.TempDir()

	if _, err := Create(root); err != nil {
		t.Fatal(err)
	}
	res, err := Create(root)
	if err != nil {
		t.Fatal(err)
	}
	if !res.AlreadyExisted {
		t.Errorf("AlreadyExisted = false on the second run, want true")
	}
}

// snapshot returns a sorted, slash-normalized recursive listing of dir, used to
// assert that an operation left the existing tree untouched.
func snapshot(t *testing.T, dir string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(dir, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	return paths
}

// TestPlanListsMissingWithoutWriting verifies Plan reports every canonical entry
// as missing on a fresh target and writes nothing to disk.
func TestPlanListsMissingWithoutWriting(t *testing.T) {
	root := t.TempDir()

	plan, err := Detect(root)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if plan.WorkspaceExisted {
		t.Errorf("WorkspaceExisted = true on a fresh root, want false")
	}
	if len(plan.MissingDirs) != len(Subdirs) {
		t.Errorf("MissingDirs = %v, want all %d subdirs", plan.MissingDirs, len(Subdirs))
	}
	if len(plan.MissingFiles) != len(RootArtifacts) {
		t.Errorf("MissingFiles = %v, want all %d artifacts", plan.MissingFiles, len(RootArtifacts))
	}
	if plan.IsEmpty() {
		t.Errorf("IsEmpty = true, want false for a fresh target")
	}

	// Detection must be side-effect free: the workspace must not exist yet.
	if _, err := os.Stat(filepath.Join(root, Name)); !os.IsNotExist(err) {
		t.Errorf("Plan created the workspace directory (stat err=%v), want it untouched", err)
	}
}

// TestPlanMissingIsDeterministic verifies the missing entries come back in the
// fixed canonical order, not in filesystem-iteration order.
func TestPlanMissingIsDeterministic(t *testing.T) {
	root := t.TempDir()

	plan, err := Detect(root)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	for i, sub := range Subdirs {
		if plan.MissingDirs[i] != sub {
			t.Errorf("MissingDirs[%d] = %q, want %q (canonical order)", i, plan.MissingDirs[i], sub)
		}
	}
	for i, name := range RootArtifacts {
		if plan.MissingFiles[i] != name {
			t.Errorf("MissingFiles[%d] = %q, want %q (canonical order)", i, plan.MissingFiles[i], name)
		}
	}
}

// TestApplyCompletesMissingWithoutTouchingRest verifies an upgrade adds only the
// missing entries and leaves every existing file byte-for-byte intact.
func TestApplyCompletesMissingWithoutTouchingRest(t *testing.T) {
	root := t.TempDir()

	// Start from a complete workspace, then remove one canonical subdir to
	// simulate an incomplete (e.g. partially deleted) workspace.
	if _, err := Create(root); err != nil {
		t.Fatal(err)
	}
	wsPath := filepath.Join(root, Name)
	if err := os.RemoveAll(filepath.Join(wsPath, "docs")); err != nil {
		t.Fatal(err)
	}

	// Seed manual content that must survive the upgrade untouched.
	manualAgent := filepath.Join(wsPath, "agents", "my-agent.yaml")
	if err := os.WriteFile(manualAgent, []byte("name: my-agent"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := Detect(root)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !plan.WorkspaceExisted {
		t.Errorf("WorkspaceExisted = false on an existing workspace, want true")
	}
	if got, want := plan.MissingDirs, []string{"docs"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("MissingDirs = %v, want %v", got, want)
	}
	if len(plan.MissingFiles) != 0 {
		t.Errorf("MissingFiles = %v, want none", plan.MissingFiles)
	}

	res, err := plan.Apply()
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if !res.AlreadyExisted {
		t.Errorf("AlreadyExisted = false, want true on upgrade")
	}
	if len(res.CreatedDirs) != 1 || res.CreatedDirs[0] != "docs" {
		t.Errorf("CreatedDirs = %v, want [docs]", res.CreatedDirs)
	}

	if info, err := os.Stat(filepath.Join(wsPath, "docs")); err != nil || !info.IsDir() {
		t.Errorf("docs/ not recreated (stat err=%v)", err)
	}
	if b, _ := os.ReadFile(manualAgent); string(b) != "name: my-agent" {
		t.Errorf("manual agent file was modified: %q", b)
	}
}

// TestPlanEmptyOnCompleteWorkspace verifies a re-run over a complete workspace
// plans nothing (idempotency) and applying it changes the tree not at all.
func TestPlanEmptyOnCompleteWorkspace(t *testing.T) {
	root := t.TempDir()

	if _, err := Create(root); err != nil {
		t.Fatal(err)
	}
	wsPath := filepath.Join(root, Name)
	before := snapshot(t, wsPath)

	plan, err := Detect(root)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if !plan.IsEmpty() {
		t.Errorf("IsEmpty = false on a complete workspace, want true (missing dirs=%v files=%v)",
			plan.MissingDirs, plan.MissingFiles)
	}

	if _, err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if after := snapshot(t, wsPath); !equal(before, after) {
		t.Errorf("tree changed on an empty plan:\nbefore=%v\nafter =%v", before, after)
	}
}

// TestApplyPreservesManualEdit verifies a manually edited root artifact keeps its
// content across an upgrade — the core non-destruction guarantee (CA2).
func TestApplyPreservesManualEdit(t *testing.T) {
	root := t.TempDir()

	if _, err := Create(root); err != nil {
		t.Fatal(err)
	}
	initMD := filepath.Join(root, Name, "init.md")
	const marker = "MANUAL-EDIT-123"
	if err := os.WriteFile(initMD, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}

	// Force the upgrade path to do real work by removing a subdir, then re-run.
	if err := os.RemoveAll(filepath.Join(root, Name, "epics")); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(root); err != nil {
		t.Fatalf("Create (upgrade): %v", err)
	}

	if b, _ := os.ReadFile(initMD); string(b) != marker {
		t.Errorf("manual edit to init.md was lost: %q", b)
	}
}

// equal reports whether two string slices are element-wise equal.
func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
