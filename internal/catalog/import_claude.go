package catalog

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Claude Code → canonical mapping (R2/CA2).
//
// A Claude Code agent (a `.claude/agents/*.md` file) is Markdown with a leading
// YAML frontmatter block delimited by `---` lines, followed by the prompt body.
// Its documented frontmatter keys are: name, description, tools, model, color.
// We map them to the backend-agnostic canonical model as follows, and document
// what we deliberately drop:
//
//   - name        -> id          (normalized to kebab-case via NormalizeID; if
//                                  absent, derived from the file base name)
//   - description -> role        (the human-facing role/description)
//   - <body>      -> prompt      (everything after the closing `---`)
//   - model       -> parameter "model" (string)   [only if present]
//   - tools       -> DROPPED     (backend-specific: the set of tools a Claude Code
//                                  agent may call. There is no Phase-1 canonical
//                                  concept for it; tool wiring is a compile-time
//                                  concern of the backend adapter, epic-06. We do
//                                  not invent a canonical parameter the PRD does
//                                  not define.)
//   - color       -> DROPPED     (purely a Claude Code UI affordance; no canonical
//                                  meaning and no backend-agnostic purpose.)
//
// Dropping tools/color is the honest, simplest choice: preserving them as opaque
// parameters would fabricate semantics Daedalus cannot yet honor and would muddy
// the canonical definition. When the Claude adapter (epic-06) compiles canonical
// agents back to `.claude/`, it will derive tools from the agent's role/config
// then; round-tripping those fields is out of scope here (R8: import only
// transforms, it does not execute or fully model the backend).

// fromClaudeCode converts a Claude Code agent file (frontmatter + body) into a
// canonical Agent. It is tolerant of the frontmatter keys it does not use and
// strict about the ones it does: a missing/empty description or empty body yields
// an actionable error (surfaced via the agent later failing Validate, or here for
// id derivation). path is used to derive the id when `name` is absent and to make
// error messages name the source file.
func fromClaudeCode(content, path string) (Agent, error) {
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return Agent{}, fmt.Errorf("%s: %v", filepath.Base(path), err)
	}

	fields, err := parseFrontmatter(fm)
	if err != nil {
		return Agent{}, fmt.Errorf("%s: frontmatter: %v", filepath.Base(path), err)
	}

	// id: from `name`, falling back to the file base name without extension. Either
	// way it is normalized to kebab-case; a name that cannot be normalized is an
	// actionable error rather than a fabricated id (R6/CA5).
	rawID := fields["name"]
	if strings.TrimSpace(rawID) == "" {
		rawID = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	id, err := NormalizeID(rawID)
	if err != nil {
		return Agent{}, fmt.Errorf("%s: %v", filepath.Base(path), err)
	}

	agent := Agent{
		ID:     id,
		Role:   strings.TrimSpace(fields["description"]),
		Prompt: strings.TrimSpace(body),
	}

	// model -> a single string parameter, only when present and non-empty. We do
	// not synthesize a default; an absent model simply means no parameter, which
	// the canonical model represents as an empty `parameters` block.
	if model := strings.TrimSpace(fields["model"]); model != "" {
		agent.Params = append(agent.Params, Param{Key: "model", Type: ParamString, Value: model})
	}

	return agent, nil
}

// splitFrontmatter separates a leading YAML frontmatter block from the body. The
// block opens with `---` on the first line and closes with the next line that is
// exactly `---`; everything after it is the body. A frontmatter that is never
// closed is an error (a truncated/corrupt file), so we never silently treat the
// whole document as frontmatter or as body. Both LF and CRLF line endings are
// accepted so files authored on Windows import cleanly.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	// Normalize CRLF to LF for scanning; the body is re-derived from the original
	// offsets so we do not alter the prompt's own line endings beyond this.
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return "", "", fmt.Errorf("expected YAML frontmatter opening '---'")
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			fm := strings.Join(lines[1:i], "\n")
			bodyLines := lines[i+1:]
			return fm, strings.Join(bodyLines, "\n"), nil
		}
	}
	return "", "", fmt.Errorf("unterminated YAML frontmatter (missing closing '---')")
}

// parseFrontmatter scans a YAML frontmatter block into a flat key→value map for
// the simple scalar keys we care about (name, description, model, and others as
// raw strings). It is a minimal, hand-rolled scanner — no YAML dependency, by
// project rule — tolerant of keys it does not use: every `key: value` line is
// captured, list values (e.g. `tools: a, b`) are captured as their raw text since
// we do not consume them, and indented continuation/list-item lines are skipped.
// It is strict only about basic shape (a top-level line must be `key: value`);
// a duplicate key is an error because it makes the source ambiguous.
func parseFrontmatter(fm string) (map[string]string, error) {
	fields := make(map[string]string)
	for _, raw := range strings.Split(fm, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		// Indented lines are continuations or block/list items of a key we do not
		// consume (e.g. a multi-line `tools:` list); skip them rather than mistake
		// them for top-level keys.
		if line[0] == ' ' || line[0] == '\t' || strings.HasPrefix(strings.TrimSpace(line), "- ") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("line is not key: value: %q", line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, fmt.Errorf("empty key in line %q", line)
		}
		if _, dup := fields[key]; dup {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		fields[key] = stripFrontmatterScalar(strings.TrimSpace(val))
	}
	return fields, nil
}

// stripFrontmatterScalar removes the optional surrounding quotes a YAML scalar may
// carry. Claude Code frontmatter values are plain text (often unquoted), but a
// value may be quoted to protect a leading special character; we strip a matching
// pair of single or double quotes so the canonical model stores the bare value.
// We do not interpret escape sequences: these are short human-authored labels, and
// the renderer re-quotes on output as needed, so a literal backslash is preserved.
func stripFrontmatterScalar(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// fromCanonicalFile converts a single-file canonical definition into an Agent.
// This supports importing a canonical agent that lives as one file (the inline
// form) rather than the materialized directory pair that Load reads. It reuses the
// same definition parser as Load for the metadata; the prompt is taken from an
// inline body if present after a `---` separator, otherwise from the `prompt:`
// pointer's target is not resolved here (a bare canonical file with no inline body
// is reported as needing its prompt, which Validate then flags as empty).
//
// In practice the common canonical import is the materialized directory, handled
// by ImportPlanFor delegating per-file; this single-file path exists so a hand-
// written canonical `*.yaml`/`*.md` is also importable (R3).
func fromCanonicalFile(content, path string) (Agent, error) {
	// A canonical single file may carry the prompt inline after a `---` separator
	// (definition block, then body). Split on the first such separator if present.
	defBlock, body := splitCanonicalInline(content)

	agent, err := parseDefinition(defBlock)
	if err != nil {
		return Agent{}, fmt.Errorf("%s: %v", filepath.Base(path), err)
	}
	// The id derived from the directory name is not available for a loose file, so
	// trust the `id:` key here; normalize it defensively so a slightly-off id
	// (e.g. stray case) still imports under a valid kebab-case name (R6/CA5).
	id, err := NormalizeID(agent.ID)
	if err != nil {
		return Agent{}, fmt.Errorf("%s: %v", filepath.Base(path), err)
	}
	agent.ID = id
	agent.Prompt = strings.TrimSpace(body)
	return agent, nil
}

// splitCanonicalInline splits a single-file canonical definition into its
// definition block and an optional inline prompt body. The convention is the same
// `---` separator used by frontmatter, but here the *first* line is a key (not
// `---`): the definition comes first, then an optional `---` then the body. If
// there is no separator, the whole content is the definition block and the body is
// empty (the agent's prompt then comes from validation as "missing").
func splitCanonicalInline(content string) (defBlock, body string) {
	lines := strings.Split(content, "\n")
	for i, raw := range lines {
		if i == 0 {
			continue // the first line is a definition key, never the separator
		}
		if strings.TrimRight(raw, "\r") == "---" {
			return strings.Join(lines[:i], "\n"), strings.Join(lines[i+1:], "\n")
		}
	}
	return content, ""
}
