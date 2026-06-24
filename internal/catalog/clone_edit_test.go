package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// readFile reads a materialized agent file under agentsRoot, failing the test if
// it is absent.
func readFile(t *testing.T, agentsRoot, id, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(agentsRoot, id, name))
	if err != nil {
		t.Fatalf("read %s/%s: %v", id, name, err)
	}
	return string(b)
}

// TestCloneCreatesIndependentDefinition covers CA1/CA6: cloning a built-in agent
// to a kebab-case dest id creates a new canonical definition under that id.
func TestCloneCreatesIndependentDefinition(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	res, err := Builtin.Clone(agentsRoot, "analyst", "analyst-custom")
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if res.AlreadyExisted() {
		t.Errorf("AlreadyExisted = true on a clean clone, want false")
	}
	if len(res.Created) != 2 {
		t.Errorf("Created = %v, want both files", res.Created)
	}

	// The clone's definition carries the dest id, not the source id.
	def := readFile(t, agentsRoot, "analyst-custom", DefinitionFileName)
	if !strings.Contains(def, "id: analyst-custom") {
		t.Errorf("clone definition does not carry dest id; got:\n%s", def)
	}
}

// TestCloneRejectsNonKebabDest covers CA6 negative: a non-kebab dest id is
// rejected before any write.
func TestCloneRejectsNonKebabDest(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	if _, err := Builtin.Clone(agentsRoot, "analyst", "Bad_Id"); err == nil {
		t.Errorf("Clone accepted a non-kebab dest id, want error")
	}
	if _, err := os.Stat(filepath.Join(agentsRoot, "Bad_Id")); !os.IsNotExist(err) {
		t.Errorf("a rejected clone created files (stat err=%v), want none", err)
	}
}

// TestCloneUnknownSource covers the source-lookup failure: an unknown source id
// yields ErrAgentNotFound and writes nothing.
func TestCloneUnknownSource(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	if _, err := Builtin.Clone(agentsRoot, "nope", "dest"); !errors.Is(err, ErrAgentNotFound) {
		t.Errorf("Clone(unknown source): err = %v, want ErrAgentNotFound", err)
	}
}

// TestCloneIsNonDestructive covers CA4: cloning over an existing dest id does not
// overwrite it; the conflict is reported and a manual edit survives.
func TestCloneIsNonDestructive(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	if _, err := Builtin.Clone(agentsRoot, "analyst", "analyst-custom"); err != nil {
		t.Fatalf("first Clone: %v", err)
	}
	prompt := filepath.Join(agentsRoot, "analyst-custom", PromptFileName)
	const marker = "MANUAL-CLONE-EDIT"
	if err := os.WriteFile(prompt, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Builtin.Clone(agentsRoot, "analyst", "analyst-custom")
	if err != nil {
		t.Fatalf("second Clone: %v", err)
	}
	if !res.AlreadyExisted() {
		t.Errorf("AlreadyExisted = false on a re-clone, want true")
	}
	if len(res.Created) != 0 {
		t.Errorf("Created = %v on re-clone, want none", res.Created)
	}
	if b, _ := os.ReadFile(prompt); string(b) != marker {
		t.Errorf("re-clone overwrote the manual edit: %q", b)
	}
}

// TestEditPersistsRolePromptParams covers CA3: editing role, prompt and a
// parameter persists all three to the clone's canonical definition.
func TestEditPersistsRolePromptParams(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
	if _, err := Builtin.Clone(agentsRoot, "analyst", "analyst-custom"); err != nil {
		t.Fatalf("Clone: %v", err)
	}

	edited, err := Builtin.Edit(agentsRoot, "analyst-custom", EditSpec{
		SetRole:   true,
		Role:      "Custom analyst role.",
		SetPrompt: true,
		Prompt:    "# Custom prompt\n\nDo the custom thing.",
		SetParams: []Param{{Key: "model", Type: ParamString, Value: "custom-model"}},
	})
	if err != nil {
		t.Fatalf("Edit: %v", err)
	}
	if edited.Role != "Custom analyst role." {
		t.Errorf("returned role = %q, want the edited value", edited.Role)
	}

	// Re-load from disk to confirm the changes were persisted, not just returned.
	reloaded, err := Load(agentsRoot, "analyst-custom")
	if err != nil {
		t.Fatalf("Load after edit: %v", err)
	}
	if reloaded.Role != "Custom analyst role." {
		t.Errorf("persisted role = %q, want the edited value", reloaded.Role)
	}
	if !strings.Contains(reloaded.Prompt, "Do the custom thing.") {
		t.Errorf("persisted prompt missing edit; got:\n%s", reloaded.Prompt)
	}
	var found bool
	for _, p := range reloaded.Params {
		if p.Key == "model" && p.Value == "custom-model" {
			found = true
		}
	}
	if !found {
		t.Errorf("persisted params missing edited model=custom-model; got %+v", reloaded.Params)
	}
}

// TestEditAddRemoveParam covers parameter add and removal, including the
// order-preserving and idempotent-removal behavior.
func TestEditAddRemoveParam(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
	if _, err := Builtin.Clone(agentsRoot, "analyst", "analyst-custom"); err != nil {
		t.Fatalf("Clone: %v", err)
	}

	// Add a new param, then remove the original model param.
	if _, err := Builtin.Edit(agentsRoot, "analyst-custom", EditSpec{
		SetParams: []Param{{Key: "temperature", Type: ParamString, Value: "0.5"}},
	}); err != nil {
		t.Fatalf("Edit (add): %v", err)
	}
	if _, err := Builtin.Edit(agentsRoot, "analyst-custom", EditSpec{
		RemoveParams: []string{"model"},
	}); err != nil {
		t.Fatalf("Edit (remove): %v", err)
	}
	// Removing an absent key is a no-op, not an error.
	if _, err := Builtin.Edit(agentsRoot, "analyst-custom", EditSpec{
		RemoveParams: []string{"does-not-exist"},
	}); err != nil {
		t.Fatalf("Edit (remove absent) should be a no-op: %v", err)
	}

	reloaded, err := Load(agentsRoot, "analyst-custom")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	for _, p := range reloaded.Params {
		if p.Key == "model" {
			t.Errorf("removed param 'model' is still present: %+v", reloaded.Params)
		}
	}
	var hasTemp bool
	for _, p := range reloaded.Params {
		if p.Key == "temperature" {
			hasTemp = true
		}
	}
	if !hasTemp {
		t.Errorf("added param 'temperature' missing; got %+v", reloaded.Params)
	}
}

// TestEditDoesNotMutateBuiltin covers CA2: cloning then editing the clone leaves
// the built-in catalog agent unchanged — the core independence guarantee.
func TestEditDoesNotMutateBuiltin(t *testing.T) {
	// Capture the built-in's canonical state before any clone/edit.
	before, err := Builtin.Get("analyst")
	if err != nil {
		t.Fatalf("Get(analyst): %v", err)
	}

	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
	if _, err := Builtin.Clone(agentsRoot, "analyst", "analyst-custom"); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if _, err := Builtin.Edit(agentsRoot, "analyst-custom", EditSpec{
		SetRole:   true,
		Role:      "Totally different role.",
		SetPrompt: true,
		Prompt:    "Totally different prompt.",
		SetParams: []Param{{Key: "model", Type: ParamString, Value: "changed"}},
	}); err != nil {
		t.Fatalf("Edit: %v", err)
	}

	after, err := Builtin.Get("analyst")
	if err != nil {
		t.Fatalf("Get(analyst) after edit: %v", err)
	}
	if after.Role != before.Role {
		t.Errorf("built-in analyst role changed: %q -> %q", before.Role, after.Role)
	}
	if after.Prompt != before.Prompt {
		t.Errorf("built-in analyst prompt changed after editing a clone")
	}
	if len(after.Params) == 0 || after.Params[0].Value != before.Params[0].Value {
		t.Errorf("built-in analyst params changed: %+v -> %+v", before.Params, after.Params)
	}
}

// TestEditInvalidLeavesFileIntact covers CA5: an edit that would make the
// definition invalid (empty role) is rejected with an actionable error and the
// existing files are left byte-for-byte intact.
func TestEditInvalidLeavesFileIntact(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
	if _, err := Builtin.Clone(agentsRoot, "analyst", "analyst-custom"); err != nil {
		t.Fatalf("Clone: %v", err)
	}

	defBefore := readFile(t, agentsRoot, "analyst-custom", DefinitionFileName)
	promptBefore := readFile(t, agentsRoot, "analyst-custom", PromptFileName)

	_, err := Builtin.Edit(agentsRoot, "analyst-custom", EditSpec{SetRole: true, Role: "   "})
	if err == nil {
		t.Fatalf("Edit accepted an empty role, want an actionable error")
	}
	if !strings.Contains(err.Error(), "role") {
		t.Errorf("error does not name the offending field; got: %v", err)
	}

	// The definition must be untouched — not half-written, not blanked.
	if got := readFile(t, agentsRoot, "analyst-custom", DefinitionFileName); got != defBefore {
		t.Errorf("invalid edit modified the definition file:\n--- before ---\n%s\n--- after ---\n%s", defBefore, got)
	}
	if got := readFile(t, agentsRoot, "analyst-custom", PromptFileName); got != promptBefore {
		t.Errorf("invalid edit modified the prompt file")
	}
}

// TestEditUnknownAgent covers editing an agent that is not in the workspace.
func TestEditUnknownAgent(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
	if _, err := Builtin.Edit(agentsRoot, "ghost", EditSpec{SetRole: true, Role: "x"}); !errors.Is(err, ErrAgentNotFound) {
		t.Errorf("Edit(absent): err = %v, want ErrAgentNotFound", err)
	}
}
