// Package traceability consolidates and verifies the SDD backlog's
// spec -> epic -> ticket chain across the workspace (ticket 05-04).
//
// # Role: aggregator/consumer, not a format owner
//
// Tickets 05-01 (specs), 05-02 (architecture) and 05-03 (epics/tickets) each own a
// self-contained format package that deliberately does NOT import its siblings, so no
// package's on-disk format can reshape another's. This package is the exception by
// design: it is the AGGREGATOR. It imports internal/specs, internal/architecture and
// internal/backlog to READ their models (their List/Load functions) and assemble the
// cross-artifact chain. The "don't import siblings" rule applied to the FORMAT packages
// to keep their formats independent; it never applied to a consumer whose whole job is
// to read all of them. Traceability holds NO format of its own and never writes — it is
// pure, read-only analysis (R4/R5/CA5): the links it reasons over are the ones already
// recorded in each artifact's frontmatter, so the source of truth is never duplicated.
//
// # What it produces
//
//   - A navigable Graph (graph.go): from a spec you can reach its epics and their
//     tickets (descending); from a ticket you can climb to its epic and its origin
//     spec/architecture (ascending) — R1/CA1/CA2.
//   - A verification Report (verify.go): an ordered, deterministic list of findings
//     describing every inconsistency in the chain — R2/R3/R6/CA3/CA4/CA6.
//
// # Severity model (the cross-cutting tension with 05-03)
//
// In 05-03 the epic/ticket origin link (spec/architecture) is OPTIONAL. So a
// "traceability gap" (no origin recorded) cannot be conflated with a "broken link" (an
// origin IS recorded but points at something that does not exist) without contradicting
// the already-shipped 05-03. We therefore split severity (see Severity):
//
//   - SeverityError — a HARD inconsistency: a recorded reference that does not resolve
//     (broken link / dangling reference) or a ticket whose parent epic does not exist
//     (orphan ticket). These break the chain and affect the verify exit code.
//   - SeverityWarning — a SOFT gap: an artifact that records NO origin link at all (an
//     epic/ticket without spec/architecture). 05-03 made this legal, so we report it as
//     an informational gap, NOT as an error, and it does NOT affect the exit code.
//
// This satisfies R3/CA4 ("detect and report inconsistencies … orphan epics without an
// origin spec") while respecting 05-03's optional-link decision: the gap is detected and
// reported, just at a severity that does not pretend the workspace is broken.
//
// # Determinism (R6/CA6)
//
// Findings are collected in a fixed pass order and stable-sorted by (severity, kind,
// subject, observed), so the same workspace always yields byte-identical output.
package traceability

import (
	"fmt"
	"sort"
)

// Severity classifies how serious a finding is, and — crucially — whether it should
// affect a verify command's exit code. The set is closed so a caller can branch on it.
type Severity string

const (
	// SeverityError is a hard inconsistency that breaks the traceability chain: a
	// recorded reference that does not resolve, or a ticket with no existing parent
	// epic. Errors make a verification FAIL (non-zero exit) — the chain is genuinely
	// inconsistent (R3/CA4).
	SeverityError Severity = "error"
	// SeverityWarning is a soft traceability gap: an epic/ticket that records no origin
	// link at all. 05-03 made the origin link optional, so this is reported for
	// visibility but does NOT make verification fail (it does not affect the exit code).
	SeverityWarning Severity = "warning"
)

// rank gives a deterministic ordering weight to a severity (errors before warnings) so
// a report reads worst-first.
func (s Severity) rank() int {
	switch s {
	case SeverityError:
		return 0
	case SeverityWarning:
		return 1
	default:
		return 2
	}
}

// FindingKind is the precise class of a traceability problem. The set is closed and
// small so a caller (CLI/TUI) can branch on the kind and render an appropriate message.
type FindingKind string

const (
	// KindBrokenLink marks a reference that IS present in an artifact's frontmatter but
	// points at an artifact that does not exist in the workspace — a dangling reference
	// (R3/CA4). Hard error.
	KindBrokenLink FindingKind = "broken-link"
	// KindOrphanTicket marks a ticket whose parent epic does not exist (R3/CA4). In
	// 05-03 the parent epic is mandatory, so this only arises if the epic was deleted or
	// the ticket's epic reference was corrupted. Hard error.
	KindOrphanTicket FindingKind = "orphan-ticket"
	// KindMissingOrigin marks an epic or ticket that records NO origin link
	// (spec/architecture) at all — a traceability gap, not a broken link (R3/CA4). 05-03
	// made the origin link optional, so this is a warning, never an error.
	KindMissingOrigin FindingKind = "missing-origin"
)

// rank gives a deterministic within-severity ordering weight to a kind, so findings of
// the same severity read in a stable, sensible sequence.
func (k FindingKind) rank() int {
	switch k {
	case KindOrphanTicket:
		return 0
	case KindBrokenLink:
		return 1
	case KindMissingOrigin:
		return 2
	default:
		return 3
	}
}

// Finding is a single actionable traceability problem (R3/CA4). It mirrors the
// field/observed/expected spirit of the workflows graph validator's Finding but is
// scoped to the cross-artifact chain: every finding names the affected artifact
// (Subject), its severity and kind, the concrete value at fault (Observed), and a
// plain-language reason the user can act on (Reason).
type Finding struct {
	// Subject is the id of the affected artifact (a spec slug, an epic id, or a ticket
	// id), so a finding is always anchored to a concrete artifact.
	Subject string
	// Severity is whether this is a hard inconsistency (error) or a soft gap (warning).
	Severity Severity
	// Kind is the precise class of problem.
	Kind FindingKind
	// Observed is the concrete value at fault: the dangling reference, the missing epic
	// id, or — for a missing origin — the empty link.
	Observed string
	// Reason is a clear, self-contained explanation of why this is a problem and what
	// would make it valid.
	Reason string
}

// Error renders one finding as a single actionable line, anchored to its subject,
// severity and kind, so a Report reads top-to-bottom without extra formatting.
func (f Finding) Error() string {
	return fmt.Sprintf("[%s] %s: %s: observed %q; %s", f.Severity, f.Subject, f.Kind, f.Observed, f.Reason)
}

// sortFindings imposes a fully deterministic order on findings (R6/CA6), independent of
// collection order: by severity (errors first), then by kind rank, then by subject, then
// by observed text as a final tie-break. Stable so equal keys keep their collection
// order.
func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if ri, rj := findings[i].Severity.rank(), findings[j].Severity.rank(); ri != rj {
			return ri < rj
		}
		if ri, rj := findings[i].Kind.rank(), findings[j].Kind.rank(); ri != rj {
			return ri < rj
		}
		if findings[i].Subject != findings[j].Subject {
			return findings[i].Subject < findings[j].Subject
		}
		return findings[i].Observed < findings[j].Observed
	})
}
