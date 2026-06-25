package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests pin the DECISION PATHS of `init` at the CLI orchestration boundary
// (runInit), which the spec for ticket 09-02 calls out explicitly (CA1: "los
// caminos de decisión de init"): create-from-scratch vs. non-destructive upgrade
// vs. already-complete no-op, and the --preview dry run that must write nothing.
// The underlying workspace package already unit-tests Plan/Apply; these assert the
// branch the COMMAND takes and the wording it reports, which is the user-facing
// contract a refactor could silently break.

// TestRunInitCreatesFromScratch covers the create branch: over a directory with no
// .daedalus/, init scaffolds the workspace and reports a from-scratch creation.
func TestRunInitCreatesFromScratch(t *testing.T) {
	dir := t.TempDir()

	code, stdout, stderr := runInitInDir(dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "from scratch") {
		t.Errorf("stdout does not report a from-scratch creation; got:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, ".daedalus", "daedalus.yaml")); err != nil {
		t.Errorf("workspace manifest was not created: %v", err)
	}
}

// TestRunInitUpgradesExistingNonDestructively covers the upgrade branch: re-running
// init over a workspace missing one required subdirectory completes ONLY the
// missing piece and reports an upgrade, without disturbing the rest.
func TestRunInitUpgradesExistingNonDestructively(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runInitInDir(dir); code != 0 {
		t.Fatalf("seed init failed (%d): %s", code, stderr)
	}

	// Drop a required subdirectory so the next init has exactly one thing to add,
	// forcing the upgrade (not create, not no-op) branch.
	agentsDir := filepath.Join(dir, ".daedalus", "agents")
	if err := os.RemoveAll(agentsDir); err != nil {
		t.Fatal(err)
	}
	// A user file elsewhere in the workspace must survive the upgrade untouched.
	sentinel := filepath.Join(dir, ".daedalus", "prompts", "keep.md")
	if err := os.WriteFile(sentinel, []byte("manual content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runInitInDir(dir)
	if code != 0 {
		t.Fatalf("upgrade init failed (%d): %s", code, stderr)
	}
	if !strings.Contains(stdout, "Upgraded existing") {
		t.Errorf("stdout does not report an upgrade; got:\n%s", stdout)
	}
	if _, err := os.Stat(agentsDir); err != nil {
		t.Errorf("upgrade did not restore the missing agents/ directory: %v", err)
	}
	got, err := os.ReadFile(sentinel)
	if err != nil || string(got) != "manual content\n" {
		t.Errorf("upgrade disturbed a manual file: content=%q err=%v", string(got), err)
	}
}

// TestRunInitAlreadyCompleteIsNoOp covers the no-op branch: re-running init over a
// complete workspace reports "already complete" and adds nothing.
func TestRunInitAlreadyCompleteIsNoOp(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runInitInDir(dir); code != 0 {
		t.Fatalf("seed init failed (%d): %s", code, stderr)
	}

	code, stdout, stderr := runInitInDir(dir)
	if code != 0 {
		t.Fatalf("second init failed (%d): %s", code, stderr)
	}
	if !strings.Contains(stdout, "already complete") {
		t.Errorf("stdout does not report an already-complete no-op; got:\n%s", stdout)
	}
}

// TestRunInitPreviewWritesNothing covers the --preview decision path: a fresh
// preview reports the proposed changes but leaves the filesystem untouched (no
// .daedalus/ is created), so a dry run can never mutate the target.
func TestRunInitPreviewWritesNothing(t *testing.T) {
	dir := t.TempDir()

	code, stdout, stderr := runInitInDir(dir, "--preview")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "Preview") {
		t.Errorf("stdout is not a preview; got:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, ".daedalus")); !os.IsNotExist(err) {
		t.Errorf("preview created a workspace despite being a dry run (stat err=%v)", err)
	}
}
