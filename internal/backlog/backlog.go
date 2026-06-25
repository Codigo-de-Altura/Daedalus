// Package backlog owns the domain model and on-disk persistence of the SDD
// backlog's epics and tickets, living under a project's `.daedalus/epics/`
// workspace directory (ticket 05-03).
//
// # Why epics and tickets share ONE package (deviation from the per-domain molde)
//
// Tickets 05-01 (specs) and 05-02 (architecture) each got their own package, one per
// artifact. Epics and tickets are deliberately handled together here because they are
// a COUPLED domain, not two independent ones:
//
//   - A ticket physically lives UNDER its epic on disk (see layout below): a ticket
//     cannot exist without an epic, so the two share location logic and identity rules.
//   - Their ids are interlocking: a ticket id `ticket-NN-MM-<slug>` embeds its epic's
//     number NN (R2/CA2), so parsing/validating one needs the other's convention.
//   - Dependencies cross both (R4/CA4): a ticket may depend on another ticket or on an
//     epic, and an epic on another epic; the dependency vocabulary is shared.
//
// Splitting them would force an epics package and a tickets package to import each
// other (or a third shared one) just to agree on ids and paths — exactly the
// build-time coupling the prompts package documents avoiding. So backlog keeps both
// in one self-contained package. It still does NOT import other internal packages: the
// optional existence check of referenced specs/architecture lives in the CLI layer (as
// in 05-02), and the provenance anchor (planner / sdd-default / epics|tickets) is
// duplicated here as constants and pinned to the real workflow phases by a test
// (provenance_link_test.go).
//
// # Canonical on-disk layout (nested; orchestrator adjudication)
//
//	.daedalus/epics/
//	  epic-NN-<slug>/
//	    epic.md
//	    tickets/
//	      ticket-NN-MM-<slug>/
//	        ticket.md
//
// Tickets are nested UNDER their epic, matching CLAUDE.md §6 and the validation
// (check 2: a ticket "under the epic"). The flat `.daedalus/tickets/` subdir that
// exists in workspace.Subdirs is intentionally NOT used by this ticket and is left
// reserved/untouched (changing workspace.Subdirs would be a cross-package change out
// of scope here). The canonical home of a ticket is its nested folder; the folder name
// IS the id (CLAUDE.md §6), so the markdown file inside is named by kind (`epic.md` /
// `ticket.md`), stable and independent of the slug — a rename of the slug renames only
// the folder, keeping the artifact file name constant and the diff clean.
//
// # Phase 1: Daedalus manages the definition, it does NOT run the agent (R7)
//
// As in 05-01/05-02, the planner's execution (and the implementation it would drive)
// lives OUTSIDE Daedalus, on the user's backend (PRD decision D5). This package never
// launches a process. Links to a spec/architecture and to the parent epic are pure
// metadata recorded in stable frontmatter (R5/CA5); the body is a placeholder the user
// replaces. `generated: false` is written explicitly to make R7/CA7 self-evident.
//
// # Determinism and non-destruction are first-class (R6/R8)
//
// The same epic/ticket always renders byte-identical content (fixed key order, sorted
// nothing the user authored, trailing newline). Creating never overwrites an existing
// artifact, and editing touches only its own file. The body is persisted verbatim as
// arbitrary Markdown; this package never reinterprets it.
package backlog

import (
	"fmt"
	"regexp"
	"strings"
)

// EpicsDir is the workspace subdirectory that roots the nested epic/ticket tree. It
// mirrors the canonical layout (workspace.Subdirs "epics"); kept as a constant here so
// this package does not import the workspace package just for a directory name.
const EpicsDir = "epics"

// TicketsSubdir is the per-epic subdirectory that holds the epic's tickets, matching
// CLAUDE.md §6 (`epic-NN-<slug>/tickets/ticket-NN-MM-<slug>/`).
const TicketsSubdir = "tickets"

// EpicFile and TicketFile are the canonical markdown file names inside an epic's and a
// ticket's folder. The folder name is the id (CLAUDE.md §6), so the file is named by
// kind rather than by slug, keeping it stable across a slug rename (R6 diff-friendly).
const (
	EpicFile   = "epic.md"
	TicketFile = "ticket.md"
)

// Provenance anchor: the default SDD pipeline's `epics` and `tickets` steps, both run
// by the planner (init.md §6). Duplicated here — not imported from internal/workflows —
// so this package stays self-contained; provenance_link_test.go pins them to
// internal/workflows.DefaultWorkflow so they can never drift from the real phases.
const (
	// PlannerAgent is the agent that derives epics and tickets from spec+architecture
	// (init.md §6). It matches both the epics and tickets phases' Agent in the default
	// workflow.
	PlannerAgent = "planner"
	// DefaultWorkflowName is the factory SDD workflow these artifacts belong to. It
	// matches workflows.DefaultWorkflowName.
	DefaultWorkflowName = "sdd-default"
	// PhaseEpics and PhaseTickets are the phase ids of the planner's two steps. They
	// match workflows.DefaultPhaseEpics and workflows.DefaultPhaseTickets.
	PhaseEpics   = "epics"
	PhaseTickets = "tickets"
)

// Status is the lifecycle state of an epic or ticket. The set is closed and small
// (R6/CA6): a value outside it is a validation error, so a backlog never drifts into
// ad-hoc states that tooling cannot reason about. The vocabulary is shared by epics
// and tickets because both move through the same lifecycle.
type Status string

const (
	// StatusTodo is the default state of a freshly created artifact: defined, not yet
	// started.
	StatusTodo Status = "todo"
	// StatusInProgress marks work underway.
	StatusInProgress Status = "in-progress"
	// StatusBlocked marks work that cannot proceed (e.g. an unmet dependency).
	StatusBlocked Status = "blocked"
	// StatusDone marks completed work.
	StatusDone Status = "done"
)

// DefaultStatus is the status assigned when none is specified: a new artifact is
// defined but not started.
const DefaultStatus = StatusTodo

// IsValid reports whether s is one of the closed set of statuses (R6/CA6). Centralized
// so the renderer, parser and validator share one definition of "known status".
func (s Status) IsValid() bool {
	switch s {
	case StatusTodo, StatusInProgress, StatusBlocked, StatusDone:
		return true
	default:
		return false
	}
}

// Statuses lists the valid statuses in canonical order, for actionable error messages
// and CLI help (so the set the user sees matches the validator's set exactly).
func Statuses() []Status { return []Status{StatusTodo, StatusInProgress, StatusBlocked, StatusDone} }

// Priority is the scheduling weight of an epic or ticket. Like Status it is a closed,
// validated set (R6/CA6).
type Priority string

const (
	PriorityLow      Priority = "low"
	PriorityMedium   Priority = "medium"
	PriorityHigh     Priority = "high"
	PriorityCritical Priority = "critical"
)

// DefaultPriority is the priority assigned when none is specified: medium, the neutral
// middle of the scale so an unset priority is never accidentally the most or least
// urgent.
const DefaultPriority = PriorityMedium

// IsValid reports whether p is one of the closed set of priorities (R6/CA6).
func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return true
	default:
		return false
	}
}

// Priorities lists the valid priorities in canonical (ascending urgency) order, for
// actionable error messages and CLI help.
func Priorities() []Priority {
	return []Priority{PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical}
}

// epicIDPattern matches a well-formed epic id `epic-NN-<slug>`: the literal `epic-`,
// a numeric NN (one or more digits, the epic number), a dash, then a kebab-case slug.
// The slug segment is itself kebab-case (lowercase letters/digits in dash-separated
// segments). This is the single source of truth for "is this epic id well-formed"
// (R1/CA1), per CLAUDE.md §6.
var epicIDPattern = regexp.MustCompile(`^epic-([0-9]+)-([a-z0-9]+(?:-[a-z0-9]+)*)$`)

// ticketIDPattern matches a well-formed ticket id `ticket-NN-MM-<slug>`: the literal
// `ticket-`, NN (epic number), MM (sequence within the epic), then a kebab-case slug
// (R2/CA2), per CLAUDE.md §6.
var ticketIDPattern = regexp.MustCompile(`^ticket-([0-9]+)-([0-9]+)-([a-z0-9]+(?:-[a-z0-9]+)*)$`)

// slugPattern matches a bare kebab-case slug, used to validate the user-supplied slug
// before it is composed into an id. Duplicated from specs/architecture (not imported)
// so backlog owns its own slug rule (init.md §7).
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// IsKebabCase reports whether s is a well-formed bare kebab-case slug. Exported so the
// CLI can validate a slug argument before composing an id.
func IsKebabCase(s string) bool {
	return slugPattern.MatchString(s)
}

// IsEpicID reports whether id is a well-formed `epic-NN-<slug>` id (R1/CA1).
func IsEpicID(id string) bool {
	return epicIDPattern.MatchString(id)
}

// IsTicketID reports whether id is a well-formed `ticket-NN-MM-<slug>` id (R2/CA2).
func IsTicketID(id string) bool {
	return ticketIDPattern.MatchString(id)
}

// EpicID composes the canonical epic id from a number and a slug: `epic-<number>-<slug>`.
// The number is formatted verbatim (the caller supplies the exact NN, e.g. "05"); the
// composed id is validated by the caller via IsEpicID. Numbering is explicit and
// deterministic (the user provides NN), never auto-incremented, so the same inputs
// always yield the same id with no hidden state.
func EpicID(number, slug string) string {
	return fmt.Sprintf("epic-%s-%s", number, slug)
}

// TicketID composes the canonical ticket id: `ticket-<epicNumber>-<sequence>-<slug>`.
// The epic number ties the ticket to its epic (NN); the sequence is the MM within the
// epic. Both are supplied explicitly by the caller (no auto-increment).
func TicketID(epicNumber, sequence, slug string) string {
	return fmt.Sprintf("ticket-%s-%s-%s", epicNumber, sequence, slug)
}

// EpicNumberOf extracts the NN number from an epic id, or "" if the id is malformed.
// Used to derive a ticket's epic number from its parent epic id so the two stay
// consistent (a ticket under epic-05-x must be ticket-05-MM-y).
func EpicNumberOf(epicID string) string {
	m := epicIDPattern.FindStringSubmatch(epicID)
	if m == nil {
		return ""
	}
	return m[1]
}

// trimmedEmpty reports whether s is empty after trimming surrounding whitespace.
// Shared by the validator and the renderer so they agree on what "empty" means.
func trimmedEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}

// fmtQuote renders an observed value with quotes consistently for findings.
func fmtQuote(s string) string {
	return fmt.Sprintf("%q", s)
}
