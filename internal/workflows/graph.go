package workflows

import (
	"fmt"
	"sort"
	"strings"
)

// Semantic validation of a workflow DAG (ticket 04-03).
//
// This is the *semantic* validator: distinct from validate.go, which is the
// *structural* schema validator (each phase has a kebab-case unique id, a
// non-empty agent and gate, etc.). Schema validation answers "is each phase
// well-formed in isolation?"; semantic validation answers "does the set of
// phases describe a coherent, runnable DAG?". The two are deliberately separate
// types and separate passes so a caller can run them independently and so the
// conceptual difference stays legible:
//
//   - Structural (Workflow.Validate, *ValidationError): per-phase field rules.
//   - Semantic   (Workflow.ValidateGraph, *GraphReport): whole-graph rules —
//     cycles, artifact availability, agent existence.
//
// ValidateGraph assumes the workflow already passed schema validation in spirit
// (it does not re-check field rules) but never panics regardless of input: a
// degenerate workflow (empty, phases with no deps, empty lists, a depends_on that
// names a non-existent phase) is handled as a defined case (R8).
//
// # Backend-agnosticism: why knownAgents is injected (R9)
//
// "Agent exists" can only be answered against the workspace's agent catalog — but
// this package must NOT import internal/catalog (it owns no backend/tool
// knowledge, the isolation documented in workflow.go). So the caller injects the
// set of known agent ids as a plain predicate over strings: the core validates
// the canonical model and asks "is this agent known?" without ever learning what
// a catalog is or how agents are resolved. The CLI, which already imports the
// catalog, is the layer that builds the set and passes it in. A nil predicate is
// treated as "agent existence is not checked" so callers that only care about the
// graph shape can opt out cleanly.
//
// # Predecessor / artifact-availability semantics (R3)
//
// An artifact a phase lists in `inputs` is *available* to that phase when it is
// either:
//
//   - the initial artifact "brief" (the literal pipeline entry artifact, init.md
//     §6), or
//   - an `output` of one of the phase's *predecessors*, where a predecessor is a
//     transitive ancestor reachable by following `depends_on` edges from this
//     phase back toward the roots.
//
// Transitive (not just direct) ancestry is the right rule: if phase C depends on
// B and B depends on A, then A's outputs have been produced by the time C runs,
// so C may legitimately consume them even though C does not list A directly. We
// compute each phase's ancestor set by walking depends_on edges among the phases
// that actually exist. A depends_on entry that does not name an existing phase
// contributes no ancestor (it is reported separately, see below) and is otherwise
// ignored for availability so it cannot crash or distort the walk.
//
// # Findings beyond the three mandatory classes
//
// The three required classes are cycle, missing-artifact and unknown-agent. We
// additionally report an `unknown-dependency` finding when a depends_on entry
// names neither an existing phase nor the initial "brief": a dangling edge is
// almost always a typo and surfacing it is more helpful than silently dropping
// it. It is reported deterministically alongside the others and never panics; it
// is an additive diagnostic, not a substitute for the mandatory three.

// initialArtifact is the literal name of the pipeline's entry artifact (init.md
// §6): the one input that is available to a phase without any predecessor having
// produced it. It is also accepted as a valid depends_on target so a first phase
// can declare `depends_on: [brief]` without that edge being flagged as dangling.
const initialArtifact = "brief"

// FindingKind classifies a semantic finding. The set is closed and small so a
// caller (CLI/TUI) can branch on the kind and render an appropriate message.
type FindingKind string

const (
	// KindCycle marks a dependency cycle: the phases in Observed form a loop, so
	// the graph is not a DAG (R2).
	KindCycle FindingKind = "cycle"
	// KindMissingArtifact marks a phase input that no predecessor produces and that
	// is not the initial artifact (R3).
	KindMissingArtifact FindingKind = "missing-artifact"
	// KindUnknownAgent marks a phase whose agent is not in the injected known set
	// (R4).
	KindUnknownAgent FindingKind = "unknown-agent"
	// KindUnknownDependency marks a depends_on entry that names neither an existing
	// phase nor the initial artifact — an additive diagnostic, not one of the three
	// mandatory classes.
	KindUnknownDependency FindingKind = "unknown-dependency"
)

// Finding is a single actionable semantic problem (R5). It mirrors the
// field/observed/expected spirit of the schema validator's SchemaError but is
// scoped to whole-graph concerns: every finding names the affected phase, the
// kind of problem, the observed value (the artifact, agent, dependency, or cycle
// chain), and a plain-language reason the user can act on.
type Finding struct {
	// Phase is the id of the affected phase. For a cycle it is the entry phase the
	// cycle was detected from (the first member of Observed), so a finding is always
	// anchored to a concrete phase.
	Phase string
	// Kind is the class of problem (cycle / missing-artifact / unknown-agent /
	// unknown-dependency).
	Kind FindingKind
	// Observed is the concrete value at fault: the missing artifact name, the
	// unknown agent id, the dangling dependency reference, or — for a cycle — the
	// loop rendered as "a -> b -> a".
	Observed string
	// Reason is a clear, self-contained explanation of why this is a problem and
	// what would make it valid.
	Reason string
}

// Error renders one finding as a single actionable line, anchored to its phase
// and kind, so a *GraphReport reads top-to-bottom without extra formatting.
func (f Finding) Error() string {
	return fmt.Sprintf("phase %q: %s: observed %s; %s", f.Phase, f.Kind, fmtQuote(f.Observed), f.Reason)
}

// GraphReport is the result of semantic validation (R1): valid/invalid plus the
// ordered list of findings. A valid workflow yields Valid==true and no findings
// (R6). It implements error so an invalid report can flow through error-returning
// gates, but callers typically inspect Valid and Findings directly.
type GraphReport struct {
	// WorkflowName echoes the validated workflow's name for context in messages.
	WorkflowName string
	// Findings are every semantic problem detected, in deterministic order (R7).
	// Empty when the workflow is valid.
	Findings []Finding
}

// Valid reports whether the workflow is semantically valid (no findings). It is a
// method rather than a field so it cannot fall out of sync with Findings.
func (r *GraphReport) Valid() bool {
	return len(r.Findings) == 0
}

// Error renders all findings, one per line, prefixed with the workflow name, in
// the deterministic order ValidateGraph produced, so the message is byte-stable
// for a given workflow (R7). It is safe to call on a valid report (it says so).
func (r *GraphReport) Error() string {
	var b strings.Builder
	name := r.WorkflowName
	if name == "" {
		name = "(unnamed)"
	}
	if r.Valid() {
		fmt.Fprintf(&b, "workflow %q is semantically valid", name)
		return b.String()
	}
	fmt.Fprintf(&b, "workflow %q is semantically invalid (%d finding%s):", name, len(r.Findings), pluralS(len(r.Findings)))
	for _, f := range r.Findings {
		b.WriteString("\n  - ")
		b.WriteString(f.Error())
	}
	return b.String()
}

// ValidateGraph runs the semantic validation of the workflow's DAG (R1) and
// returns a *GraphReport. knownAgents is the injected agent-existence predicate
// (R4/R9): it reports whether an agent id is known to the workspace. A nil
// predicate disables the unknown-agent check (the caller does not care about agent
// existence), so graph-shape-only validation is a clean opt-out.
//
// The pass is pure (no I/O, no backend calls, R9) and deterministic (R7): findings
// are collected per phase in document order, each phase's checks in a fixed order
// (cycle, then unknown-dependency, then missing-artifact, then unknown-agent), and
// then stable-sorted by (phase position, kind rank) so the same workflow always
// yields the same report. It never panics on degenerate input (R8).
func (w Workflow) ValidateGraph(knownAgents func(string) bool) *GraphReport {
	report := &GraphReport{WorkflowName: w.Name}

	// Index phases by id once. When two phases share an id (a schema violation that
	// semantic validation does not re-check), the first wins for lookups; this keeps
	// the graph walk well-defined rather than panicking on the malformed input.
	byID := make(map[string]Phase, len(w.Phases))
	pos := make(map[string]int, len(w.Phases))
	for i, p := range w.Phases {
		if _, ok := byID[p.ID]; !ok {
			byID[p.ID] = p
			pos[p.ID] = i
		}
	}

	// 1) Cycle detection over the phase dependency graph. A cycle makes the rest of
	// the analysis (ancestor walks) ill-defined, so we run it first and, when the
	// graph has cycles, we still run the other checks but guard the ancestor walk
	// against infinite loops (the walk uses a visited set, so it terminates).
	cycleFindings := w.detectCycles(byID)
	report.Findings = append(report.Findings, cycleFindings...)

	// 2) Per-phase checks: dangling dependencies, missing artifacts, unknown agents.
	for _, p := range w.Phases {
		report.Findings = append(report.Findings, w.checkDependencies(p, byID)...)
		report.Findings = append(report.Findings, w.checkArtifacts(p, byID)...)
		if f, ok := checkAgent(p, knownAgents); ok {
			report.Findings = append(report.Findings, f)
		}
	}

	sortFindingsGraph(report.Findings, pos)
	return report
}

// detectCycles finds dependency cycles using a depth-first search with an active
// recursion stack, the same technique as prompts.Resolve's cycle detection
// adapted to the phase graph: a node re-entered while still on the active path
// closes a cycle, and the active stack yields the exact loop chain. Edges are
// followed in each phase's declared depends_on order, and phases are visited in
// document order, so the set of reported cycles and their chains are deterministic
// (R7). Only edges to existing phases are followed (a dangling edge is not part of
// any cycle and is reported separately). Each distinct cycle is reported once,
// keyed by its normalized member set, so the same loop reached from two entry
// points is not double-counted.
func (w Workflow) detectCycles(byID map[string]Phase) []Finding {
	const (
		white = 0 // unvisited
		gray  = 1 // on the active recursion stack
		black = 2 // fully explored
	)
	color := make(map[string]int, len(byID))
	var stack []string
	var findings []Finding
	reported := make(map[string]bool) // normalized cycle key -> already reported

	var visit func(id string)
	visit = func(id string) {
		color[id] = gray
		stack = append(stack, id)

		p := byID[id]
		for _, dep := range p.DependsOn {
			// Only existing phases form graph edges; the initial artifact and dangling
			// references are not nodes.
			if _, ok := byID[dep]; !ok {
				continue
			}
			switch color[dep] {
			case white:
				visit(dep)
			case gray:
				// Re-entering a node on the active path closes a cycle. The chain runs
				// from that node's first appearance on the stack to the current node,
				// then back to the node, so the loop is self-evident.
				chain := cyclePhaseChain(stack, dep)
				key := cycleKey(chain)
				if !reported[key] {
					reported[key] = true
					findings = append(findings, Finding{
						Phase:    chain[0],
						Kind:     KindCycle,
						Observed: strings.Join(chain, " -> "),
						Reason:   "these phases form a dependency cycle, so the workflow is not a DAG; break the loop by removing one of the depends_on edges",
					})
				}
			}
		}

		stack = stack[:len(stack)-1]
		color[id] = black
	}

	// Visit in document order so cycle discovery is deterministic.
	for _, p := range w.Phases {
		if color[p.ID] == white {
			visit(p.ID)
		}
	}
	return findings
}

// cyclePhaseChain builds the cycle path to report: the active stack from the first
// occurrence of id onward, with id appended again to close the loop visibly (e.g.
// ["a", "b", "a"]). Mirrors prompts.cycleChain.
func cyclePhaseChain(stack []string, id string) []string {
	start := 0
	for i, s := range stack {
		if s == id {
			start = i
			break
		}
	}
	chain := make([]string, 0, len(stack)-start+1)
	chain = append(chain, stack[start:]...)
	chain = append(chain, id)
	return chain
}

// cycleKey normalizes a cycle's member set into a stable key so the same loop,
// discovered from different entry points, is reported exactly once. It drops the
// duplicated closing element, sorts the members and joins them; the rendered chain
// in the finding still preserves the discovery order for readability.
func cycleKey(chain []string) string {
	if len(chain) == 0 {
		return ""
	}
	members := make([]string, 0, len(chain)-1)
	members = append(members, chain[:len(chain)-1]...)
	sort.Strings(members)
	return strings.Join(members, "\x00")
}

// checkDependencies reports a depends_on entry that names neither an existing
// phase nor the initial artifact (R8 degenerate handling, additive diagnostic). A
// reference to the initial artifact "brief" is legitimate (a first phase depends
// on the pipeline entry), so it is accepted. Entries are checked in declared
// order; duplicates within one phase each yield a finding so nothing is hidden.
func (w Workflow) checkDependencies(p Phase, byID map[string]Phase) []Finding {
	var findings []Finding
	for _, dep := range p.DependsOn {
		if dep == initialArtifact {
			continue
		}
		if _, ok := byID[dep]; ok {
			continue
		}
		findings = append(findings, Finding{
			Phase:    p.ID,
			Kind:     KindUnknownDependency,
			Observed: dep,
			Reason:   "depends_on references no existing phase (and is not the initial artifact " + fmtQuote(initialArtifact) + "); fix the reference or add the missing phase",
		})
	}
	return findings
}

// checkArtifacts reports each input artifact of a phase that is not available: not
// the initial artifact and not produced as an output by any transitive predecessor
// (R3). The predecessor set is computed by walking depends_on edges among existing
// phases (see ancestorsOf), which is also robust to cycles because the walk uses a
// visited set. Inputs are checked in declared order so findings are deterministic.
func (w Workflow) checkArtifacts(p Phase, byID map[string]Phase) []Finding {
	if len(p.Inputs) == 0 {
		return nil
	}

	// Collect every output produced by a transitive predecessor of p.
	produced := make(map[string]bool)
	for ancestorID := range w.ancestorsOf(p.ID, byID) {
		for _, out := range byID[ancestorID].Outputs {
			produced[out] = true
		}
	}

	var findings []Finding
	for _, in := range p.Inputs {
		if in == initialArtifact || produced[in] {
			continue
		}
		findings = append(findings, Finding{
			Phase:    p.ID,
			Kind:     KindMissingArtifact,
			Observed: in,
			Reason:   "no predecessor phase produces this input artifact and it is not the initial artifact " + fmtQuote(initialArtifact) + "; add a predecessor that outputs it, or correct the input",
		})
	}
	return findings
}

// ancestorsOf returns the set of phase ids that are transitive predecessors of the
// phase id, reachable by following depends_on edges back toward the roots. It
// considers only edges to existing phases (a dangling reference contributes no
// ancestor) and uses a visited set so it terminates even if the graph contains a
// cycle (R8) — a cycle is reported elsewhere; here we just must not loop forever.
// The phase itself is not included in its own ancestor set.
func (w Workflow) ancestorsOf(id string, byID map[string]Phase) map[string]bool {
	ancestors := make(map[string]bool)
	var walk func(cur string)
	walk = func(cur string) {
		p, ok := byID[cur]
		if !ok {
			return
		}
		for _, dep := range p.DependsOn {
			if _, exists := byID[dep]; !exists {
				continue
			}
			if ancestors[dep] {
				continue // already accounted for (also breaks cycles)
			}
			ancestors[dep] = true
			walk(dep)
		}
	}
	walk(id)
	// Defensive: a self-loop would have added id to its own ancestor set; drop it so
	// a phase never counts as its own predecessor.
	delete(ancestors, id)
	return ancestors
}

// checkAgent reports a phase whose agent is not known to the workspace (R4), using
// the injected predicate. A nil predicate means the caller opted out of agent
// existence checking, so nothing is reported. An empty agent is a schema concern
// (validate.go), not re-reported here.
func checkAgent(p Phase, knownAgents func(string) bool) (Finding, bool) {
	if knownAgents == nil || trimmedEmpty(p.Agent) {
		return Finding{}, false
	}
	if knownAgents(p.Agent) {
		return Finding{}, false
	}
	return Finding{
		Phase:    p.ID,
		Kind:     KindUnknownAgent,
		Observed: p.Agent,
		Reason:   "no agent with this id exists in the workspace catalog; create or import the agent, or correct the reference",
	}, true
}

// sortFindingsGraph imposes a fully deterministic order on findings (R7),
// independent of collection order: by the affected phase's position in the
// document, then by a fixed kind rank, then by observed text as a final tie-break.
// A phase that is not in the position map (e.g. a cycle anchored to a phase id, or
// a defensive edge case) sorts after the known phases but still deterministically.
func sortFindingsGraph(findings []Finding, pos map[string]int) {
	rankPhase := func(id string) int {
		if i, ok := pos[id]; ok {
			return i
		}
		return len(pos)
	}
	sort.SliceStable(findings, func(i, j int) bool {
		pi, pj := rankPhase(findings[i].Phase), rankPhase(findings[j].Phase)
		if pi != pj {
			return pi < pj
		}
		ri, rj := kindRank(findings[i].Kind), kindRank(findings[j].Kind)
		if ri != rj {
			return ri < rj
		}
		return findings[i].Observed < findings[j].Observed
	})
}

// kindRank fixes the within-phase ordering of finding kinds so a phase's findings
// read in a stable, sensible sequence (cycle first as the most structural, then
// dangling dependency, then missing artifact, then unknown agent).
func kindRank(k FindingKind) int {
	switch k {
	case KindCycle:
		return 0
	case KindUnknownDependency:
		return 1
	case KindMissingArtifact:
		return 2
	case KindUnknownAgent:
		return 3
	default:
		return 4
	}
}
