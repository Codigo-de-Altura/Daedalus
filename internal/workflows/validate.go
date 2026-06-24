package workflows

import (
	"fmt"
	"sort"
	"strings"
)

// The canonical workflow/phase schema (R1/R7/R8).
//
// This is the single, legible source of truth for what makes a workflow's
// *structure* valid. It mirrors the actionable, single-pass style of the prompts
// and catalog validators — duplicated, not imported, because workflows owns its
// own (different) field set. The rules, per phase:
//
//	Field         Required  Rule
//	-----         --------  ----
//	id            yes       non-empty, kebab-case, unique within the workflow
//	agent         yes       non-empty (trimmed)
//	gate          yes       non-empty (trimmed)
//	inputs        no        list of references; not validated for existence
//	outputs       no        list of references; not validated for existence
//	depends_on    no        list of references (the DAG edges); not validated here
//
// Crucially, this is STRUCTURAL validation only (R8 / ticket scope): it checks
// each phase against the schema and that phase ids are unique, but it does NOT
// perform *semantic* graph validation — no cycle detection, no check that a
// depends_on/inputs reference resolves to an existing phase or artifact, no check
// that an agent exists. Those are ticket 04-03's responsibility. Validate is a
// pure function of the in-memory Workflow: no I/O, no backend calls (R8).

// SchemaError is a single actionable validation finding (R7): which field failed,
// what was observed, and what the schema expected. The three parts let a user fix
// the problem without guessing. For a phase-scoped finding, Phase carries the
// offending phase's id (or its position when the id itself is the problem) so the
// message names exactly which phase is invalid.
type SchemaError struct {
	// Phase identifies the offending phase (its id, or a positional label like
	// "#2" when the id is missing/invalid). Empty for a workflow-level finding.
	Phase string
	// Field is the canonical field the finding is about.
	Field string
	// Observed describes what the phase actually contained.
	Observed string
	// Expected describes what the schema requires instead.
	Expected string
}

// Error renders one finding as a single actionable line, scoped to its phase when
// it has one, self-contained with both the observed and expected halves.
func (e SchemaError) Error() string {
	if e.Phase != "" {
		return fmt.Sprintf("phase %s: %s: observed %s; expected %s", e.Phase, e.Field, e.Observed, e.Expected)
	}
	return fmt.Sprintf("%s: observed %s; expected %s", e.Field, e.Observed, e.Expected)
}

// ValidationError aggregates every finding for one workflow (R7): the validator
// reports all detectable problems in a single pass rather than stopping at the
// first, so a user fixes them in one cycle. It implements error so it flows
// through the existing `error`-returning gates unchanged. A non-nil
// *ValidationError always carries at least one finding.
type ValidationError struct {
	// WorkflowName is the name of the offending workflow, echoed for context. It
	// may be empty (an in-memory model being created) and is included so a caller
	// can tell which source failed.
	WorkflowName string
	// Errors are the findings in stable, deterministic order (R4).
	Errors []SchemaError
}

// Error renders all findings, one per line, prefixed with the workflow name so
// the message is self-describing. The order is the deterministic order Validate
// produced, so the rendered message is byte-stable for a given workflow.
func (e *ValidationError) Error() string {
	var b strings.Builder
	name := e.WorkflowName
	if name == "" {
		name = "(unnamed)"
	}
	fmt.Fprintf(&b, "workflow %q is invalid (%d issue%s):", name, len(e.Errors), pluralS(len(e.Errors)))
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

// Validate checks a workflow against the canonical structural schema and returns
// nil when it is valid or a *ValidationError listing every finding when it is not
// (R7). It is the single gate reused by the persisting operations (store.go) so
// "what is a structurally valid workflow" is defined in exactly one place.
//
// Findings are collected phase-by-phase in document order, each phase's fields in
// schema order, then stable-sorted by (phase position, field rank) so the output
// is fully deterministic (R4). The function is pure and performs no I/O (R8).
func (w Workflow) Validate() error {
	var findings []SchemaError

	// Track seen ids to flag duplicates: a phase id is the DAG node key and must be
	// unique within the workflow.
	seen := make(map[string]int)

	for i, p := range w.Phases {
		// A stable per-phase label for the finding: the id when usable, else the
		// 1-based position, so a phase with an empty/invalid id is still pinpointed.
		label := p.ID
		if trimmedEmpty(label) {
			label = fmt.Sprintf("#%d", i+1)
		}

		// id: required, non-empty, kebab-case, unique.
		switch {
		case trimmedEmpty(p.ID):
			findings = append(findings, SchemaError{
				Phase: label, Field: FieldID,
				Observed: "empty",
				Expected: "a non-empty kebab-case identifier (e.g. spec)",
			})
		case !IsKebabCase(p.ID):
			findings = append(findings, SchemaError{
				Phase: label, Field: FieldID,
				Observed: fmtQuote(p.ID),
				Expected: "kebab-case: lowercase letters/digits in dash-separated segments (e.g. write-spec)",
			})
		default:
			if prev, dup := seen[p.ID]; dup {
				findings = append(findings, SchemaError{
					Phase: label, Field: FieldID,
					Observed: fmt.Sprintf("%s, already used by phase #%d", fmtQuote(p.ID), prev+1),
					Expected: "a phase id unique within the workflow",
				})
			} else {
				seen[p.ID] = i
			}
		}

		// agent: required, non-empty.
		if trimmedEmpty(p.Agent) {
			findings = append(findings, SchemaError{
				Phase: label, Field: FieldAgent,
				Observed: "empty",
				Expected: "a non-empty agent reference (e.g. analyst)",
			})
		}

		// gate: required, non-empty.
		if trimmedEmpty(p.Gate) {
			findings = append(findings, SchemaError{
				Phase: label, Field: FieldGate,
				Observed: "empty",
				Expected: "a non-empty gate reference (e.g. spec-gate)",
			})
		}
	}

	if len(findings) == 0 {
		return nil
	}

	sortFindings(findings, w)
	return &ValidationError{WorkflowName: w.Name, Errors: findings}
}

// sortFindings imposes a fully deterministic order on the findings (R4),
// independent of how they were collected: by the offending phase's position in
// the document, then by a fixed schema-order field rank, then by observed text as
// a final tie-break. A workflow-level finding (no phase) sorts first.
func sortFindings(findings []SchemaError, w Workflow) {
	pos := make(map[string]int)
	for i, p := range w.Phases {
		if _, ok := pos[p.ID]; !ok {
			pos[p.ID] = i
		}
	}
	rankPhase := func(label string) int {
		if label == "" {
			return -1
		}
		if i, ok := pos[label]; ok {
			return i
		}
		return len(w.Phases) // positional "#n" labels sort after id-labelled ones
	}
	sort.SliceStable(findings, func(i, j int) bool {
		pi, pj := rankPhase(findings[i].Phase), rankPhase(findings[j].Phase)
		if pi != pj {
			return pi < pj
		}
		ri, rj := fieldRank(findings[i].Field), fieldRank(findings[j].Field)
		if ri != rj {
			return ri < rj
		}
		return findings[i].Observed < findings[j].Observed
	})
}

// fieldRank maps a field to its schema-order rank so findings within a phase sort
// in the order a reader expects (id, agent, gate) rather than alphabetically.
func fieldRank(field string) int {
	switch field {
	case FieldID:
		return 0
	case FieldAgent:
		return 1
	case FieldGate:
		return 2
	default:
		return 3
	}
}
