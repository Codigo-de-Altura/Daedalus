package workflows

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrWorkflowNotFound is returned (wrapped) when a requested workflow name has no
// file under the workflows root. Exposed as a sentinel so callers (the CLI/TUI)
// can map "no such workflow" to a usage error via errors.Is without string
// matching.
var ErrWorkflowNotFound = errors.New("workflow not found")

// ErrMalformedWorkflow is the sentinel returned (wrapped) when a workflow file on
// disk cannot be parsed against the canonical format (R7). It is distinct from
// ErrWorkflowNotFound (the file is absent) so a caller can tell "no such
// workflow" from "this workflow's file is corrupt", and from *ValidationError
// (the file parsed but its schema is invalid).
var ErrMalformedWorkflow = errors.New("malformed workflow")

// fileName returns the on-disk file name for a workflow name: `<name>.yaml` (R2).
// The name is assumed to be valid kebab-case (callers validate first), so it is a
// safe single path segment.
func fileName(name string) string {
	return name + FileExt
}

// Load reads a persisted workflow by name from workflowsRoot and reconstructs the
// in-memory model (R3/CA1). The directory entry's base name (the `<name>` of
// `<name>.yaml`) is the canonical identity — the document carries no name of its
// own — so Load always returns the workflow under the name the caller asked for,
// and a file renamed on disk cannot disagree with its contents.
//
// We hand-roll the parser instead of pulling in a YAML library: go.mod carries
// none and stdlib-first is the rule, and we own this small, fixed format.
// Anything that does not match the shape Render writes is rejected as malformed
// (wrapping ErrMalformedWorkflow) rather than silently half-read (R7).
func Load(workflowsRoot, name string) (Workflow, error) {
	if !IsKebabCase(name) {
		return Workflow{}, fmt.Errorf("workflow name %q is not valid kebab-case", name)
	}

	path := filepath.Join(workflowsRoot, fileName(name))
	raw, err := os.ReadFile(path)
	if err != nil {
		// Surface absence as not-found so a caller can distinguish it from a parse
		// failure; other I/O errors propagate as-is.
		if errors.Is(err, os.ErrNotExist) {
			return Workflow{}, fmt.Errorf("%w: %q", ErrWorkflowNotFound, name)
		}
		return Workflow{}, err
	}

	w, err := parse(string(raw))
	if err != nil {
		return Workflow{}, fmt.Errorf("%w: workflow %q: %v", ErrMalformedWorkflow, name, err)
	}

	// The file name is the canonical identity; supply it on the model.
	w.Name = name
	return w, nil
}

// parse turns a workflow file's bytes into a Workflow. It accepts exactly the
// shape Render emits: a single top-level `phases:` key whose value is a block
// sequence of phase mappings, each a fixed set of `key: value` lines (id, agent,
// inputs, outputs, gate, depends_on) where the three list keys are flow-style
// `[a, b, c]` (or `[]`). It is a minimal, hand-rolled scanner — no YAML
// dependency, by project rule — strict about the shape it knows so a malformed
// file is flagged (R7) rather than loaded half-read. It does not validate the
// schema's *semantics* (that the ids are kebab-case, the agent is non-empty,
// etc.): that is Validate's job, run separately by the persisting callers.
func parse(content string) (Workflow, error) {
	lines := splitLines(content)

	// Find the top-level `phases:` key. We accept leading blank/comment lines so a
	// hand-authored file with a header comment still loads.
	i := 0
	for i < len(lines) && isBlankOrComment(lines[i]) {
		i++
	}
	if i >= len(lines) {
		return Workflow{}, fmt.Errorf("missing required top-level key %q", keyPhases)
	}

	key, rest, ok := strings.Cut(lines[i], ":")
	if !ok || strings.TrimSpace(key) != keyPhases {
		return Workflow{}, fmt.Errorf("expected top-level key %q, got %q", keyPhases, strings.TrimSpace(lines[i]))
	}
	rest = strings.TrimSpace(rest)
	i++

	// `phases: []` (or `phases:` with no items) is a valid, empty workflow.
	if rest == "[]" {
		return Workflow{}, nil
	}
	if rest != "" {
		return Workflow{}, fmt.Errorf("%q must be a block sequence or %q, got inline %q", keyPhases, "[]", rest)
	}

	phases, err := parsePhases(lines[i:])
	if err != nil {
		return Workflow{}, err
	}
	return Workflow{Phases: phases}, nil
}

// parsePhases scans the block sequence under `phases:`. Each item opens with a
// `  - ` marker carrying the first `key: value`; subsequent keys of the same item
// are indented continuation lines until the next marker. Blank and comment lines
// between/within items are tolerated. Anything that is neither a marker nor a
// recognized continuation line at the expected shape is reported as malformed.
func parsePhases(lines []string) ([]Phase, error) {
	var phases []Phase
	var cur map[string]string
	flush := func() error {
		if cur == nil {
			return nil
		}
		p, err := phaseFromFields(cur)
		if err != nil {
			return err
		}
		phases = append(phases, p)
		cur = nil
		return nil
	}

	for _, raw := range lines {
		line := strings.TrimRight(raw, "\r")
		if isBlankOrComment(line) {
			continue
		}

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "- ") {
			// New sequence item: flush the previous one, then read the inline key.
			if err := flush(); err != nil {
				return nil, err
			}
			cur = make(map[string]string)
			k, v, err := splitKeyValue(strings.TrimPrefix(trimmed, "- "))
			if err != nil {
				return nil, err
			}
			cur[k] = v
			continue
		}

		// A continuation key must belong to an open item.
		if cur == nil {
			return nil, fmt.Errorf("unexpected line outside a phase item: %q", trimmed)
		}
		k, v, err := splitKeyValue(trimmed)
		if err != nil {
			return nil, err
		}
		if _, dup := cur[k]; dup {
			return nil, fmt.Errorf("duplicate key %q in phase", k)
		}
		cur[k] = v
	}

	if err := flush(); err != nil {
		return nil, err
	}
	return phases, nil
}

// phaseFromFields builds a Phase from a phase item's parsed key→raw-value map. It
// requires every canonical key to be present so the on-disk shape Render writes
// round-trips exactly and a hand-authored file missing a key is flagged as
// malformed (R7) rather than loaded with silent zero values. The three list keys
// are parsed from flow style; the scalars are taken as-is (already unquoted by
// splitKeyValue).
func phaseFromFields(fields map[string]string) (Phase, error) {
	for _, required := range []string{FieldID, FieldAgent, FieldInputs, FieldOutputs, FieldGate, FieldDependsOn} {
		if _, ok := fields[required]; !ok {
			return Phase{}, fmt.Errorf("phase is missing required key %q", required)
		}
	}
	// Reject any key we do not recognize so a typo'd key (e.g. `input:`) is caught
	// at load instead of silently dropped.
	for k := range fields {
		switch k {
		case FieldID, FieldAgent, FieldInputs, FieldOutputs, FieldGate, FieldDependsOn:
		default:
			return Phase{}, fmt.Errorf("phase has unknown key %q", k)
		}
	}

	inputs, err := parseFlowList(fields[FieldInputs])
	if err != nil {
		return Phase{}, fmt.Errorf("%s: %v", FieldInputs, err)
	}
	outputs, err := parseFlowList(fields[FieldOutputs])
	if err != nil {
		return Phase{}, fmt.Errorf("%s: %v", FieldOutputs, err)
	}
	dependsOn, err := parseFlowList(fields[FieldDependsOn])
	if err != nil {
		return Phase{}, fmt.Errorf("%s: %v", FieldDependsOn, err)
	}

	return Phase{
		ID:        fields[FieldID],
		Agent:     fields[FieldAgent],
		Inputs:    inputs,
		Outputs:   outputs,
		Gate:      fields[FieldGate],
		DependsOn: dependsOn,
	}, nil
}

// splitKeyValue splits a `key: value` line into its bare key and unquoted scalar
// value, reversing the renderer's per-line format. The value is returned raw for
// list keys (the caller parses the flow list) and unquoted for scalar keys. It is
// strict about basic shape so a line that is not `key: value` is malformed.
func splitKeyValue(line string) (key, value string, err error) {
	k, v, ok := strings.Cut(line, ":")
	if !ok {
		return "", "", fmt.Errorf("line is not key: value: %q", line)
	}
	key = strings.TrimSpace(k)
	if key == "" {
		return "", "", fmt.Errorf("empty key in line %q", line)
	}
	value = strings.TrimSpace(v)
	// List keys keep their raw `[...]` value for parseFlowList; scalar keys are
	// unquoted here so the in-memory value is bare.
	switch key {
	case FieldInputs, FieldOutputs, FieldDependsOn:
		return key, value, nil
	default:
		scalar, uerr := unquoteScalar(value)
		if uerr != nil {
			return "", "", fmt.Errorf("%s: %v", key, uerr)
		}
		return key, scalar, nil
	}
}

// parseFlowList parses a YAML flow sequence `[a, b, c]` (or `[]`) into a slice of
// unquoted scalars, reversing writePhaseList. It is limited to flow style on a
// single line — the only form Render emits — so a value that is not bracketed is
// malformed. Elements are split on commas at the top level; because the values we
// write are simple artifact references (quoted only for safety), a comma inside a
// quoted element is not expected, but quoted elements are still unquoted faithfully.
func parseFlowList(raw string) ([]string, error) {
	if !strings.HasPrefix(raw, "[") || !strings.HasSuffix(raw, "]") {
		return nil, fmt.Errorf("expected a flow sequence [a, b, c], got %q", raw)
	}
	inner := strings.TrimSpace(raw[1 : len(raw)-1])
	if inner == "" {
		// Empty list. Return an empty (non-nil) slice so a round-tripped `[]` stays
		// `[]` and equality with a freshly built empty slice holds.
		return []string{}, nil
	}

	parts := splitFlowElements(inner)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		token := strings.TrimSpace(part)
		if token == "" {
			return nil, fmt.Errorf("empty element in flow sequence %q", raw)
		}
		scalar, err := unquoteScalar(token)
		if err != nil {
			return nil, err
		}
		out = append(out, scalar)
	}
	return out, nil
}

// splitFlowElements splits a flow-sequence body on top-level commas, respecting
// double-quoted elements so a comma inside quotes does not split. It is the
// minimal amount of structure-awareness needed to faithfully reverse what
// writePhaseList emits.
func splitFlowElements(inner string) []string {
	var parts []string
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		switch {
		case c == '"':
			inQuote = !inQuote
			b.WriteByte(c)
		case c == '\\' && inQuote && i+1 < len(inner):
			// Preserve an escape pair verbatim so unquoteScalar can interpret it.
			b.WriteByte(c)
			i++
			b.WriteByte(inner[i])
		case c == ',' && !inQuote:
			parts = append(parts, b.String())
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	parts = append(parts, b.String())
	return parts
}

// unquoteScalar reverses yamlScalar (render.go): it strips surrounding double
// quotes and undoes the backslash/quote escaping the renderer applied, leaving a
// bare scalar untouched. It is limited to the escapes the renderer produces (\\
// and \") because that is the entire surface of values we ever write; an
// unterminated or unexpectedly-escaped quoted scalar is reported as malformed.
func unquoteScalar(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	if token[0] != '"' {
		// Bare scalar: emitted only when yamlScalar judged it safe, so no escaping.
		return token, nil
	}
	if len(token) < 2 || token[len(token)-1] != '"' {
		return "", fmt.Errorf("unterminated quoted scalar: %q", token)
	}
	inner := token[1 : len(token)-1]

	var b strings.Builder
	for i := 0; i < len(inner); i++ {
		c := inner[i]
		if c != '\\' {
			b.WriteByte(c)
			continue
		}
		i++
		if i >= len(inner) {
			return "", fmt.Errorf("dangling escape in quoted scalar: %q", token)
		}
		switch inner[i] {
		case '\\':
			b.WriteByte('\\')
		case '"':
			b.WriteByte('"')
		default:
			return "", fmt.Errorf("unsupported escape %q in quoted scalar: %q", string(inner[i]), token)
		}
	}
	return b.String(), nil
}

// splitLines splits content into lines on LF, leaving any trailing CR on each
// line for the caller to trim. Both LF and CRLF endings are thus accepted so
// files authored on Windows load cleanly.
func splitLines(content string) []string {
	return strings.Split(content, "\n")
}

// isBlankOrComment reports whether a line is empty (after trimming) or a comment.
// Such lines carry no document structure and are skipped by the parser, so a
// header comment or spacing in a hand-authored file does not break the load.
func isBlankOrComment(line string) bool {
	t := strings.TrimSpace(strings.TrimRight(line, "\r"))
	return t == "" || strings.HasPrefix(t, "#")
}
