package prompts

import (
	"fmt"
	"strings"
)

// On-disk format of a persisted prompt (R1/R2/R5).
//
// A prompt is a single Markdown file `.daedalus/prompts/<id>.md` made of a YAML
// frontmatter block delimited by `---` lines, followed by the body verbatim:
//
//	---
//	id: my-prompt
//	kind: global
//	title: My Prompt
//	description: optional one-liner
//	---
//	<body markdown, verbatim>
//
// Key order is FIXED and deterministic: id, then kind, then title, then
// description. The order is chosen for the reader (identity first, then class,
// then the human label) and is what guarantees the same Prompt always renders
// byte-identical bytes (R5). `description` is emitted ONLY when it is non-empty:
// an absent description is meaningful ("no summary") and emitting `description:`
// with an empty value would be noise in the diff and ambiguous to a reader, so we
// omit the key entirely rather than write an empty scalar. Every other key is
// always present so the shape is stable.
//
// The renderer is hand-rolled and stdlib-only — go.mod carries no YAML
// dependency and stdlib-first is the project rule — and is duplicated from (not
// shared with) the catalog renderer so prompts owns its own format. Output always
// ends with a single trailing newline so the file is POSIX-clean.

const (
	// frontmatterDelim is the line that opens and closes the YAML frontmatter.
	frontmatterDelim = "---"
)

// Render serializes a prompt to its canonical on-disk bytes. It assumes the
// prompt is already valid (callers validate before rendering); it does not
// validate here so it stays a pure formatting function. Output is byte-stable
// for a given Prompt (R5).
func Render(p Prompt) string {
	var b strings.Builder

	b.WriteString(frontmatterDelim)
	b.WriteByte('\n')

	// Fixed key order. id/kind/title are always present; description only when set.
	writeScalar(&b, FieldID, p.ID)
	writeScalar(&b, FieldKind, string(p.Kind))
	writeScalar(&b, FieldTitle, p.Title)
	if !trimmedEmpty(p.Description) {
		writeScalar(&b, "description", p.Description)
	}

	b.WriteString(frontmatterDelim)
	b.WriteByte('\n')

	// The body is persisted verbatim (R7). We normalize only the trailing newline
	// so the file is byte-stable regardless of how the body was authored: strip any
	// trailing newlines and re-append exactly one, which also guarantees the file
	// ends cleanly even for an empty body.
	body := strings.TrimRight(p.Body, "\n")
	if body != "" {
		b.WriteString(body)
		b.WriteByte('\n')
	}

	return b.String()
}

// writeScalar writes a `key: value` line with a YAML-safe value. Mirrors the
// catalog renderer's helper of the same name; duplicated rather than shared
// because the two packages own independent canonical formats.
func writeScalar(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s: %s\n", key, yamlScalar(value))
}

// yamlScalar renders a string as a YAML scalar, quoting it only when the plain
// form could be misread by a parser. Titles and descriptions are free human text
// and can contain ':' and other indicator characters, so the quoting is
// conservative: when in doubt, quote.
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
// It is intentionally conservative. Mirrors the catalog renderer's helper.
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
