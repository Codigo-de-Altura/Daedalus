package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TestReadManifestRoundTrips covers the reader as the inverse of the writer: a
// scaffolded workspace's manifest reads back with the name, version and the
// default backend intact.
func TestReadManifestRoundTrips(t *testing.T) {
	root := t.TempDir()
	if _, err := Create(root); err != nil {
		t.Fatalf("Create: %v", err)
	}

	m, err := ReadManifest(root)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if m.Version != SchemaVersion {
		t.Errorf("version = %q, want %q", m.Version, SchemaVersion)
	}
	if len(m.Backends) != 1 || m.Backends[0] != DefaultBackend {
		t.Errorf("backends = %v, want [%s]", m.Backends, DefaultBackend)
	}
	if m.Name == "" {
		t.Error("name is empty")
	}
}

// TestReadManifestPreservesMultipleBackends ensures the reader returns backends
// in manifest order, which the build orchestration relies on for determinism.
func TestReadManifestPreservesMultipleBackends(t *testing.T) {
	root := t.TempDir()
	path := ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	body := renderManifest(Manifest{
		Name:        "demo",
		Version:     SchemaVersion,
		Backends:    []string{"claude-code", "codex"},
		Conventions: []convention{{Key: "naming", Value: "kebab-case"}},
	})
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	m, err := ReadManifest(root)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	want := []string{"claude-code", "codex"}
	if len(m.Backends) != len(want) {
		t.Fatalf("backends = %v, want %v", m.Backends, want)
	}
	for i := range want {
		if m.Backends[i] != want[i] {
			t.Errorf("backends[%d] = %q, want %q", i, m.Backends[i], want[i])
		}
	}
}

// TestReadManifestNotFound covers the absent-workspace signal the build command
// turns into "run daedalus init first".
func TestReadManifestNotFound(t *testing.T) {
	root := t.TempDir()
	_, err := ReadManifest(root)
	if !errors.Is(err, ErrManifestNotFound) {
		t.Fatalf("err = %v, want ErrManifestNotFound", err)
	}
}

// TestReadManifestMalformed covers a corrupt manifest: it must be rejected as
// malformed, not silently half-read.
func TestReadManifestMalformed(t *testing.T) {
	root := t.TempDir()
	path := ManifestPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	// Missing the required `backends` block.
	if err := os.WriteFile(path, []byte("name: demo\nversion: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ReadManifest(root)
	if !errors.Is(err, ErrManifestMalformed) {
		t.Fatalf("err = %v, want ErrManifestMalformed", err)
	}
}
