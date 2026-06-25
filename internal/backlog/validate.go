package backlog

import (
	"fmt"
	"sort"
	"strings"
)

// The canonical epic/ticket schema (R1/R2/R3/R6).
//
// This is the single, legible source of truth for what makes an epic or a ticket
// valid. It mirrors the actionable, single-pass style of the sibling validators. The
// rules:
//
//	Epic          Required  Rule
//	----          --------  ----
//	id            yes       well-formed epic-NN-<slug>
//	title         yes       non-empty (trimmed)
//	status        yes       one of the closed Status set
//	priority      yes       one of the closed Priority set
//	spec/arch     no        optional origin links (R5); not validated for existence
//	depends_on    no        list of ids; each non-empty (consistency, R4)
//	body          no        arbitrary Markdown; persisted verbatim (R6), not validated
//
//	Ticket        Required  Rule
//	------        --------  ----
//	id            yes       well-formed ticket-NN-MM-<slug>
//	epic          yes       well-formed epic-NN-<slug> (the parent link, R5)
//	title         yes       non-empty
//	status        yes       closed Status set
//	priority      yes       closed Priority set
//	spec/arch     no        optional; not validated for existence
//	depends_on    no        list of ids; each non-empty
//	body          no        verbatim
//
// Existence of referenced specs/architecture/epics is NOT checked here: Validate is a
// pure function of the in-memory model (no I/O). A friendly existence check happens at
// the CLI layer (as in 05-02); end-to-end traceability/graph verification is 05-04.

// SchemaError is a single actionable validation finding. Mirrors the sibling type.
type SchemaError struct {
	Field    string
	Observed string
	Expected string
}

// Error renders one finding as a single actionable line.
func (e SchemaError) Error() string {
	return fmt.Sprintf("%s: observed %s; expected %s", e.Field, e.Observed, e.Expected)
}

// ValidationError aggregates every finding for one artifact in a single pass, so a user
// fixes them in one cycle. It implements error. A non-nil *ValidationError always
// carries at least one finding.
type ValidationError struct {
	// ID is the id of the offending artifact, echoed for context (may itself be one of
	// the findings).
	ID string
	// Kind names which artifact failed ("epic" or "ticket").
	Kind string
	// Errors are the findings in stable, deterministic order (R6).
	Errors []SchemaError
}

// Error renders all findings, one per line, prefixed with kind and id.
func (e *ValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %q is invalid (%d issue%s):", e.Kind, e.ID, len(e.Errors), pluralS(len(e.Errors)))
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

// Validate checks an epic against the canonical schema and returns nil when valid or a
// *ValidationError listing every finding. Single-pass and deterministic (R6).
func (e Epic) Validate() error {
	var f []SchemaError

	if e.ID == "" {
		f = append(f, SchemaError{Field: FieldID, Observed: "empty",
			Expected: "a well-formed epic id (e.g. epic-05-sdd-backlog)"})
	} else if !IsEpicID(e.ID) {
		f = append(f, SchemaError{Field: FieldID, Observed: fmtQuote(e.ID),
			Expected: "epic-NN-<slug>: 'epic-', a number, '-', then a kebab-case slug"})
	}

	f = appendTitleFinding(f, e.Title)
	f = appendStatusFinding(f, e.Status)
	f = appendPriorityFinding(f, e.Priority)
	f = appendDependsOnFindings(f, e.DependsOn)

	if len(f) == 0 {
		return nil
	}
	sortFindings(f)
	return &ValidationError{ID: e.ID, Kind: kindEpic, Errors: f}
}

// Validate checks a ticket against the canonical schema, additionally requiring a
// well-formed parent epic id (R5/CA5). Same single-pass, deterministic contract.
func (t Ticket) Validate() error {
	var f []SchemaError

	if t.ID == "" {
		f = append(f, SchemaError{Field: FieldID, Observed: "empty",
			Expected: "a well-formed ticket id (e.g. ticket-05-03-epics-tickets-management)"})
	} else if !IsTicketID(t.ID) {
		f = append(f, SchemaError{Field: FieldID, Observed: fmtQuote(t.ID),
			Expected: "ticket-NN-MM-<slug>: 'ticket-', epic number, sequence, then a kebab-case slug"})
	}

	// The parent epic link is mandatory for a ticket (R5/CA5).
	if t.EpicID == "" {
		f = append(f, SchemaError{Field: FieldEpic, Observed: "empty",
			Expected: "the parent epic id (every ticket references its epic)"})
	} else if !IsEpicID(t.EpicID) {
		f = append(f, SchemaError{Field: FieldEpic, Observed: fmtQuote(t.EpicID),
			Expected: "a well-formed epic id (e.g. epic-05-sdd-backlog)"})
	}

	f = appendTitleFinding(f, t.Title)
	f = appendStatusFinding(f, t.Status)
	f = appendPriorityFinding(f, t.Priority)
	f = appendDependsOnFindings(f, t.DependsOn)

	if len(f) == 0 {
		return nil
	}
	sortFindings(f)
	return &ValidationError{ID: t.ID, Kind: kindTicket, Errors: f}
}

// appendTitleFinding adds a title finding when title is empty after trimming.
func appendTitleFinding(f []SchemaError, title string) []SchemaError {
	if trimmedEmpty(title) {
		return append(f, SchemaError{Field: FieldTitle, Observed: "empty", Expected: "a non-empty title"})
	}
	return f
}

// appendStatusFinding adds a status finding when status is not one of the closed set.
func appendStatusFinding(f []SchemaError, s Status) []SchemaError {
	if !s.IsValid() {
		observed := "empty"
		if s != "" {
			observed = fmtQuote(string(s))
		}
		return append(f, SchemaError{Field: FieldStatus, Observed: observed,
			Expected: "one of: " + joinStatuses()})
	}
	return f
}

// appendPriorityFinding adds a priority finding when priority is not one of the closed
// set.
func appendPriorityFinding(f []SchemaError, p Priority) []SchemaError {
	if !p.IsValid() {
		observed := "empty"
		if p != "" {
			observed = fmtQuote(string(p))
		}
		return append(f, SchemaError{Field: FieldPriority, Observed: observed,
			Expected: "one of: " + joinPriorities()})
	}
	return f
}

// appendDependsOnFindings adds a finding for any empty dependency id, since an empty id
// in the list is meaningless and would make the dependency set inconsistent (R4/CA4).
// Existence of the referenced ids is NOT checked here (that is 05-04 / a CLI-layer
// concern); this is purely a shape/consistency check.
func appendDependsOnFindings(f []SchemaError, deps []string) []SchemaError {
	for _, d := range deps {
		if trimmedEmpty(d) {
			return append(f, SchemaError{Field: FieldDependsOn, Observed: "an empty id",
				Expected: "every dependency to be a non-empty artifact id"})
		}
	}
	return f
}

// joinStatuses / joinPriorities render the closed sets for error messages and CLI help,
// so the values a user sees match the validator's set exactly.
func joinStatuses() string {
	parts := make([]string, 0, len(Statuses()))
	for _, s := range Statuses() {
		parts = append(parts, string(s))
	}
	return strings.Join(parts, ", ")
}

func joinPriorities() string {
	parts := make([]string, 0, len(Priorities()))
	for _, p := range Priorities() {
		parts = append(parts, string(p))
	}
	return strings.Join(parts, ", ")
}

// sortFindings imposes a fully deterministic order (R6): by a fixed schema-order rank,
// then field name, then observed text.
func sortFindings(f []SchemaError) {
	sort.SliceStable(f, func(i, j int) bool {
		ri, rj := fieldRank(f[i].Field), fieldRank(f[j].Field)
		if ri != rj {
			return ri < rj
		}
		if f[i].Field != f[j].Field {
			return f[i].Field < f[j].Field
		}
		return f[i].Observed < f[j].Observed
	})
}

// fieldRank maps a field to its schema-order rank so findings sort in reader order
// (id, epic, title, status, priority, depends_on).
func fieldRank(field string) int {
	switch field {
	case FieldID:
		return 0
	case FieldEpic:
		return 1
	case FieldTitle:
		return 2
	case FieldStatus:
		return 3
	case FieldPriority:
		return 4
	case FieldDependsOn:
		return 5
	default:
		return 6
	}
}
