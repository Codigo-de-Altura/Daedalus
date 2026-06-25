package specs

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readFile reads a file, failing the test if it is absent. Helper kept local so the
// tests read top-down.
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// listFiles returns the base names of the regular files directly under dir, so a
// test can assert on the exact set of files present.
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

// TestCapturePersistsBriefAndSpec covers Check 1/3 (CA1/CA3): capturing a brief
// persists `<slug>.brief.md` and materializes the canonical `<slug>.md` spec.
func TestCapturePersistsBriefAndSpec(t *testing.T) {
	root := t.TempDir()

	res, err := Capture(root, Brief{Slug: "user-auth", Title: "User Auth", Body: "Let users log in."})
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if !res.BriefCreated || !res.SpecCreated {
		t.Fatalf("expected both artifacts created; got brief=%v spec=%v", res.BriefCreated, res.SpecCreated)
	}

	briefContent := readFile(t, filepath.Join(root, "user-auth.brief.md"))
	if !strings.Contains(briefContent, "kind: brief") || !strings.Contains(briefContent, "slug: user-auth") {
		t.Errorf("brief frontmatter wrong; got:\n%s", briefContent)
	}
	if !strings.Contains(briefContent, "Let users log in.") {
		t.Errorf("brief body not persisted verbatim; got:\n%s", briefContent)
	}

	specContent := readFile(t, filepath.Join(root, "user-auth.md"))
	if !strings.Contains(specContent, "kind: spec") {
		t.Errorf("spec frontmatter missing kind: spec; got:\n%s", specContent)
	}
}

// TestSpecLinksBriefAndAnalyst covers Check 2/7 (CA2/CA7): the spec's frontmatter
// references its originating brief (brief -> spec trace) and records the analyst-step
// provenance anchored to the default workflow phase.
func TestSpecLinksBriefAndAnalyst(t *testing.T) {
	root := t.TempDir()
	if _, err := Capture(root, Brief{Slug: "payments", Title: "Payments", Body: "Take money."}); err != nil {
		t.Fatalf("Capture: %v", err)
	}

	specContent := readFile(t, filepath.Join(root, "payments.md"))
	for _, want := range []string{
		"brief: payments.brief.md", // R8/CA7 trace
		"agent: analyst",           // R2/CA2 link
		"workflow: sdd-default",
		"phase: spec",
		"generated: false", // R5/CA5: Daedalus did not run the agent
	} {
		if !strings.Contains(specContent, want) {
			t.Errorf("spec frontmatter missing %q; got:\n%s", want, specContent)
		}
	}

	briefContent := readFile(t, filepath.Join(root, "payments.brief.md"))
	for _, want := range []string{"consumed-by: analyst", "workflow: sdd-default", "phase: spec"} {
		if !strings.Contains(briefContent, want) {
			t.Errorf("brief frontmatter missing %q; got:\n%s", want, briefContent)
		}
	}
}

// TestSpecBodyReferencesBrief covers CA7 in the human-readable body: the seeded spec
// names the brief it must be generated from, and states Daedalus did not run the
// agent (R5).
func TestSpecBodyReferencesBrief(t *testing.T) {
	root := t.TempDir()
	if _, err := Capture(root, Brief{Slug: "search", Title: "Search", Body: "Find things."}); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	spec, err := LoadSpec(root, "search")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if !strings.Contains(spec.Body, "search.brief.md") {
		t.Errorf("seeded spec body should name the source brief; got:\n%s", spec.Body)
	}
	if !strings.Contains(spec.Body, "analyst") {
		t.Errorf("seeded spec body should mention the analyst agent; got:\n%s", spec.Body)
	}
}

// TestRecaptureDoesNotOverwriteEditedSpec covers Check 4 (CA4): a user edits the
// materialized spec, then a re-capture of the same brief leaves the user's edits
// untouched and reports the existing files as preserved (non-destructive).
func TestRecaptureDoesNotOverwriteEditedSpec(t *testing.T) {
	root := t.TempDir()
	if _, err := Capture(root, Brief{Slug: "user-auth", Title: "User Auth", Body: "v1"}); err != nil {
		t.Fatalf("Capture: %v", err)
	}

	// The user refines the spec manually.
	edited, err := EditSpecArtifact(root, "user-auth", SpecEditSpec{SetBody: true, Body: "REFINED BY HUMAN"})
	if err != nil {
		t.Fatalf("EditSpecArtifact: %v", err)
	}
	if edited.Body != "REFINED BY HUMAN" {
		t.Fatalf("edit not applied; got %q", edited.Body)
	}

	before := snapshotDir(t, root)

	// Re-capture the same brief — must NOT clobber the user's refined spec.
	res, err := Capture(root, Brief{Slug: "user-auth", Title: "User Auth", Body: "v2-different"})
	if err != nil {
		t.Fatalf("re-Capture: %v", err)
	}
	if res.BriefCreated || res.SpecCreated {
		t.Errorf("re-capture should create nothing; got brief=%v spec=%v", res.BriefCreated, res.SpecCreated)
	}

	after := snapshotDir(t, root)
	if before["user-auth.md"] != after["user-auth.md"] {
		t.Errorf("re-capture overwrote the user-refined spec\nbefore:\n%s\nafter:\n%s",
			before["user-auth.md"], after["user-auth.md"])
	}
	if before["user-auth.brief.md"] != after["user-auth.brief.md"] {
		t.Errorf("re-capture overwrote the existing brief")
	}
	if !strings.Contains(after["user-auth.md"], "REFINED BY HUMAN") {
		t.Errorf("user's refinement lost after re-capture; got:\n%s", after["user-auth.md"])
	}
}

// TestPartialPairCompleted covers the non-destructive completion case: if the spec
// was deleted but the brief remains, a re-capture re-creates only the missing spec
// and preserves the brief.
func TestPartialPairCompleted(t *testing.T) {
	root := t.TempDir()
	if _, err := Capture(root, Brief{Slug: "orders", Title: "Orders", Body: "b"}); err != nil {
		t.Fatalf("Capture: %v", err)
	}
	if err := os.Remove(filepath.Join(root, "orders.md")); err != nil {
		t.Fatal(err)
	}
	briefBefore := readFile(t, filepath.Join(root, "orders.brief.md"))

	res, err := Capture(root, Brief{Slug: "orders", Title: "Orders", Body: "b"})
	if err != nil {
		t.Fatalf("re-Capture: %v", err)
	}
	if res.BriefCreated {
		t.Errorf("brief should have been preserved, not re-created")
	}
	if !res.SpecCreated {
		t.Errorf("missing spec should have been re-created")
	}
	if got := readFile(t, filepath.Join(root, "orders.brief.md")); got != briefBefore {
		t.Errorf("brief was altered while completing the pair")
	}
}

// TestEditDoesNotTouchBriefOrOtherSpecs covers Check 4 (R4/R7): editing one spec
// changes only its file; the brief and every other artifact stay byte-identical.
func TestEditDoesNotTouchBriefOrOtherSpecs(t *testing.T) {
	root := t.TempDir()
	if _, err := Capture(root, Brief{Slug: "alpha", Title: "Alpha", Body: "a"}); err != nil {
		t.Fatalf("Capture alpha: %v", err)
	}
	if _, err := Capture(root, Brief{Slug: "beta", Title: "Beta", Body: "b"}); err != nil {
		t.Fatalf("Capture beta: %v", err)
	}

	before := snapshotDir(t, root)

	if _, err := EditSpecArtifact(root, "beta", SpecEditSpec{SetTitle: true, Title: "Beta v2"}); err != nil {
		t.Fatalf("EditSpecArtifact: %v", err)
	}

	after := snapshotDir(t, root)
	for _, untouched := range []string{"alpha.brief.md", "alpha.md", "beta.brief.md"} {
		if before[untouched] != after[untouched] {
			t.Errorf("editing beta spec altered %s", untouched)
		}
	}
	if before["beta.md"] == after["beta.md"] {
		t.Errorf("edit did not change beta.md")
	}
}

// TestDeterminism covers Check 6 (CA6): rendering the same brief/spec twice produces
// byte-identical content, and capturing into two clean roots yields identical files.
func TestDeterminism(t *testing.T) {
	b := Brief{Slug: "feature-x", Title: "Feature X", Body: "line one\nline two"}
	if RenderBrief(b) != RenderBrief(b) {
		t.Errorf("RenderBrief is not deterministic")
	}
	s := Spec{Slug: "feature-x", Title: "Feature X", BriefRef: "feature-x.brief.md", Body: "a\nb"}
	if RenderSpec(s) != RenderSpec(s) {
		t.Errorf("RenderSpec is not deterministic")
	}

	rootA, rootB := t.TempDir(), t.TempDir()
	if _, err := Capture(rootA, b); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(rootB, b); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"feature-x.brief.md", "feature-x.md"} {
		if readFile(t, filepath.Join(rootA, name)) != readFile(t, filepath.Join(rootB, name)) {
			t.Errorf("two captures with identical input produced different %s", name)
		}
	}
}

// TestInvalidSlugRejected covers slug validation (R3): an empty or non-kebab-case
// slug is rejected with a *ValidationError and no file is created.
func TestInvalidSlugRejected(t *testing.T) {
	cases := []struct {
		name string
		slug string
	}{
		{"empty", ""},
		{"uppercase", "MyFeature"},
		{"spaces", "my feature"},
		{"leading dash", "-bad"},
		{"underscore", "my_feature"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			_, err := Capture(root, Brief{Slug: tc.slug, Title: "T", Body: "b"})
			var ve *ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("Capture(%q) error = %v, want *ValidationError", tc.slug, err)
			}
			if names := listFiles(t, root); len(names) != 0 {
				t.Errorf("invalid slug created files: %v", names)
			}
		})
	}
}

// TestEmptyTitleRejected covers the title rule, including via edit (set title to
// empty must be rejected, leaving the file intact).
func TestEmptyTitleRejected(t *testing.T) {
	root := t.TempDir()
	if _, err := Capture(root, Brief{Slug: "x", Title: "   ", Body: "b"}); err == nil {
		t.Fatalf("Capture with whitespace title succeeded, want rejection")
	}
	if names := listFiles(t, root); len(names) != 0 {
		t.Errorf("rejected capture created files: %v", names)
	}

	if _, err := Capture(root, Brief{Slug: "ok", Title: "OK", Body: "b"}); err != nil {
		t.Fatalf("Capture ok: %v", err)
	}
	before := readFile(t, filepath.Join(root, "ok.md"))
	_, err := EditSpecArtifact(root, "ok", SpecEditSpec{SetTitle: true, Title: ""})
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("Edit to empty title error = %v, want *ValidationError", err)
	}
	if got := readFile(t, filepath.Join(root, "ok.md")); got != before {
		t.Errorf("rejected edit altered the spec file")
	}
}

// TestListReflectsCapturesAndSpecState covers listing (R6): briefs are listed sorted
// by slug, each flagged with whether its spec exists.
func TestListReflectsCapturesAndSpecState(t *testing.T) {
	root := t.TempDir()
	if _, err := Capture(root, Brief{Slug: "zeta", Title: "Zeta", Body: "z"}); err != nil {
		t.Fatal(err)
	}
	if _, err := Capture(root, Brief{Slug: "alpha", Title: "Alpha", Body: "a"}); err != nil {
		t.Fatal(err)
	}
	// Delete alpha's spec so it should report HasSpec=false.
	if err := os.Remove(filepath.Join(root, "alpha.md")); err != nil {
		t.Fatal(err)
	}

	entries, err := List(root)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("List returned %d entries, want 2: %+v", len(entries), entries)
	}
	// Sorted by slug: alpha before zeta.
	if entries[0].Slug != "alpha" || entries[1].Slug != "zeta" {
		t.Errorf("List not sorted by slug; got %q, %q", entries[0].Slug, entries[1].Slug)
	}
	if entries[0].HasSpec {
		t.Errorf("alpha spec was deleted; HasSpec should be false")
	}
	if !entries[1].HasSpec {
		t.Errorf("zeta spec exists; HasSpec should be true")
	}
}

// TestListEmptyWorkspace covers the well-defined empty case: listing a workspace
// with no specs directory yields an empty list, not an error.
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

// TestLoadRoundTrip covers load<->render stability for both artifacts: a freshly
// captured pair loads back equal and re-renders to the same bytes.
func TestLoadRoundTrip(t *testing.T) {
	root := t.TempDir()
	b := Brief{Slug: "round-trip", Title: "Round: Trip", Body: "line one\nline two"}
	if _, err := Capture(root, b); err != nil {
		t.Fatalf("Capture: %v", err)
	}

	loadedBrief, err := LoadBrief(root, "round-trip")
	if err != nil {
		t.Fatalf("LoadBrief: %v", err)
	}
	if loadedBrief.Slug != b.Slug || loadedBrief.Title != b.Title || loadedBrief.Body != b.Body {
		t.Errorf("brief round-trip mismatch\nwant: %+v\ngot:  %+v", b, loadedBrief)
	}
	if RenderBrief(loadedBrief) != readFile(t, filepath.Join(root, "round-trip.brief.md")) {
		t.Errorf("re-render of a loaded brief is not byte-stable")
	}

	loadedSpec, err := LoadSpec(root, "round-trip")
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if loadedSpec.BriefRef != "round-trip.brief.md" {
		t.Errorf("spec brief ref = %q, want round-trip.brief.md", loadedSpec.BriefRef)
	}
	if RenderSpec(loadedSpec) != readFile(t, filepath.Join(root, "round-trip.md")) {
		t.Errorf("re-render of a loaded spec is not byte-stable")
	}
}

// TestNotFoundSentinels covers the not-found sentinels for Load/Edit.
func TestNotFoundSentinels(t *testing.T) {
	root := t.TempDir()
	if _, err := LoadBrief(root, "nope"); !errors.Is(err, ErrBriefNotFound) {
		t.Errorf("LoadBrief absent error = %v, want ErrBriefNotFound", err)
	}
	if _, err := LoadSpec(root, "nope"); !errors.Is(err, ErrSpecNotFound) {
		t.Errorf("LoadSpec absent error = %v, want ErrSpecNotFound", err)
	}
	if _, err := EditSpecArtifact(root, "nope", SpecEditSpec{SetBody: true, Body: "x"}); !errors.Is(err, ErrSpecNotFound) {
		t.Errorf("EditSpecArtifact absent error = %v, want ErrSpecNotFound", err)
	}
}

// TestMalformedArtifact covers the malformed sentinel: a file without frontmatter is
// rejected on load.
func TestMalformedArtifact(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "broken.brief.md"), []byte("no frontmatter"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadBrief(root, "broken"); !errors.Is(err, ErrMalformed) {
		t.Errorf("LoadBrief malformed error = %v, want ErrMalformed", err)
	}
}

// TestPlanCaptureIsNonWriting covers the plan/apply split (R6/R7): PlanCapture
// computes content without writing, and the planned content equals what Apply writes.
func TestPlanCaptureIsNonWriting(t *testing.T) {
	root := t.TempDir()
	plan, err := PlanCapture(root, Brief{Slug: "preview-me", Title: "Preview Me", Body: "x"})
	if err != nil {
		t.Fatalf("PlanCapture: %v", err)
	}
	// Nothing written yet.
	if _, err := os.Stat(root); err == nil {
		if names := listFiles(t, root); len(names) != 0 {
			t.Errorf("PlanCapture wrote files: %v", names)
		}
	}
	if _, err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if got := readFile(t, plan.BriefPath); got != plan.BriefContent {
		t.Errorf("written brief != planned content")
	}
	if got := readFile(t, plan.SpecPath); got != plan.SpecContent {
		t.Errorf("written spec != planned content")
	}
}
