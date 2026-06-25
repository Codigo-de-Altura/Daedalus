package backlog

import (
	"fmt"
	"strings"
)

// On-disk format of an epic and a ticket (R3/R5/R6).
//
// Both are single Markdown files with a YAML frontmatter block delimited by `---`
// lines, followed by the body verbatim. The frontmatter carries the stable metadata
// (status, priority, dependencies, origin links); the body is the human-authored
// objective/feature description.
//
// An epic `.daedalus/epics/epic-NN-<slug>/epic.md`:
//
//	---
//	id: epic-05-sdd-backlog
//	kind: epic
//	title: SDD Backlog
//	status: todo
//	priority: medium
//	spec: sdd-backlog
//	architecture: sdd-backlog-arch
//	depends_on: [epic-04-workflows]
//	agent: planner
//	workflow: sdd-default
//	phase: epics
//	generated: false
//	---
//	<epic markdown, verbatim>
//
// A ticket `.daedalus/epics/epic-NN-<slug>/tickets/ticket-NN-MM-<slug>/ticket.md`:
//
//	---
//	id: ticket-05-03-epics-tickets-management
//	kind: ticket
//	title: Epics & Tickets Management
//	epic: epic-05-sdd-backlog
//	status: todo
//	priority: high
//	spec: sdd-backlog
//	architecture: sdd-backlog-arch
//	depends_on: [ticket-05-02-architecture-docs]
//	agent: planner
//	workflow: sdd-default
//	phase: tickets
//	generated: false
//	---
//	<ticket markdown, verbatim>
//
// # Key order and why (R3/R6)
//
// The order is FIXED and chosen for the reader: identity first (id, kind, title), then
// — for a ticket — its parent epic (the R5/CA5 link, mandatory), then the lifecycle
// metadata (status, priority), then the optional origin links (spec, architecture),
// then the explicit dependency list (depends_on), then — when the artifact is linked
// to any origin — the planner-step provenance. status/priority/depends_on are ALWAYS
// present (depends_on as `[]` when empty) so the metadata shape is stable and a diff
// never has to distinguish "absent" from "empty". spec/architecture are omitted when
// empty (an unlinked artifact must not carry an empty link), mirroring the omit-empty
// rule of the other packages.
//
// The planner-step provenance group (agent/workflow/phase/generated) is emitted only
// when the artifact records at least one origin link (spec or architecture): it
// describes the `... -> epics/tickets` step that produced the artifact, which is only
// meaningful when an origin is recorded. `generated: false` is a real YAML boolean,
// written raw, to make R7/CA7 machine-checkable. The phase differs by kind: `epics`
// for an epic, `tickets` for a ticket (the two planner steps).
//
// List-valued keys (depends_on) render in YAML flow style on a single line —
// `depends_on: [a, b]` — mirroring the workflows renderer, with `[]` for empty. The
// renderer is hand-rolled and stdlib-only and is duplicated from the sibling packages
// so backlog owns its own format. Output always ends with a single trailing newline.

const (
	frontmatterDelim = "---"
	kindEpic         = "epic"
	kindTicket       = "ticket"
)

// Frontmatter key names, exported so the parser, validator and tests refer to fields
// by a stable identifier instead of hardcoding strings.
const (
	FieldID           = "id"
	FieldKind         = "kind"
	FieldTitle        = "title"
	FieldEpic         = "epic"
	FieldStatus       = "status"
	FieldPriority     = "priority"
	FieldSpec         = "spec"
	FieldArchitecture = "architecture"
	FieldDependsOn    = "depends_on"
	FieldAgent        = "agent"
	FieldWorkflow     = "workflow"
	FieldPhase        = "phase"
	FieldGenerated    = "generated"
)

// RenderEpic serializes an epic to its canonical on-disk bytes (R6). It assumes the
// epic is already valid (callers validate before rendering); it does not validate here
// so it stays a pure formatting function. Output is byte-stable for a given Epic.
func RenderEpic(e Epic) string {
	var b strings.Builder

	b.WriteString(frontmatterDelim)
	b.WriteByte('\n')

	writeScalar(&b, FieldID, e.ID)
	writeScalar(&b, FieldKind, kindEpic)
	writeScalar(&b, FieldTitle, e.Title)
	writeScalar(&b, FieldStatus, string(e.Status))
	writeScalar(&b, FieldPriority, string(e.Priority))
	writeOptionalScalar(&b, FieldSpec, e.SpecRef)
	writeOptionalScalar(&b, FieldArchitecture, e.ArchitectureRef)
	writeList(&b, FieldDependsOn, e.DependsOn)
	writeProvenance(&b, PhaseEpics, hasOrigin(e.SpecRef, e.ArchitectureRef))

	b.WriteString(frontmatterDelim)
	b.WriteByte('\n')

	writeBody(&b, e.Body)
	return b.String()
}

// RenderTicket serializes a ticket to its canonical on-disk bytes (R6). The `epic` key
// (the mandatory R5/CA5 link) is emitted right after identity; the planner-step phase
// is `tickets`.
func RenderTicket(t Ticket) string {
	var b strings.Builder

	b.WriteString(frontmatterDelim)
	b.WriteByte('\n')

	writeScalar(&b, FieldID, t.ID)
	writeScalar(&b, FieldKind, kindTicket)
	writeScalar(&b, FieldTitle, t.Title)
	writeScalar(&b, FieldEpic, t.EpicID)
	writeScalar(&b, FieldStatus, string(t.Status))
	writeScalar(&b, FieldPriority, string(t.Priority))
	writeOptionalScalar(&b, FieldSpec, t.SpecRef)
	writeOptionalScalar(&b, FieldArchitecture, t.ArchitectureRef)
	writeList(&b, FieldDependsOn, t.DependsOn)
	writeProvenance(&b, PhaseTickets, hasOrigin(t.SpecRef, t.ArchitectureRef))

	b.WriteString(frontmatterDelim)
	b.WriteByte('\n')

	writeBody(&b, t.Body)
	return b.String()
}

// hasOrigin reports whether the artifact records at least one origin link, which is
// what gates the planner-step provenance group.
func hasOrigin(specRef, archRef string) bool {
	return !trimmedEmpty(specRef) || !trimmedEmpty(archRef)
}

// writeProvenance emits the planner-step provenance group (agent/workflow/phase/
// generated) for the given phase, but only when the artifact is linked to an origin.
// The group is all-or-nothing because the four keys are only meaningful together — they
// describe one `... -> phase` step. `generated` is a real YAML boolean (written raw) so
// R7/CA7 is machine-checkable, not a coincidentally-named string.
func writeProvenance(b *strings.Builder, phase string, linked bool) {
	if !linked {
		return
	}
	writeScalar(b, FieldAgent, PlannerAgent)
	writeScalar(b, FieldWorkflow, DefaultWorkflowName)
	writeScalar(b, FieldPhase, phase)
	writeRaw(b, FieldGenerated, "false")
}

// writeBody appends the verbatim body, normalizing only the trailing newline so the
// file is byte-stable (R6).
func writeBody(b *strings.Builder, body string) {
	trimmed := strings.TrimRight(body, "\n")
	if trimmed != "" {
		b.WriteString(trimmed)
		b.WriteByte('\n')
	}
}

// writeScalar writes a `key: value` line with a YAML-safe value (always present key).
func writeScalar(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s: %s\n", key, yamlScalar(value))
}

// writeOptionalScalar writes the key only when value is non-empty; an empty optional
// link is omitted entirely rather than emitted as `key: ""`, so an unlinked artifact
// carries no misleading empty reference (mirrors the omit-empty rule of the siblings).
func writeOptionalScalar(b *strings.Builder, key, value string) {
	if trimmedEmpty(value) {
		return
	}
	writeScalar(b, key, value)
}

// writeList writes a `key: [a, b]` line in YAML flow style, or `key: []` for an empty
// list. Each element passes through yamlScalar so a value ambiguous in flow context
// (containing a comma, bracket, etc.) is quoted. Order is preserved verbatim, so the
// same list always renders the same bytes (R6). Mirrors the workflows renderer.
func writeList(b *strings.Builder, key string, items []string) {
	if len(items) == 0 {
		fmt.Fprintf(b, "%s: []\n", key)
		return
	}
	rendered := make([]string, len(items))
	for i, it := range items {
		rendered[i] = yamlScalar(it)
	}
	fmt.Fprintf(b, "%s: [%s]\n", key, strings.Join(rendered, ", "))
}

// writeRaw writes a `key: value` line with the value emitted verbatim, for known-safe
// YAML literals (e.g. the boolean keyword `false`) that must NOT pass through the
// conservative scalar quoter.
func writeRaw(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s: %s\n", key, value)
}

// yamlScalar renders a string as a YAML scalar, quoting it only when the plain form
// could be misread by a parser. Mirrors the sibling packages' helper; conservative.
func yamlScalar(s string) string {
	if s == "" {
		return `""`
	}
	if needsQuoting(s) {
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s
}

// needsQuoting reports whether a YAML scalar must be quoted to round-trip safely. It is
// intentionally conservative and mirrors the sibling helper, with the brackets/comma
// cases relevant here because list elements render inline in flow style.
func needsQuoting(s string) bool {
	if s != strings.TrimSpace(s) {
		return true
	}
	switch s {
	case "true", "false", "null", "yes", "no", "on", "off", "~":
		return true
	}
	if first := s[0]; first >= '0' && first <= '9' {
		return true
	}
	return strings.ContainsAny(s, ":#{}[],&*!|>'\"%@`")
}
