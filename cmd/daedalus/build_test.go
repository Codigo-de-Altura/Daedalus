package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runBuildInDir runs `daedalus build` against dir with extra flags, capturing
// stdout/stderr, so tests assert on behavior without spawning a process.
func runBuildInDir(dir string, extra ...string) (code int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	args := append([]string{"--path", dir}, extra...)
	code = runBuild(args, &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// scaffold runs `daedalus init` so the build has a workspace to read.
func scaffold(t *testing.T, dir string) {
	t.Helper()
	if code, _, stderr := runInitInDir(dir); code != 0 {
		t.Fatalf("init failed (%d): %s", code, stderr)
	}
}

// TestRunBuildMissingWorkspace covers REQ-2/REQ-8: building where there is no
// workspace exits with the compile/write code and an actionable message, and
// writes nothing.
func TestRunBuildMissingWorkspace(t *testing.T) {
	dir := t.TempDir()

	code, _, stderr := runBuildInDir(dir)
	if code != exitBuildCompile {
		t.Fatalf("exit code = %d, want %d; stderr:\n%s", code, exitBuildCompile, stderr)
	}
	if !strings.Contains(stderr, "init") {
		t.Errorf("stderr does not point at 'daedalus init': %s", stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude")); !os.IsNotExist(err) {
		t.Errorf("build wrote artifacts despite no workspace (err=%v)", err)
	}
}

// TestRunBuildInvalidDefinition covers REQ-3/REQ-8: an invalid canonical
// definition aborts with the validation exit code (distinct from the compile
// code) and writes nothing.
func TestRunBuildInvalidDefinition(t *testing.T) {
	dir := t.TempDir()
	scaffold(t, dir)
	// Corrupt an agent (missing role) so validation fails.
	agentDir := filepath.Join(dir, ".daedalus", "agents", "broken")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "agent.yaml"), []byte("id: broken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "prompt.md"), []byte("body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, _, stderr := runBuildInDir(dir)
	if code != exitBuildValidation {
		t.Fatalf("exit code = %d, want %d (validation); stderr:\n%s", code, exitBuildValidation, stderr)
	}
	if code == exitBuildCompile {
		t.Error("validation error must not share the compile/write exit code")
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude")); !os.IsNotExist(err) {
		t.Errorf("build wrote artifacts despite an invalid definition (err=%v)", err)
	}
}

// TestRunBuildNotYetImplementedAdapter documents the 06-01/06-02 boundary at the
// CLI: with a valid workspace the build routes to the claude stub, which reports
// it is not implemented — a compile error (not a validation error).
func TestRunBuildNotYetImplementedAdapter(t *testing.T) {
	dir := t.TempDir()
	scaffold(t, dir)
	if code, _, stderr := runAgentCmd("add", "analyst", "--path", dir); code != 0 {
		t.Fatalf("agent add failed (%d): %s", code, stderr)
	}

	code, _, stderr := runBuildInDir(dir)
	if code != exitBuildCompile {
		t.Fatalf("exit code = %d, want %d (compile); stderr:\n%s", code, exitBuildCompile, stderr)
	}
	if !strings.Contains(stderr, "build failed") {
		t.Errorf("stderr does not read as a build failure: %s", stderr)
	}
}

// TestBuildSyncAliasRoutesIdentically covers REQ-1: `sync` dispatches to the same
// handler as `build`. We assert via run() that both verbs reach runBuild (same
// exit code and stderr for the same missing-workspace input).
func TestBuildSyncAliasRoutesIdentically(t *testing.T) {
	dir := t.TempDir()

	var bOut, bErr bytes.Buffer
	buildCode := runBuild([]string{"--path", dir}, &bOut, &bErr)

	// run() must route both "build" and "sync" into runBuild; exercise the alias
	// path through the top-level dispatcher by confirming it is a known command
	// (an unknown command would exit 2 with "unknown command").
	if got := run([]string{"sync", "--path", dir}); got != buildCode {
		t.Errorf("sync exit code = %d, want %d (same as build)", got, buildCode)
	}
}

// TestRunBuildHelp covers the usage surface: --help exits 0 and prints the alias.
func TestRunBuildHelp(t *testing.T) {
	var outBuf, errBuf bytes.Buffer
	code := runBuild([]string{"--help"}, &outBuf, &errBuf)
	if code != exitBuildOK {
		t.Fatalf("--help exit code = %d, want 0", code)
	}
	if !strings.Contains(errBuf.String(), "sync") {
		t.Errorf("usage does not mention the sync alias:\n%s", errBuf.String())
	}
}
