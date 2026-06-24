package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

// TestRoundTripIsByteIdentical covers the central determinism guarantee edit
// relies on: materialize a built-in, load it back, and re-render — the bytes must
// be identical to the originals for every built-in agent. This proves Load is the
// exact inverse of the renderer (no type/quoting/ordering drift on the way out).
func TestRoundTripIsByteIdentical(t *testing.T) {
	for _, e := range Builtin.List() {
		t.Run(e.ID, func(t *testing.T) {
			agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
			if _, err := Builtin.Materialize(agentsRoot, e.ID); err != nil {
				t.Fatalf("Materialize: %v", err)
			}

			origDef := readFile(t, agentsRoot, e.ID, DefinitionFileName)
			origPrompt := readFile(t, agentsRoot, e.ID, PromptFileName)

			loaded, err := Load(agentsRoot, e.ID)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}

			if got := renderDefinition(loaded); got != origDef {
				t.Errorf("definition not byte-identical after load->render:\n--- orig ---\n%s\n--- got ---\n%s", origDef, got)
			}
			if got := renderPrompt(loaded); got != origPrompt {
				t.Errorf("prompt not byte-identical after load->render:\n--- orig ---\n%s\n--- got ---\n%s", origPrompt, got)
			}
		})
	}
}

// TestLoadReconstructsModel checks the loaded model matches the source agent's
// fields (id, role, prompt, params) — Load reads back what was written.
func TestLoadReconstructsModel(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
	if _, err := Builtin.Materialize(agentsRoot, "analyst"); err != nil {
		t.Fatalf("Materialize: %v", err)
	}
	src, err := Builtin.Get("analyst")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	loaded, err := Load(agentsRoot, "analyst")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.ID != src.ID {
		t.Errorf("id = %q, want %q", loaded.ID, src.ID)
	}
	if loaded.Role != src.Role {
		t.Errorf("role = %q, want %q", loaded.Role, src.Role)
	}
	if loaded.Prompt != src.Prompt {
		t.Errorf("prompt mismatch:\n--- loaded ---\n%s\n--- src ---\n%s", loaded.Prompt, src.Prompt)
	}
	if len(loaded.Params) != len(src.Params) {
		t.Fatalf("params length = %d, want %d", len(loaded.Params), len(src.Params))
	}
	for i := range src.Params {
		// Description is intentionally not persisted on disk (render.go keeps the
		// canonical YAML minimal), so a loaded param has none — compare only the
		// persisted fields (key, type, value).
		lp, sp := loaded.Params[i], src.Params[i]
		if lp.Key != sp.Key || lp.Type != sp.Type || lp.Value != sp.Value {
			t.Errorf("param[%d] = {%s %s %q}, want {%s %s %q}",
				i, lp.Key, lp.Type, lp.Value, sp.Key, sp.Type, sp.Value)
		}
	}
}

// TestParseParamValueTyping pins the type re-inference that keeps the round-trip
// stable: bare booleans/numbers recover their type, quoted scalars recover as
// strings (unquoted), bare safe text is a string.
func TestParseParamValueTyping(t *testing.T) {
	cases := []struct {
		token     string
		wantType  ParamType
		wantValue string
	}{
		{"true", ParamBool, "true"},
		{"false", ParamBool, "false"},
		{"0.2", ParamNumber, "0.2"},
		{"42", ParamNumber, "42"},
		{"default", ParamString, "default"},
		{`"true"`, ParamString, "true"}, // quoted boolean-looking text is a string
		{`"0.2"`, ParamString, "0.2"},   // quoted number-looking text is a string
		{`"a: b"`, ParamString, "a: b"}, // quoted scalar with an indicator char
		{`"a\"b"`, ParamString, `a"b`},  // escaped quote
		{`"a\\b"`, ParamString, `a\b`},  // escaped backslash
	}
	for _, c := range cases {
		gotType, gotVal, err := parseParamValue(c.token)
		if err != nil {
			t.Errorf("parseParamValue(%q): unexpected error %v", c.token, err)
			continue
		}
		if gotType != c.wantType || gotVal != c.wantValue {
			t.Errorf("parseParamValue(%q) = (%s, %q), want (%s, %q)",
				c.token, gotType, gotVal, c.wantType, c.wantValue)
		}
	}
}

// TestLoadRejectsMalformed covers the loader's defensive parsing: corrupt or
// out-of-shape definitions are reported as malformed, not half-loaded.
func TestLoadRejectsMalformed(t *testing.T) {
	cases := []struct {
		name string
		def  string
	}{
		{"unknown key", "id: x\nrole: r\nbogus: y\n"},
		{"missing id", "role: r\n"},
		{"missing role", "id: x\n"},
		{"params not block or empty", "id: x\nrole: r\nparameters: nope\n"},
		{"unterminated quote", "id: x\nrole: \"unterminated\nparameters: {}\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := parseDefinition(c.def); err == nil {
				t.Errorf("parseDefinition accepted malformed input (%s)", c.name)
			}
		})
	}
}

// TestLoadAbsentAgentIsNotFound covers the not-found path: loading an agent whose
// files are absent yields ErrAgentNotFound, distinct from a parse failure.
func TestLoadAbsentAgentIsNotFound(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
	if _, err := Load(agentsRoot, "ghost"); err == nil {
		t.Errorf("Load(absent) succeeded, want ErrAgentNotFound")
	}
}

// TestLoadEditedAgentRoundTrips checks that an edited (CLI-style, string-typed)
// definition also round-trips: load->render is stable after a real edit, not just
// for pristine built-ins.
func TestLoadEditedAgentRoundTrips(t *testing.T) {
	agentsRoot := filepath.Join(t.TempDir(), AgentsDir)
	if _, err := Builtin.Clone(agentsRoot, "analyst", "analyst-custom"); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if _, err := Builtin.Edit(agentsRoot, "analyst-custom", EditSpec{
		SetRole:   true,
		Role:      "Role with: a colon and # hash",
		SetParams: []Param{{Key: "note", Type: ParamString, Value: "value with spaces"}},
	}); err != nil {
		t.Fatalf("Edit: %v", err)
	}

	defBytes, err := os.ReadFile(filepath.Join(agentsRoot, "analyst-custom", DefinitionFileName))
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(agentsRoot, "analyst-custom")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got := renderDefinition(loaded); got != string(defBytes) {
		t.Errorf("edited definition not stable on load->render:\n--- on disk ---\n%s\n--- got ---\n%s", defBytes, got)
	}
}
