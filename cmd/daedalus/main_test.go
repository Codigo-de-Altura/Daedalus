package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runInitInDir runs `daedalus init` against dir with the given extra flags,
// capturing stdout/stderr. It returns the exit code and both streams so tests
// can assert on behavior without spawning a process.
func runInitInDir(dir string, extra ...string) (code int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	args := append([]string{"--path", dir}, extra...)
	code = runInit(args, &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// readManifest reads the generated manifest under dir, failing the test if it is
// absent.
func readManifest(t *testing.T, dir string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, ".daedalus", "daedalus.yaml"))
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	return string(b)
}

// TestRunInitDefaultRecordsClaudeCode covers CA2/check1: init with no --backend
// records the MVP default in the manifest.
func TestRunInitDefaultRecordsClaudeCode(t *testing.T) {
	dir := t.TempDir()

	code, _, stderr := runInitInDir(dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if manifest := readManifest(t, dir); !strings.Contains(manifest, "- claude-code") {
		t.Errorf("default manifest missing claude-code backend; got:\n%s", manifest)
	}
}

// TestRunInitExplicitBackendRecorded covers CA1/CA3/check2: choosing claude-code
// explicitly records it in the manifest.
func TestRunInitExplicitBackendRecorded(t *testing.T) {
	dir := t.TempDir()

	code, _, stderr := runInitInDir(dir, "--backend", "claude-code")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if manifest := readManifest(t, dir); !strings.Contains(manifest, "- claude-code") {
		t.Errorf("explicit manifest missing claude-code backend; got:\n%s", manifest)
	}
}

// TestRunInitUnsupportedBackendRejected covers R5/CA4/check3: an unsupported
// backend exits non-zero with a clear stderr message and writes nothing — the
// workspace must not exist, so no invalid value can leak into a manifest.
func TestRunInitUnsupportedBackendRejected(t *testing.T) {
	dir := t.TempDir()

	code, _, stderr := runInitInDir(dir, "--backend", "foo")
	if code == 0 {
		t.Fatalf("exit code = 0 for an unsupported backend, want non-zero")
	}
	// Usage-style errors use exit code 2 like the other CLI rejections.
	if code != 2 {
		t.Errorf("exit code = %d, want 2 for a usage error", code)
	}
	if !strings.Contains(stderr, "foo") {
		t.Errorf("stderr does not name the offending backend; got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "claude-code") {
		t.Errorf("stderr does not list the supported backend(s); got:\n%s", stderr)
	}
	// The filesystem must be untouched: no .daedalus/ at all.
	if _, err := os.Stat(filepath.Join(dir, ".daedalus")); !os.IsNotExist(err) {
		t.Errorf("workspace was created despite an unsupported backend (stat err=%v)", err)
	}
}

// TestRunInitMultiBackendInputShape covers R6: the --backend flag accepts a
// comma-separated list (the multi-backend shape) and records the selection. The
// MVP set is a single backend, so a repeated claude-code collapses to one entry,
// proving the input path parses lists while the supported set stays MVP.
func TestRunInitMultiBackendInputShape(t *testing.T) {
	dir := t.TempDir()

	code, _, stderr := runInitInDir(dir, "--backend", "claude-code, claude-code")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	manifest := readManifest(t, dir)
	if got := strings.Count(manifest, "- claude-code"); got != 1 {
		t.Errorf("manifest lists claude-code %d times, want 1 (deduplicated); got:\n%s", got, manifest)
	}
}
