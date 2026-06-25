package specs

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden rewrites golden files from current output when set. Mirrors the
// adapter golden harness in internal/compile (RF-8.2 / RNF-5).
var updateGolden = flag.Bool("update", false, "rewrite golden files from current output")

// goldenBrief and goldenSpec are fixed canonical artifacts exercising the
// renderer surface: a title with a colon (so the conservative YAML quoting kicks
// in) and a multi-line body. The `generated: false` literal is a fixed boolean,
// never a volatile value, so the rendered bytes are reproducible.
func goldenBrief() Brief {
	return Brief{Slug: "my-feature", Title: "My Feature: a demo", Body: "The brief body.\nSecond line."}
}

func goldenSpec() Spec {
	return Spec{Slug: "my-feature", Title: "My Feature: a demo", BriefRef: "my-feature.brief.md", Body: "The spec body.\nSecond line."}
}

// TestSpecsGolden renders the canonical brief and spec and asserts each matches
// its golden file byte-for-byte. Run with -update to regenerate.
func TestSpecsGolden(t *testing.T) {
	assertGolden(t, "my-feature.brief.md", []byte(RenderBrief(goldenBrief())))
	assertGolden(t, "my-feature.md", []byte(RenderSpec(goldenSpec())))
}

// TestSpecsRenderDeterministic covers RF-8.2 / RNF-5: rendering the same brief
// and spec twice yields identical bytes (no volatile fields).
func TestSpecsRenderDeterministic(t *testing.T) {
	if RenderBrief(goldenBrief()) != RenderBrief(goldenBrief()) {
		t.Error("RenderBrief is not deterministic")
	}
	if RenderSpec(goldenSpec()) != RenderSpec(goldenSpec()) {
		t.Error("RenderSpec is not deterministic")
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
