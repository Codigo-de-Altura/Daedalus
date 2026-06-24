package catalog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// canonicalAgents are the five agents the spec requires the built-in catalog to
// expose at minimum (R2/CA1, init.md §8). Tests assert these are present rather
// than pinning the exact set, so adding future built-ins never breaks them.
var canonicalAgents = []string{"analyst", "architect", "planner", "validator", "documenter"}

// TestListIncludesCanonicalAgents covers CA1: the catalog lists at least the five
// canonical agents, each with a non-empty role.
func TestListIncludesCanonicalAgents(t *testing.T) {
	entries := Builtin.List()
	if len(entries) < len(canonicalAgents) {
		t.Fatalf("List returned %d agents, want at least %d", len(entries), len(canonicalAgents))
	}

	byID := make(map[string]Entry, len(entries))
	for _, e := range entries {
		byID[e.ID] = e
	}
	for _, id := range canonicalAgents {
		e, ok := byID[id]
		if !ok {
			t.Errorf("catalog is missing canonical agent %q", id)
			continue
		}
		if strings.TrimSpace(e.Role) == "" {
			t.Errorf("agent %q has an empty role in the listing", id)
		}
	}
}

// TestListIsSortedAndDeterministic covers R8: List is ordered by id and stable
// across calls, independent of the catalog's internal storage order.
func TestListIsSortedAndDeterministic(t *testing.T) {
	a := Builtin.List()
	b := Builtin.List()
	if len(a) != len(b) {
		t.Fatalf("List length differs across calls: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("List not deterministic at %d: %+v vs %+v", i, a[i], b[i])
		}
		if i > 0 && a[i-1].ID > a[i].ID {
			t.Errorf("List not sorted by id: %q before %q", a[i-1].ID, a[i].ID)
		}
	}
}

// TestEveryAgentIsStructurallyValid covers CA2: each catalog agent has a
// non-empty role and prompt, a kebab-case id, and passes structural validation
// (the stand-in for the formal schema until ticket-02-04 lands).
func TestEveryAgentIsStructurallyValid(t *testing.T) {
	for _, e := range Builtin.List() {
		t.Run(e.ID, func(t *testing.T) {
			a, err := Builtin.Get(e.ID)
			if err != nil {
				t.Fatalf("Get(%q): %v", e.ID, err)
			}
			if err := a.Validate(); err != nil {
				t.Errorf("agent %q failed structural validation: %v", e.ID, err)
			}
			if strings.TrimSpace(a.Role) == "" {
				t.Errorf("agent %q has an empty role", e.ID)
			}
			if strings.TrimSpace(a.Prompt) == "" {
				t.Errorf("agent %q has an empty prompt", e.ID)
			}
		})
	}
}

// TestAllIdsAreKebabCase covers CA5: every catalog id is kebab-case.
func TestAllIdsAreKebabCase(t *testing.T) {
	for _, e := range Builtin.List() {
		if !IsKebabCase(e.ID) {
			t.Errorf("agent id %q is not kebab-case", e.ID)
		}
	}
}

// TestIsKebabCase pins the id convention predicate (R7/CA5) the catalog and
// future importers rely on.
func TestIsKebabCase(t *testing.T) {
	cases := []struct {
		id   string
		want bool
	}{
		{"analyst", true},
		{"agent-2", true},
		{"multi-word-id", true},
		{"", false},
		{"Analyst", false},
		{"agent_underscore", false},
		{"-leading", false},
		{"trailing-", false},
		{"double--dash", false},
		{"has space", false},
	}
	for _, c := range cases {
		if got := IsKebabCase(c.id); got != c.want {
			t.Errorf("IsKebabCase(%q) = %v, want %v", c.id, got, c.want)
		}
	}
}

// TestGetUnknownAgentReturnsSentinel covers the lookup-failure path: an unknown
// id yields ErrAgentNotFound so the CLI can map it to a usage error.
func TestGetUnknownAgentReturnsSentinel(t *testing.T) {
	if _, err := Builtin.Get("does-not-exist"); !errors.Is(err, ErrAgentNotFound) {
		t.Errorf("Get(unknown): err = %v, want ErrAgentNotFound", err)
	}
}

// TestGetReturnsDefensiveCopy guards against a caller mutating the catalog's
// source of truth through the returned Agent's params slice.
func TestGetReturnsDefensiveCopy(t *testing.T) {
	a, err := Builtin.Get("analyst")
	if err != nil {
		t.Fatalf("Get(analyst): %v", err)
	}
	if len(a.Params) == 0 {
		t.Skip("analyst has no params to mutate")
	}
	a.Params[0].Value = "mutated"

	again, err := Builtin.Get("analyst")
	if err != nil {
		t.Fatalf("Get(analyst): %v", err)
	}
	if again.Params[0].Value == "mutated" {
		t.Errorf("mutating a returned Agent corrupted the catalog's source of truth")
	}
}

// TestMaterializeCreatesCanonicalFiles covers CA3: materializing an agent into a
// clean workspace creates its canonical definition (YAML + prompt MD) under the
// agents directory.
func TestMaterializeCreatesCanonicalFiles(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	res, err := Builtin.Materialize(agentsRoot, "analyst")
	if err != nil {
		t.Fatalf("Materialize(analyst): %v", err)
	}
	if res.AlreadyExisted() {
		t.Errorf("AlreadyExisted = true on a clean workspace, want false")
	}
	if len(res.Created) != 2 {
		t.Errorf("Created = %v, want both definition and prompt", res.Created)
	}

	def := filepath.Join(agentsRoot, "analyst", DefinitionFileName)
	prompt := filepath.Join(agentsRoot, "analyst", PromptFileName)
	for _, f := range []string{def, prompt} {
		if _, err := os.Stat(f); err != nil {
			t.Errorf("expected materialized file %q: %v", f, err)
		}
	}

	// The YAML must carry the canonical contract fields, and the prompt MD must
	// be non-empty — the editable source of truth (R5).
	yaml, _ := os.ReadFile(def)
	for _, key := range []string{"id:", "role:", "prompt:", "parameters:"} {
		if !strings.Contains(string(yaml), key) {
			t.Errorf("definition missing %q; got:\n%s", key, yaml)
		}
	}
	if b, _ := os.ReadFile(prompt); strings.TrimSpace(string(b)) == "" {
		t.Errorf("materialized prompt.md is empty")
	}
}

// TestMaterializeAllCanonicalAgents covers CA1+CA3 together: every canonical
// agent can be materialized and produces both files.
func TestMaterializeAllCanonicalAgents(t *testing.T) {
	for _, id := range canonicalAgents {
		t.Run(id, func(t *testing.T) {
			agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
			res, err := Builtin.Materialize(agentsRoot, id)
			if err != nil {
				t.Fatalf("Materialize(%q): %v", id, err)
			}
			if len(res.Created) != 2 {
				t.Errorf("Created = %v, want 2 files", res.Created)
			}
		})
	}
}

// TestMaterializeIsNonDestructive covers CA4: materializing an agent that already
// exists does not overwrite it; the conflict is reported via Skipped and the
// existing (manually edited) content survives verbatim.
func TestMaterializeIsNonDestructive(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	if _, err := Builtin.Materialize(agentsRoot, "analyst"); err != nil {
		t.Fatalf("first Materialize: %v", err)
	}

	// Hand-edit the materialized prompt: it is now the user's source of truth.
	prompt := filepath.Join(agentsRoot, "analyst", PromptFileName)
	const marker = "MANUAL-EDIT-456"
	if err := os.WriteFile(prompt, []byte(marker), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Builtin.Materialize(agentsRoot, "analyst")
	if err != nil {
		t.Fatalf("second Materialize: %v", err)
	}
	if !res.AlreadyExisted() {
		t.Errorf("AlreadyExisted = false on re-materialize, want true")
	}
	if len(res.Created) != 0 {
		t.Errorf("Created = %v on re-materialize, want none", res.Created)
	}
	if len(res.Skipped) != 2 {
		t.Errorf("Skipped = %v, want both files reported as conflicts", res.Skipped)
	}
	if b, _ := os.ReadFile(prompt); string(b) != marker {
		t.Errorf("re-materialize overwrote the manual edit: %q", b)
	}
}

// TestMaterializeIsDeterministic covers CA6: materializing the same agent into
// two independent clean workspaces yields byte-identical content for both files.
func TestMaterializeIsDeterministic(t *testing.T) {
	render := func() (yaml, prompt []byte) {
		agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
		if _, err := Builtin.Materialize(agentsRoot, "architect"); err != nil {
			t.Fatalf("Materialize: %v", err)
		}
		y, err := os.ReadFile(filepath.Join(agentsRoot, "architect", DefinitionFileName))
		if err != nil {
			t.Fatalf("read definition: %v", err)
		}
		p, err := os.ReadFile(filepath.Join(agentsRoot, "architect", PromptFileName))
		if err != nil {
			t.Fatalf("read prompt: %v", err)
		}
		return y, p
	}

	y1, p1 := render()
	y2, p2 := render()
	if string(y1) != string(y2) {
		t.Errorf("definition not deterministic:\n--- a ---\n%s\n--- b ---\n%s", y1, y2)
	}
	if string(p1) != string(p2) {
		t.Errorf("prompt not deterministic:\n--- a ---\n%s\n--- b ---\n%s", p1, p2)
	}
}

// TestMaterializePlanWritesNothing covers the plan/apply split mirrored from the
// workspace package: computing a plan renders the content without touching disk,
// so a caller can preview it before committing.
func TestMaterializePlanWritesNothing(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	plan, err := Builtin.MaterializePlanFor(agentsRoot, "planner")
	if err != nil {
		t.Fatalf("MaterializePlanFor: %v", err)
	}
	if len(plan.Files) != 2 {
		t.Fatalf("plan has %d files, want 2", len(plan.Files))
	}
	for _, f := range plan.Files {
		if strings.TrimSpace(f.Content) == "" {
			t.Errorf("planned file %q has empty content", f.Name)
		}
	}
	// Planning must be side-effect free.
	if _, err := os.Stat(filepath.Join(agentsRoot, "planner")); !os.IsNotExist(err) {
		t.Errorf("MaterializePlanFor created files on disk (stat err=%v), want none", err)
	}
}

// TestMaterializeUnknownAgent covers the failure path at the materialize
// boundary: an unknown id is rejected before any I/O.
func TestMaterializeUnknownAgent(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)

	if _, err := Builtin.Materialize(agentsRoot, "nope"); !errors.Is(err, ErrAgentNotFound) {
		t.Fatalf("Materialize(unknown): err = %v, want ErrAgentNotFound", err)
	}
	if _, err := os.Stat(filepath.Join(agentsRoot, "nope")); !os.IsNotExist(err) {
		t.Errorf("an unknown agent created a directory, want none (stat err=%v)", err)
	}
}

// TestValidateRejectsMalformedAgents covers the structural validator's negative
// cases — the guarantees that keep a bad definition out of a workspace and that
// every built-in must satisfy.
func TestValidateRejectsMalformedAgents(t *testing.T) {
	base := Agent{ID: "ok", Role: "role", Prompt: "prompt"}

	cases := []struct {
		name  string
		agent Agent
	}{
		{"empty id", func() Agent { a := base; a.ID = ""; return a }()},
		{"non-kebab id", func() Agent { a := base; a.ID = "Bad_Id"; return a }()},
		{"empty role", func() Agent { a := base; a.Role = "  "; return a }()},
		{"empty prompt", func() Agent { a := base; a.Prompt = ""; return a }()},
		{"empty param key", func() Agent {
			a := base
			a.Params = []Param{{Key: "", Type: ParamString, Value: "x"}}
			return a
		}()},
		{"duplicate param key", func() Agent {
			a := base
			a.Params = []Param{{Key: "k", Type: ParamString, Value: "1"}, {Key: "k", Type: ParamString, Value: "2"}}
			return a
		}()},
		{"unknown param type", func() Agent {
			a := base
			a.Params = []Param{{Key: "k", Type: ParamType("weird"), Value: "x"}}
			return a
		}()},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.agent.Validate(); err == nil {
				t.Errorf("Validate accepted a malformed agent (%s)", c.name)
			}
		})
	}

	// A well-formed agent with valid typed params must pass.
	good := base
	good.Params = []Param{
		{Key: "model", Type: ParamString, Value: "default"},
		{Key: "temperature", Type: ParamNumber, Value: "0.2"},
		{Key: "stream", Type: ParamBool, Value: "true"},
	}
	if err := good.Validate(); err != nil {
		t.Errorf("Validate rejected a well-formed agent: %v", err)
	}
}

// TestRenderParamTypesAreUnquoted covers the type→syntax mapping: numbers and
// bools render bare while strings are quoted when needed, so a parser reads each
// value as its declared type.
func TestRenderParamValueByType(t *testing.T) {
	cases := []struct {
		param Param
		want  string
	}{
		{Param{Key: "n", Type: ParamNumber, Value: "0.2"}, "0.2"},
		{Param{Key: "b", Type: ParamBool, Value: "true"}, "true"},
		{Param{Key: "s", Type: ParamString, Value: "default"}, "default"},
		{Param{Key: "s", Type: ParamString, Value: "true"}, `"true"`}, // string that looks boolean must be quoted
	}
	for _, c := range cases {
		if got := renderParamValue(c.param); got != c.want {
			t.Errorf("renderParamValue(%+v) = %q, want %q", c.param, got, c.want)
		}
	}
}
