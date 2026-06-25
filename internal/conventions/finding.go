// Package conventions validates a Daedalus `.daedalus/` workspace against the
// team conventions stated in init.md §7 and CLAUDE.md §6 (RF-8.3). It is the
// single, referenceable, machine-checkable expression of those conventions: the
// constants and checks below ARE the canonical convention catalog, and the user
// manual (docs/) points here. The package is a read-only AGGREGATOR — like
// internal/traceability — that reuses each domain package's identity rules
// (backlog.IsKebabCase / IsEpicID / IsTicketID), layout constants
// (catalog.AgentsDir, prompts.PromptsDir, …) and the cross-artifact chain
// (traceability.Build/Verify) instead of re-deriving them. It NEVER writes and
// NEVER auto-fixes: it reports, so a human (or CI) decides what to change.
//
// # The four convention families (RF-8.3)
//
//   - Naming: kebab-case for files/ids; `epic-NN-<slug>` and `ticket-NN-MM-<slug>`
//     id patterns for epics, tickets, agents and workflows.
//   - Structure: the canonical `.daedalus/` layout (the workspace.Subdirs tree)
//     and the nested epics/<epic>/tickets/<ticket> backlog shape, with required
//     directories and documents present and nothing out of place.
//   - Format: YAML with ordered keys and structured Markdown — verified by
//     re-rendering each artifact through its own deterministic renderer and
//     comparing (the same guarantee ticket 08-02 pins with golden files).
//   - Traceability: every ticket references its epic and every epic its origin,
//     delegated to internal/traceability.
//
// # Determinism (R7/CA7)
//
// Findings are collected in a fixed pass order (naming, structure, format,
// traceability) and stable-sorted by (severity, family, location, convention),
// so the same workspace always yields byte-identical output.
package conventions

import (
	"fmt"
	"sort"
)

// Severity classifies how serious a finding is and whether it should fail a
// `validate` command's exit code. The set is closed so a caller can branch on it.
type Severity string

const (
	// SeverityError is a hard convention violation: a malformed id, a broken
	// kebab-case name, a missing required directory/document, a format that does
	// not round-trip, or a hard traceability break. Errors make validation FAIL
	// (non-zero exit).
	SeverityError Severity = "error"
	// SeverityWarning is a soft, advisory finding that does NOT fail validation —
	// e.g. a missing OPTIONAL origin link, which the backlog model (ticket 05-03)
	// deliberately allows. Reported for visibility, never affecting the exit code.
	SeverityWarning Severity = "warning"
)

// rank orders severities so a report reads worst-first (errors before warnings).
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

// Family is the convention family a finding belongs to, so a reader can group
// violations and a caller can branch on the class. The set mirrors the four
// families in the package doc and is closed.
type Family string

const (
	// FamilyNaming covers kebab-case and id-pattern violations.
	FamilyNaming Family = "naming"
	// FamilyStructure covers missing/misplaced directories and documents.
	FamilyStructure Family = "structure"
	// FamilyFormat covers YAML key-order / Markdown-structure deviations.
	FamilyFormat Family = "format"
	// FamilyTraceability covers ticket->epic and epic->origin link problems.
	FamilyTraceability Family = "traceability"
)

// rank gives a deterministic ordering weight to a family so findings read in a
// stable, sensible sequence (the order they appear in the package doc).
func (f Family) rank() int {
	switch f {
	case FamilyNaming:
		return 0
	case FamilyStructure:
		return 1
	case FamilyFormat:
		return 2
	case FamilyTraceability:
		return 3
	default:
		return 4
	}
}

// Finding is a single actionable convention violation (R6/CA6). Every finding is
// anchored to a concrete Location (a workspace-relative path or artifact id),
// names the Convention it breaks (a short, referenceable rule id) and carries a
// plain-language Reason the user can act on.
type Finding struct {
	// Family is the convention family this violation belongs to.
	Family Family
	// Severity is whether this is a hard violation (error) or a soft advisory
	// (warning).
	Severity Severity
	// Location is the affected element: a workspace-relative path (forward slashes)
	// or an artifact id, so a finding always points at something concrete.
	Location string
	// Convention is the short id of the rule that was broken (e.g. "kebab-case",
	// "epic-id-pattern", "required-directory"), referenceable from the manual.
	Convention string
	// Reason is a self-contained explanation of why this is a violation and what
	// would make it valid.
	Reason string
}

// Error renders one finding as a single actionable line, anchored to its
// location, severity and convention, so a Report reads top-to-bottom without
// extra formatting.
func (f Finding) Error() string {
	return fmt.Sprintf("[%s] %s: %s: %s", f.Severity, f.Location, f.Convention, f.Reason)
}

// sortFindings imposes a fully deterministic order on findings (R7/CA7),
// independent of collection order: by severity (errors first), then by family,
// then by location, then by convention as a final tie-break. Stable so equal
// keys keep their collection order.
func sortFindings(findings []Finding) {
	sort.SliceStable(findings, func(i, j int) bool {
		if ri, rj := findings[i].Severity.rank(), findings[j].Severity.rank(); ri != rj {
			return ri < rj
		}
		if ri, rj := findings[i].Family.rank(), findings[j].Family.rank(); ri != rj {
			return ri < rj
		}
		if findings[i].Location != findings[j].Location {
			return findings[i].Location < findings[j].Location
		}
		return findings[i].Convention < findings[j].Convention
	})
}
