package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// runPromptCmd runs the `daedalus prompt` subcommand with the given args,
// capturing stdout/stderr so tests can assert on behavior without spawning a
// process.
func runPromptCmd(args ...string) (code int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	code = runPrompt(args, &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// promptFile reads a persisted prompt file under dir, failing if absent.
func promptFile(t *testing.T, dir, id string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, ".daedalus", "prompts", id+".md"))
	if err != nil {
		t.Fatalf("read prompt %q: %v", id, err)
	}
	return string(b)
}

// TestRunPromptCreateGlobal covers manual-validation Caso 1: creating a global
// prompt persists <id>.md with kind: global.
func TestRunPromptCreateGlobal(t *testing.T) {
	dir := t.TempDir()

	code, stdout, stderr := runPromptCmd("create", "project-style", "--kind", "global",
		"--title", "Project Style", "--body", "Use English.", "--path", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "Created prompt") {
		t.Errorf("stdout does not confirm creation; got:\n%s", stdout)
	}
	if content := promptFile(t, dir, "project-style"); !strings.Contains(content, "kind: global") {
		t.Errorf("created file missing kind: global; got:\n%s", content)
	}
}

// TestRunPromptCreateShared covers manual-validation Caso 2: a shared prompt is
// persisted and the global one (created first) is left untouched.
func TestRunPromptCreateShared(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runPromptCmd("create", "project-style", "--kind", "global",
		"--title", "Style", "--body", "x", "--path", dir); code != 0 {
		t.Fatalf("setup create failed: %s", stderr)
	}
	before := promptFile(t, dir, "project-style")

	code, _, stderr := runPromptCmd("create", "glossary", "--kind", "shared",
		"--title", "Glossary", "--body", "terms", "--path", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if content := promptFile(t, dir, "glossary"); !strings.Contains(content, "kind: shared") {
		t.Errorf("created file missing kind: shared; got:\n%s", content)
	}
	if after := promptFile(t, dir, "project-style"); after != before {
		t.Errorf("creating glossary altered project-style.md")
	}
}

// TestRunPromptListFilter covers manual-validation Caso 3 and the --kind filter.
func TestRunPromptListFilter(t *testing.T) {
	dir := t.TempDir()
	runPromptCmd("create", "project-style", "--kind", "global", "--title", "Style", "--body", "x", "--path", dir)
	runPromptCmd("create", "glossary", "--kind", "shared", "--title", "Glossary", "--body", "y", "--path", dir)

	code, stdout, stderr := runPromptCmd("list", "--path", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "project-style") || !strings.Contains(stdout, "glossary") {
		t.Errorf("list missing prompts; got:\n%s", stdout)
	}

	code, stdout, _ = runPromptCmd("list", "--kind", "shared", "--path", dir)
	if code != 0 {
		t.Fatalf("filtered list exit = %d", code)
	}
	if strings.Contains(stdout, "project-style") || !strings.Contains(stdout, "glossary") {
		t.Errorf("shared filter wrong; got:\n%s", stdout)
	}

	// An invalid kind filter is a usage error.
	if code, _, _ := runPromptCmd("list", "--kind", "weird", "--path", dir); code != 2 {
		t.Errorf("invalid --kind exit = %d, want 2", code)
	}
}

// TestRunPromptEditPreservesOthers covers manual-validation Caso 4: editing one
// prompt's body leaves the other byte-identical.
func TestRunPromptEditPreservesOthers(t *testing.T) {
	dir := t.TempDir()
	runPromptCmd("create", "project-style", "--kind", "global", "--title", "Style", "--body", "x", "--path", dir)
	runPromptCmd("create", "glossary", "--kind", "shared", "--title", "Glossary", "--body", "terms", "--path", dir)
	styleBefore := promptFile(t, dir, "project-style")

	code, _, stderr := runPromptCmd("edit", "glossary", "--body", "terms\nmore", "--path", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(promptFile(t, dir, "glossary"), "more") {
		t.Errorf("edit did not persist new body")
	}
	if promptFile(t, dir, "project-style") != styleBefore {
		t.Errorf("editing glossary altered project-style.md")
	}
}

// TestRunPromptDuplicateRejected covers manual-validation Caso 5: a duplicate id
// is a usage error and does not overwrite.
func TestRunPromptDuplicateRejected(t *testing.T) {
	dir := t.TempDir()
	runPromptCmd("create", "glossary", "--kind", "shared", "--title", "Glossary", "--body", "original", "--path", dir)
	before := promptFile(t, dir, "glossary")

	code, _, stderr := runPromptCmd("create", "glossary", "--kind", "global",
		"--title", "Other", "--body", "overwrite", "--path", dir)
	if code != 2 {
		t.Fatalf("duplicate create exit = %d, want 2; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stderr, "already exists") {
		t.Errorf("stderr does not explain the conflict; got:\n%s", stderr)
	}
	if promptFile(t, dir, "glossary") != before {
		t.Errorf("duplicate create overwrote the original file")
	}
}

// TestRunPromptRemove covers manual-validation Caso 6: remove deletes only the
// named prompt's file.
func TestRunPromptRemove(t *testing.T) {
	dir := t.TempDir()
	runPromptCmd("create", "project-style", "--kind", "global", "--title", "Style", "--body", "x", "--path", dir)
	runPromptCmd("create", "glossary", "--kind", "shared", "--title", "Glossary", "--body", "y", "--path", dir)

	code, _, stderr := runPromptCmd("remove", "project-style", "--path", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(dir, ".daedalus", "prompts", "project-style.md")); !os.IsNotExist(err) {
		t.Errorf("removed file still present (stat err=%v)", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".daedalus", "prompts", "glossary.md")); err != nil {
		t.Errorf("remove deleted the wrong file: %v", err)
	}

	// Removing an absent prompt is a usage error.
	if code, _, _ := runPromptCmd("remove", "project-style", "--path", dir); code != 2 {
		t.Errorf("removing absent prompt exit = %d, want 2", code)
	}
}

// TestRunPromptInvalidID covers Check 8 at the CLI boundary: a non-kebab-case id
// is rejected and nothing is written.
func TestRunPromptInvalidID(t *testing.T) {
	dir := t.TempDir()
	code, _, stderr := runPromptCmd("create", "Bad-ID", "--kind", "global", "--title", "T", "--path", dir)
	if code != 2 {
		t.Fatalf("invalid id exit = %d, want 2; stderr:\n%s", code, stderr)
	}
	if entries, err := os.ReadDir(filepath.Join(dir, ".daedalus", "prompts")); err == nil {
		if len(entries) != 0 {
			t.Errorf("invalid id created files: %d", len(entries))
		}
	}
}

// TestRunPromptShow covers the show verb: it prints the file content verbatim.
func TestRunPromptShow(t *testing.T) {
	dir := t.TempDir()
	runPromptCmd("create", "glossary", "--kind", "shared", "--title", "Glossary", "--body", "terms", "--path", dir)

	code, stdout, stderr := runPromptCmd("show", "glossary", "--path", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if stdout != promptFile(t, dir, "glossary") {
		t.Errorf("show output is not the verbatim file content\ngot:\n%s", stdout)
	}
}

// TestRunPromptRenderComposes covers the render verb: it resolves inclusions and
// prints the composed text (distinct from show, which is raw).
func TestRunPromptRenderComposes(t *testing.T) {
	dir := t.TempDir()
	runPromptCmd("create", "glossary", "--kind", "shared", "--title", "Glossary", "--body", "Terms.", "--path", dir)
	runPromptCmd("create", "main", "--kind", "global", "--title", "Main",
		"--body", "Intro.\n{{include: glossary}}\nOutro.", "--path", dir)

	code, stdout, stderr := runPromptCmd("render", "main", "--path", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "Terms.") || strings.Contains(stdout, "{{include:") {
		t.Errorf("render did not expand the inclusion; got:\n%s", stdout)
	}

	// `show` must still print the raw directive (unresolved).
	_, rawOut, _ := runPromptCmd("show", "main", "--path", dir)
	if !strings.Contains(rawOut, "{{include: glossary}}") {
		t.Errorf("show should print the raw directive; got:\n%s", rawOut)
	}
}

// TestRunPromptRenderCycle covers the render verb's cycle error path (exit 2).
func TestRunPromptRenderCycle(t *testing.T) {
	dir := t.TempDir()
	runPromptCmd("create", "a", "--kind", "global", "--title", "A", "--body", "{{include: b}}", "--path", dir)
	runPromptCmd("create", "b", "--kind", "shared", "--title", "B", "--body", "{{include: a}}", "--path", dir)

	code, _, stderr := runPromptCmd("render", "a", "--path", dir)
	if code != 2 {
		t.Fatalf("cycle render exit = %d, want 2; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stderr, "cycle") {
		t.Errorf("stderr does not name the cycle; got:\n%s", stderr)
	}
}

// TestRunPromptRenderMissingInclude covers the render verb's missing-reference
// error path (exit 2), naming the missing id.
func TestRunPromptRenderMissingInclude(t *testing.T) {
	dir := t.TempDir()
	runPromptCmd("create", "a", "--kind", "global", "--title", "A", "--body", "{{include: ghost}}", "--path", dir)

	code, _, stderr := runPromptCmd("render", "a", "--path", dir)
	if code != 2 {
		t.Fatalf("missing-include render exit = %d, want 2; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stderr, "ghost") {
		t.Errorf("stderr does not name the missing id; got:\n%s", stderr)
	}
}

// TestRunPromptCreatePreview covers --preview: it renders the file without writing.
func TestRunPromptCreatePreview(t *testing.T) {
	dir := t.TempDir()
	code, stdout, stderr := runPromptCmd("create", "glossary", "--kind", "shared",
		"--title", "Glossary", "--body", "terms", "--preview", "--path", dir)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stderr:\n%s", code, stderr)
	}
	if !strings.Contains(stdout, "kind: shared") {
		t.Errorf("preview did not render the file; got:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(dir, ".daedalus", "prompts", "glossary.md")); !os.IsNotExist(err) {
		t.Errorf("preview wrote a file (stat err=%v)", err)
	}
}
