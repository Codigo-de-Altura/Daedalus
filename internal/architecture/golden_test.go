package architecture

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

// updateGolden rewrites golden files from current output when set. Mirrors the
// adapter golden harness in internal/compile (RF-8.2 / RNF-5).
var updateGolden = flag.Bool("update", false, "rewrite golden files from current output")

// goldenLinked and goldenUnlinked are fixed canonical documents exercising both
// branches of the renderer: a document linked to a spec (the full provenance
// block, including the fixed `generated: false` boolean) and an unlinked one (the
// provenance block omitted). Neither embeds volatile data, so the rendered bytes
// are reproducible.
func goldenLinked() Document {
	return Document{Slug: "payments-arch", Title: "Payments Architecture", SpecRef: "payments.md", Body: "The architecture body.\nSecond line."}
}

func goldenUnlinked() Document {
	return Document{Slug: "notes-arch", Title: "Notes Architecture", Body: "Unlinked body."}
}

// TestArchitectureGolden renders the linked and unlinked documents and asserts
// each matches its golden file byte-for-byte. Run with -update to regenerate.
func TestArchitectureGolden(t *testing.T) {
	assertGolden(t, "payments-arch.md", []byte(Render(goldenLinked())))
	assertGolden(t, "notes-arch.md", []byte(Render(goldenUnlinked())))
}

// TestArchitectureRenderDeterministic covers RF-8.2 / RNF-5: rendering the same
// document twice yields identical bytes (no volatile fields).
func TestArchitectureRenderDeterministic(t *testing.T) {
	for _, d := range []Document{goldenLinked(), goldenUnlinked()} {
		if Render(d) != Render(d) {
			t.Errorf("Render is not deterministic for %q", d.Slug)
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
