package architecture

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readFile reads a file, failing the test if it is absent.
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

// snapshotDir captures the content of every file under dir keyed by name, so a test
// can prove byte-for-byte that unrelated files were left untouched.
func snapshotDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	snap := make(map[string]string)
	for _, name := range listFiles(t, dir) {
		snap[name] = readFile(t, filepath.Join(dir, name))
	}
	return snap
}

// mustCreate creates a document and fails the test on error.
func mustCreate(t *testing.T, root string, d Document) {
	t.Helper()
	if err := Create(root, d); err != nil {
		t.Fatalf("Create(%q): %v", d.Slug, err)
	}
}

// TestCreatePersistsCanonicalLocation covers Check 1/2 (CA1/CA2): creating a document
// persists `.daedalus/architecture/<slug>.md` and it is gestionable (loadable).
func TestCreatePersistsCanonicalLocation(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Document{Slug: "payments-arch", Title: "Payments Architecture"})

	path := filepath.Join(root, "payments-arch.md")
	content := readFile(t, path)
	if !strings.Contains(content, "kind: architecture") || !strings.Contains(content, "slug: payments-arch") {
		t.Errorf("frontmatter wrong; got:\n%s", content)
	}
	if !strings.Contains(content, "title: Payments Architecture") {
		t.Errorf("missing title; got:\n%s", content)
	}
}

// TestUnlinkedDocumentOmitsProvenance covers the optional-link rule (R3): a document
// without a spec link carries no architect provenance keys at all.
func TestUnlinkedDocumentOmitsProvenance(t *testing.T) {
	content := Render(Document{Slug: "x", Title: "X", Body: "b"})
	for _, key := range []string{"spec:", "agent:", "workflow:", "phase:", "generated:"} {
		if strings.Contains(content, key) {
			t.Errorf("unlinked document should omit %q; got:\n%s", key, content)
		}
	}
}

// TestLinkedDocumentRecordsSpecProvenance covers Check 3 (CA3): a document linked to
// a spec references it (spec -> architecture trace) and records the architect-step
// provenance anchored to the default workflow phase.
func TestLinkedDocumentRecordsSpecProvenance(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Document{Slug: "payments-arch", Title: "Payments Architecture", SpecRef: "payments.md"})

	content := readFile(t, filepath.Join(root, "payments-arch.md"))
	for _, want := range []string{
		"spec: payments.md", // R3/CA3 trace
		"agent: architect",
		"workflow: sdd-default",
		"phase: architecture",
		"generated: false", // R5/CA5: Daedalus did not run the agent
	} {
		if !strings.Contains(content, want) {
			t.Errorf("linked document frontmatter missing %q; got:\n%s", want, content)
		}
	}
	// generated must be a real YAML boolean, not the quoted string "false".
	if strings.Contains(content, `generated: "false"`) {
		t.Errorf("generated should be an unquoted boolean; got:\n%s", content)
	}
}

// TestSeededBodyMentionsSpecAndAgent covers R5/CA5 in the human-readable body: the
// seeded placeholder states Daedalus did not run the agent and, when linked, names
// the source spec.
func TestSeededBodyMentionsSpecAndAgent(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Document{Slug: "search-arch", Title: "Search", SpecRef: "search.md"})
	d, err := Load(root, "search-arch")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !strings.Contains(d.Body, "search.md") {
		t.Errorf("seeded body should name the source spec; got:\n%s", d.Body)
	}
	if !strings.Contains(d.Body, "architect") {
		t.Errorf("seeded body should mention the architect agent; got:\n%s", d.Body)
	}
}

// TestSuppliedBodyKeptVerbatim covers that a caller-supplied body is not replaced by
// the placeholder.
func TestSuppliedBodyKeptVerbatim(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Document{Slug: "x", Title: "X", Body: "MY OWN BODY"})
	content := readFile(t, filepath.Join(root, "x.md"))
	if !strings.Contains(content, "MY OWN BODY") || strings.Contains(content, "placeholder") {
		t.Errorf("supplied body should be kept verbatim; got:\n%s", content)
	}
}

// TestDuplicateSlugFails covers Check 4 (CA4/R7): creating a document with an existing
// slug fails with ErrDocumentExists and does not overwrite the original.
func TestDuplicateSlugFails(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Document{Slug: "arch", Title: "Original", Body: "original"})
	original := readFile(t, filepath.Join(root, "arch.md"))

	err := Create(root, Document{Slug: "arch", Title: "Other", Body: "overwrite attempt"})
	if !errors.Is(err, ErrDocumentExists) {
		t.Fatalf("duplicate create error = %v, want ErrDocumentExists", err)
	}
	if got := readFile(t, filepath.Join(root, "arch.md")); got != original {
		t.Errorf("original file was overwritten by a duplicate create")
	}
}

// TestEditDoesNotDestroyOtherFiles covers Check 4 (R4/R7): editing one document
// changes only its file; every other file stays byte-identical.
func TestEditDoesNotDestroyOtherFiles(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Document{Slug: "alpha", Title: "Alpha", Body: "a"})
	mustCreate(t, root, Document{Slug: "beta", Title: "Beta", Body: "b"})

	before := snapshotDir(t, root)

	edited, err := Edit(root, "beta", EditSpec{SetBody: true, Body: "REFINED BY HUMAN"})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if edited.Body != "REFINED BY HUMAN" {
		t.Errorf("edit not applied; got %q", edited.Body)
	}

	after := snapshotDir(t, root)
	if before["alpha.md"] != after["alpha.md"] {
		t.Errorf("editing beta altered alpha.md")
	}
	if before["beta.md"] == after["beta.md"] {
		t.Errorf("edit did not change beta.md")
	}
	if !strings.Contains(after["beta.md"], "REFINED BY HUMAN") {
		t.Errorf("edited file missing new body; got:\n%s", after["beta.md"])
	}
}

// TestEditAttachesAndClearsSpecLink covers the editable spec link (R3): a user can
// attach a spec to a previously unlinked document and later clear it.
func TestEditAttachesAndClearsSpecLink(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Document{Slug: "arch", Title: "Arch", Body: "b"})

	// Initially unlinked.
	if strings.Contains(readFile(t, filepath.Join(root, "arch.md")), "spec:") {
		t.Fatalf("document should start unlinked")
	}

	// Attach a spec link.
	if _, err := Edit(root, "arch", EditSpec{SetSpec: true, Spec: "orders.md"}); err != nil {
		t.Fatalf("Edit attach: %v", err)
	}
	linked := readFile(t, filepath.Join(root, "arch.md"))
	if !strings.Contains(linked, "spec: orders.md") || !strings.Contains(linked, "agent: architect") {
		t.Errorf("attach did not record the spec provenance; got:\n%s", linked)
	}

	// Clear the spec link (SetSpec with empty value).
	if _, err := Edit(root, "arch", EditSpec{SetSpec: true, Spec: ""}); err != nil {
		t.Fatalf("Edit clear: %v", err)
	}
	cleared := readFile(t, filepath.Join(root, "arch.md"))
	for _, key := range []string{"spec:", "agent:", "workflow:", "phase:", "generated:"} {
		if strings.Contains(cleared, key) {
			t.Errorf("clearing the link should drop %q; got:\n%s", key, cleared)
		}
	}
}

// TestDeterminism covers Check 6 (CA6): rendering the same document twice produces
// byte-identical content, for both linked and unlinked documents.
func TestDeterminism(t *testing.T) {
	for _, d := range []Document{
		{Slug: "arch", Title: "Arch", Body: "a\nb"},
		{Slug: "arch", Title: "Arch", SpecRef: "spec.md", Body: "a\nb"},
	} {
		if Render(d) != Render(d) {
			t.Errorf("Render is not deterministic for %+v", d)
		}
	}

	rootA, rootB := t.TempDir(), t.TempDir()
	d := Document{Slug: "arch", Title: "Arch", SpecRef: "spec.md", Body: "x"}
	mustCreate(t, rootA, d)
	mustCreate(t, rootB, d)
	if readFile(t, filepath.Join(rootA, "arch.md")) != readFile(t, filepath.Join(rootB, "arch.md")) {
		t.Errorf("two creates with identical input produced different files")
	}
}

// TestInvalidSlugRejected covers slug validation (R1): an empty or non-kebab-case slug
// is rejected with a *ValidationError and no file is created.
func TestInvalidSlugRejected(t *testing.T) {
	cases := []struct {
		name string
		slug string
	}{
		{"empty", ""},
		{"uppercase", "MyArch"},
		{"spaces", "my arch"},
		{"leading dash", "-bad"},
		{"underscore", "my_arch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			err := Create(root, Document{Slug: tc.slug, Title: "T", Body: "b"})
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("Create(%q) error = %v, want *ValidationError", tc.slug, err)
			}
			if names := listFiles(t, root); len(names) != 0 {
				t.Errorf("invalid slug created files: %v", names)
			}
		})
	}
}

// TestEmptyTitleRejected covers the title rule, including via Edit (set title to empty
// must be rejected, leaving the file intact).
func TestEmptyTitleRejected(t *testing.T) {
	root := t.TempDir()
	if err := Create(root, Document{Slug: "x", Title: "   ", Body: "b"}); err == nil {
		t.Fatalf("Create with whitespace title succeeded, want rejection")
	}
	if names := listFiles(t, root); len(names) != 0 {
		t.Errorf("rejected create left files: %v", names)
	}

	mustCreate(t, root, Document{Slug: "ok", Title: "OK", Body: "b"})
	before := readFile(t, filepath.Join(root, "ok.md"))
	_, err := Edit(root, "ok", EditSpec{SetTitle: true, Title: ""})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("Edit to empty title error = %v, want *ValidationError", err)
	}
	if got := readFile(t, filepath.Join(root, "ok.md")); got != before {
		t.Errorf("rejected edit altered the file")
	}
}

// TestListSortedWithSpecState covers listing (R6): documents are listed sorted by
// slug, each carrying its spec link state.
func TestListSortedWithSpecState(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Document{Slug: "zeta", Title: "Zeta", SpecRef: "zeta-spec.md", Body: "z"})
	mustCreate(t, root, Document{Slug: "alpha", Title: "Alpha", Body: "a"})

	entries, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("List returned %d entries, want 2: %+v", len(entries), entries)
	}
	if entries[0].Slug != "alpha" || entries[1].Slug != "zeta" {
		t.Errorf("List not sorted by slug; got %q, %q", entries[0].Slug, entries[1].Slug)
	}
	if entries[0].SpecRef != "" {
		t.Errorf("alpha is unlinked; SpecRef should be empty, got %q", entries[0].SpecRef)
	}
	if entries[1].SpecRef != "zeta-spec.md" {
		t.Errorf("zeta SpecRef = %q, want zeta-spec.md", entries[1].SpecRef)
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

// TestLoadRoundTrip covers load<->render stability: a freshly created document loads
// back equal and re-renders to the same bytes, for both linked and unlinked.
func TestLoadRoundTrip(t *testing.T) {
	for _, d := range []Document{
		{Slug: "round-trip", Title: "Round: Trip", Body: "line one\nline two"},
		{Slug: "round-trip", Title: "Round Trip", SpecRef: "src.md", Body: "line one\nline two"},
	} {
		root := t.TempDir()
		mustCreate(t, root, d)

		loaded, err := Load(root, "round-trip")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		if loaded.Slug != d.Slug || loaded.Title != d.Title || loaded.SpecRef != d.SpecRef || loaded.Body != d.Body {
			t.Errorf("round-trip mismatch\nwant: %+v\ngot:  %+v", d, loaded)
		}
		if Render(loaded) != readFile(t, filepath.Join(root, "round-trip.md")) {
			t.Errorf("re-render of a loaded document is not byte-stable")
		}
	}
}

// TestNotFoundSentinel covers the not-found sentinel for Load/Edit/Remove.
func TestNotFoundSentinel(t *testing.T) {
	root := t.TempDir()
	if _, err := Load(root, "nope"); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("Load absent error = %v, want ErrDocumentNotFound", err)
	}
	if _, err := Edit(root, "nope", EditSpec{SetBody: true, Body: "x"}); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("Edit absent error = %v, want ErrDocumentNotFound", err)
	}
	if err := Remove(root, "nope"); !errors.Is(err, ErrDocumentNotFound) {
		t.Errorf("Remove absent error = %v, want ErrDocumentNotFound", err)
	}
}

// TestRemoveRemovesOnlyItsFile covers Remove: removing one document deletes only its
// file; the others remain.
func TestRemoveRemovesOnlyItsFile(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Document{Slug: "alpha", Title: "Alpha", Body: "x"})
	mustCreate(t, root, Document{Slug: "beta", Title: "Beta", Body: "y"})

	if err := Remove(root, "alpha"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	names := listFiles(t, root)
	if len(names) != 1 || names[0] != "beta.md" {
		t.Errorf("Remove left wrong files: %v", names)
	}
}

// TestMalformedDocument covers the malformed sentinel: a file without frontmatter is
// rejected on Load.
func TestMalformedDocument(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.md"), []byte("no frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root, "broken"); !errors.Is(err, ErrMalformed) {
		t.Errorf("Load malformed error = %v, want ErrMalformed", err)
	}
}

// TestPlanCreateIsNonWriting covers the plan/apply split (R6/R7): PlanCreate computes
// content without writing, and the planned content equals what Apply writes.
func TestPlanCreateIsNonWriting(t *testing.T) {
	root := t.TempDir()
	plan, err := PlanCreate(root, Document{Slug: "preview-me", Title: "Preview Me", SpecRef: "s.md"})
	if err != nil {
		t.Fatalf("PlanCreate: %v", err)
	}
	if _, err := os.Stat(root); err == nil {
		if names := listFiles(t, root); len(names) != 0 {
			t.Errorf("PlanCreate wrote files: %v", names)
		}
	}
	if err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := readFile(t, plan.Path); got != plan.Content {
		t.Errorf("written content != planned content")
	}
}
