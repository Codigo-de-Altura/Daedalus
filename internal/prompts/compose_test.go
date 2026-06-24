package prompts

import (
	"errors"
	"strings"
	"testing"
)

// TestResolveSimpleInclusion covers Check 2: a prompt A that includes B expands
// B's content at the directive's position.
func TestResolveSimpleInclusion(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "glossary", Kind: KindShared, Title: "Glossary", Body: "Term: Daedalus."})
	mustCreate(t, root, Prompt{ID: "main", Kind: KindGlobal, Title: "Main",
		Body: "Intro.\n{{include: glossary}}\nOutro."})

	got, err := Resolve(root, "main")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := "Intro.\nTerm: Daedalus.\nOutro."
	if got != want {
		t.Errorf("composed text mismatch\nwant:\n%q\ngot:\n%q", want, got)
	}
}

// TestResolveRecursive covers Check 3: A includes B, B includes C; C is resolved
// through B into the final text.
func TestResolveRecursive(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "c", Kind: KindShared, Title: "C", Body: "C-content"})
	mustCreate(t, root, Prompt{ID: "b", Kind: KindShared, Title: "B", Body: "B-before\n{{include: c}}\nB-after"})
	mustCreate(t, root, Prompt{ID: "a", Kind: KindGlobal, Title: "A", Body: "A-top\n{{include: b}}\nA-bottom"})

	got, err := Resolve(root, "a")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	want := "A-top\nB-before\nC-content\nB-after\nA-bottom"
	if got != want {
		t.Errorf("recursive composition mismatch\nwant:\n%q\ngot:\n%q", want, got)
	}
}

// TestResolveDeterministic covers Check 4: resolving the same prompt twice yields
// byte-identical output.
func TestResolveDeterministic(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "shared", Kind: KindShared, Title: "Shared", Body: "shared body"})
	mustCreate(t, root, Prompt{ID: "x", Kind: KindGlobal, Title: "X",
		Body: "one\n{{include: shared}}\ntwo\n{{include: shared}}\nthree"})

	first, err := Resolve(root, "x")
	if err != nil {
		t.Fatalf("Resolve first: %v", err)
	}
	second, err := Resolve(root, "x")
	if err != nil {
		t.Fatalf("Resolve second: %v", err)
	}
	if first != second {
		t.Errorf("Resolve is not deterministic\nfirst:\n%q\nsecond:\n%q", first, second)
	}
	// The shared fragment must appear once per reference (a diamond/repeat is fine).
	if strings.Count(first, "shared body") != 2 {
		t.Errorf("expected shared body twice, got:\n%q", first)
	}
}

// TestResolveCycleDetected covers Check 5: a cycle A→B→A is reported as a typed
// error without hanging or overflowing.
func TestResolveCycleDetected(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "a", Kind: KindGlobal, Title: "A", Body: "{{include: b}}"})
	mustCreate(t, root, Prompt{ID: "b", Kind: KindShared, Title: "B", Body: "{{include: a}}"})

	_, err := Resolve(root, "a")
	if !errors.Is(err, ErrIncludeCycle) {
		t.Fatalf("cycle error = %v, want ErrIncludeCycle", err)
	}
	var ce *IncludeCycleError
	if !errors.As(err, &ce) {
		t.Fatalf("error is not *IncludeCycleError: %v", err)
	}
	// The chain must name the loop and close it (a -> b -> a).
	chain := strings.Join(ce.Chain, " -> ")
	if !strings.Contains(chain, "a -> b -> a") {
		t.Errorf("cycle chain = %q, want it to contain 'a -> b -> a'", chain)
	}
}

// TestResolveSelfCycle covers self-inclusion A→A as a cycle.
func TestResolveSelfCycle(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "loop", Kind: KindGlobal, Title: "Loop", Body: "x\n{{include: loop}}\ny"})

	_, err := Resolve(root, "loop")
	if !errors.Is(err, ErrIncludeCycle) {
		t.Fatalf("self-cycle error = %v, want ErrIncludeCycle", err)
	}
}

// TestResolveNotFound covers Check 6: a reference to a non-existent id fails with
// an explicit error naming the missing id.
func TestResolveNotFound(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "a", Kind: KindGlobal, Title: "A", Body: "{{include: ghost}}"})

	_, err := Resolve(root, "a")
	if !errors.Is(err, ErrIncludeNotFound) {
		t.Fatalf("not-found error = %v, want ErrIncludeNotFound", err)
	}
	var nf *IncludeNotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("error is not *IncludeNotFoundError: %v", err)
	}
	if nf.MissingID != "ghost" {
		t.Errorf("MissingID = %q, want %q", nf.MissingID, "ghost")
	}
	if nf.ReferencedBy != "a" {
		t.Errorf("ReferencedBy = %q, want %q", nf.ReferencedBy, "a")
	}
	if !strings.Contains(err.Error(), "ghost") {
		t.Errorf("error message does not name the missing id: %v", err)
	}
}

// TestResolveDRYSingleSourceFile covers Check 7: two prompts referencing the same
// fragment, and the fragment exists as exactly one file on disk.
func TestResolveDRYSingleSourceFile(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "policy", Kind: KindShared, Title: "Policy", Body: "Commit policy."})
	mustCreate(t, root, Prompt{ID: "agent-one", Kind: KindGlobal, Title: "One", Body: "{{include: policy}}"})
	mustCreate(t, root, Prompt{ID: "agent-two", Kind: KindGlobal, Title: "Two", Body: "{{include: policy}}"})

	// Both compose to the shared fragment.
	for _, id := range []string{"agent-one", "agent-two"} {
		got, err := Resolve(root, id)
		if err != nil {
			t.Fatalf("Resolve(%q): %v", id, err)
		}
		if got != "Commit policy." {
			t.Errorf("Resolve(%q) = %q, want the shared fragment", id, got)
		}
	}

	// The fragment exists exactly once on disk: one policy.md, no duplicates.
	names := listFiles(t, root)
	policyCount := 0
	for _, n := range names {
		if strings.Contains(n, "policy") {
			policyCount++
		}
	}
	if policyCount != 1 {
		t.Errorf("expected exactly one policy file, found %d in %v", policyCount, names)
	}
}

// TestResolveDoesNotMutateSources covers Check 8: resolving leaves every source
// file byte-identical.
func TestResolveDoesNotMutateSources(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "frag", Kind: KindShared, Title: "Frag", Body: "fragment"})
	mustCreate(t, root, Prompt{ID: "host", Kind: KindGlobal, Title: "Host", Body: "before\n{{include: frag}}\nafter"})

	before := snapshotDir(t, root)

	if _, err := Resolve(root, "host"); err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	after := snapshotDir(t, root)
	for name, content := range before {
		if after[name] != content {
			t.Errorf("Resolve mutated source %s\nbefore:\n%s\nafter:\n%s", name, content, after[name])
		}
	}
	if len(after) != len(before) {
		t.Errorf("Resolve changed the set of files: before %d, after %d", len(before), len(after))
	}
}

// TestResolveNoIncludes covers the trivial case: a prompt with no directives
// composes to its own body, with trailing newlines trimmed.
func TestResolveNoIncludes(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "plain", Kind: KindGlobal, Title: "Plain", Body: "just text\nsecond line"})

	got, err := Resolve(root, "plain")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "just text\nsecond line" {
		t.Errorf("Resolve plain = %q", got)
	}
}

// TestResolveRootNotFound covers a missing root prompt (distinct from a missing
// include): the underlying ErrPromptNotFound surfaces.
func TestResolveRootNotFound(t *testing.T) {
	root := t.TempDir()
	if _, err := Resolve(root, "nope"); !errors.Is(err, ErrPromptNotFound) {
		t.Errorf("Resolve missing root error = %v, want ErrPromptNotFound", err)
	}
}

// TestResolveDirectiveOnlyWholeLine covers the syntax rule: a line that merely
// mentions the token in prose is NOT a directive and is left verbatim.
func TestResolveDirectiveNotWholeLine(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "frag", Kind: KindShared, Title: "Frag", Body: "FRAG"})
	mustCreate(t, root, Prompt{ID: "host", Kind: KindGlobal, Title: "Host",
		Body: "use {{include: frag}} inline\n{{include: frag}}"})

	got, err := Resolve(root, "host")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	// The inline mention stays literal; only the standalone directive expands.
	want := "use {{include: frag}} inline\nFRAG"
	if got != want {
		t.Errorf("whole-line rule failed\nwant:\n%q\ngot:\n%q", want, got)
	}
}

// TestResolveIndentedDirective covers that an indented directive line is still a
// directive (whitespace around it is not significant) and is replaced wholesale.
func TestResolveIndentedDirective(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "frag", Kind: KindShared, Title: "Frag", Body: "FRAG"})
	mustCreate(t, root, Prompt{ID: "host", Kind: KindGlobal, Title: "Host", Body: "a\n    {{include: frag}}\nb"})

	got, err := Resolve(root, "host")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != "a\nFRAG\nb" {
		t.Errorf("indented directive not replaced wholesale; got:\n%q", got)
	}
}

// TestResolveDiamond covers that a fragment reached via two branches (A→B→D and
// A→C→D) is not a cycle and expands on each branch.
func TestResolveDiamond(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "d", Kind: KindShared, Title: "D", Body: "D"})
	mustCreate(t, root, Prompt{ID: "b", Kind: KindShared, Title: "B", Body: "{{include: d}}"})
	mustCreate(t, root, Prompt{ID: "c", Kind: KindShared, Title: "C", Body: "{{include: d}}"})
	mustCreate(t, root, Prompt{ID: "a", Kind: KindGlobal, Title: "A", Body: "{{include: b}}\n{{include: c}}"})

	got, err := Resolve(root, "a")
	if err != nil {
		t.Fatalf("Resolve diamond: %v", err)
	}
	if got != "D\nD" {
		t.Errorf("diamond expansion = %q, want \"D\\nD\"", got)
	}
}

// TestResolveWritesNothing guards that the composition path never adds a file:
// the file count under the prompts root is unchanged by a Resolve.
func TestResolveWritesNothing(t *testing.T) {
	root := t.TempDir()
	mustCreate(t, root, Prompt{ID: "frag", Kind: KindShared, Title: "Frag", Body: "FRAG"})
	mustCreate(t, root, Prompt{ID: "host", Kind: KindGlobal, Title: "Host", Body: "{{include: frag}}"})

	beforeCount := len(listFiles(t, root))
	if _, err := Resolve(root, "host"); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if afterCount := len(listFiles(t, root)); afterCount != beforeCount {
		t.Errorf("Resolve changed file count: before %d, after %d", beforeCount, afterCount)
	}
}
