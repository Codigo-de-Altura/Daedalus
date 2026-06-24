package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestNormalizeBackendsDefaultsWhenEmpty covers R3/CA2: with no explicit
// selection (the non-interactive default path) the result is exactly the MVP
// default backend.
func TestNormalizeBackendsDefaultsWhenEmpty(t *testing.T) {
	got, err := NormalizeBackends(nil)
	if err != nil {
		t.Fatalf("NormalizeBackends(nil): %v", err)
	}
	if len(got) != 1 || got[0] != DefaultBackend {
		t.Errorf("NormalizeBackends(nil) = %v, want [%s]", got, DefaultBackend)
	}

	// An empty (non-nil) slice must default the same way as nil, so callers that
	// build a slice and find nothing to add still get the default.
	if got, err := NormalizeBackends([]string{}); err != nil || len(got) != 1 || got[0] != DefaultBackend {
		t.Errorf("NormalizeBackends([]) = %v, %v; want [%s], nil", got, err, DefaultBackend)
	}
}

// TestNormalizeBackendsDefaultIsFreshSlice guards against the default aliasing
// the package-level SupportedBackends: a caller mutating the returned slice must
// not corrupt the supported set for every later call.
func TestNormalizeBackendsDefaultIsFreshSlice(t *testing.T) {
	got, err := NormalizeBackends(nil)
	if err != nil {
		t.Fatalf("NormalizeBackends(nil): %v", err)
	}
	got[0] = "mutated"
	if SupportedBackends[0] != DefaultBackend {
		t.Errorf("mutating the returned default corrupted SupportedBackends: %v", SupportedBackends)
	}
}

// TestNormalizeBackendsAcceptsSupported covers CA3: an explicit, supported
// selection is accepted and returned verbatim.
func TestNormalizeBackendsAcceptsSupported(t *testing.T) {
	got, err := NormalizeBackends([]string{DefaultBackend})
	if err != nil {
		t.Fatalf("NormalizeBackends(%q): %v", DefaultBackend, err)
	}
	if len(got) != 1 || got[0] != DefaultBackend {
		t.Errorf("got %v, want [%s]", got, DefaultBackend)
	}
}

// TestNormalizeBackendsRejectsUnsupported covers R5/CA4: an unsupported backend
// is rejected with the sentinel error, and the message names the offending value
// so the user knows what to fix. nil result prevents persisting a bad slice.
func TestNormalizeBackendsRejectsUnsupported(t *testing.T) {
	got, err := NormalizeBackends([]string{"foo"})
	if err == nil {
		t.Fatalf("NormalizeBackends([foo]) succeeded, want error")
	}
	if !errors.Is(err, ErrUnsupportedBackend) {
		t.Errorf("error %v is not ErrUnsupportedBackend", err)
	}
	if !strings.Contains(err.Error(), "foo") {
		t.Errorf("error %q does not name the offending backend", err)
	}
	if got != nil {
		t.Errorf("got %v on error, want nil so a bad selection is never persisted", got)
	}
}

// TestNormalizeBackendsRejectsMixedWithSupported covers the multi-input path
// (R6): one unsupported entry rejects the whole selection rather than silently
// recording only the valid ones, which would persist a selection the user did
// not ask for.
func TestNormalizeBackendsRejectsMixedWithSupported(t *testing.T) {
	if _, err := NormalizeBackends([]string{DefaultBackend, "foo"}); !errors.Is(err, ErrUnsupportedBackend) {
		t.Errorf("mixed valid+invalid selection: err = %v, want ErrUnsupportedBackend", err)
	}
}

// TestNormalizeBackendsDeduplicatesPreservingOrder covers R6/R7: a multi-backend
// input is deduplicated while keeping first-seen order, so the manifest is
// stable regardless of accidental repeats.
func TestNormalizeBackendsDeduplicatesPreservingOrder(t *testing.T) {
	got, err := NormalizeBackends([]string{DefaultBackend, DefaultBackend})
	if err != nil {
		t.Fatalf("NormalizeBackends: %v", err)
	}
	if len(got) != 1 || got[0] != DefaultBackend {
		t.Errorf("got %v, want a single %s (duplicates collapsed)", got, DefaultBackend)
	}
}

// TestIsSupportedBackend pins the membership predicate the CLI relies on.
func TestIsSupportedBackend(t *testing.T) {
	if !IsSupportedBackend(DefaultBackend) {
		t.Errorf("IsSupportedBackend(%q) = false, want true", DefaultBackend)
	}
	if IsSupportedBackend("foo") {
		t.Errorf("IsSupportedBackend(foo) = true, want false")
	}
}

// TestDetectWithOptionsRecordsSelectedBackend covers CA1/CA3 at the package
// boundary: an explicit selection rides the Plan to the manifest on disk.
func TestDetectWithOptionsRecordsSelectedBackend(t *testing.T) {
	root := t.TempDir()

	plan, err := DetectWithOptions(root, Options{Backends: []string{DefaultBackend}})
	if err != nil {
		t.Fatalf("DetectWithOptions: %v", err)
	}
	if len(plan.Backends) != 1 || plan.Backends[0] != DefaultBackend {
		t.Errorf("plan.Backends = %v, want [%s]", plan.Backends, DefaultBackend)
	}
	if _, err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	manifest := readManifest(t, root)
	if !strings.Contains(manifest, "- "+DefaultBackend) {
		t.Errorf("manifest does not record selected backend %q; got:\n%s", DefaultBackend, manifest)
	}
}

// TestDetectWithOptionsDefaultsBackend covers CA2: a zero Options (the Detect
// path) records the MVP default.
func TestDetectWithOptionsDefaultsBackend(t *testing.T) {
	root := t.TempDir()

	plan, err := DetectWithOptions(root, Options{})
	if err != nil {
		t.Fatalf("DetectWithOptions: %v", err)
	}
	if len(plan.Backends) != 1 || plan.Backends[0] != DefaultBackend {
		t.Errorf("plan.Backends = %v, want default [%s]", plan.Backends, DefaultBackend)
	}
}

// TestDetectWithOptionsRejectsUnsupportedBeforeWriting covers R5/CA4 end-to-end:
// an unsupported backend fails detection and leaves the filesystem untouched —
// no .daedalus/ directory and therefore no invalid value persisted.
func TestDetectWithOptionsRejectsUnsupportedBeforeWriting(t *testing.T) {
	root := t.TempDir()

	if _, err := DetectWithOptions(root, Options{Backends: []string{"foo"}}); !errors.Is(err, ErrUnsupportedBackend) {
		t.Fatalf("DetectWithOptions(foo): err = %v, want ErrUnsupportedBackend", err)
	}
	if _, err := os.Stat(filepath.Join(root, Name)); !os.IsNotExist(err) {
		t.Errorf("workspace was created despite an unsupported backend (stat err=%v)", err)
	}
}

// TestSelectedBackendManifestIsDeterministic covers R7/CA6: the same selection
// over two independent roots that share a derived project name produces a
// byte-identical manifest. The fixed-name-subdir trick neutralizes the
// project-name input so only the backend selection is under test.
func TestSelectedBackendManifestIsDeterministic(t *testing.T) {
	render := func() string {
		root := fixedNameRoot(t, "fixed-name")
		plan, err := DetectWithOptions(root, Options{Backends: []string{DefaultBackend}})
		if err != nil {
			t.Fatalf("DetectWithOptions: %v", err)
		}
		if _, err := plan.Apply(); err != nil {
			t.Fatalf("Apply: %v", err)
		}
		return readManifest(t, root)
	}

	if a, b := render(), render(); a != b {
		t.Errorf("manifest not deterministic for the same backend selection:\n--- a ---\n%s\n--- b ---\n%s", a, b)
	}
}

// TestSelectedBackendIsNonDestructive covers R8 (consistency with 01-02): an
// explicit backend selection must not overwrite a manifest that already exists.
func TestSelectedBackendIsNonDestructive(t *testing.T) {
	root := t.TempDir()

	if _, err := Create(root); err != nil {
		t.Fatalf("Create: %v", err)
	}
	manifestPath := filepath.Join(root, Name, "daedalus.yaml")
	const marker = "name: hand-edited"
	if err := os.WriteFile(manifestPath, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}

	// Re-run init with an explicit selection; the existing manifest must survive
	// verbatim — the --backend flag does not break the 01-02 guarantee.
	plan, err := DetectWithOptions(root, Options{Backends: []string{DefaultBackend}})
	if err != nil {
		t.Fatalf("DetectWithOptions: %v", err)
	}
	if _, err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if b, _ := os.ReadFile(manifestPath); string(b) != marker {
		t.Errorf("existing manifest was overwritten by a backend selection: %q", b)
	}
}

// readManifest reads the rendered daedalus.yaml under root's workspace.
func readManifest(t *testing.T, root string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, Name, "daedalus.yaml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return string(b)
}
