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

// runAgent runs the `daedalus agent` subcommand with the given args, capturing
// stdout/stderr so tests can assert on behavior without spawning a process.
func runAgentCmd(args ...string) (code int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	code = runAgent(args, &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// TestRunAgentListIncludesCanonical covers manual-validation Caso 1: `agent list`
// prints at least the five canonical agents to stdout.
func TestRunAgentListIncludesCanonical(t *testing.T) {
	code, stdout, stderr := runAgentCmd("list")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	for _, id := range []string{"analyst", "architect", "planner", "validator", "documenter"} {
		if !strings.Contains(stdout, id) {
			t.Errorf("agent list missing canonical agent %q; got:\n%s", id, stdout)
		}
	}
}

// TestRunAgentAddCreatesFiles covers manual-validation Caso 2: `agent add analyst`
// materializes the agent's canonical files under .daedalus/agents/.
func TestRunAgentAddCreatesFiles(t *testing.T) {
	dir := t.TempDir()

	code, stdout, stderr := runAgentCmd("add", "analyst", "--path", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "Materialized") {
		t.Errorf("stdout does not confirm materialization; got:\n%s", stdout)
	}

	agentDir := filepath.Join(dir, ".daedalus", "agents", "analyst")
	for _, name := range []string{"agent.yaml", "prompt.md"} {
		if _, err := os.Stat(filepath.Join(agentDir, name)); err != nil {
			t.Errorf("expected materialized file %q: %v", name, err)
		}
	}
}

// TestRunAgentAddIsNonDestructive covers manual-validation Caso 3: a second add of
// the same agent does not overwrite it; the conflict is reported and a hand-edited
// file survives verbatim.
func TestRunAgentAddIsNonDestructive(t *testing.T) {
	dir := t.TempDir()

	if code, _, stderr := runAgentCmd("add", "analyst", "--path", dir); code != 0 {
		t.Fatalf("first add exit code = %d, want 0; stderr:\n%s", code, stderr)
	}

	// Hand-edit the prompt: it is now the user's source of truth and must survive.
	prompt := filepath.Join(dir, ".daedalus", "agents", "analyst", "prompt.md")
	const marker = "MANUAL-EDIT-CLI"
	if err := os.WriteFile(prompt, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runAgentCmd("add", "analyst", "--path", dir)
	if code != 0 {
		t.Fatalf("second add exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "already exists") || !strings.Contains(stdout, "not overwritten") {
		t.Errorf("stdout does not report the non-destructive conflict; got:\n%s", stdout)
	}
	if b, _ := os.ReadFile(prompt); string(b) != marker {
		t.Errorf("second add overwrote the manual edit: %q", b)
	}
}

// TestRunAgentAddPreviewWritesNothing covers the dry-run path: --preview reports
// the files that would be created without touching the filesystem.
func TestRunAgentAddPreviewWritesNothing(t *testing.T) {
	dir := t.TempDir()

	code, stdout, stderr := runAgentCmd("add", "analyst", "--path", dir, "--preview")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "Preview") {
		t.Errorf("stdout is not a preview; got:\n%s", stdout)
	}
	// Nothing must have been written: no agents directory at all.
	if _, err := os.Stat(filepath.Join(dir, ".daedalus", "agents", "analyst")); !os.IsNotExist(err) {
		t.Errorf("preview wrote files to disk (stat err=%v), want none", err)
	}
}

// TestRunAgentAddUnknownID covers the failure path: an unknown agent id is a
// usage error (exit 2) with an actionable stderr message and no files written.
func TestRunAgentAddUnknownID(t *testing.T) {
	dir := t.TempDir()

	code, _, stderr := runAgentCmd("add", "does-not-exist", "--path", dir)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 for an unknown agent", code)
	}
	if !strings.Contains(stderr, "does-not-exist") {
		t.Errorf("stderr does not name the unknown agent; got:\n%s", stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, ".daedalus", "agents")); !os.IsNotExist(err) {
		t.Errorf("an unknown agent created files (stat err=%v), want none", err)
	}
}

// TestRunAgentNoOperation covers the dispatcher guard: `agent` with no operation
// is a usage error (exit 2).
func TestRunAgentNoOperation(t *testing.T) {
	if code, _, _ := runAgentCmd(); code != 2 {
		t.Errorf("exit code = %d for `agent` with no operation, want 2", code)
	}
	if code, _, _ := runAgentCmd("bogus"); code != 2 {
		t.Errorf("exit code = %d for an unknown operation, want 2", code)
	}
}

// TestRunAgentAddMissingID covers the positional-argument guard: `add` without an
// id is a usage error (exit 2).
func TestRunAgentAddMissingID(t *testing.T) {
	if code, _, _ := runAgentCmd("add", "--path", t.TempDir()); code != 2 {
		t.Errorf("exit code = %d for `add` with no id, want 2", code)
	}
}

// TestRunAgentDispatchedFromRun covers the run() wiring: the `agent` token routes
// to runAgent rather than the unknown-command path.
func TestRunAgentDispatchedFromRun(t *testing.T) {
	if code := run([]string{"agent", "list"}); code != 0 {
		t.Errorf("run([agent list]) = %d, want 0", code)
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
