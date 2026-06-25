package architecture

import (
	"fmt"
	"sort"
	"strings"
)

// The canonical architecture-document schema (R1/R2).
//
// This is the single, legible source of truth for what makes a document valid. It
// mirrors the actionable, single-pass style of the specs/prompts validators —
// duplicated, not imported, because architecture owns its own field set. The rules:
//
//	Field        Required  Rule
//	-----        --------  ----
//	slug         yes       non-empty, kebab-case
//	title        yes       non-empty (trimmed)
//	spec         no        optional link to the originating spec (R3); not validated
//	body         no        arbitrary Markdown; persisted verbatim (R6), not validated
//
// The spec link is intentionally NOT validated for existence here: Validate is a
// pure function of the in-memory model (no I/O), and whether a referenced spec file
// exists is a filesystem concern resolved at the store/CLI layer (a friendly check at
// create time), not a property of the document's own shape. End-to-end traceability
// is ticket 05-04.

// SchemaError is a single actionable validation finding: which field failed, what
// was observed, and what the schema expected. Mirrors specs.SchemaError.
type SchemaError struct {
	Field    string
	Observed string
	Expected string
}

// Error renders one finding as a single actionable line, self-contained with both
// the observed and expected halves.
func (e SchemaError) Error() string {
	return fmt.Sprintf("%s: observed %s; expected %s", e.Field, e.Observed, e.Expected)
}

// ValidationError aggregates every finding for one document: the validator reports
// all detectable problems in a single pass rather than stopping at the first, so a
// user fixes them in one cycle. It implements error so it flows through the existing
// `error`-returning gates unchanged. A non-nil *ValidationError always carries at
// least one finding. Mirrors specs.ValidationError.
type ValidationError struct {
	// Slug is the slug of the offending document, echoed for context. It may itself be
	// invalid (that is one of the findings); it is included so a caller can tell which
	// source failed.
	Slug string
	// Errors are the findings in stable, deterministic order (R6).
	Errors []SchemaError
}

// Error renders all findings, one per line, prefixed with the document slug so the
// message is self-describing and byte-stable for a given model.
func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "architecture document %q is invalid (%d issue%s):", e.Slug, len(e.Errors), pluralS(len(e.Errors)))
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

// Validate checks a document against the canonical schema and returns nil when it is
// valid (R2) or a *ValidationError listing every finding when it is not. It is the
// single gate reused by Create and Edit so "what is a valid document" is defined in
// exactly one place. Findings are collected in a fixed sequence (slug, then title)
// and stable-sorted by a schema-order rank so the output is fully deterministic (R6).
// The function is pure and performs no I/O.
func (d Document) Validate() error {
	var findings []SchemaError

	switch {
	case d.Slug == "":
		findings = append(findings, SchemaError{
			Field:    FieldSlug,
			Observed: "empty",
			Expected: "a non-empty kebab-case slug (e.g. payments-arch)",
		})
	case !IsKebabCase(d.Slug):
		findings = append(findings, SchemaError{
			Field:    FieldSlug,
			Observed: fmtQuote(d.Slug),
			Expected: "kebab-case: lowercase letters/digits in dash-separated segments (e.g. payments-arch)",
		})
	}

	if trimmedEmpty(d.Title) {
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
	return &ValidationError{Slug: d.Slug, Errors: findings}
}

// sortFindings imposes a fully deterministic order on the findings (R6), independent
// of how they were collected: by a fixed schema-order rank (slug < title), then by
// field name, then by observed text as a final tie-break.
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
// reader expects (slug, title) rather than alphabetically.
func fieldRank(field string) int {
	switch field {
	case FieldSlug:
		return 0
	case FieldTitle:
		return 1
	default:
		return 2
	}
}
