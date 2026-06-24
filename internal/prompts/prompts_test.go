package prompts

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readFile reads a file under root, failing the test if it is absent. Helper kept
// local so the test reads top-down.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// listFiles returns the base names of the regular files directly under dir,
// sorted, so a test can assert on the exact set of prompt files present.
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

// TestCreateGlobalPersistsFile covers Check 2 / CA "create global": a global
// prompt is persisted as `<id>.md` with kind: global in its frontmatter.
func TestCreateGlobalPersistsFile(t *testing.T) {
	root := t.TempDir()

	p := Prompt{ID: "project-style", Kind: KindGlobal, Title: "Project Style", Body: "Use English."}
	if err := Create(root, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	path := filepath.Join(root, "project-style.md")
	content := readFile(t, path)
	if !strings.Contains(content, "kind: global") {
		t.Errorf("frontmatter missing 'kind: global'; got:\n%s", content)
	}
	if !strings.Contains(content, "id: project-style") {
		t.Errorf("frontmatter missing 'id: project-style'; got:\n%s", content)
	}
	if !strings.HasSuffix(content, "Use English.\n") {
		t.Errorf("body not persisted verbatim with trailing newline; got:\n%s", content)
	}
}

// TestCreateSharedPersistsFile covers Check 3: a shared prompt is persisted with
// kind: shared.
func TestCreateSharedPersistsFile(t *testing.T) {
	root := t.TempDir()

	p := Prompt{ID: "glossary", Kind: KindShared, Title: "Glossary", Body: "Terms."}
	if err := Create(root, p); err != nil {
		t.Fatalf("Create: %v", err)
	}

	content := readFile(t, filepath.Join(root, "glossary.md"))
	if !strings.Contains(content, "kind: shared") {
		t.Errorf("frontmatter missing 'kind: shared'; got:\n%s", content)
	}
}

// TestListAndFilterByKind covers Check 4: listing returns all prompts with their
// kind, and a filter returns only the matching kind.
func TestListAndFilterByKind(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "project-style", Kind: KindGlobal, Title: "Style", Body: "x"})
	mustCreate(t, root, Prompt{ID: "glossary", Kind: KindShared, Title: "Glossary", Body: "y"})

	all, err := List(root, "")
	if err != nil {
		t.Fatalf("List all: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("List all returned %d entries, want 2: %+v", len(all), all)
	}
	// Deterministic order: sorted by id => glossary before project-style.
	if all[0].ID != "glossary" || all[1].ID != "project-style" {
		t.Errorf("List not sorted by id; got %q, %q", all[0].ID, all[1].ID)
	}

	globals, err := List(root, KindGlobal)
	if err != nil {
		t.Fatalf("List global: %v", err)
	}
	if len(globals) != 1 || globals[0].ID != "project-style" {
		t.Errorf("global filter wrong; got %+v", globals)
	}

	shared, err := List(root, KindShared)
	if err != nil {
		t.Fatalf("List shared: %v", err)
	}
	if len(shared) != 1 || shared[0].ID != "glossary" {
		t.Errorf("shared filter wrong; got %+v", shared)
	}
}

// TestListEmptyWorkspace covers the well-defined empty case: listing a workspace
// with no prompts directory yields an empty list, not an error.
func TestListEmptyWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "does-not-exist")
	entries, err := List(root, "")
	if err != nil {
		t.Fatalf("List on absent dir: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected empty list, got %+v", entries)
	}
}

// TestEditDoesNotDestroyOtherFiles covers Check 5: editing one prompt changes
// only its file; every other file stays byte-identical.
func TestEditDoesNotDestroyOtherFiles(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "project-style", Kind: KindGlobal, Title: "Style", Body: "original"})
	mustCreate(t, root, Prompt{ID: "glossary", Kind: KindShared, Title: "Glossary", Body: "terms"})

	before := snapshotDir(t, root)

	edited, err := Edit(root, "glossary", EditSpec{SetBody: true, Body: "terms\nmore terms"})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if edited.Body != "terms\nmore terms" {
		t.Errorf("edited body = %q, want updated", edited.Body)
	}

	after := snapshotDir(t, root)

	// The untouched prompt must be byte-identical.
	if before["project-style.md"] != after["project-style.md"] {
		t.Errorf("editing glossary altered project-style.md\nbefore:\n%s\nafter:\n%s",
			before["project-style.md"], after["project-style.md"])
	}
	// The edited prompt must have actually changed.
	if before["glossary.md"] == after["glossary.md"] {
		t.Errorf("edit did not change glossary.md")
	}
	if !strings.Contains(after["glossary.md"], "more terms") {
		t.Errorf("edited file missing new body; got:\n%s", after["glossary.md"])
	}
}

// TestDuplicateIDFails covers Check 6: creating a prompt with an existing id
// fails with ErrPromptExists and does not overwrite the original.
func TestDuplicateIDFails(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "glossary", Kind: KindShared, Title: "Glossary", Body: "original"})

	original := readFile(t, filepath.Join(root, "glossary.md"))

	err := Create(root, Prompt{ID: "glossary", Kind: KindGlobal, Title: "Other", Body: "overwrite attempt"})
	if !errors.Is(err, ErrPromptExists) {
		t.Fatalf("duplicate create error = %v, want ErrPromptExists", err)
	}
	if got := readFile(t, filepath.Join(root, "glossary.md")); got != original {
		t.Errorf("original file was overwritten by a duplicate create\nwant:\n%s\ngot:\n%s", original, got)
	}
}

// TestDeterminism covers Check 7: rendering the same prompt twice produces
// byte-identical content.
func TestDeterminism(t *testing.T) {
	p := Prompt{ID: "glossary", Kind: KindShared, Title: "Glossary", Description: "Terms used", Body: "a\nb"}
	first := Render(p)
	second := Render(p)
	if first != second {
		t.Errorf("Render is not deterministic\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	// Creating into a clean root twice yields identical bytes too.
	rootA, rootB := t.TempDir(), t.TempDir()
	mustCreate(t, rootA, p)
	mustCreate(t, rootB, p)
	if readFile(t, filepath.Join(rootA, "glossary.md")) != readFile(t, filepath.Join(rootB, "glossary.md")) {
		t.Errorf("two creates with identical input produced different files")
	}
}

// TestInvalidIDRejected covers Check 8: an empty or non-kebab-case id is rejected
// with a *ValidationError and no file is created.
func TestInvalidIDRejected(t *testing.T) {
	cases := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"uppercase", "MyPrompt"},
		{"spaces", "my prompt"},
		{"leading dash", "-bad"},
		{"underscore", "my_prompt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			err := Create(root, Prompt{ID: tc.id, Kind: KindGlobal, Title: "T", Body: "b"})
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("Create(%q) error = %v, want *ValidationError", tc.id, err)
			}
			// No file must have been created.
			if names := listFiles(t, root); len(names) != 0 {
				t.Errorf("invalid id created files: %v", names)
			}
		})
	}
}

// TestInvalidKindRejected covers the kind half of validation: a kind outside
// {global, shared} is rejected.
func TestInvalidKindRejected(t *testing.T) {
	root := t.TempDir()
	err := Create(root, Prompt{ID: "x", Kind: Kind("weird"), Title: "T", Body: "b"})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("Create with bad kind error = %v, want *ValidationError", err)
	}
}

// TestEmptyTitleRejected covers the title rule, including via Edit (set title to
// empty must be rejected, leaving the file intact).
func TestEmptyTitleRejected(t *testing.T) {
	root := t.TempDir()
	if err := Create(root, Prompt{ID: "x", Kind: KindGlobal, Title: "   ", Body: "b"}); err == nil {
		t.Fatalf("Create with whitespace title succeeded, want rejection")
	}

	mustCreate(t, root, Prompt{ID: "g", Kind: KindShared, Title: "Glossary", Body: "b"})
	before := readFile(t, filepath.Join(root, "g.md"))
	_, err := Edit(root, "g", EditSpec{SetTitle: true, Title: ""})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("Edit to empty title error = %v, want *ValidationError", err)
	}
	if got := readFile(t, filepath.Join(root, "g.md")); got != before {
		t.Errorf("rejected edit altered the file\nbefore:\n%s\nafter:\n%s", before, got)
	}
}

// TestRemoveRemovesOnlyItsFile covers Check 9: removing one prompt deletes only
// its file; the others remain.
func TestRemoveRemovesOnlyItsFile(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "project-style", Kind: KindGlobal, Title: "Style", Body: "x"})
	mustCreate(t, root, Prompt{ID: "glossary", Kind: KindShared, Title: "Glossary", Body: "y"})

	if err := Remove(root, "project-style"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	names := listFiles(t, root)
	if len(names) != 1 || names[0] != "glossary.md" {
		t.Errorf("Remove left wrong files: %v", names)
	}

	// Removing an absent prompt is an explicit error.
	if err := Remove(root, "project-style"); !errors.Is(err, ErrPromptNotFound) {
		t.Errorf("Remove absent error = %v, want ErrPromptNotFound", err)
	}
}

// TestLoadNotFound covers the not-found sentinel for Load/Edit.
func TestLoadNotFound(t *testing.T) {
	root := t.TempDir()
	if _, err := Load(root, "nope"); !errors.Is(err, ErrPromptNotFound) {
		t.Errorf("Load absent error = %v, want ErrPromptNotFound", err)
	}
	if _, err := Edit(root, "nope", EditSpec{SetBody: true, Body: "x"}); !errors.Is(err, ErrPromptNotFound) {
		t.Errorf("Edit absent error = %v, want ErrPromptNotFound", err)
	}
}

// TestLoadRoundTrip covers load↔render stability: a freshly created prompt loads
// back equal and re-renders to the same bytes.
func TestLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	p := Prompt{ID: "glossary", Kind: KindShared, Title: "Glossary: terms", Description: "all the terms", Body: "line one\nline two"}
	mustCreate(t, root, p)

	loaded, err := Load(root, "glossary")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.ID != p.ID || loaded.Kind != p.Kind || loaded.Title != p.Title ||
		loaded.Description != p.Description || loaded.Body != p.Body {
		t.Errorf("round-trip mismatch\nwant: %+v\ngot:  %+v", p, loaded)
	}
	if Render(loaded) != readFile(t, filepath.Join(root, "glossary.md")) {
		t.Errorf("re-render of a loaded prompt is not byte-stable")
	}
}

// TestMalformedPrompt covers the malformed sentinel: a file without frontmatter
// is rejected on Load.
func TestMalformedPrompt(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.md"), []byte("no frontmatter here"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, "broken"); !errors.Is(err, ErrMalformedPrompt) {
		t.Errorf("Load malformed error = %v, want ErrMalformedPrompt", err)
	}
}

// TestDescriptionOmittedWhenEmpty documents the frontmatter decision: an empty
// description is omitted entirely, a non-empty one is emitted.
func TestDescriptionOmittedWhenEmpty(t *testing.T) {
	withoutDesc := Render(Prompt{ID: "a", Kind: KindGlobal, Title: "A", Body: "x"})
	if strings.Contains(withoutDesc, "description:") {
		t.Errorf("empty description should be omitted; got:\n%s", withoutDesc)
	}
	withDesc := Render(Prompt{ID: "a", Kind: KindGlobal, Title: "A", Description: "d", Body: "x"})
	if !strings.Contains(withDesc, "description: d") {
		t.Errorf("non-empty description should be emitted; got:\n%s", withDesc)
	}
}

// mustCreate creates a prompt and fails the test on error. Keeps the test bodies
// focused on the behavior under test.
func mustCreate(t *testing.T, root string, p Prompt) {
	t.Helper()
	if err := Create(root, p); err != nil {
		t.Fatalf("Create(%q): %v", p.ID, err)
	}
}
