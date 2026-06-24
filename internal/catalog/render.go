package catalog

import (
	"fmt"
	"strings"
)

// DefinitionVersion is the schema version stamped into a materialized agent's
// YAML. Like workspace.SchemaVersion it identifies the *format* of the agent
// definition, not the binary: bumping it signals a breaking change to the
// canonical agent layout (a migration concern for ticket-02-04 and beyond).
// Keeping it explicit and constant is also what lets the rendered output stay
// byte-stable across binary releases (R8/CA6).
const DefinitionVersion = "1"

// renderDefinition serializes an agent's canonical *definition* to deterministic
// YAML using only the standard library — the same hand-rolled approach as the
// workspace manifest (content.go): go.mod carries no YAML dependency and
// stdlib-first is the rule, and a fixed, small shape gives us guaranteed key
// order without trusting a library's map iteration. Output ends with a trailing
// newline and is byte-for-byte stable for a given Agent (R8/CA6).
//
// The prompt is intentionally *not* inlined here: it lives in a sibling
// prompt.md (referenced by `prompt: prompt.md`) so the canonical definition
// stays a small, diff-friendly metadata file and the prompt can be edited as
// Markdown. The key order is fixed by the writes below, not by the struct, so it
// is unambiguous when reading this function top-to-bottom.
func renderDefinition(a Agent) string {
	var b strings.Builder

	b.WriteString("# Daedalus canonical agent definition.\n")
	b.WriteString("# Generated from the built-in catalog. Keys are ordered and stable for clean diffs.\n")
	b.WriteString("# This file is the editable source of truth; the prompt lives in prompt.md.\n")

	writeScalar(&b, "id", a.ID)
	writeScalar(&b, "version", DefinitionVersion)
	writeScalar(&b, "role", a.Role)
	// The prompt is stored alongside as Markdown; the definition only points to
	// it so the two never drift and the YAML stays small.
	writeScalar(&b, "prompt", PromptFileName)

	if len(a.Params) == 0 {
		// Emit an explicit empty mapping so the key is always present (stable
		// shape) and a reader never has to distinguish "absent" from "empty".
		b.WriteString("parameters: {}\n")
		return b.String()
	}

	b.WriteString("parameters:\n")
	for _, p := range a.Params {
		fmt.Fprintf(&b, "  %s: %s\n", p.Key, renderParamValue(p))
	}

	return b.String()
}

// renderParamValue renders a parameter's value according to its declared type.
// Numbers and bools are emitted bare (so a parser reads them as their type, not
// as a string), while strings go through yamlScalar's conservative quoting. This
// is the single spot that maps the canonical string Value back to a typed YAML
// scalar, keeping the type→syntax decision in one place.
func renderParamValue(p Param) string {
	switch p.Type {
	case ParamNumber, ParamBool:
		// Trust the curated canonical form; these come from the built-in catalog
		// (and, later, a validated schema), so they are already well-formed
		// literals and must not be quoted.
		return p.Value
	default:
		return yamlScalar(p.Value)
	}
}

// writeScalar writes a `key: value` line with a YAML-safe value. It mirrors the
// workspace renderer's helper of the same name; duplicated rather than shared
// because the two packages own independent canonical formats and we do not want
// a change to one silently reshaping the other.
func writeScalar(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s: %s\n", key, yamlScalar(value))
}

// yamlScalar renders a string as a YAML scalar, quoting it only when the plain
// form could be misread by a parser. Curated catalog values are simple, but
// roles can contain ':' and other indicator characters, so we keep the same
// conservative quoting as the workspace renderer.
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
// It is intentionally conservative: when in doubt, quote.
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

// renderPrompt produces the prompt.md body for an agent. The catalog stores
// prompts already as Markdown, so this only guarantees a single trailing newline
// so the file is POSIX-clean and byte-stable regardless of how the literal was
// authored (R8/CA6).
func renderPrompt(a Agent) string {
	return strings.TrimRight(a.Prompt, "\n") + "\n"
}
