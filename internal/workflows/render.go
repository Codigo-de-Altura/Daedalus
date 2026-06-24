package workflows

import (
	"fmt"
	"strings"
)

// On-disk format of a persisted workflow (R2/R3/R4).
//
// A workflow is a single YAML file `.daedalus/workflows/<name>.yaml` with exactly
// one top-level key, `phases:`, holding a block sequence of phase mappings:
//
//	phases:
//	  - id: spec
//	    agent: analyst
//	    inputs: [brief]
//	    outputs: [spec]
//	    gate: spec-gate
//	    depends_on: [brief]
//	  - id: build
//	    agent: architect
//	    inputs: [spec]
//	    outputs: [design]
//	    gate: design-gate
//	    depends_on: [spec]
//
// The workflow name is deliberately NOT a key in the document: it is the file's
// base name (`<name>.yaml`), the canonical identity, exactly as a prompt's id is
// its file name and an agent's id is its directory name. Carrying the name twice
// (file + a `name:` key) would let the two drift and would add a key with no
// information, so the format omits it and Load supplies the name from the path.
//
// Key order within each phase is FIXED and deterministic: id, agent, inputs,
// outputs, gate, depends_on — the same order the epic's schema lists them, chosen
// for the reader (identity, then who runs it, then what flows in/out, then the
// gate, then the edges). Phases are emitted in the model's order; they are NEVER
// reordered, because the ordered list is itself meaningful (it is the authored
// pipeline order). Every phase key is always present so the shape is stable and a
// reader never has to distinguish "absent" from "empty".
//
// List-valued keys (inputs, outputs, depends_on) are rendered in YAML flow style
// on a single line — `inputs: [brief, design]` — with each element passing
// through yamlScalar's conservative quoting, and an empty list as `[]`. Flow
// style keeps a phase compact and its diff a single line per list, which is the
// git-friendly choice for these short artifact-reference lists (RNF-5/RNF-6).
//
// The renderer is hand-rolled and stdlib-only — go.mod carries no YAML
// dependency and stdlib-first is the project rule — and is duplicated from (not
// shared with) the prompts/catalog renderers so workflows owns its own format.
// Output always ends with a single trailing newline so the file is POSIX-clean.

// Top-level and per-phase key names, exported as constants so the parser, the
// validator and tests refer to fields by a stable identifier instead of
// hardcoding strings, and so the names a user sees in an error match the
// serialized form exactly.
const (
	keyPhases      = "phases"
	FieldID        = "id"
	FieldAgent     = "agent"
	FieldInputs    = "inputs"
	FieldOutputs   = "outputs"
	FieldGate      = "gate"
	FieldDependsOn = "depends_on"
)

// Render serializes a workflow to its canonical on-disk bytes. It assumes the
// workflow is already valid (callers validate before rendering); it does not
// validate here so it stays a pure formatting function. Output is byte-stable for
// a given Workflow (R3/R4): same model in, same bytes out, every time.
func Render(w Workflow) string {
	var b strings.Builder

	// An empty workflow still emits the top-level key with an explicit empty
	// sequence, so the file always has the stable `phases:` shape and a reader
	// never sees a bare or missing key.
	if len(w.Phases) == 0 {
		b.WriteString(keyPhases)
		b.WriteString(": []\n")
		return b.String()
	}

	b.WriteString(keyPhases)
	b.WriteString(":\n")

	for _, p := range w.Phases {
		// The sequence item marker carries the first key inline ("- id: ..."); the
		// remaining keys align under it at four-space indent, the conventional YAML
		// block-mapping-in-sequence layout.
		fmt.Fprintf(&b, "  - %s: %s\n", FieldID, yamlScalar(p.ID))
		writePhaseScalar(&b, FieldAgent, p.Agent)
		writePhaseList(&b, FieldInputs, p.Inputs)
		writePhaseList(&b, FieldOutputs, p.Outputs)
		writePhaseScalar(&b, FieldGate, p.Gate)
		writePhaseList(&b, FieldDependsOn, p.DependsOn)
	}

	return b.String()
}

// writePhaseScalar writes a `    key: value` line for a continuation key of a
// phase mapping (four-space indent so it aligns under the "- id:" marker), with a
// YAML-safe scalar value.
func writePhaseScalar(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "    %s: %s\n", key, yamlScalar(value))
}

// writePhaseList writes a `    key: [a, b, c]` line in flow style, or `key: []`
// for an empty list. Each element is rendered through yamlScalar so a value that
// would be ambiguous in flow context (containing a comma, bracket, etc.) is
// quoted. Determinism comes for free: the slice order is preserved verbatim, so
// the same list always renders the same bytes (R4).
func writePhaseList(b *strings.Builder, key string, items []string) {
	if len(items) == 0 {
		fmt.Fprintf(b, "    %s: []\n", key)
		return
	}
	rendered := make([]string, len(items))
	for i, it := range items {
		rendered[i] = yamlScalar(it)
	}
	fmt.Fprintf(b, "    %s: [%s]\n", key, strings.Join(rendered, ", "))
}

// yamlScalar renders a string as a YAML scalar, quoting it only when the plain
// form could be misread by a parser. It mirrors the prompts/catalog renderers'
// helper of the same name; duplicated rather than shared because the two packages
// own independent canonical formats. The quoting is conservative: when in doubt,
// quote.
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

// needsQuoting reports whether a YAML scalar must be quoted to round-trip safely.
// It is intentionally conservative and mirrors the prompts/catalog helper, with
// the brackets/comma cases especially relevant here because list elements are
// rendered inline in flow style where `[`, `]` and `,` are structural.
func needsQuoting(s string) bool {
	if s != strings.TrimSpace(s) {
		return true // leading/trailing whitespace is significant only when quoted
	}
	switch s {
	case "true", "false", "null", "yes", "no", "on", "off", "~":
		return true
	}
	if first := s[0]; first >= '0' && first <= '9' {
		return true // could be parsed as a number
	}
	return strings.ContainsAny(s, ":#{}[],&*!|>'\"%@`")
}
