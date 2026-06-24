package catalog

import (
	"fmt"
	"sort"
	"strings"
)

// The canonical agent schema (R1/CA1).
//
// This is the single, legible source of truth for what makes an agent definition
// valid. Each field below states its rule; ValidateAgent enforces exactly these
// rules and nothing dispersed elsewhere. The schema is intentionally small and
// closed — it covers the Phase-1 canonical agent (init.md §5): an identifier, a
// role, a prompt, optional parameters and a format version. PRD §15 leaves the
// *fine-grained* schema as an open decision; this ticket fixes only the basic
// rules the epic requires, so extending it later (e.g. richer parameter typing)
// is an additive change to this one file.
//
//	Field         Required  Rule
//	-----         --------  ----
//	id            yes       non-empty, kebab-case (init.md §7)
//	role          yes       non-empty (trimmed)
//	prompt        yes       non-empty (trimmed)
//	parameters    no        each key non-empty and unique; each value a known
//	                        ParamType (string | number | bool)
//	version       —         stamped by the renderer (DefinitionVersion); not a
//	                        user-authored field, so it is not validated as input
//
// The schema validator does not execute the agent or touch a backend (R7): it is
// a pure function of the in-memory definition.

// Schema field names. They are exported as constants so callers (the CLI's
// actionable output, tests, a future TUI form) refer to fields by a stable
// identifier instead of hardcoding strings, and so the field names a user sees in
// an error match the canonical vocabulary exactly.
const (
	FieldID         = "id"
	FieldRole       = "role"
	FieldPrompt     = "prompt"
	FieldParameters = "parameters"
)

// SchemaError is a single actionable validation finding (R3/CA3): which field
// failed, what was observed, and what the schema expected. The three parts let a
// user fix the problem without guessing — the RF-9.3 "actionable error" contract.
type SchemaError struct {
	// Field is the canonical field (or "parameters[<key>]" for a parameter) the
	// finding is about.
	Field string
	// Observed describes what the definition actually contained.
	Observed string
	// Expected describes what the schema requires instead.
	Expected string
}

// Error renders one finding as a single actionable line. The field name is first
// so findings read as "field: …", and both the observed and expected halves are
// included so the message is self-contained.
func (e SchemaError) Error() string {
	return fmt.Sprintf("%s: observed %s; expected %s", e.Field, e.Observed, e.Expected)
}

// ValidationError aggregates every finding for one agent (R4/CA5): the validator
// reports all detectable problems in a single pass rather than stopping at the
// first, so a user fixes them in one cycle. It implements error so it flows
// through the existing `error`-returning gates unchanged. A non-nil
// *ValidationError always carries at least one finding.
type ValidationError struct {
	// AgentID is the id of the offending agent, echoed for context. It may itself
	// be invalid (that is one of the findings); it is included so a batch caller
	// (directory import) can tell which source failed.
	AgentID string
	// Errors are the findings in stable, deterministic order (R5/CA6).
	Errors []SchemaError
}

// Error renders all findings, one per line, prefixed with the agent id so the
// message is self-describing even when several agents are validated in a batch.
// The order is the deterministic order ValidateAgent produced, so the rendered
// message is byte-stable for a given definition.
func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "agent %q is invalid (%d issue%s):", e.AgentID, len(e.Errors), plivalS(len(e.Errors)))
	for _, se := range e.Errors {
		b.WriteString("\n  - ")
		b.WriteString(se.Error())
	}
	return b.String()
}

// plivalS is a tiny local pluralizer for the aggregate message (the cmd package
// has its own; the core must not depend on it).
func plivalS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// ValidateAgent checks an agent against the canonical schema and returns nil when
// it is valid (R2/CA2) or a *ValidationError listing every finding when it is not
// (R3/R4). It is the single gate reused by the catalog flows — materialize, clone,
// edit, import — so "what is a valid agent" is defined in exactly one place (R6).
//
// It collects findings in a fixed sequence — id, then role, then prompt, then each
// parameter in declaration order with its sub-rules in a fixed order — and then
// stable-sorts them by field so the output order is fully deterministic and never
// depends on map iteration (R5/CA6). The function is pure and performs no I/O and
// no backend calls (R7).
func ValidateAgent(a Agent) error {
	var findings []SchemaError

	// id: required, non-empty, kebab-case.
	switch {
	case a.ID == "":
		findings = append(findings, SchemaError{
			Field:    FieldID,
			Observed: "empty",
			Expected: "a non-empty kebab-case identifier (e.g. my-agent)",
		})
	case !IsKebabCase(a.ID):
		findings = append(findings, SchemaError{
			Field:    FieldID,
			Observed: fmt.Sprintf("%q", a.ID),
			Expected: "kebab-case: lowercase letters/digits in dash-separated segments (e.g. my-agent)",
		})
	}

	// role: required, non-empty after trimming.
	if strings.TrimSpace(a.Role) == "" {
		findings = append(findings, SchemaError{
			Field:    FieldRole,
			Observed: "empty",
			Expected: "a non-empty role/description",
		})
	}

	// prompt: required, non-empty after trimming.
	if strings.TrimSpace(a.Prompt) == "" {
		findings = append(findings, SchemaError{
			Field:    FieldPrompt,
			Observed: "empty",
			Expected: "a non-empty prompt",
		})
	}

	// parameters: optional; each key non-empty and unique, each value a known type.
	findings = append(findings, validateParams(a.Params)...)

	if len(findings) == 0 {
		return nil
	}

	sortFindings(findings)
	return &ValidationError{AgentID: a.ID, Errors: findings}
}

// validateParams checks the optional parameters block. Findings are produced in
// declaration order (already deterministic for a given Agent); duplicate detection
// uses first-seen tracking so the *second* occurrence is the one flagged, which
// matches how a user reads the list top-to-bottom.
func validateParams(params []Param) []SchemaError {
	var findings []SchemaError
	seen := make(map[string]int, len(params))
	for i, p := range params {
		field := fmt.Sprintf("%s[%d]", FieldParameters, i)
		if strings.TrimSpace(p.Key) == "" {
			findings = append(findings, SchemaError{
				Field:    field,
				Observed: "empty key",
				Expected: "a non-empty parameter key",
			})
			// Without a key the remaining checks for this entry are meaningless.
			continue
		}
		// Re-key the field on the parameter name now that we know it, so the finding
		// points at the human-recognizable key rather than just an index.
		field = fmt.Sprintf("%s[%s]", FieldParameters, p.Key)
		if first, dup := seen[p.Key]; dup {
			findings = append(findings, SchemaError{
				Field:    field,
				Observed: fmt.Sprintf("duplicate key (first defined at index %d)", first),
				Expected: "each parameter key to be unique",
			})
		} else {
			seen[p.Key] = i
		}
		switch p.Type {
		case ParamString, ParamNumber, ParamBool:
			// known type
		default:
			findings = append(findings, SchemaError{
				Field:    field,
				Observed: fmt.Sprintf("type %q", string(p.Type)),
				Expected: fmt.Sprintf("a known type: %s, %s or %s", ParamString, ParamNumber, ParamBool),
			})
		}
	}
	return findings
}

// sortFindings imposes a fully deterministic order on the findings (R5/CA6),
// independent of how they were collected: primarily by field name, then by the
// observed text as a tie-breaker. Because field names embed the canonical order
// implicitly ("id" < "parameters…" < "prompt" < "role" lexically is not the
// schema order), we map each top-level field to a fixed rank first so the headline
// fields read in schema order (id, role, prompt, parameters), with parameter
// findings grouped last and ordered by their field key.
func sortFindings(findings []SchemaError) {
	sort.SliceStable(findings, func(i, j int) bool {
		ri, rj := fieldRank(findings[i].Field), fieldRank(findings[j].Field)
		if ri != rj {
			return ri < rj
		}
		if findings[i].Field != findings[j].Field {
			return findings[i].Field < findings[j].Field
		}
		return findings[i].Observed < findings[j].Observed
	})
}

// fieldRank maps a field to its schema-order rank so findings sort in the order a
// reader expects (id, role, prompt, then parameters) rather than alphabetically.
// Parameter fields ("parameters[...]") all share the last rank and are then
// ordered by their full field string in sortFindings.
func fieldRank(field string) int {
	switch {
	case field == FieldID:
		return 0
	case field == FieldRole:
		return 1
	case field == FieldPrompt:
		return 2
	case strings.HasPrefix(field, FieldParameters):
		return 3
	default:
		return 4
	}
}
