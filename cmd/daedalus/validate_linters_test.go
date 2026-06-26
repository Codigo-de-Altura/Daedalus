package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestValidateSurfacesDefinitionLinterFindings covers the wired surface (RF-9.3):
// `daedalus validate` runs the definition linters alongside the conventions check,
// prints a Definitions section, and exits 1 when a definition is invalid even if the
// conventions are clean. We make an agent schema-invalid (empty prompt) and assert
// the command reports it and fails.
func TestValidateSurfacesDefinitionLinterFindings(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runInitInDir(dir); code != 0 {
		t.Fatalf("init failed (%d): %s", code, stderr)
	}

	// A schema-invalid agent (parses, but the prompt body is empty).
	agentDir := filepath.Join(dir, ".daedalus", "agents", "hollow")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "agent.yaml"), []byte("id: hollow\nrole: tester\nprompt: prompt.md\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, "prompt.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, _ := runValidateInDir(t, dir)
	if code != 1 {
		t.Fatalf("validate exit = %d, want 1 (an invalid definition is present)", code)
	}
	if !strings.Contains(stdout, "Definitions:") {
		t.Errorf("stdout has no Definitions section; got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "hollow") {
		t.Errorf("stdout does not name the invalid agent; got:\n%s", stdout)
	}
}

// TestValidateCleanWorkspacePassesBothAxes covers CA6 at the command level: a fresh,
// conformant workspace passes both the conventions and the definition linters and
// exits 0, with the Definitions section reporting all definitions valid.
func TestValidateCleanWorkspacePassesBothAxes(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runInitInDir(dir); code != 0 {
		t.Fatalf("init failed (%d): %s", code, stderr)
	}

	code, stdout, _ := runValidateInDir(t, dir)
	if code != 0 {
		t.Fatalf("validate exit = %d, want 0 on a clean workspace; stdout:\n%s", code, stdout)
	}
	if !strings.Contains(stdout, "Definitions: all agents, workflows and manifest are valid.") {
		t.Errorf("stdout does not confirm valid definitions; got:\n%s", stdout)
	}
}
