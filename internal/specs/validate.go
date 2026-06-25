package specs

import (
	"fmt"
	"sort"
	"strings"
)

// The canonical brief/spec schema (R1/R3).
//
// This is the single, legible source of truth for what makes a brief or a spec
// valid. It mirrors the actionable, single-pass style of the prompts validator —
// duplicated, not imported, because specs owns its own field set. The rules:
//
//	Brief        Required  Rule
//	-----        --------  ----
//	slug         yes       non-empty, kebab-case
//	title        yes       non-empty (trimmed)
//	body         no        arbitrary Markdown; persisted verbatim (R6), not validated
//
//	Spec         Required  Rule
//	----         --------  ----
//	slug         yes       non-empty, kebab-case
//	title        yes       non-empty (trimmed)
//	brief        yes       non-empty (the R8/CA7 trace to the originating brief)
//	body         no        arbitrary Markdown; persisted verbatim (R6), not validated
//
// Validate is a pure function of the in-memory model: no I/O, no backend calls.

// SchemaError is a single actionable validation finding: which field failed, what
// was observed, and what the schema expected. The three parts let a user fix the
// problem without guessing. Mirrors prompts.SchemaError.
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

// ValidationError aggregates every finding for one artifact: the validator reports
// all detectable problems in a single pass rather than stopping at the first, so a
// user fixes them in one cycle. It implements error so it flows through the
// existing `error`-returning gates unchanged. A non-nil *ValidationError always
// carries at least one finding. Mirrors prompts.ValidationError.
type ValidationError struct {
	// Slug is the slug of the offending artifact, echoed for context. It may itself
	// be invalid (that is one of the findings); it is included so a caller can tell
	// which source failed.
	Slug string
	// Kind names which artifact failed ("brief" or "spec") so the message is
	// unambiguous when both share a slug.
	Kind string
	// Errors are the findings in stable, deterministic order (R6).
	Errors []SchemaError
}

// Error renders all findings, one per line, prefixed with the artifact kind and
// slug so the message is self-describing and byte-stable for a given model.
func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %q is invalid (%d issue%s):", e.Kind, e.Slug, len(e.Errors), pluralS(len(e.Errors)))
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

// Validate checks a brief against the canonical schema and returns nil when it is
// valid (R1) or a *ValidationError listing every finding when it is not. It is the
// single gate reused by the capture flow so "what is a valid brief" is defined in
// exactly one place. Findings are collected in a fixed sequence and stable-sorted
// so the output is fully deterministic (R6). The function is pure and performs no
// I/O.
func (b Brief) Validate() error {
	var findings []SchemaError
	findings = appendSlugFinding(findings, b.Slug)
	findings = appendTitleFinding(findings, b.Title)

	if len(findings) == 0 {
		return nil
	}
	sortFindings(findings)
	return &ValidationError{Slug: b.Slug, Kind: kindBrief, Errors: findings}
}

// Validate checks a spec against the canonical schema, additionally requiring the
// `brief` trace (R8/CA7). Same single-pass, deterministic contract as Brief.Validate.
func (s Spec) Validate() error {
	var findings []SchemaError
	findings = appendSlugFinding(findings, s.Slug)
	findings = appendTitleFinding(findings, s.Title)

	if trimmedEmpty(s.BriefRef) {
		findings = append(findings, SchemaError{
			Field:    FieldBrief,
			Observed: "empty",
			Expected: "a reference to the originating brief (the brief -> spec trace)",
		})
	}

	if len(findings) == 0 {
		return nil
	}
	sortFindings(findings)
	return &ValidationError{Slug: s.Slug, Kind: kindSpec, Errors: findings}
}

// appendSlugFinding adds a slug finding when slug is empty or not kebab-case. Shared
// by both artifacts so the slug rule (R3) is defined once.
func appendSlugFinding(findings []SchemaError, slug string) []SchemaError {
	switch {
	case slug == "":
		return append(findings, SchemaError{
			Field:    FieldSlug,
			Observed: "empty",
			Expected: "a non-empty kebab-case slug (e.g. my-feature)",
		})
	case !IsKebabCase(slug):
		return append(findings, SchemaError{
			Field:    FieldSlug,
			Observed: fmtQuote(slug),
			Expected: "kebab-case: lowercase letters/digits in dash-separated segments (e.g. my-feature)",
		})
	}
	return findings
}

// appendTitleFinding adds a title finding when title is empty after trimming.
func appendTitleFinding(findings []SchemaError, title string) []SchemaError {
	if trimmedEmpty(title) {
		return append(findings, SchemaError{
			Field:    FieldTitle,
			Observed: "empty",
			Expected: "a non-empty title",
		})
	}
	return findings
}

// sortFindings imposes a fully deterministic order on the findings (R6),
// independent of how they were collected: by a fixed schema-order rank (slug <
// title < brief), then by field name, then by observed text as a final tie-break.
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
// reader expects (slug, title, brief) rather than alphabetically.
func fieldRank(field string) int {
	switch field {
	case FieldSlug:
		return 0
	case FieldTitle:
		return 1
	case FieldBrief:
		return 2
	default:
		return 3
	}
}
