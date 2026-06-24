package workspace

import (
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
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

// rootYAMLKeys extracts the top-level keys of a small, hand-rendered YAML
// document in document order. It deliberately avoids a YAML dependency
// (stdlib-first; go.mod has none): the manifest is a fixed, flat-at-the-root
// shape, so a top-level key is any line that starts in column 0 (no leading
// whitespace), is not a comment, and contains a ':'. Nested block entries are
// indented and list items start with '-', so both are skipped.
func rootYAMLKeys(t *testing.T, yaml string) []string {
	t.Helper()
	var keys []string
	for _, line := range strings.Split(yaml, "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Indented lines are children of a block (e.g. conventions entries) and
		// list items belong to a sequence (e.g. backends); neither is a root key.
		if line[0] == ' ' || line[0] == '\t' || strings.HasPrefix(line, "-") {
			continue
		}
		key, _, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		keys = append(keys, strings.TrimSpace(key))
	}
	return keys
}

// fixedNameRoot creates a subdirectory with a fixed name inside a fresh TempDir
// and returns it. deriveProjectName derives the project name from the basename
// of the root, so two distinct t.TempDir() paths would otherwise yield different
// project names (and different content). Nesting a fixed-name directory under
// each TempDir forces both runs to share the same derived project name, which is
// what lets us assert byte-for-byte determinism across two independent roots.
func fixedNameRoot(t *testing.T, name string) string {
	t.Helper()
	root := filepath.Join(t.TempDir(), name)
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestManifestHasRequiredContent covers CA1/CA5: after Create, daedalus.yaml
// holds the required root keys and the backends list includes the MVP default.
func TestManifestHasRequiredContent(t *testing.T) {
	root := fixedNameRoot(t, "demo-project")

	if _, err := Create(root); err != nil {
		t.Fatalf("Create: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(root, Name, "daedalus.yaml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	manifest := string(b)

	// CA1: the four required root keys are present.
	keys := rootYAMLKeys(t, manifest)
	for _, want := range []string{"name", "version", "backends", "conventions"} {
		found := false
		for _, k := range keys {
			if k == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("manifest missing required root key %q; got keys %v", want, keys)
		}
	}

	// name is derived from the (fixed) root basename, so it is deterministic.
	if !strings.Contains(manifest, "name: demo-project") {
		t.Errorf("manifest does not carry the derived project name; got:\n%s", manifest)
	}

	// CA5: the backends list ships the MVP default (claude-code).
	if !strings.Contains(manifest, DefaultBackend) {
		t.Errorf("manifest backends missing MVP default %q; got:\n%s", DefaultBackend, manifest)
	}
}

// TestInitMDHasStructureAndConventions covers CA2: init.md carries the
// `.daedalus/` structure map and a conventions section.
func TestInitMDHasStructureAndConventions(t *testing.T) {
	root := fixedNameRoot(t, "demo-project")

	if _, err := Create(root); err != nil {
		t.Fatalf("Create: %v", err)
	}
	b, err := os.ReadFile(filepath.Join(root, Name, "init.md"))
	if err != nil {
		t.Fatalf("read init.md: %v", err)
	}
	initMD := string(b)

	// The structure map must enumerate the workspace layout: the root dir line
	// plus every canonical subdir, so the documented map cannot drift from what
	// the scaffolding creates.
	if !strings.Contains(initMD, ".daedalus/") {
		t.Errorf("init.md missing the .daedalus/ structure map; got:\n%s", initMD)
	}
	for _, sub := range Subdirs {
		if !strings.Contains(initMD, sub+"/") {
			t.Errorf("init.md structure map missing subdir %q", sub)
		}
	}
	// A conventions section must be present (case-insensitive to tolerate heading
	// vs. inline phrasing without coupling the test to exact wording).
	if !strings.Contains(strings.ToLower(initMD), "conventions") {
		t.Errorf("init.md missing a conventions section; got:\n%s", initMD)
	}
}

// TestArtifactContentIsByteForByteDeterministic covers CA3/CA4: generating the
// artifacts twice over two independent roots that share the same derived project
// name yields byte-identical daedalus.yaml and init.md, and the manifest's root
// key order is identical across runs.
func TestArtifactContentIsByteForByteDeterministic(t *testing.T) {
	// Same fixed name under two distinct TempDirs => same derived project name
	// => the content generators receive identical input on both runs.
	generate := func() (manifest, initMD []byte) {
		root := fixedNameRoot(t, "fixed-name")
		if _, err := Create(root); err != nil {
			t.Fatalf("Create: %v", err)
		}
		m, err := os.ReadFile(filepath.Join(root, Name, "daedalus.yaml"))
		if err != nil {
			t.Fatalf("read manifest: %v", err)
		}
		i, err := os.ReadFile(filepath.Join(root, Name, "init.md"))
		if err != nil {
			t.Fatalf("read init.md: %v", err)
		}
		return m, i
	}

	manifest1, initMD1 := generate()
	manifest2, initMD2 := generate()

	// CA3: byte-for-byte equality of both artifacts across runs.
	if string(manifest1) != string(manifest2) {
		t.Errorf("daedalus.yaml not deterministic:\n--- run 1 ---\n%s\n--- run 2 ---\n%s", manifest1, manifest2)
	}
	if string(initMD1) != string(initMD2) {
		t.Errorf("init.md not deterministic:\n--- run 1 ---\n%s\n--- run 2 ---\n%s", initMD1, initMD2)
	}

	// CA4: the root key order is identical and stable between runs.
	keys1 := rootYAMLKeys(t, string(manifest1))
	keys2 := rootYAMLKeys(t, string(manifest2))
	if !equal(keys1, keys2) {
		t.Errorf("manifest key order not stable across runs: %v vs %v", keys1, keys2)
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
