package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

// This file owns the read-only rendering of a workflow DAG (ticket 04-02). It
// turns the canonical workflows.Workflow model (ticket 04-01) into a legible
// terminal graph: each phase is a bordered node showing its id + agent (and a
// compact view of its inputs/outputs/gate), and each depends_on dependency is a
// vertical connector drawn so the pipeline reads top-to-bottom in dependency
// order (predecessor above its dependents).
//
// It is presentation only (R4/CA5): nothing here mutates or executes the
// workflow. It also computes nothing semantic (cycle/reference validation is
// ticket 04-03) beyond the layout-local cycle guard below, which exists purely so
// the renderer cannot loop forever on a malformed graph (R7/CA7).

// topoOrder returns the workflow's phases in a deterministic topological order:
// roots (phases nothing depends on, i.e. no satisfiable predecessor) first, then
// their dependents, so the rendered pipeline reads brief → spec → … → docs (R3/CA3).
//
// It uses Kahn's algorithm with a stable tie-break (declared phase order) so the
// same workflow always lays out identically. Edges that reference a non-existent
// phase are ignored for ordering (that dangling reference is ticket 04-03's
// semantic concern, not a layout error). If a cycle prevents ordering every phase
// — Kahn cannot drain all nodes — we fall back to the declared phase order for the
// remainder and return cyclic=true, so the view degrades to "show it in declared
// order" instead of hanging or panicking (R7/CA7).
func topoOrder(w workflows.Workflow) (order []workflows.Phase, cyclic bool) {
	n := len(w.Phases)
	if n == 0 {
		return nil, false
	}

	// idxByID maps a phase id to its declared position; only real phases count as
	// dependencies, so a depends_on pointing at a non-phase contributes no edge.
	idxByID := make(map[string]int, n)
	for i, p := range w.Phases {
		idxByID[p.ID] = i
	}

	// indegree[i] counts how many of phase i's depends_on entries resolve to an
	// actual phase; adj[from] lists the phases that depend on `from`.
	indegree := make([]int, n)
	adj := make([][]int, n)
	for i, p := range w.Phases {
		for _, dep := range p.DependsOn {
			if j, ok := idxByID[dep]; ok {
				indegree[i]++
				adj[j] = append(adj[j], i)
			}
		}
	}

	// Seed the queue with every root (indegree 0) in declared order so the layout
	// is deterministic; process by repeatedly taking the smallest-declared ready
	// node so siblings keep their authored order.
	ready := make([]int, 0, n)
	for i := 0; i < n; i++ {
		if indegree[i] == 0 {
			ready = append(ready, i)
		}
	}

	visited := make([]bool, n)
	order = make([]workflows.Phase, 0, n)
	for len(ready) > 0 {
		// Stable selection: pick the ready node with the smallest declared index.
		sort.Ints(ready)
		cur := ready[0]
		ready = ready[1:]
		if visited[cur] {
			continue
		}
		visited[cur] = true
		order = append(order, w.Phases[cur])
		for _, next := range adj[cur] {
			indegree[next]--
			if indegree[next] == 0 {
				ready = append(ready, next)
			}
		}
	}

	if len(order) == n {
		return order, false
	}

	// A cycle left some phases unprocessed. Append them in declared order so every
	// phase is still rendered exactly once, and flag the workflow as cyclic so the
	// view can warn the reader. This is the layout-local guard against an infinite
	// loop — it does not attempt the semantic cycle diagnosis of ticket 04-03.
	for i := 0; i < n; i++ {
		if !visited[i] {
			order = append(order, w.Phases[i])
		}
	}
	return order, true
}

// renderDAG renders the whole workflow as a top-to-bottom graph string: one node
// box per phase in topological order, joined by vertical edge connectors that
// carry the names of the predecessors each phase depends on. The result is meant
// to be placed inside a scrollable viewport by the caller, so it does no height
// clamping itself.
func (m Model) renderDAG(w workflows.Workflow) string {
	order, cyclic := topoOrder(w)

	var b strings.Builder
	if cyclic {
		// Surface the layout fallback without computing the semantic diagnosis
		// (ticket 04-03): the order may not be a true topological one, so say so.
		b.WriteString(m.theme.dagMeta.Render(
			"! dependency cycle detected — phases shown in declared order"))
		b.WriteString("\n\n")
	}

	for i, p := range order {
		if i > 0 {
			// Draw the edge into this phase: a connector annotated with the
			// predecessors it depends on, so the pipeline direction is explicit (R3).
			b.WriteString(m.renderEdge(p))
		}
		b.WriteString(m.renderNode(p))
		if i < len(order)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderNode renders one phase as a bordered card. The headline is the phase id
// and the agent that runs it (R2/CA2, the mandatory minimum); below it, when
// present, a compact one-line-each summary of inputs, outputs and gate so the
// reader sees the data flow without opening the YAML. Empty fields are omitted so
// a sparse phase stays compact.
func (m Model) renderNode(p workflows.Phase) string {
	var lines []string

	head := m.theme.dagNodeID.Render(p.ID)
	if agent := strings.TrimSpace(p.Agent); agent != "" {
		head += "  " + m.theme.dagAgent.Render("@"+agent)
	}
	lines = append(lines, head)

	if len(p.Inputs) > 0 {
		lines = append(lines, m.theme.dagMeta.Render("in:  "+strings.Join(p.Inputs, ", ")))
	}
	if len(p.Outputs) > 0 {
		lines = append(lines, m.theme.dagMeta.Render("out: "+strings.Join(p.Outputs, ", ")))
	}
	if gate := strings.TrimSpace(p.Gate); gate != "" {
		lines = append(lines, m.theme.dagMeta.Render("gate: "+gate))
	}

	return m.theme.dagNode.Render(strings.Join(lines, "\n"))
}

// renderEdge draws the vertical connector that leads into a phase, labelled with
// the predecessors it depends on so the edge reads as "these run before this"
// (R3/CA3). A phase with no resolvable dependency (a root reached out of order, or
// a leaf laid out after an unrelated branch) still gets a plain connector so the
// vertical flow is unbroken and nothing ever renders edgeless mid-graph.
func (m Model) renderEdge(p workflows.Phase) string {
	connector := m.theme.dagEdge.Render("  │")
	if len(p.DependsOn) == 0 {
		return connector + "\n" + m.theme.dagEdge.Render("  ↓") + "\n"
	}
	label := m.theme.dagMeta.Render(
		fmt.Sprintf("after %s", strings.Join(p.DependsOn, ", ")))
	return connector + "  " + label + "\n" + m.theme.dagEdge.Render("  ↓") + "\n"
}

// dagViewportContent builds the full DAG body for the viewport, or a clear empty
// state when the workflow has no phases (R7/CA7). An empty workflow is a valid
// document — it parsed fine, it just has nothing to draw — so it is an empty
// state, not an error.
func (m Model) dagViewportContent(w workflows.Workflow) string {
	if len(w.Phases) == 0 {
		return m.theme.emptyState.Render(
			"This workflow is empty.\n\n" +
				"Add phases with `daedalus workflow add-phase`.")
	}
	return m.renderDAG(w)
}
