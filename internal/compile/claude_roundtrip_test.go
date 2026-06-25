package compile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
)

// TestAgentRoundTripBuildThenImport asserts the build↔import coherence the
// adapter is designed for: an agent compiled to a Claude Code `.claude/agents/*.md`
// file re-imports (catalog.ImportPlanFor) to a canonical agent with the same id,
// role, prompt and model. This is the inverse of catalog/import_claude.go, so the
// frontmatter keys and quoting must line up — the test fails if they ever drift.
//
// Fields the canonical model does not carry (tools, color) are intentionally not
// emitted and so cannot round-trip; the test asserts the fields that DO have a
// canonical home survive a build→import cycle unchanged.
func TestAgentRoundTripBuildThenImport(t *testing.T) {
	original := catalog.Agent{
		ID:     "architect",
		Role:   "System architect: designs the system",
		Prompt: "You are the architect.\nDesign deliberately.",
		Params: []catalog.Param{
			{Key: "model", Type: catalog.ParamString, Value: "opus"},
		},
	}

	// 1. Build the agent to its Claude Code native file on disk.
	rendered := renderAgent(original)
	srcDir := t.TempDir()
	srcFile := filepath.Join(srcDir, original.ID+".md")
	if err := os.WriteFile(srcFile, []byte(rendered), 0o644); err != nil {
		t.Fatal(err)
	}

	// 2. Import it back into a fresh canonical agents root.
	agentsRoot := filepath.Join(t.TempDir(), "agents")
	plan, err := catalog.ImportPlanFor(agentsRoot, srcFile)
	if err != nil {
		t.Fatalf("ImportPlanFor: %v", err)
	}
	if len(plan.Errors) != 0 {
		t.Fatalf("import reported errors: %+v", plan.Errors)
	}
	if _, err := plan.Apply(); err != nil {
		t.Fatalf("import apply: %v", err)
	}

	// 3. Load the round-tripped canonical agent and compare the fields that have a
	// canonical home.
	got, err := catalog.Load(agentsRoot, original.ID)
	if err != nil {
		t.Fatalf("load round-tripped agent: %v", err)
	}
	if got.ID != original.ID {
		t.Errorf("id = %q, want %q", got.ID, original.ID)
	}
	if got.Role != original.Role {
		t.Errorf("role = %q, want %q", got.Role, original.Role)
	}
	if got.Prompt != original.Prompt {
		t.Errorf("prompt = %q, want %q", got.Prompt, original.Prompt)
	}
	if model := agentModel(got); model != "opus" {
		t.Errorf("model param = %q, want %q", model, "opus")
	}
}
