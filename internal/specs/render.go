package specs

import (
	"fmt"
	"strings"
)

// On-disk format of a brief and a spec (R1/R3/R6/R8).
//
// Both artifacts are single Markdown files made of a YAML frontmatter block
// delimited by `---` lines, followed by the body verbatim. The frontmatter carries
// the stable metadata; the body is the human-authored / human-refined Markdown.
//
// A brief `.daedalus/specs/<slug>.brief.md`:
//
//	---
//	slug: my-feature
//	kind: brief
//	title: My Feature
//	consumed-by: analyst
//	workflow: sdd-default
//	phase: spec
//	---
//	<brief markdown, verbatim>
//
// A spec `.daedalus/specs/<slug>.md`:
//
//	---
//	slug: my-feature
//	kind: spec
//	title: My Feature
//	brief: my-feature.brief.md
//	agent: analyst
//	workflow: sdd-default
//	phase: spec
//	generated: false
//	---
//	<spec markdown, verbatim>
//
// # Why these keys, and why this order (R2/R5/R8)
//
// The order is FIXED and chosen for the reader: identity first (slug, kind, title),
// then the provenance — the link Daedalus manages in phase 1. For a brief the
// provenance is "who consumes me, in which workflow, at which phase" (the analyst
// step of sdd-default), which is the R2/CA2 link. For a spec it is "which brief I
// came from (the R8/CA7 trace), produced by which agent, in which workflow/phase,
// and whether Daedalus generated my body". `generated` is always `false` in phase 1
// and is written explicitly to make R5/CA5 self-evident in the file itself: Daedalus
// seeded the placeholder; the user (running the analyst on their backend) produces
// the real content. Every key is always present so the shape is stable and a diff
// never has to distinguish "absent" from "empty".
//
// The renderer is hand-rolled and stdlib-only — go.mod carries no YAML dependency
// and stdlib-first is the project rule — and is duplicated from (not shared with)
// the prompts/workflows renderers so specs owns its own format. Output always ends
// with a single trailing newline so the file is POSIX-clean.

const (
	// frontmatterDelim is the line that opens and closes the YAML frontmatter.
	frontmatterDelim = "---"

	// kindBrief and kindSpec are the `kind:` discriminators written into each
	// artifact's frontmatter so a reader (and the parser) can tell the two apart
	// even out of context.
	kindBrief = "brief"
	kindSpec  = "spec"
)

// Frontmatter key names, exported as constants so the parser, the validator and
// tests refer to fields by a stable identifier instead of hardcoding strings, and
// so the names a user sees in an error match the serialized form exactly.
const (
	FieldSlug  = "slug"
	FieldKind  = "kind"
	FieldTitle = "title"
	// Brief provenance keys.
	FieldConsumedBy = "consumed-by"
	// Spec provenance keys.
	FieldBrief     = "brief"
	FieldAgent     = "agent"
	FieldGenerated = "generated"
	// Shared provenance keys (both artifacts anchor to the same workflow phase).
	FieldWorkflow = "workflow"
	FieldPhase    = "phase"
)

// RenderBrief serializes a brief to its canonical on-disk bytes. It assumes the
// brief is already valid (callers validate before rendering); it does not validate
// here so it stays a pure formatting function. The provenance keys are constant —
// every brief is consumed by the analyst step of the default workflow (R2/CA2) — so
// they are written from the package constants, not from per-brief fields. Output is
// byte-stable for a given Brief (R6).
func RenderBrief(b Brief) string {
	var sb strings.Builder

	sb.WriteString(frontmatterDelim)
	sb.WriteByte('\n')

	// Identity, then the analyst-step provenance (the R2/CA2 link).
	writeScalar(&sb, FieldSlug, b.Slug)
	writeScalar(&sb, FieldKind, kindBrief)
	writeScalar(&sb, FieldTitle, b.Title)
	writeScalar(&sb, FieldConsumedBy, AnalystAgent)
	writeScalar(&sb, FieldWorkflow, DefaultWorkflowName)
	writeScalar(&sb, FieldPhase, DefaultPhase)

	sb.WriteString(frontmatterDelim)
	sb.WriteByte('\n')

	writeBody(&sb, b.Body)
	return sb.String()
}

// RenderSpec serializes a spec to its canonical on-disk bytes. Like RenderBrief it
// is a pure formatting function. The `brief:` key carries the R8/CA7 trace to the
// originating brief; `generated: false` records that Daedalus did not run the agent
// (R5/CA5). Output is byte-stable for a given Spec (R6).
func RenderSpec(s Spec) string {
	var sb strings.Builder

	sb.WriteString(frontmatterDelim)
	sb.WriteByte('\n')

	// Identity, then the brief trace, then the producing-step provenance.
	writeScalar(&sb, FieldSlug, s.Slug)
	writeScalar(&sb, FieldKind, kindSpec)
	writeScalar(&sb, FieldTitle, s.Title)
	writeScalar(&sb, FieldBrief, s.BriefRef)
	writeScalar(&sb, FieldAgent, AnalystAgent)
	writeScalar(&sb, FieldWorkflow, DefaultWorkflowName)
	writeScalar(&sb, FieldPhase, DefaultPhase)
	// Phase 1: Daedalus never generates the body, so this is always false. It is a
	// real YAML boolean (written raw, not via yamlScalar, which would conservatively
	// quote the keyword "false" into the string "false"). A genuine boolean makes the
	// R5/CA5 fact machine-checkable, not just a coincidentally-named string.
	writeRaw(&sb, FieldGenerated, "false")

	sb.WriteString(frontmatterDelim)
	sb.WriteByte('\n')

	writeBody(&sb, s.Body)
	return sb.String()
}

// writeBody appends the verbatim body, normalizing only the trailing newline so the
// file is byte-stable regardless of how the body was authored: strip any trailing
// newlines and re-append exactly one, which also guarantees the file ends cleanly
// even for an empty body. The body is otherwise persisted verbatim (R6).
func writeBody(b *strings.Builder, body string) {
	trimmed := strings.TrimRight(body, "\n")
	if trimmed != "" {
		b.WriteString(trimmed)
		b.WriteByte('\n')
	}
}

// writeRaw writes a `key: value` line with the value emitted verbatim, for the few
// values that are known-safe YAML literals (e.g. the boolean keyword `false`) and
// must NOT pass through the conservative scalar quoter. Callers are responsible for
// the value being a valid bare scalar.
func writeRaw(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s: %s\n", key, value)
}

// writeScalar writes a `key: value` line with a YAML-safe value. Mirrors the
// prompts/workflows renderers' helper of the same name; duplicated rather than
// shared because the packages own independent canonical formats.
func writeScalar(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s: %s\n", key, yamlScalar(value))
}

// yamlScalar renders a string as a YAML scalar, quoting it only when the plain form
// could be misread by a parser. Titles are free human text and can contain ':' and
// other indicator characters, so the quoting is conservative: when in doubt, quote.
// Mirrors the prompts/workflows helper.
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
// is intentionally conservative and mirrors the prompts/workflows helper.
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
