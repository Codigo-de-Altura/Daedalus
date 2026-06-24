package catalog

import (
	"errors"
	"strings"
	"testing"
)

// validAgent is a minimal agent that satisfies the canonical schema, used as the
// baseline that the negative cases mutate.
func validAgent() Agent {
	return Agent{
		ID:     "my-agent",
		Role:   "Does a thing.",
		Prompt: "# Prompt\n\nDo the thing.",
		Params: []Param{{Key: "model", Type: ParamString, Value: "default"}},
	}
}

// asValidationError type-asserts err to *ValidationError, failing the test if it
// is not one. Used by the negative cases to inspect the findings.
func asValidationError(t *testing.T, err error) *ValidationError {
	t.Helper()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("error %v (%T) is not a *ValidationError", err, err)
	}
	return ve
}

// findingFor returns the first finding whose Field matches, or nil.
func findingFor(ve *ValidationError, field string) *SchemaError {
	for i := range ve.Errors {
		if ve.Errors[i].Field == field {
			return &ve.Errors[i]
		}
	}
	return nil
}

// TestSchemaValidAgentPasses covers CA2: a valid definition validates with no
// errors.
func TestSchemaValidAgentPasses(t *testing.T) {
	if err := ValidateAgent(validAgent()); err != nil {
		t.Errorf("ValidateAgent(valid) = %v, want nil", err)
	}
	// An agent with no parameters at all is also valid (parameters is optional).
	a := validAgent()
	a.Params = nil
	if err := ValidateAgent(a); err != nil {
		t.Errorf("ValidateAgent(no params) = %v, want nil", err)
	}
}

// TestSchemaMissingRequiredField covers CA3: a missing required field fails with a
// finding carrying field, observed and expected.
func TestSchemaMissingRequiredField(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Agent)
		field   string
		obsWant string
	}{
		{"missing role", func(a *Agent) { a.Role = "  " }, FieldRole, "empty"},
		{"missing prompt", func(a *Agent) { a.Prompt = "" }, FieldPrompt, "empty"},
		{"missing id", func(a *Agent) { a.ID = "" }, FieldID, "empty"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := validAgent()
			c.mutate(&a)
			ve := asValidationError(t, ValidateAgent(a))
			f := findingFor(ve, c.field)
			if f == nil {
				t.Fatalf("no finding for field %q; got %+v", c.field, ve.Errors)
			}
			if f.Observed != c.obsWant {
				t.Errorf("Observed = %q, want %q", f.Observed, c.obsWant)
			}
			if strings.TrimSpace(f.Expected) == "" {
				t.Errorf("Expected is empty; an actionable error must say what was expected")
			}
		})
	}
}

// TestSchemaNonKebabID covers CA4: a non-kebab-case id fails with an actionable
// finding naming the id field and the kebab-case expectation.
func TestSchemaNonKebabID(t *testing.T) {
	a := validAgent()
	a.ID = "Bad_Id"
	ve := asValidationError(t, ValidateAgent(a))
	f := findingFor(ve, FieldID)
	if f == nil {
		t.Fatalf("no id finding; got %+v", ve.Errors)
	}
	if !strings.Contains(f.Observed, "Bad_Id") {
		t.Errorf("Observed %q does not echo the offending id", f.Observed)
	}
	if !strings.Contains(f.Expected, "kebab-case") {
		t.Errorf("Expected %q does not mention kebab-case", f.Expected)
	}
}

// TestSchemaParameterRules covers the optional parameters block rules: empty key,
// duplicate key and unknown type each produce a finding.
func TestSchemaParameterRules(t *testing.T) {
	a := validAgent()
	a.Params = []Param{
		{Key: "", Type: ParamString, Value: "x"},               // empty key
		{Key: "model", Type: ParamString, Value: "a"},          // ok
		{Key: "model", Type: ParamString, Value: "b"},          // duplicate
		{Key: "weird", Type: ParamType("mystery"), Value: "y"}, // unknown type
	}
	ve := asValidationError(t, ValidateAgent(a))

	// Expect at least: empty-key, duplicate, unknown-type findings.
	var emptyKey, dup, unknown bool
	for _, e := range ve.Errors {
		switch {
		case strings.Contains(e.Observed, "empty key"):
			emptyKey = true
		case strings.Contains(e.Observed, "duplicate"):
			dup = true
		case strings.Contains(e.Observed, "type"):
			unknown = true
		}
	}
	if !emptyKey || !dup || !unknown {
		t.Errorf("missing a parameter finding: emptyKey=%v dup=%v unknown=%v; got %+v",
			emptyKey, dup, unknown, ve.Errors)
	}
}

// TestSchemaReportsAllInOnePass covers CA5: a definition with several problems
// reports them all at once, not just the first.
func TestSchemaReportsAllInOnePass(t *testing.T) {
	bad := Agent{
		ID:     "Bad_Id",                                              // non-kebab
		Role:   "",                                                    // empty
		Prompt: "",                                                    // empty
		Params: []Param{{Key: "k", Type: ParamType("x"), Value: "v"}}, // unknown type
	}
	ve := asValidationError(t, ValidateAgent(bad))
	if len(ve.Errors) < 4 {
		t.Errorf("got %d findings, want >= 4 (id, role, prompt, param) in one pass: %+v",
			len(ve.Errors), ve.Errors)
	}
	for _, field := range []string{FieldID, FieldRole, FieldPrompt} {
		if findingFor(ve, field) == nil {
			t.Errorf("missing finding for %q; got %+v", field, ve.Errors)
		}
	}
}

// TestSchemaDeterministicOrder covers CA6: the same definition produces the same
// verdict and the same findings in the same order across repeated runs, and the
// order follows the schema sequence (id, role, prompt, parameters).
func TestSchemaDeterministicOrder(t *testing.T) {
	bad := Agent{
		ID:     "Bad_Id",
		Role:   "",
		Prompt: "",
		Params: []Param{{Key: "k", Type: ParamType("x"), Value: "v"}},
	}

	first := asValidationError(t, ValidateAgent(bad))
	for i := 0; i < 5; i++ {
		again := asValidationError(t, ValidateAgent(bad))
		if len(again.Errors) != len(first.Errors) {
			t.Fatalf("finding count not stable: %d vs %d", len(again.Errors), len(first.Errors))
		}
		for j := range first.Errors {
			if again.Errors[j] != first.Errors[j] {
				t.Errorf("finding %d not stable: %+v vs %+v", j, again.Errors[j], first.Errors[j])
			}
		}
	}

	// Order must follow the schema: id before role before prompt before parameters.
	order := map[string]int{}
	for i, e := range first.Errors {
		base := e.Field
		if idx := strings.IndexByte(base, '['); idx >= 0 {
			base = base[:idx] // collapse "parameters[k]" -> "parameters"
		}
		if _, seen := order[base]; !seen {
			order[base] = i
		}
	}
	if order[FieldID] > order[FieldRole] || order[FieldRole] > order[FieldPrompt] || order[FieldPrompt] > order[FieldParameters] {
		t.Errorf("findings not in schema order: %+v", first.Errors)
	}
}

// TestValidationErrorMessage covers the aggregate error rendering: it names the
// agent and lists every finding (one per line), so the CLI/log output is
// self-describing.
func TestValidationErrorMessage(t *testing.T) {
	a := validAgent()
	a.Role = ""
	a.Prompt = ""
	msg := ValidateAgent(a).Error()
	if !strings.Contains(msg, "my-agent") {
		t.Errorf("message does not name the agent; got:\n%s", msg)
	}
	if !strings.Contains(msg, FieldRole) || !strings.Contains(msg, FieldPrompt) {
		t.Errorf("message does not list all failing fields; got:\n%s", msg)
	}
}

// TestValidateMethodDelegates covers R6 integration at the method level: the
// Agent.Validate() alias returns the same rich error type as ValidateAgent.
func TestValidateMethodDelegates(t *testing.T) {
	a := validAgent()
	a.ID = ""
	var ve *ValidationError
	if !errors.As(a.Validate(), &ve) {
		t.Errorf("Agent.Validate() did not return a *ValidationError")
	}
}

// TestBuiltinsPassSchema is the integration guarantee that the formal schema does
// not retroactively reject the shipped built-ins (the 02-01 catalog), nor the
// agents produced by clone/import round-trips elsewhere in the suite.
func TestBuiltinsPassSchema(t *testing.T) {
	for _, e := range Builtin.List() {
		a, err := Builtin.Get(e.ID)
		if err != nil {
			t.Fatalf("Get(%q): %v", e.ID, err)
		}
		if err := ValidateAgent(a); err != nil {
			t.Errorf("built-in %q fails the canonical schema: %v", e.ID, err)
		}
	}
}
