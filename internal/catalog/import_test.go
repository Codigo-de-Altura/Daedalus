package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// claudeAgent is a small, valid Claude Code agent file for import tests.
const claudeAgent = `---
name: my-helper
description: Helps with things.
tools: Read, Write, Bash
model: opus
color: blue
---

# My Helper

You are a helpful agent. Do helpful things.
`

// writeSource writes content to a file under dir and returns its path.
func writeSource(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestNormalizeID covers R6/CA5: arbitrary source identifiers fold to kebab-case,
// and an unnormalizable source is an error rather than a fabricated id.
func TestNormalizeID(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"analyst", "analyst", false},
		{"My Helper", "my-helper", false},
		{"My_Helper_2", "my-helper-2", false},
		{"Code Reviewer!!", "code-reviewer", false},
		{"--weird--name--", "weird-name", false},
		{"CamelCase", "camelcase", false},
		{"  spaced  ", "spaced", false},
		{"", "", true},
		{"!!!", "", true},
		{"___", "", true},
	}
	for _, c := range cases {
		got, err := NormalizeID(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("NormalizeID(%q) = %q, want error", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("NormalizeID(%q): unexpected error %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalizeID(%q) = %q, want %q", c.in, got, c.want)
		}
		if !IsKebabCase(got) {
			t.Errorf("NormalizeID(%q) = %q is not kebab-case", c.in, got)
		}
	}
}

// TestSplitFrontmatter covers the frontmatter scanner's boundary handling.
func TestSplitFrontmatter(t *testing.T) {
	fm, body, err := splitFrontmatter(claudeAgent)
	if err != nil {
		t.Fatalf("splitFrontmatter: %v", err)
	}
	if !strings.Contains(fm, "name: my-helper") {
		t.Errorf("frontmatter missing name; got:\n%s", fm)
	}
	if !strings.Contains(body, "You are a helpful agent") {
		t.Errorf("body missing prompt; got:\n%s", body)
	}

	// CRLF line endings must be accepted.
	crlf := strings.ReplaceAll(claudeAgent, "\n", "\r\n")
	if _, _, err := splitFrontmatter(crlf); err != nil {
		t.Errorf("splitFrontmatter(CRLF): %v", err)
	}

	// Missing opening / closing delimiters are errors.
	if _, _, err := splitFrontmatter("no frontmatter here\n"); err == nil {
		t.Errorf("splitFrontmatter accepted content without opening '---'")
	}
	if _, _, err := splitFrontmatter("---\nname: x\nno closing\n"); err == nil {
		t.Errorf("splitFrontmatter accepted unterminated frontmatter")
	}
}

// TestParseFrontmatterTolerant covers that the scanner captures the keys we use
// and tolerates (skips) list items and keys we do not.
func TestParseFrontmatterTolerant(t *testing.T) {
	fm := "name: x\ndescription: a desc\ntools: Read, Write\nmodel: opus\ncolor: blue\n"
	fields, err := parseFrontmatter(fm)
	if err != nil {
		t.Fatalf("parseFrontmatter: %v", err)
	}
	if fields["name"] != "x" || fields["description"] != "a desc" || fields["model"] != "opus" {
		t.Errorf("parsed fields wrong: %+v", fields)
	}

	// A duplicate key is an error (ambiguous source).
	if _, err := parseFrontmatter("name: a\nname: b\n"); err == nil {
		t.Errorf("parseFrontmatter accepted a duplicate key")
	}
}

// TestImportClaudeCodeToCanonical covers CA2: a Claude Code file converts to a
// valid canonical agent under .daedalus/agents/, with the documented mapping.
func TestImportClaudeCodeToCanonical(t *testing.T) {
	src := t.TempDir()
	file := writeSource(t, src, "my-helper.md", claudeAgent)
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	outcomes, err := Import(agentsRoot, file)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(outcomes) != 1 || outcomes[0].Err != nil {
		t.Fatalf("outcomes = %+v, want one successful import", outcomes)
	}
	if outcomes[0].AgentID != "my-helper" {
		t.Errorf("AgentID = %q, want my-helper", outcomes[0].AgentID)
	}

	// Load it back and confirm the mapping: description->role, body->prompt,
	// model->parameter, tools/color dropped.
	loaded, err := Load(agentsRoot, "my-helper")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Role != "Helps with things." {
		t.Errorf("role = %q, want the description", loaded.Role)
	}
	if !strings.Contains(loaded.Prompt, "You are a helpful agent") {
		t.Errorf("prompt missing body; got:\n%s", loaded.Prompt)
	}
	var hasModel bool
	for _, p := range loaded.Params {
		if p.Key == "model" {
			hasModel = true
			if p.Value != "opus" {
				t.Errorf("model param = %q, want opus", p.Value)
			}
		}
		if p.Key == "tools" || p.Key == "color" {
			t.Errorf("backend-specific field %q was not dropped: %+v", p.Key, loaded.Params)
		}
	}
	if !hasModel {
		t.Errorf("model parameter missing; got %+v", loaded.Params)
	}
}

// TestImportDerivesIDFromFilename covers the id fallback: a Claude Code file with
// no `name` derives its id from the file base name, normalized to kebab-case.
func TestImportDerivesIDFromFilename(t *testing.T) {
	src := t.TempDir()
	noName := "---\ndescription: d\nmodel: opus\n---\n\nBody here.\n"
	file := writeSource(t, src, "Code Reviewer.md", noName)
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	outcomes, err := Import(agentsRoot, file)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(outcomes) != 1 || outcomes[0].AgentID != "code-reviewer" {
		t.Fatalf("outcomes = %+v, want id code-reviewer derived from filename", outcomes)
	}
}

// TestImportCanonicalFile covers CA1/R3: importing a single-file canonical
// definition (definition block + inline body) creates its canonical agent.
func TestImportCanonicalFile(t *testing.T) {
	src := t.TempDir()
	canonical := "id: ported-agent\nversion: \"1\"\nrole: A ported role.\nprompt: prompt.md\nparameters:\n  model: default\n---\n# Ported\n\nThe prompt body.\n"
	file := writeSource(t, src, "ported.txt", canonical)
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	outcomes, err := Import(agentsRoot, file)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(outcomes) != 1 || outcomes[0].Err != nil || outcomes[0].AgentID != "ported-agent" {
		t.Fatalf("outcomes = %+v, want one import of ported-agent", outcomes)
	}
	loaded, err := Load(agentsRoot, "ported-agent")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Role != "A ported role." || !strings.Contains(loaded.Prompt, "The prompt body.") {
		t.Errorf("canonical import lost data: role=%q prompt=%q", loaded.Role, loaded.Prompt)
	}
}

// TestImportInvalidIsActionableAndNotWritten covers CA3: a source that fails the
// canonical schema (here: empty description -> empty role) produces an actionable
// error and is not written.
func TestImportInvalidIsActionableAndNotWritten(t *testing.T) {
	src := t.TempDir()
	invalid := "---\nname: broken\ndescription:\nmodel: opus\n---\n\nA body.\n" // empty description
	file := writeSource(t, src, "broken.md", invalid)
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	outcomes, err := Import(agentsRoot, file)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(outcomes) != 1 || outcomes[0].Err == nil {
		t.Fatalf("outcomes = %+v, want one failed outcome", outcomes)
	}
	if !strings.Contains(outcomes[0].Err.Error(), "role") {
		t.Errorf("error does not name the offending field; got: %v", outcomes[0].Err)
	}
	// Nothing must have been written for the invalid agent.
	if _, err := os.Stat(filepath.Join(agentsRoot, "broken")); !os.IsNotExist(err) {
		t.Errorf("invalid source created files (stat err=%v), want none", err)
	}
}

// TestImportUnrecognizedSource covers a file that is neither frontmatter nor
// canonical: it is reported as an unrecognized source, not silently skipped.
func TestImportUnrecognizedSource(t *testing.T) {
	src := t.TempDir()
	file := writeSource(t, src, "random.md", "just some markdown with no frontmatter\n")
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	outcomes, err := Import(agentsRoot, file)
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if len(outcomes) != 1 || outcomes[0].Err == nil {
		t.Fatalf("outcomes = %+v, want one failed outcome", outcomes)
	}
	if !errors.Is(outcomes[0].Err, ErrUnrecognizedSource) {
		t.Errorf("err = %v, want ErrUnrecognizedSource", outcomes[0].Err)
	}
}

// TestImportNonDestructive covers CA4: importing over an existing id does not
// overwrite it; the conflict is reported and a manual edit survives.
func TestImportNonDestructive(t *testing.T) {
	src := t.TempDir()
	file := writeSource(t, src, "my-helper.md", claudeAgent)
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	if _, err := Import(agentsRoot, file); err != nil {
		t.Fatalf("first Import: %v", err)
	}
	prompt := filepath.Join(agentsRoot, "my-helper", PromptFileName)
	const marker = "MANUAL-IMPORT-EDIT"
	if err := os.WriteFile(prompt, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}

	outcomes, err := Import(agentsRoot, file)
	if err != nil {
		t.Fatalf("second Import: %v", err)
	}
	if len(outcomes) != 1 || !outcomes[0].AlreadyExisted() {
		t.Fatalf("outcomes = %+v, want a non-destructive skip", outcomes)
	}
	if b, _ := os.ReadFile(prompt); string(b) != marker {
		t.Errorf("re-import overwrote the manual edit: %q", b)
	}
}

// TestImportIsDeterministic covers CA6: importing the same source into two clean
// workspaces yields byte-identical canonical files.
func TestImportIsDeterministic(t *testing.T) {
	src := t.TempDir()
	file := writeSource(t, src, "my-helper.md", claudeAgent)

	render := func() (def, prompt []byte) {
		agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
		if _, err := Import(agentsRoot, file); err != nil {
			t.Fatalf("Import: %v", err)
		}
		d, err := os.ReadFile(filepath.Join(agentsRoot, "my-helper", DefinitionFileName))
		if err != nil {
			t.Fatalf("read def: %v", err)
		}
		p, err := os.ReadFile(filepath.Join(agentsRoot, "my-helper", PromptFileName))
		if err != nil {
			t.Fatalf("read prompt: %v", err)
		}
		return d, p
	}

	d1, p1 := render()
	d2, p2 := render()
	if string(d1) != string(d2) {
		t.Errorf("definition not deterministic:\n--- a ---\n%s\n--- b ---\n%s", d1, d2)
	}
	if string(p1) != string(p2) {
		t.Errorf("prompt not deterministic")
	}
}

// TestImportDirectoryMixedValidInvalid covers the multi-agent directory policy:
// valid sources import, invalid ones are reported, and one bad file never aborts
// the good ones.
func TestImportDirectoryMixedValidInvalid(t *testing.T) {
	src := t.TempDir()
	writeSource(t, src, "good-one.md", strings.Replace(claudeAgent, "my-helper", "good-one", 1))
	writeSource(t, src, "good-two.md", strings.Replace(claudeAgent, "my-helper", "good-two", 1))
	writeSource(t, src, "bad.md", "---\nname: bad\ndescription:\n---\n\nbody\n") // empty role
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	outcomes, err := Import(agentsRoot, src)
	if err != nil {
		t.Fatalf("Import dir: %v", err)
	}

	var imported, failed int
	for _, o := range outcomes {
		if o.Err != nil {
			failed++
		} else {
			imported++
		}
	}
	if imported != 2 {
		t.Errorf("imported = %d, want 2 valid agents", imported)
	}
	if failed != 1 {
		t.Errorf("failed = %d, want 1 invalid source", failed)
	}
	// The valid ones must actually be on disk despite the bad sibling.
	for _, id := range []string{"good-one", "good-two"} {
		if _, err := os.Stat(filepath.Join(agentsRoot, id)); err != nil {
			t.Errorf("valid agent %q not imported: %v", id, err)
		}
	}
}

// TestImportPlanWritesNothing covers the preview path: planning renders without
// touching the filesystem.
func TestImportPlanWritesNothing(t *testing.T) {
	src := t.TempDir()
	file := writeSource(t, src, "my-helper.md", claudeAgent)
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	plan, err := ImportPlanFor(agentsRoot, file)
	if err != nil {
		t.Fatalf("ImportPlanFor: %v", err)
	}
	if len(plan.Agents) != 1 {
		t.Fatalf("plan has %d agents, want 1", len(plan.Agents))
	}
	if _, err := os.Stat(filepath.Join(agentsRoot, "my-helper")); !os.IsNotExist(err) {
		t.Errorf("planning wrote files (stat err=%v), want none", err)
	}
}

// TestImportRealClaudeAgentsRoundTrips imports Daedalus's own .claude/agents and
// confirms each converts to a valid canonical agent — the concrete CA2 case the
// validation references (a real .claude/agents structure).
func TestImportRealClaudeAgentsRoundTrips(t *testing.T) {
	repoAgents := filepath.Join("..", "..", ".claude", "agents")
	if _, err := os.Stat(repoAgents); err != nil {
		t.Skipf("repo .claude/agents not available: %v", err)
	}
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	outcomes, err := Import(agentsRoot, repoAgents)
	if err != nil {
		t.Fatalf("Import .claude/agents: %v", err)
	}
	if len(outcomes) == 0 {
		t.Fatal("no agents imported from .claude/agents")
	}
	for _, o := range outcomes {
		if o.Err != nil {
			t.Errorf("import failed for %s: %v", o.SourcePath, o.Err)
			continue
		}
		// Every imported agent must be a valid, loadable canonical definition.
		loaded, err := Load(agentsRoot, o.AgentID)
		if err != nil {
			t.Errorf("Load(%q): %v", o.AgentID, err)
			continue
		}
		if err := loaded.Validate(); err != nil {
			t.Errorf("imported agent %q invalid: %v", o.AgentID, err)
		}
	}
}
