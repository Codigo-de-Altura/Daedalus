package prompts

import (
	"fmt"
	"sort"
	"strings"
)

// The canonical prompt schema (R2/R8).
//
// This is the single, legible source of truth for what makes a prompt valid. It
// mirrors the actionable, single-pass style of the catalog's schema validator
// (catalog/schema.go) — duplicated, not imported, because prompts owns its own
// (different) field set. The rules:
//
//	Field         Required  Rule
//	-----         --------  ----
//	id            yes       non-empty, kebab-case
//	kind          yes       one of: global | shared
//	title         yes       non-empty (trimmed)
//	description   no        free text; not validated
//	body          no        arbitrary Markdown; persisted verbatim (R7), not validated
//
// Validate is a pure function of the in-memory Prompt: no I/O, no backend calls.

// Schema field names, exported as constants so callers (the CLI's actionable
// output, tests) refer to fields by a stable identifier instead of hardcoding
// strings, and so the field names a user sees in an error match the canonical
// vocabulary exactly.
const (
	FieldID    = "id"
	FieldKind  = "kind"
	FieldTitle = "title"
)

// SchemaError is a single actionable validation finding (R8): which field
// failed, what was observed, and what the schema expected. The three parts let a
// user fix the problem without guessing. It mirrors catalog.SchemaError but is a
// distinct type so prompts does not depend on the catalog package.
type SchemaError struct {
	// Field is the canonical field the finding is about.
	Field string
	// Observed describes what the prompt actually contained.
	Observed string
	// Expected describes what the schema requires instead.
	Expected string
}

// Error renders one finding as a single actionable line ("field: …"), self-
// contained with both the observed and expected halves.
func (e SchemaError) Error() string {
	return fmt.Sprintf("%s: observed %s; expected %s", e.Field, e.Observed, e.Expected)
}

// ValidationError aggregates every finding for one prompt (R8): the validator
// reports all detectable problems in a single pass rather than stopping at the
// first, so a user fixes them in one cycle. It implements error so it flows
// through the existing `error`-returning gates unchanged. A non-nil
// *ValidationError always carries at least one finding.
type ValidationError struct {
	// PromptID is the id of the offending prompt, echoed for context. It may
	// itself be invalid (that is one of the findings); it is included so a caller
	// can tell which source failed.
	PromptID string
	// Errors are the findings in stable, deterministic order (R5).
	Errors []SchemaError
}

// Error renders all findings, one per line, prefixed with the prompt id so the
// message is self-describing. The order is the deterministic order Validate
// produced, so the rendered message is byte-stable for a given prompt.
func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "prompt %q is invalid (%d issue%s):", e.PromptID, len(e.Errors), pluralS(len(e.Errors)))
	for _, se := range e.Errors {
		b.WriteString("\n  - ")
		b.WriteString(se.Error())
	}
	return b.String()
}

// pluralS is a tiny local pluralizer for the aggregate message.
func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// Validate checks a prompt against the canonical schema and returns nil when it
// is valid (R2) or a *ValidationError listing every finding when it is not (R8).
// It is the single gate reused by Create and Edit so "what is a valid prompt" is
// defined in exactly one place. Findings are collected in a fixed sequence (id,
// then kind, then title) and stable-sorted by a schema-order rank so the output
// is fully deterministic (R5). The function is pure and performs no I/O.
func (p Prompt) Validate() error {
	var findings []SchemaError

	// id: required, non-empty, kebab-case.
	switch {
	case p.ID == "":
		findings = append(findings, SchemaError{
			Field:    FieldID,
			Observed: "empty",
			Expected: "a non-empty kebab-case identifier (e.g. my-prompt)",
		})
	case !IsKebabCase(p.ID):
		findings = append(findings, SchemaError{
			Field:    FieldID,
			Observed: fmtQuote(p.ID),
			Expected: "kebab-case: lowercase letters/digits in dash-separated segments (e.g. my-prompt)",
		})
	}

	// kind: required, one of the known kinds.
	if !p.Kind.IsValidKind() {
		observed := "empty"
		if p.Kind != "" {
			observed = fmtQuote(string(p.Kind))
		}
		findings = append(findings, SchemaError{
			Field:    FieldKind,
			Observed: observed,
			Expected: fmt.Sprintf("one of: %s, %s", KindGlobal, KindShared),
		})
	}

	// title: required, non-empty after trimming.
	if trimmedEmpty(p.Title) {
		findings = append(findings, SchemaError{
			Field:    FieldTitle,
			Observed: "empty",
			Expected: "a non-empty title",
		})
	}

	if len(findings) == 0 {
		return nil
	}

	sortFindings(findings)
	return &ValidationError{PromptID: p.ID, Errors: findings}
}

// sortFindings imposes a fully deterministic order on the findings (R5),
// independent of how they were collected: by a fixed schema-order rank (id <
// kind < title), then by field name, then by observed text as a final tie-break.
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
// reader expects (id, kind, title) rather than alphabetically.
func fieldRank(field string) int {
	switch field {
	case FieldID:
		return 0
	case FieldKind:
		return 1
	case FieldTitle:
		return 2
	default:
		return 3
	}
}
