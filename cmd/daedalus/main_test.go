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

// TestRunAgentCloneCreatesIndependentDef covers manual-validation Caso 1: `agent
// clone analyst analyst-custom` creates a new definition under the dest id.
func TestRunAgentCloneCreatesIndependentDef(t *testing.T) {
	dir := t.TempDir()

	code, stdout, stderr := runAgentCmd("clone", "analyst", "analyst-custom", "--path", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "Materialized") {
		t.Errorf("stdout does not confirm the clone; got:\n%s", stdout)
	}
	agentDir := filepath.Join(dir, ".daedalus", "agents", "analyst-custom")
	for _, name := range []string{"agent.yaml", "prompt.md"} {
		if _, err := os.Stat(filepath.Join(agentDir, name)); err != nil {
			t.Errorf("expected cloned file %q: %v", name, err)
		}
	}
}

// TestRunAgentEditPersistsAndLeavesBuiltinIntact covers manual-validation Casos
// 2-3: editing the clone (role/prompt/param) persists, and the built-in original
// is unaffected (it is an in-binary literal, so `agent list` still shows its
// original role).
func TestRunAgentEditPersistsAndLeavesBuiltinIntact(t *testing.T) {
	dir := t.TempDir()

	if code, _, stderr := runAgentCmd("clone", "analyst", "analyst-custom", "--path", dir); code != 0 {
		t.Fatalf("clone exit code = %d, want 0; stderr:\n%s", code, stderr)
	}

	code, stdout, stderr := runAgentCmd("edit", "analyst-custom", "--path", dir,
		"--role", "Custom role", "--prompt", "Custom prompt body", "--set-param", "model=custom-x")
	if code != 0 {
		t.Fatalf("edit exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "Edited") {
		t.Errorf("stdout does not confirm the edit; got:\n%s", stdout)
	}

	def, _ := os.ReadFile(filepath.Join(dir, ".daedalus", "agents", "analyst-custom", "agent.yaml"))
	prompt, _ := os.ReadFile(filepath.Join(dir, ".daedalus", "agents", "analyst-custom", "prompt.md"))
	if !strings.Contains(string(def), "Custom role") {
		t.Errorf("edited role not persisted; got:\n%s", def)
	}
	if !strings.Contains(string(def), "model: custom-x") {
		t.Errorf("edited param not persisted; got:\n%s", def)
	}
	if !strings.Contains(string(prompt), "Custom prompt body") {
		t.Errorf("edited prompt not persisted; got:\n%s", prompt)
	}

	// The built-in original is unchanged: list still shows analyst's original role.
	_, listOut, _ := runAgentCmd("list")
	if strings.Contains(listOut, "Custom role") {
		t.Errorf("editing a clone leaked into the built-in catalog listing:\n%s", listOut)
	}
}

// TestRunAgentEditFromPromptFile covers --prompt-file and its precedence over
// --prompt.
func TestRunAgentEditFromPromptFile(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runAgentCmd("clone", "analyst", "analyst-custom", "--path", dir); code != 0 {
		t.Fatalf("clone exit code = %d; stderr:\n%s", code, stderr)
	}

	pf := filepath.Join(dir, "prompt.txt")
	if err := os.WriteFile(pf, []byte("PROMPT FROM FILE"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Both --prompt and --prompt-file given: the file must win.
	code, _, stderr := runAgentCmd("edit", "analyst-custom", "--path", dir,
		"--prompt", "INLINE", "--prompt-file", pf)
	if code != 0 {
		t.Fatalf("edit exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	prompt, _ := os.ReadFile(filepath.Join(dir, ".daedalus", "agents", "analyst-custom", "prompt.md"))
	if !strings.Contains(string(prompt), "PROMPT FROM FILE") || strings.Contains(string(prompt), "INLINE") {
		t.Errorf("--prompt-file did not take precedence over --prompt; got:\n%s", prompt)
	}
}

// TestRunAgentEditInvalidLeavesIntact covers Check 5/CA5: an edit that empties the
// role is rejected (exit 2) with an actionable message, and the file is intact.
func TestRunAgentEditInvalidLeavesIntact(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runAgentCmd("clone", "analyst", "analyst-custom", "--path", dir); code != 0 {
		t.Fatalf("clone exit code = %d; stderr:\n%s", code, stderr)
	}
	defPath := filepath.Join(dir, ".daedalus", "agents", "analyst-custom", "agent.yaml")
	before, _ := os.ReadFile(defPath)

	code, _, stderr := runAgentCmd("edit", "analyst-custom", "--path", dir, "--role", "")
	if code != 2 {
		t.Fatalf("exit code = %d for an invalid edit, want 2", code)
	}
	if !strings.Contains(stderr, "role") {
		t.Errorf("stderr does not name the offending field; got:\n%s", stderr)
	}
	if after, _ := os.ReadFile(defPath); string(after) != string(before) {
		t.Errorf("invalid edit modified the definition on disk")
	}
}

// TestRunAgentEditNoFlagsIsUsageError covers the no-op guard: edit with no edit
// flag is a usage error (exit 2), not a silent rewrite.
func TestRunAgentEditNoFlagsIsUsageError(t *testing.T) {
	dir := t.TempDir()
	if code, _, _ := runAgentCmd("clone", "analyst", "analyst-custom", "--path", dir); code != 0 {
		t.Fatalf("clone failed")
	}
	if code, _, _ := runAgentCmd("edit", "analyst-custom", "--path", dir); code != 2 {
		t.Errorf("exit code = %d for edit with no flags, want 2", code)
	}
}

// TestRunAgentCloneNonDestructive covers manual-validation Caso 4: a second clone
// over the same dest id reports the conflict and does not overwrite.
func TestRunAgentCloneNonDestructive(t *testing.T) {
	dir := t.TempDir()
	if code, _, _ := runAgentCmd("clone", "analyst", "analyst-custom", "--path", dir); code != 0 {
		t.Fatalf("first clone failed")
	}
	prompt := filepath.Join(dir, ".daedalus", "agents", "analyst-custom", "prompt.md")
	const marker = "MANUAL"
	if err := os.WriteFile(prompt, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runAgentCmd("clone", "analyst", "analyst-custom", "--path", dir)
	if code != 0 {
		t.Fatalf("second clone exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "already exists") {
		t.Errorf("stdout does not report the non-destructive conflict; got:\n%s", stdout)
	}
	if b, _ := os.ReadFile(prompt); string(b) != marker {
		t.Errorf("second clone overwrote the manual edit: %q", b)
	}
}

// TestRunAgentClonePreviewWritesNothing covers the dry-run path for clone.
func TestRunAgentClonePreviewWritesNothing(t *testing.T) {
	dir := t.TempDir()
	code, stdout, stderr := runAgentCmd("clone", "analyst", "analyst-custom", "--path", dir, "--preview")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "Preview") {
		t.Errorf("stdout is not a preview; got:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, ".daedalus", "agents", "analyst-custom")); !os.IsNotExist(err) {
		t.Errorf("preview wrote files (stat err=%v), want none", err)
	}
}

// TestRunAgentCloneArgErrors covers the positional-arg and id validation guards.
func TestRunAgentCloneArgErrors(t *testing.T) {
	dir := t.TempDir()
	// Missing dest id.
	if code, _, _ := runAgentCmd("clone", "analyst", "--path", dir); code != 2 {
		t.Errorf("exit code = %d for clone with one id, want 2", code)
	}
	// Non-kebab dest id.
	if code, _, _ := runAgentCmd("clone", "analyst", "Bad_Id", "--path", dir); code != 2 {
		t.Errorf("exit code = %d for non-kebab dest, want 2", code)
	}
	// Unknown source.
	if code, _, _ := runAgentCmd("clone", "ghost", "dest-id", "--path", dir); code != 2 {
		t.Errorf("exit code = %d for unknown source, want 2", code)
	}
}

// TestRunAgentEditUnknownAgent covers editing an agent absent from the workspace.
func TestRunAgentEditUnknownAgent(t *testing.T) {
	dir := t.TempDir()
	code, _, stderr := runAgentCmd("edit", "ghost", "--path", dir, "--role", "x")
	if code != 2 {
		t.Errorf("exit code = %d for editing an absent agent, want 2", code)
	}
	if !strings.Contains(stderr, "ghost") {
		t.Errorf("stderr does not name the absent agent; got:\n%s", stderr)
	}
}

// claudeAgentFile is a minimal valid Claude Code agent for CLI import tests.
const claudeAgentFile = `---
name: imported-agent
description: An imported agent.
tools: Read, Write
model: opus
color: blue
---

# Imported

You are the imported agent.
`

// writeFile writes content under dir/name and returns the path.
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestRunAgentImportFile covers Check 1/2/CA1/CA2: importing a Claude Code file
// creates a canonical agent under .daedalus/agents/.
func TestRunAgentImportFile(t *testing.T) {
	ws := t.TempDir()
	src := writeFile(t, t.TempDir(), "imported-agent.md", claudeAgentFile)

	code, stdout, stderr := runAgentCmd("import", src, "--path", ws)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "imported") {
		t.Errorf("stdout does not confirm the import; got:\n%s", stdout)
	}
	agentDir := filepath.Join(ws, ".daedalus", "agents", "imported-agent")
	for _, name := range []string{"agent.yaml", "prompt.md"} {
		if _, err := os.Stat(filepath.Join(agentDir, name)); err != nil {
			t.Errorf("expected imported file %q: %v", name, err)
		}
	}
}

// TestRunAgentImportInvalid covers Check 3/CA3: an invalid source is reported with
// an actionable error, exit 2, and is not written.
func TestRunAgentImportInvalid(t *testing.T) {
	ws := t.TempDir()
	invalid := "---\nname: broken\ndescription:\n---\n\nbody\n"
	src := writeFile(t, t.TempDir(), "broken.md", invalid)

	code, _, stderr := runAgentCmd("import", src, "--path", ws)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 for an invalid source", code)
	}
	if !strings.Contains(stderr, "role") {
		t.Errorf("stderr does not name the offending field; got:\n%s", stderr)
	}
	if _, err := os.Stat(filepath.Join(ws, ".daedalus", "agents", "broken")); !os.IsNotExist(err) {
		t.Errorf("invalid source created files (stat err=%v), want none", err)
	}
}

// TestRunAgentImportNonDestructive covers Check 4/CA4: importing over an existing
// id reports the conflict and does not overwrite.
func TestRunAgentImportNonDestructive(t *testing.T) {
	ws := t.TempDir()
	src := writeFile(t, t.TempDir(), "imported-agent.md", claudeAgentFile)

	if code, _, _ := runAgentCmd("import", src, "--path", ws); code != 0 {
		t.Fatalf("first import failed")
	}
	prompt := filepath.Join(ws, ".daedalus", "agents", "imported-agent", "prompt.md")
	const marker = "MANUAL"
	if err := os.WriteFile(prompt, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}

	code, stdout, stderr := runAgentCmd("import", src, "--path", ws)
	if code != 0 {
		t.Fatalf("second import exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "already exists") {
		t.Errorf("stdout does not report the conflict; got:\n%s", stdout)
	}
	if b, _ := os.ReadFile(prompt); string(b) != marker {
		t.Errorf("re-import overwrote the manual edit: %q", b)
	}
}

// TestRunAgentImportPreviewWritesNothing covers the dry-run path.
func TestRunAgentImportPreviewWritesNothing(t *testing.T) {
	ws := t.TempDir()
	src := writeFile(t, t.TempDir(), "imported-agent.md", claudeAgentFile)

	code, stdout, stderr := runAgentCmd("import", src, "--path", ws, "--preview")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "Preview") {
		t.Errorf("stdout is not a preview; got:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(ws, ".daedalus", "agents", "imported-agent")); !os.IsNotExist(err) {
		t.Errorf("preview wrote files (stat err=%v), want none", err)
	}
}

// TestRunAgentImportDirectory covers the multi-agent directory path with a mix of
// valid and invalid sources: valid ones import, invalid ones are reported, exit 2.
func TestRunAgentImportDirectory(t *testing.T) {
	ws := t.TempDir()
	src := t.TempDir()
	writeFile(t, src, "one.md", strings.Replace(claudeAgentFile, "imported-agent", "one", 1))
	writeFile(t, src, "two.md", strings.Replace(claudeAgentFile, "imported-agent", "two", 1))
	writeFile(t, src, "bad.md", "---\nname: bad\ndescription:\n---\n\nbody\n")

	code, stdout, stderr := runAgentCmd("import", src, "--path", ws)
	if code != 2 {
		t.Fatalf("exit code = %d, want 2 (mixed valid/invalid); stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "2 imported") {
		t.Errorf("summary does not report 2 imports; got:\n%s", stdout)
	}
	for _, id := range []string{"one", "two"} {
		if _, err := os.Stat(filepath.Join(ws, ".daedalus", "agents", id)); err != nil {
			t.Errorf("valid agent %q not imported despite a bad sibling: %v", id, err)
		}
	}
}

// TestRunAgentImportMissingSource covers the missing-source-path I/O error.
func TestRunAgentImportMissingSource(t *testing.T) {
	ws := t.TempDir()
	code, _, _ := runAgentCmd("import", filepath.Join(ws, "does-not-exist.md"), "--path", ws)
	if code != 1 {
		t.Errorf("exit code = %d for a missing source path, want 1 (I/O error)", code)
	}
}

// TestRunAgentImportNoSource covers the positional-arg guard.
func TestRunAgentImportNoSource(t *testing.T) {
	if code, _, _ := runAgentCmd("import", "--path", t.TempDir()); code != 2 {
		t.Errorf("exit code = %d for import with no source, want 2", code)
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
