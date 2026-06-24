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
