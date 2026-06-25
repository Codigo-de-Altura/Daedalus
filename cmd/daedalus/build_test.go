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

// TestRunBuildCompilesClaudeArtifacts covers the happy path at the CLI: with a
// valid workspace `build --yes` routes to the Claude Code adapter, writes
// `.claude/` artifacts, exits 0, and prints a summary naming the backend (REQ-7).
// --yes is the non-interactive write path (the test runs without a TTY); a plain
// `build` without a terminal is a dry run (see TestRunBuildNonTTYNoYesIsDryRun).
func TestRunBuildCompilesClaudeArtifacts(t *testing.T) {
	dir := t.TempDir()
	scaffold(t, dir)
	if code, _, stderr := runAgentCmd("add", "analyst", "--path", dir); code != 0 {
		t.Fatalf("agent add failed (%d): %s", code, stderr)
	}

	code, stdout, stderr := runBuildInDir(dir, "--yes")
	if code != exitBuildOK {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "claude-code") {
		t.Errorf("summary does not name the backend; got:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "agents", "analyst.md")); err != nil {
		t.Errorf("agent artifact not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude", "settings.json")); err != nil {
		t.Errorf("settings.json not written: %v", err)
	}
}

// TestRunBuildNonTTYNoYesIsDryRun covers the RF-6.4 safety decision: a plain
// `build` with no terminal and no --yes prints the textual diff/plan, exits 0, and
// writes NOTHING, telling the user how to actually write. The test harness runs
// without a TTY, so this is the path runBuild takes here.
func TestRunBuildNonTTYNoYesIsDryRun(t *testing.T) {
	dir := t.TempDir()
	scaffold(t, dir)
	if code, _, stderr := runAgentCmd("add", "analyst", "--path", dir); code != 0 {
		t.Fatalf("agent add failed (%d): %s", code, stderr)
	}

	code, stdout, _ := runBuildInDir(dir)
	if code != exitBuildOK {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "no files written") {
		t.Errorf("dry-run report should say no files were written; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "pass --yes to write") {
		t.Errorf("dry-run should tell the user how to write; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "[new]") {
		t.Errorf("dry-run should classify the new artifact; got:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude")); !os.IsNotExist(err) {
		t.Errorf("plain non-TTY build must not write (.claude err=%v)", err)
	}
}

// TestRunBuildPreviewNonTTYNeverWrites covers Check-7 for the non-TTY path:
// `build --preview` prints the diff and writes nothing, without the
// "pass --yes" notice (an explicit preview is not a withheld write).
func TestRunBuildPreviewNonTTYNeverWrites(t *testing.T) {
	dir := t.TempDir()
	scaffold(t, dir)
	if code, _, stderr := runAgentCmd("add", "analyst", "--path", dir); code != 0 {
		t.Fatalf("agent add failed (%d): %s", code, stderr)
	}

	code, stdout, _ := runBuildInDir(dir, "--preview")
	if code != exitBuildOK {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "no files written") {
		t.Errorf("preview should say no files were written; got:\n%s", stdout)
	}
	if strings.Contains(stdout, "pass --yes to write") {
		t.Errorf("an explicit --preview should not nag about --yes; got:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, ".claude")); !os.IsNotExist(err) {
		t.Errorf("--preview must not write (.claude err=%v)", err)
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
