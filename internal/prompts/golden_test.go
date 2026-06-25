package prompts

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden rewrites golden files from current output when set. Mirrors the
// adapter golden harness in internal/compile (RF-8.2 / RNF-5).
var updateGolden = flag.Bool("update", false, "rewrite golden files from current output")

// goldenPrompts are fixed canonical prompts exercising the renderer surface: one
// without a description (the key is omitted) and one with a description plus a
// title containing a colon (so the conservative YAML quoting kicks in).
func goldenPrompts() map[string]Prompt {
	return map[string]Prompt{
		"global.md": {ID: "project-style", Kind: KindGlobal, Title: "Project Style", Body: "Write in English.\nBe concise."},
		"shared.md": {ID: "glossary", Kind: KindShared, Title: "Glossary: terms", Description: "Shared vocabulary", Body: "Term one.\nTerm two."},
	}
}

// TestPromptsGolden renders the canonical prompts and asserts each matches its
// golden file byte-for-byte. Run with -update to regenerate.
func TestPromptsGolden(t *testing.T) {
	for name, p := range goldenPrompts() {
		assertGolden(t, name, []byte(Render(p)))
	}
}

// TestPromptsRenderDeterministic covers RF-8.2 / RNF-5: rendering the same prompt
// twice yields identical bytes.
func TestPromptsRenderDeterministic(t *testing.T) {
	for _, p := range goldenPrompts() {
		if Render(p) != Render(p) {
			t.Errorf("Render is not deterministic for %q", p.ID)
		}
	}
}

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	goldenPath := filepath.Join("testdata", "golden", filepath.FromSlash(name))

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenPath), 0o755); err != nil {
			t.Fatalf("mkdir golden dir: %v", err)
		}
		if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
			t.Fatalf("write golden %s: %v", goldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden %s: %v (run with -update to generate)", goldenPath, err)
	}
	if string(want) != string(got) {
		t.Errorf("artifact %s does not match golden:\n--- got ---\n%s\n--- want ---\n%s",
			name, got, want)
	}
}
