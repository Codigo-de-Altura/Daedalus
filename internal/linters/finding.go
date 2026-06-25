// Package linters validates the canonical Daedalus definitions in a `.daedalus/`
// workspace — agents, workflows (DAG) and the manifest — and returns a
// deterministic Report of actionable Findings (RF-9.3). It is a read-only
// AGGREGATOR, mirroring internal/conventions: it walks the workspace, loads each
// definition, runs the OWNING package's validator, and re-expresses every result
// as a uniform Finding. It NEVER writes and NEVER auto-fixes — it reports so a
// human (or CI) decides what to change.
//
// # Why an aggregator, not new validation logic
//
// The validation rules already live in the packages that own each definition, and
// this layer reuses them rather than re-deriving them:
//
//   - Agents:    catalog.Agent.Validate (required fields, kebab-case id, param types).
//   - Workflows: workflows.Workflow.Validate (structural: unique kebab phase ids,
//     non-empty agent/gate) AND workflows.Workflow.ValidateGraph (semantic: cycles,
//     missing artifacts, unknown agents, dangling depends_on). The unknown-agent
//     check is fed a knownAgents predicate built from the loaded agent catalog, so
//     a workflow that references a non-existent agent is caught.
//   - Manifest:  workspace.ValidateManifest (name/version/backends/conventions).
//
// The only genuinely new validation is the manifest validator (workspace package);
// everything else is consolidation and a consistent, actionable report shape.
//
// # Backend-agnosticism (R7) and determinism (R8)
//
// The linters operate on the canonical model only; no rule or message names a
// concrete backend. A backend appears solely as a value read from the manifest and
// validated against workspace.SupportedBackends (data). Findings are collected in a
// fixed family order (manifest, agents, workflows) and stable-sorted, so the same
// workspace always yields byte-identical output. Loads are wrapped so a malformed
// definition becomes a controlled finding, never a panic (R6).
package linters

import (
	"fmt"
	"sort"
)

// Severity classifies how serious a finding is. The set is closed so a caller can
// branch on it and key an exit code on the presence of errors. It mirrors the
// conventions package's split so the two reports read consistently.
type Severity string

const (
	// SeverityError is a hard violation that makes a definition invalid: a missing
	// required field, an unknown referenced agent, a DAG cycle, a missing input
	// artifact, an unsupported backend. Errors make linting FAIL (non-zero exit).
	SeverityError Severity = "error"
	// SeverityWarning is a soft, advisory finding that does NOT fail linting — e.g.
	// a manifest convention key that drifted from the canonical set. Reported for
	// visibility, never affecting the exit code.
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

// Family is the definition family a finding belongs to, so a reader can group
// findings by definition kind and a caller can branch on the class. The set is
// closed and mirrors the three families the ticket lints.
type Family string

const (
	// FamilyManifest covers the workspace manifest (`daedalus.yaml`).
	FamilyManifest Family = "manifest"
	// FamilyAgent covers canonical agent definitions.
	FamilyAgent Family = "agent"
	// FamilyWorkflow covers canonical workflow (DAG) definitions.
	FamilyWorkflow Family = "workflow"
)

// rank gives a deterministic ordering weight to a family so findings read in a
// stable, sensible sequence (manifest first as the workspace-wide config, then
// agents, then the workflows that reference them).
func (f Family) rank() int {
	switch f {
	case FamilyManifest:
		return 0
	case FamilyAgent:
		return 1
	case FamilyWorkflow:
		return 2
	default:
		return 3
	}
}

// Finding is a single actionable linter finding (R5). Every finding is anchored to
// a concrete Location (a workspace-relative file path) and names the Definition it
// concerns, the precise Spot within it (a field, phase, or key), the Rule that was
// broken (a short, referenceable id) and a plain-language Reason describing what was
// expected vs. found, so the user can fix it without guessing.
type Finding struct {
	// Family is the definition family this finding belongs to.
	Family Family
	// Severity is whether this is a hard violation (error) or a soft advisory.
	Severity Severity
	// Location is the affected file, as a workspace-relative path with forward
	// slashes (e.g. ".daedalus/agents/analyst" or ".daedalus/workflows/sdd.yaml"),
	// so a finding always points at something concrete and reads identically across
	// operating systems.
	Location string
	// Definition is the human identity of the definition (the agent id, the workflow
	// name, or "daedalus.yaml"), echoed so a batch report names which source failed.
	Definition string
	// Spot is the precise location within the definition: a field, a phase id, or a
	// convention key (e.g. "prompt", "phase write-spec", "backends[0]"). It may be
	// empty for a whole-definition finding.
	Spot string
	// Rule is the short id of the rule that was broken (e.g. "schema", "cycle",
	// "missing-artifact", "unknown-agent", "unsupported-backend").
	Rule string
	// Reason is a self-contained explanation of why this is a violation and what
	// would make it valid.
	Reason string
}

// Error renders one finding as a single actionable line, anchored to its location,
// severity, definition spot and rule, so a Report reads top-to-bottom without extra
// formatting.
func (f Finding) Error() string {
	spot := f.Spot
	if spot == "" {
		spot = "(definition)"
	}
	return fmt.Sprintf("[%s] %s: %s: %s: %s", f.Severity, f.Location, spot, f.Rule, f.Reason)
}

// sortFindings imposes a fully deterministic order on findings (R8), independent of
// collection order: by severity (errors first), then family, then location, then
// spot, then rule as a final tie-break. Stable so equal keys keep collection order
// (which is itself deterministic: the owning validators already sort their output).
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
		if findings[i].Spot != findings[j].Spot {
			return findings[i].Spot < findings[j].Spot
		}
		return findings[i].Rule < findings[j].Rule
	})
}
