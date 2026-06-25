package architecture

import (
	"fmt"
	"strings"
)

// On-disk format of an architecture document (R1/R3/R6).
//
// A document is a single Markdown file `.daedalus/architecture/<slug>.md` with a
// YAML frontmatter block delimited by `---` lines, followed by the body verbatim.
//
// Unlinked (no originating spec recorded):
//
//	---
//	slug: payments-arch
//	kind: architecture
//	title: Payments Architecture
//	---
//	<architecture markdown, verbatim>
//
// Linked to its originating spec (the R3/CA3 trace):
//
//	---
//	slug: payments-arch
//	kind: architecture
//	title: Payments Architecture
//	spec: payments.md
//	agent: architect
//	workflow: sdd-default
//	phase: architecture
//	generated: false
//	---
//	<architecture markdown, verbatim>
//
// # Why these keys, and why the provenance block is all-or-nothing (R3/R5)
//
// The order is FIXED and chosen for the reader: identity first (slug, kind, title),
// then — ONLY when the document is linked — the provenance: which spec it came from
// (the R3/CA3 trace), produced by which agent, in which workflow/phase, and whether
// Daedalus generated its body. R3 makes the spec link OPTIONAL ("can be linked"), so
// when SpecRef is empty the WHOLE provenance group is omitted rather than emitted
// with empty values: an unlinked document must not carry misleading architect wiring,
// and emitting `spec: ""` / `agent: architect` for a document with no spec would be
// ambiguous noise in the diff. The group is all-or-nothing because the four keys are
// only meaningful together — they describe one `spec -> architecture` step.
//
// `generated` is always `false` in phase 1 and is written explicitly (as a real YAML
// boolean) to make R5/CA5 self-evident in the file itself: Daedalus seeded the
// placeholder; the user (running the architect on their backend) produces the real
// content. Every key that IS present is always present for a given link state, so the
// shape is stable and a diff never has to distinguish "absent" from "empty".
//
// The renderer is hand-rolled and stdlib-only — go.mod carries no YAML dependency and
// stdlib-first is the project rule — and is duplicated from (not shared with) the
// specs/prompts/workflows renderers so architecture owns its own format. Output always
// ends with a single trailing newline so the file is POSIX-clean.

const (
	// frontmatterDelim is the line that opens and closes the YAML frontmatter.
	frontmatterDelim = "---"

	// kindArchitecture is the `kind:` discriminator written into the frontmatter so a
	// reader (and the parser) can identify the artifact even out of context.
	kindArchitecture = "architecture"
)

// Frontmatter key names, exported as constants so the parser, the validator and
// tests refer to fields by a stable identifier instead of hardcoding strings, and so
// the names a user sees in an error match the serialized form exactly.
const (
	FieldSlug      = "slug"
	FieldKind      = "kind"
	FieldTitle     = "title"
	FieldSpec      = "spec"
	FieldAgent     = "agent"
	FieldWorkflow  = "workflow"
	FieldPhase     = "phase"
	FieldGenerated = "generated"
)

// Render serializes an architecture document to its canonical on-disk bytes. It
// assumes the document is already valid (callers validate before rendering); it does
// not validate here so it stays a pure formatting function. The architect-step
// provenance keys are emitted only when the document is linked to a spec (R3); they
// are written from the package constants, not from per-document fields, because the
// step is constant for every linked document. Output is byte-stable for a given
// Document (R6).
func Render(d Document) string {
	var sb strings.Builder

	sb.WriteString(frontmatterDelim)
	sb.WriteByte('\n')

	// Identity, always present.
	writeScalar(&sb, FieldSlug, d.Slug)
	writeScalar(&sb, FieldKind, kindArchitecture)
	writeScalar(&sb, FieldTitle, d.Title)

	// Provenance group, all-or-nothing: present only for a linked document (R3/CA3).
	if !trimmedEmpty(d.SpecRef) {
		writeScalar(&sb, FieldSpec, d.SpecRef)
		writeScalar(&sb, FieldAgent, ArchitectAgent)
		writeScalar(&sb, FieldWorkflow, DefaultWorkflowName)
		writeScalar(&sb, FieldPhase, DefaultPhase)
		// Phase 1: Daedalus never generates the body, so this is always false. It is a
		// real YAML boolean (written raw, not via yamlScalar, which would conservatively
		// quote the keyword "false" into the string "false"). A genuine boolean makes the
		// R5/CA5 fact machine-checkable, not just a coincidentally-named string.
		writeRaw(&sb, FieldGenerated, "false")
	}

	sb.WriteString(frontmatterDelim)
	sb.WriteByte('\n')

	// The body is persisted verbatim (R6). We normalize only the trailing newline so
	// the file is byte-stable regardless of how the body was authored: strip any
	// trailing newlines and re-append exactly one, which also guarantees the file ends
	// cleanly even for an empty body.
	body := strings.TrimRight(d.Body, "\n")
	if body != "" {
		sb.WriteString(body)
		sb.WriteByte('\n')
	}

	return sb.String()
}

// writeRaw writes a `key: value` line with the value emitted verbatim, for the few
// values that are known-safe YAML literals (e.g. the boolean keyword `false`) and
// must NOT pass through the conservative scalar quoter. Callers are responsible for
// the value being a valid bare scalar.
func writeRaw(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s: %s\n", key, value)
}

// writeScalar writes a `key: value` line with a YAML-safe value. Mirrors the
// specs/prompts/workflows renderers' helper of the same name; duplicated rather than
// shared because the packages own independent canonical formats.
func writeScalar(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s: %s\n", key, yamlScalar(value))
}

// yamlScalar renders a string as a YAML scalar, quoting it only when the plain form
// could be misread by a parser. Titles are free human text and can contain ':' and
// other indicator characters, so the quoting is conservative: when in doubt, quote.
// Mirrors the specs/prompts/workflows helper.
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

// needsQuoting reports whether a YAML scalar must be quoted to round-trip safely. It
// is intentionally conservative and mirrors the specs/prompts/workflows helper.
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
