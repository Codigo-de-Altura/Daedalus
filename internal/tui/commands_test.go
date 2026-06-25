package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// commands_test.go verifies the per-area commands read the real core through the
// real filesystem (the same persistence the TUI consumes at runtime), so the
// presentation layer's only job — triggering a command and rendering its result —
// is exercised end to end without a terminal.

// writePrompt creates a prompt under the workspace's prompts root for the given
// workdir, using the real core.
func writePrompt(t *testing.T, workdir string, p prompts.Prompt) {
	t.Helper()
	root := filepath.Join(workdir, workspace.Name, prompts.PromptsDir)
	if err := prompts.Create(root, p); err != nil {
		t.Fatalf("create prompt %q: %v", p.ID, err)
	}
}

// TestLoadAreaCmdPromptsReadsCore verifies the prompts area lists prompts persisted
// by the core, in the core's id order, through the real filesystem.
func TestLoadAreaCmdPromptsReadsCore(t *testing.T) {
	workdir := t.TempDir()
	writePrompt(t, workdir, prompts.Prompt{ID: "style", Kind: prompts.KindGlobal, Title: "Style", Body: "x"})
	writePrompt(t, workdir, prompts.Prompt{ID: "glossary", Kind: prompts.KindShared, Title: "Glossary", Body: "y"})

	msg, ok := loadAreaCmd(workdir, areaPrompts)().(areaLoadedMsg)
	if !ok {
		t.Fatal("loadAreaCmd should produce an areaLoadedMsg")
	}
	if msg.err != nil {
		t.Fatalf("unexpected error: %v", msg.err)
	}
	if len(msg.items) != 2 {
		t.Fatalf("want 2 items, got %d", len(msg.items))
	}
	// The core sorts by id, so glossary precedes style.
	if msg.items[0].key != "glossary" || msg.items[1].key != "style" {
		t.Errorf("items not in id order: %+v", msg.items)
	}
}

// TestLoadAreaCmdPromptsMissingWorkspace verifies a directory without a workspace
// yields an empty list (a clean empty state), not an error.
func TestLoadAreaCmdPromptsMissingWorkspace(t *testing.T) {
	msg := loadAreaCmd(t.TempDir(), areaPrompts)().(areaLoadedMsg)
	if msg.err != nil {
		t.Errorf("missing workspace should not be an error, got %v", msg.err)
	}
	if len(msg.items) != 0 {
		t.Errorf("missing workspace should list no prompts, got %d", len(msg.items))
	}
}

// TestLoadAreaCmdAgentsListsCatalog verifies the agents area lists the built-in
// catalog (embedded in the binary, no disk read).
func TestLoadAreaCmdAgentsListsCatalog(t *testing.T) {
	msg := loadAreaCmd(".", areaAgents)().(areaLoadedMsg)
	if msg.err != nil {
		t.Fatalf("agents area should not error: %v", msg.err)
	}
	if len(msg.items) == 0 {
		t.Fatal("the built-in catalog should list at least one agent")
	}
}

// TestLoadSubCmdPromptComposesInclusions verifies the prompt sub-screen loader
// returns the composed text with inclusions resolved (not the raw directive). The
// markdown is now rendered off the UI thread inside loadSubCmd, so the delivered
// content is the final rendered body (asserted on its visible text).
func TestLoadSubCmdPromptComposesInclusions(t *testing.T) {
	workdir := t.TempDir()
	writePrompt(t, workdir, prompts.Prompt{ID: "frag", Kind: prompts.KindShared, Title: "Frag", Body: "FRAGMENT-BODY"})
	writePrompt(t, workdir, prompts.Prompt{ID: "host", Kind: prompts.KindGlobal, Title: "Host", Body: "before\n{{include: frag}}\nafter"})

	msg := loadSubCmd(workdir, areaPrompts, "host", defaultTheme(), 72)().(subLoadedMsg)
	if msg.err != nil {
		t.Fatalf("unexpected error: %v", msg.err)
	}
	visible := visibleText(msg.content)
	if !strings.Contains(visible, "FRAGMENT-BODY") {
		t.Errorf("composed content should include the fragment body, got:\n%s", visible)
	}
	if strings.Contains(visible, "{{include:") {
		t.Errorf("composed content should not contain raw include directives, got:\n%s", visible)
	}
}

// TestLoadSubCmdPromptCycleError verifies a cyclic inclusion surfaces an
// actionable error the sub-screen renders.
func TestLoadSubCmdPromptCycleError(t *testing.T) {
	workdir := t.TempDir()
	writePrompt(t, workdir, prompts.Prompt{ID: "a", Kind: prompts.KindGlobal, Title: "A", Body: "{{include: b}}"})
	writePrompt(t, workdir, prompts.Prompt{ID: "b", Kind: prompts.KindShared, Title: "B", Body: "{{include: a}}"})

	msg := loadSubCmd(workdir, areaPrompts, "a", defaultTheme(), 72)().(subLoadedMsg)
	if msg.err == nil {
		t.Fatal("expected a composition error for the cycle")
	}
	if !strings.Contains(msg.err.Error(), "inclusion cycle") {
		t.Errorf("cycle message not produced, got: %v", msg.err)
	}
}

// TestLoadAreaCmdInitEmptyWorkspace verifies the init area reports an empty state
// (no rows) for a directory that has no .daedalus workspace yet.
func TestLoadAreaCmdInitEmptyWorkspace(t *testing.T) {
	msg := loadAreaCmd(t.TempDir(), areaInit)().(areaLoadedMsg)
	if msg.err != nil {
		t.Fatalf("init on a fresh dir should not error: %v", msg.err)
	}
	if len(msg.items) != 0 {
		t.Errorf("init with no workspace should be an empty state, got %d items", len(msg.items))
	}
}
