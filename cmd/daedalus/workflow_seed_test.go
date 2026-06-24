package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

// runWorkflowInDir runs `daedalus workflow <op> <args...> --path dir`, capturing
// stdout/stderr, so tests can drive the workflow subcommand without spawning a
// process. The --path flag is appended so it applies regardless of the operation.
func runWorkflowInDir(dir string, opAndArgs ...string) (code int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	args := append(append([]string{}, opAndArgs...), "--path", dir)
	code = runWorkflow(args, &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// seedPath returns the path the factory workflow is seeded to under dir.
func seedPath(dir string) string {
	return filepath.Join(dir, ".daedalus", "workflows",
		workflows.DefaultWorkflowName+workflows.FileExt)
}

// TestInitSeedsDefaultWorkflow covers CA1: a fresh `daedalus init` leaves
// .daedalus/workflows/sdd-default.yaml with the deterministic factory content, and
// the run reports it.
func TestInitSeedsDefaultWorkflow(t *testing.T) {
	dir := t.TempDir()

	code, stdout, stderr := runInitInDir(dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}

	content, err := os.ReadFile(seedPath(dir))
	if err != nil {
		t.Fatalf("factory workflow not seeded: %v", err)
	}
	if got, want := string(content), workflows.Render(workflows.DefaultWorkflow()); got != want {
		t.Errorf("seeded content is not the deterministic default\ngot:\n%s\nwant:\n%s", got, want)
	}
	if !strings.Contains(stdout, "Seeded factory workflow") {
		t.Errorf("init did not report seeding; stdout:\n%s", stdout)
	}
}

// TestInitSeedIsNonDestructive covers the re-run / user-edit case: re-running init
// does not overwrite an existing sdd-default.yaml, and a user's manual edit
// survives.
func TestInitSeedIsNonDestructive(t *testing.T) {
	dir := t.TempDir()

	if code, _, stderr := runInitInDir(dir); code != 0 {
		t.Fatalf("first init exit = %d; stderr:\n%s", code, stderr)
	}

	// Simulate a user editing the seeded workflow.
	edited := "phases: []\n"
	if err := os.WriteFile(seedPath(dir), []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	// Re-run init: it must not clobber the edited file and must say so.
	code, stdout, stderr := runInitInDir(dir)
	if code != 0 {
		t.Fatalf("second init exit = %d; stderr:\n%s", code, stderr)
	}
	got, err := os.ReadFile(seedPath(dir))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != edited {
		t.Errorf("re-run clobbered the user-edited workflow\nwant:\n%s\ngot:\n%s", edited, got)
	}
	if !strings.Contains(stdout, "already present") {
		t.Errorf("re-run did not report the seed as already present; stdout:\n%s", stdout)
	}
}

// TestInitPreviewDoesNotSeed covers the preview contract: --preview writes nothing,
// so no sdd-default.yaml appears, but the preview mentions it would be seeded.
func TestInitPreviewDoesNotSeed(t *testing.T) {
	dir := t.TempDir()

	code, stdout, stderr := runInitInDir(dir, "--preview")
	if code != 0 {
		t.Fatalf("preview exit = %d; stderr:\n%s", code, stderr)
	}
	if _, err := os.Stat(seedPath(dir)); !os.IsNotExist(err) {
		t.Errorf("preview must not write the factory workflow (stat err = %v)", err)
	}
	if !strings.Contains(stdout, "factory workflow") {
		t.Errorf("preview did not mention the factory workflow; stdout:\n%s", stdout)
	}
}

// TestSeededWorkflowValidatesClean ties the seed to the validator (CA6 at the CLI
// layer): the seeded file loads and `workflow validate` reports it valid with the
// built-in agents, exit 0.
func TestSeededWorkflowValidatesClean(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runInitInDir(dir); code != 0 {
		t.Fatalf("init exit = %d; stderr:\n%s", code, stderr)
	}

	code, stdout, stderr := runWorkflowInDir(dir, "validate", workflows.DefaultWorkflowName)
	if code != 0 {
		t.Fatalf("validate exit = %d, want 0; stdout:\n%s\nstderr:\n%s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "semantically valid") {
		t.Errorf("validate did not report valid; stdout:\n%s", stdout)
	}
}
