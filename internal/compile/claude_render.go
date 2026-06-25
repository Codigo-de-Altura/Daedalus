package compile

import (
	"fmt"
	"strings"

	"github.com/Codigo-de-Altura/Daedalus/internal/buildinfo"
	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
)

// Claude Code native layout (RF-6.2). These are the paths the adapter writes,
// relative to the target repository root and always in slash form so the output
// is identical on every OS (RNF-5). They mirror PRD §8.1's example layout.
const (
	// claudeAgentsDir holds one Markdown file per canonical agent.
	claudeAgentsDir = ".claude/agents"
	// claudeCommandsDir holds one Markdown file per canonical command (prompt).
	claudeCommandsDir = ".claude/commands"
	// claudeSettingsPath is Claude Code's project settings file.
	claudeSettingsPath = ".claude/settings.json"
	// mdExt is the extension of every agent/command artifact.
	mdExt = ".md"
)

// claudeSettingsSchema is the official Claude Code settings JSON Schema URL. It is
// emitted as the `$schema` key so an editor validates the generated settings and
// a reader knows exactly which contract the file follows.
const claudeSettingsSchema = "https://json.schemastore.org/claude-code-settings.json"

// agentParamModel is the canonical parameter key whose value becomes a Claude
// Code agent's `model`. It mirrors catalog/import_claude.go, which maps a Claude
// Code `model` frontmatter key to exactly this canonical parameter on import, so
// the build is the precise inverse and the round-trip is coherent.
const agentParamModel = "model"

// renderAgent renders one canonical agent as a Claude Code `.claude/agents/<id>.md`
// file: a YAML frontmatter block followed by the prompt body.
//
// The frontmatter key order is FIXED and deterministic (RNF-5): name, then
// description, then model. It is the inverse of catalog/import_claude.go's
// Claude-Code → canonical mapping, so an agent built here re-imports to the same
// canonical agent:
//
//   - name        <- id          (the canonical kebab-case id)
//   - description <- role
//   - model       <- parameter "model"   (emitted ONLY when present; an absent
//     model means no key, exactly as import
//     synthesizes no parameter for an absent
//     model)
//
// We deliberately do NOT emit `tools` or `color`: the canonical model has no
// concept for them (import drops them), so synthesizing them here would fabricate
// configuration the user never defined and break the round-trip. The body is the
// agent's prompt, terminated by exactly one trailing newline so the file is
// POSIX-clean and byte-stable.
func renderAgent(a catalog.Agent) string {
	var b strings.Builder

	b.WriteString("---\n")
	writeFrontmatterScalar(&b, "name", a.ID)
	writeFrontmatterScalar(&b, "description", a.Role)
	if model := agentModel(a); model != "" {
		writeFrontmatterScalar(&b, "model", model)
	}
	b.WriteString("---\n")

	// The prompt is emitted verbatim with a single trailing newline. catalog.Load
	// already stripped the prompt's trailing newlines, so re-appending exactly one
	// keeps the file stable regardless of how the prompt was authored.
	body := strings.TrimRight(a.Prompt, "\n")
	if body != "" {
		b.WriteString(body)
		b.WriteByte('\n')
	}
	return b.String()
}

// agentModel returns the agent's "model" parameter value, or "" when it has none.
// It reads the canonical parameter the import side writes, so the build emits a
// `model` frontmatter key exactly when (and with the value) the agent carries one.
func agentModel(a catalog.Agent) string {
	for _, p := range a.Params {
		if p.Key == agentParamModel {
			return strings.TrimSpace(p.Value)
		}
	}
	return ""
}

// renderCommand renders one canonical command as a Claude Code
// `.claude/commands/<id>.md` slash-command file: an optional frontmatter block
// (just a `description` when the command has one) followed by the composed body.
//
// A Claude Code slash command is Markdown whose body becomes the command's
// prompt; the optional `description` frontmatter key gives the command a
// human-facing summary. We omit the frontmatter block entirely when there is no
// description so a description-less command is a clean Markdown file rather than
// an empty `---\n---\n` header — deterministic either way. The body is the
// command's already-composed text (inclusions expanded at load time), terminated
// by exactly one trailing newline.
func renderCommand(c Command) string {
	var b strings.Builder

	if strings.TrimSpace(c.Description) != "" {
		b.WriteString("---\n")
		writeFrontmatterScalar(&b, "description", c.Description)
		b.WriteString("---\n")
	}

	body := strings.TrimRight(c.Body, "\n")
	if body != "" {
		b.WriteString(body)
		b.WriteByte('\n')
	}
	return b.String()
}

// renderSettings renders Claude Code's `.claude/settings.json` as a minimal,
// honest, deterministic document (REQ-7). It declares ONLY what is true: the
// settings schema it follows and a clear marker that Daedalus generates and
// manages these `.claude/` artifacts. It fabricates no permissions, env, hooks or
// model — those are the user's to define, and inventing them would contradict the
// non-destructive philosophy of the epic.
//
// The marker lives under a single namespaced object key ("daedalus") so it never
// collides with a real Claude Code setting and a reader (or ticket-06-03's
// managed-area logic) can recognize the file as Daedalus-managed. The JSON is
// hand-rendered with a fixed key order, two-space indentation and a trailing
// newline so the bytes are stable for a given binary (RNF-5).
//
// NOTE on the schema: settings.schemastore.org's Claude Code schema does not
// forbid additional top-level properties, so the namespaced marker keeps the file
// valid. If a future schema revision were to set additionalProperties:false at
// the root, the marker would need to move (or be dropped to a `$schema`-only
// file); this is called out so the choice is auditable.
func renderSettings() string {
	// The marker records only that these artifacts are Daedalus-generated and names
	// the generator. We deliberately do NOT stamp the binary version here: a version
	// would churn the file on every release and break byte-for-byte determinism
	// across builds of different binaries over the same workspace (RNF-5). The
	// generator name is the constant buildinfo.Name, which is stable across releases.
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString("  \"$schema\": " + jsonString(claudeSettingsSchema) + ",\n")
	b.WriteString("  \"daedalus\": {\n")
	b.WriteString("    \"managed\": true,\n")
	b.WriteString("    \"generator\": " + jsonString(buildinfo.Name) + "\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
	return b.String()
}

// writeFrontmatterScalar writes a `key: value` YAML line, quoting the value only
// when the bare form could be misread by a parser. It mirrors the conservative
// quoting of the prompts/catalog renderers so the frontmatter this adapter emits
// is parseable by the same hand-rolled scanners (catalog/import_claude.go) on the
// round trip.
func writeFrontmatterScalar(b *strings.Builder, key, value string) {
	fmt.Fprintf(b, "%s: %s\n", key, yamlFrontmatterScalar(value))
}

// yamlFrontmatterScalar renders a string as a YAML scalar, quoting conservatively.
// Mirrors prompts.yamlScalar / workspace.yamlScalar; duplicated (not shared)
// because this package owns the adapter's output format and must be free to evolve
// it independently of the canonical renderers.
func yamlFrontmatterScalar(s string) string {
	if s == "" {
		return `""`
	}
	if frontmatterNeedsQuoting(s) {
		escaped := strings.ReplaceAll(s, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return `"` + escaped + `"`
	}
	return s
}

// frontmatterNeedsQuoting reports whether a YAML scalar must be quoted to
// round-trip safely. Intentionally conservative: when in doubt, quote. Mirrors the
// canonical renderers' rule so the two never disagree about a given value.
func frontmatterNeedsQuoting(s string) bool {
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

// jsonString renders s as a JSON string literal with the minimal escaping the JSON
// spec requires. We hand-render (rather than encoding/json) so the whole settings
// document has a guaranteed key order and indentation; the values we emit are
// simple ASCII identifiers/URLs, but we still escape the structural characters so
// the output is valid JSON for any value.
func jsonString(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(&b, `\u%04x`, r)
				continue
			}
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}
