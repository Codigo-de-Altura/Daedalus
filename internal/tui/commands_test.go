package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// writePrompt creates a prompt under the workspace's prompts root for the given
// workdir, using the real core so the test exercises the same persistence the
// TUI consumes at runtime.
func writePrompt(t *testing.T, workdir string, p prompts.Prompt) {
	t.Helper()
	root := filepath.Join(workdir, workspace.Name, prompts.PromptsDir)
	if err := prompts.Create(root, p); err != nil {
		t.Fatalf("create prompt %q: %v", p.ID, err)
	}
}

// TestLoadPromptsCmdReadsCore verifies loadPromptsCmd lists prompts persisted by
// the core, ordered as the core orders them, through the real filesystem.
func TestLoadPromptsCmdReadsCore(t *testing.T) {
	workdir := t.TempDir()
	writePrompt(t, workdir, prompts.Prompt{ID: "style", Kind: prompts.KindGlobal, Title: "Style", Body: "x"})
	writePrompt(t, workdir, prompts.Prompt{ID: "glossary", Kind: prompts.KindShared, Title: "Glossary", Body: "y"})

	msg, ok := loadPromptsCmd(workdir)().(promptsLoadedMsg)
	if !ok {
		t.Fatal("loadPromptsCmd should produce a promptsLoadedMsg")
	}
	if msg.err != nil {
		t.Fatalf("unexpected error: %v", msg.err)
	}
	if len(msg.entries) != 2 {
		t.Fatalf("want 2 entries, got %d", len(msg.entries))
	}
	// The core sorts by id, so glossary precedes style.
	if msg.entries[0].ID != "glossary" || msg.entries[1].ID != "style" {
		t.Errorf("entries not in id order: %+v", msg.entries)
	}
}

// TestLoadPromptsCmdMissingWorkspace verifies a directory without a workspace
// yields an empty list (a clean empty state), not an error.
func TestLoadPromptsCmdMissingWorkspace(t *testing.T) {
	msg := loadPromptsCmd(t.TempDir())().(promptsLoadedMsg)
	if msg.err != nil {
		t.Errorf("missing workspace should not be an error, got %v", msg.err)
	}
	if len(msg.entries) != 0 {
		t.Errorf("missing workspace should list no prompts, got %d", len(msg.entries))
	}
}

// TestResolvePromptCmdComposesInclusions verifies resolvePromptCmd returns the
// composed text with inclusions resolved (not the raw {{include}} directive).
func TestResolvePromptCmdComposesInclusions(t *testing.T) {
	workdir := t.TempDir()
	writePrompt(t, workdir, prompts.Prompt{ID: "frag", Kind: prompts.KindShared, Title: "Frag", Body: "FRAGMENT-BODY"})
	writePrompt(t, workdir, prompts.Prompt{ID: "host", Kind: prompts.KindGlobal, Title: "Host", Body: "before\n{{include: frag}}\nafter"})

	msg := resolvePromptCmd(workdir, "host")().(promptResolvedMsg)
	if msg.err != nil {
		t.Fatalf("unexpected error: %v", msg.err)
	}
	if !strings.Contains(msg.content, "FRAGMENT-BODY") {
		t.Errorf("composed content should include the fragment body, got:\n%s", msg.content)
	}
	if strings.Contains(msg.content, "{{include:") {
		t.Errorf("composed content should not contain raw include directives, got:\n%s", msg.content)
	}
}

// TestResolvePromptCmdCycleError verifies a cyclic inclusion surfaces a typed
// cycle error that the preview can recognize.
func TestResolvePromptCmdCycleError(t *testing.T) {
	workdir := t.TempDir()
	writePrompt(t, workdir, prompts.Prompt{ID: "a", Kind: prompts.KindGlobal, Title: "A", Body: "{{include: b}}"})
	writePrompt(t, workdir, prompts.Prompt{ID: "b", Kind: prompts.KindShared, Title: "B", Body: "{{include: a}}"})

	msg := resolvePromptCmd(workdir, "a")().(promptResolvedMsg)
	if msg.err == nil {
		t.Fatal("expected a composition error for the cycle")
	}
	if got := composeErrorMessage("a", msg.err); !strings.Contains(got, "inclusion cycle") {
		t.Errorf("cycle message not produced, got: %s", got)
	}
}
