// Package workflows owns the domain model and on-disk persistence of a project's
// DAG workflows in its `.daedalus/workflows/` workspace directory.
//
// A *workflow* in Daedalus is a declarative DAG (PRD decision D9) that describes
// the project's SDD pipeline: an ordered list of *phases* where each phase
// references an *agent*, the artifacts it consumes (`inputs`), the artifacts it
// produces (`outputs`), a validation *gate* the artifact must cross to advance,
// and the predecessor phases it `depends_on`. Those `depends_on` references are
// the *edges* of the DAG; the ordered phase list plus its edges describes the
// whole graph (R1/R5).
//
// This package is intentionally self-contained and mirrors the prompts/catalog/
// workspace packages rather than coupling to them: it owns its own kebab-case
// rule, its own deterministic hand-rolled YAML renderer/parser and its own
// non-destructive write helpers. The project duplicates these small helpers on
// purpose so each package can evolve its own canonical format without a
// build-time dependency reshaping a sibling's output. In particular this package
// imports neither internal/prompts nor internal/workspace.
//
// Scope (ticket 04-01): this package models a workflow and (de)serializes/edits
// its *definition* only. It is deliberately backend-agnostic (R8) — there is
// nothing here specific to any concrete agent tool or runtime — and it performs
// no *semantic* graph validation: cycles, missing artifacts and unknown agents are
// ticket 04-03's concern, not this one. Execution of a workflow is out of scope
// for phase 1 entirely (PRD §4.2, D5).
//
// Determinism and non-destruction are first-class (R3/R4): the same Workflow
// always renders byte-identical bytes (fixed key order, ordered phases, trailing
// newline), creating a workflow never overwrites an existing one, and editing a
// workflow touches only its own file.
package workflows

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// WorkflowsDir is the workspace subdirectory that holds persisted workflows. It
// mirrors prompts.PromptsDir / catalog.AgentsDir and matches the canonical
// layout (workspace.Subdirs "workflows"); kept as a constant here so this package
// does not import the workspace package just for a directory name, and the two
// stay in sync by convention rather than a build-time coupling.
const WorkflowsDir = "workflows"

// FileExt is the on-disk extension for a persisted workflow (R2): workflows are
// stored as YAML files so they are legible and git-friendly.
const FileExt = ".yaml"

// Workflow is the in-memory canonical model of a DAG workflow (R1). A workflow's
// identity is its name — the base name of its `<name>.yaml` file — which is NOT
// stored inside the document: the on-disk format carries only `phases:` (see
// render.go). This mirrors how prompts/agents take their identity from the file
// (or directory) name rather than a redundant `name:` key inside, so a file
// renamed on disk can never disagree with its contents and a load→render
// round-trip stays byte-stable (R3). Name is populated by Load from the file name
// and is informational for callers; it is not serialized.
type Workflow struct {
	// Name is the workflow's stable identifier in kebab-case. It is both the unique
	// key within the workspace and the on-disk file base name, so it must be
	// filesystem-safe — kebab-case guarantees that. It is not part of the YAML
	// document; Load derives it from the file name.
	Name string
	// Phases is the ordered list of phases that make up the DAG (R1). The order is
	// significant and preserved verbatim on render — phases are never reordered —
	// because it is the authored reading order of the pipeline.
	Phases []Phase
}

// Phase is a single node of the workflow DAG (R1). Its shape is exactly the
// schema the epic mandates: `{ id, agent, inputs[], outputs[], gate, depends_on[] }`.
type Phase struct {
	// ID is the phase's identifier, unique within the workflow, in kebab-case. It
	// is the handle other phases reference in their DependsOn, so it is also the
	// node key of the DAG.
	ID string
	// Agent is the id of the agent that produces this phase's outputs (e.g.
	// analyst, architect, planner, validator, documenter). It is an opaque
	// reference here: this package does not resolve or validate it against the
	// agent catalog (that is a semantic concern, ticket 04-03) and stores no
	// tool-specific knowledge about it (R8).
	Agent string
	// Inputs are the artifacts this phase consumes. The slice order is preserved
	// verbatim so the serialized form is stable and the diff is clean.
	Inputs []string
	// Outputs are the artifacts this phase produces. Order is preserved verbatim.
	Outputs []string
	// Gate is the validation criterion an artifact must satisfy to advance past
	// this phase. It is an opaque reference; this package does not interpret it.
	Gate string
	// DependsOn lists the predecessor phases/artifacts this phase depends on. These
	// references are the incoming edges of this DAG node (R5); see Workflow.Edges.
	// Order is preserved verbatim.
	DependsOn []string
}

// kebabCase matches a non-empty kebab-case identifier: lowercase ASCII letters
// and digits in dash-separated segments, no leading/trailing/double dashes. This
// is the same convention the prompts and catalog packages use for their ids
// (init.md §7); it is duplicated here (not imported) so workflows owns its own id
// rule. It is the single source of truth for "is this name/phase id well-formed".
var kebabCase = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IsKebabCase reports whether id is a well-formed kebab-case identifier. Exported
// because the rule is domain knowledge (a workflow name and a phase id are both a
// path/reference segment and a unique key), not something each caller re-encodes.
func IsKebabCase(id string) bool {
	return kebabCase.MatchString(id)
}

// Edge is a single directed dependency edge of the DAG: From depends on nothing
// here — rather, the phase To depends on From, so the edge points From -> To in
// execution order (the dependency must be satisfied before the dependent runs).
// Concretely, for a phase P with `depends_on: [A]`, Edges yields {From: A, To: P.ID}.
type Edge struct {
	// From is the predecessor reference (an entry of some phase's DependsOn).
	From string
	// To is the phase id that declared the dependency.
	To string
}

// Edges derives the directed edges of the DAG from the phases' DependsOn lists
// (R5/CA4). Every dependency of every phase becomes one From->To edge, where To
// is the dependent phase and From is the referenced predecessor. The result is
// deterministic: edges are emitted in phase order, then in each phase's DependsOn
// order, so the same Workflow always yields the same edge sequence. This is a
// pure projection of the model — it resolves nothing and validates nothing (a
// From that names no existing phase is still returned; checking that is ticket
// 04-03's semantic validation), so callers get a faithful, recoverable view of
// the graph's edge set.
func (w Workflow) Edges() []Edge {
	var edges []Edge
	for _, p := range w.Phases {
		for _, dep := range p.DependsOn {
			edges = append(edges, Edge{From: dep, To: p.ID})
		}
	}
	return edges
}

// PhaseIndex returns the index of the phase with the given id, or -1 if no phase
// has it. It is the single lookup the edit operations share so "find a phase by
// id" is defined in exactly one place.
func (w Workflow) PhaseIndex(id string) int {
	for i, p := range w.Phases {
		if p.ID == id {
			return i
		}
	}
	return -1
}

// AddPhase appends a phase to the workflow (R6). It rejects a duplicate phase id
// with ErrPhaseExists so the in-memory model never holds two phases under the
// same id (the id is the DAG node key and must be unique). It does NOT validate
// the phase's schema or the resulting graph — callers validate the whole workflow
// before persisting (see store.go) — keeping this a focused structural mutation.
func (w *Workflow) AddPhase(p Phase) error {
	if w.PhaseIndex(p.ID) >= 0 {
		return fmt.Errorf("%w: %q", ErrPhaseExists, p.ID)
	}
	w.Phases = append(w.Phases, p)
	return nil
}

// EditPhase replaces the phase identified by id with the provided phase (R6),
// preserving its position in the ordered list. The replacement may carry a
// different id (a rename); when it does, the new id must not collide with another
// existing phase. An absent id is ErrPhaseNotFound. Like AddPhase this performs no
// schema/graph validation — that is the persisting caller's gate.
func (w *Workflow) EditPhase(id string, p Phase) error {
	idx := w.PhaseIndex(id)
	if idx < 0 {
		return fmt.Errorf("%w: %q", ErrPhaseNotFound, id)
	}
	// Guard a rename against colliding with a different existing phase. Renaming to
	// the same id (idx unchanged) is allowed; only a *different* phase owning the
	// target id is a conflict.
	if p.ID != id {
		if other := w.PhaseIndex(p.ID); other >= 0 && other != idx {
			return fmt.Errorf("%w: %q", ErrPhaseExists, p.ID)
		}
	}
	w.Phases[idx] = p
	return nil
}

// RemovePhase deletes the phase identified by id from the ordered list (R6),
// shifting the remaining phases down so their relative order is preserved. An
// absent id is ErrPhaseNotFound. It does not touch other phases' DependsOn lists:
// scrubbing now-dangling references is a semantic concern (ticket 04-03), and
// silently rewriting other phases here would be a surprising, lossy edit.
func (w *Workflow) RemovePhase(id string) error {
	idx := w.PhaseIndex(id)
	if idx < 0 {
		return fmt.Errorf("%w: %q", ErrPhaseNotFound, id)
	}
	w.Phases = append(w.Phases[:idx], w.Phases[idx+1:]...)
	return nil
}

// Entry is a single listing row: the minimum a caller needs to present the
// persisted workflows for selection without loading their full phase lists. It is
// a projection of a Workflow — its name and phase count — so listing stays cheap.
type Entry struct {
	// Name is the workflow's identifier (its file base name).
	Name string
	// Phases is the number of phases in the workflow, a cheap shape summary.
	Phases int
}

// sortEntries orders entries by name so a listing is deterministic regardless of
// the order the filesystem returned the files in (R4). Centralized so every
// listing path shares one ordering rule.
func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name < entries[j].Name })
}

// fmtQuote is a tiny helper so findings render an observed value with quotes
// consistently. Mirrors the prompts helper of the same name.
func fmtQuote(s string) string {
	return fmt.Sprintf("%q", s)
}

// trimmedEmpty reports whether s is empty after trimming surrounding whitespace.
// Used by the validator so "empty" means the same thing everywhere.
func trimmedEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}
