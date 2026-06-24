package prompts

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrPromptNotFound is returned (wrapped) when a requested prompt id has no file
// under the prompts root. Exposed as a sentinel so callers (the CLI/TUI) can map
// "no such prompt" to a usage error via errors.Is without string matching (R8).
var ErrPromptNotFound = errors.New("prompt not found")

// ErrMalformedPrompt is the sentinel returned (wrapped) when a prompt file on
// disk cannot be parsed against the canonical format. It is distinct from
// ErrPromptNotFound (the file is absent) so a caller can tell "no such prompt"
// from "this prompt's file is corrupt".
var ErrMalformedPrompt = errors.New("malformed prompt")

// fileName returns the on-disk file name for a prompt id: `<id>.md` (R1). The id
// is assumed to be valid kebab-case (callers validate first), so it is a safe
// single path segment.
func fileName(id string) string {
	return id + FileExt
}

// Load reads a persisted prompt by id from promptsRoot and reconstructs the
// in-memory model. The directory entry's base name (the `<id>` of `<id>.md`) is
// the canonical identity: we trust it over any `id:` written inside the
// frontmatter, exactly as catalog.Load trusts the agent directory name. So a file
// renamed on disk cannot disagree with its contents — Load always returns the
// prompt under the id the caller asked for.
//
// We hand-roll the parser instead of pulling in a YAML library: go.mod carries
// none and stdlib-first is the rule, and we own this tiny, fixed format. Anything
// that does not match the shape Render writes is rejected as malformed rather
// than silently half-read.
func Load(promptsRoot, id string) (Prompt, error) {
	if !IsKebabCase(id) {
		return Prompt{}, fmt.Errorf("prompt id %q is not valid kebab-case", id)
	}

	path := filepath.Join(promptsRoot, fileName(id))
	raw, err := os.ReadFile(path)
	if err != nil {
		// Surface absence as not-found so a caller can distinguish it from a parse
		// failure; other I/O errors propagate as-is.
		if errors.Is(err, os.ErrNotExist) {
			return Prompt{}, fmt.Errorf("%w: %q", ErrPromptNotFound, id)
		}
		return Prompt{}, err
	}

	p, err := parse(string(raw))
	if err != nil {
		return Prompt{}, fmt.Errorf("%w: prompt %q: %v", ErrMalformedPrompt, id, err)
	}

	// The file name is the canonical identity; trust it over the parsed `id:`.
	p.ID = id
	return p, nil
}

// parse turns a prompt file's bytes into a Prompt. It accepts exactly the shape
// Render emits: a `---`-delimited frontmatter block of simple `key: value`
// scalars (id, kind, title, optional description) followed by the verbatim body.
// The `id` parsed here is informational — Load overrides it with the file name —
// but it is still required so a hand-written file without an id is flagged as
// malformed rather than loaded with an empty id.
func parse(content string) (Prompt, error) {
	fm, body, err := splitFrontmatter(content)
	if err != nil {
		return Prompt{}, err
	}

	fields, err := parseFrontmatter(fm)
	if err != nil {
		return Prompt{}, fmt.Errorf("frontmatter: %v", err)
	}

	p := Prompt{
		ID:          fields["id"],
		Kind:        Kind(fields["kind"]),
		Title:       fields["title"],
		Description: fields["description"],
		// Strip the single trailing newline the renderer normalizes onto the body so
		// an unedited load→render round-trips exactly (Render re-appends exactly one).
		// This mirrors catalog.Load's treatment of prompt.md; the body is otherwise
		// preserved verbatim (R7).
		Body: strings.TrimRight(body, "\n"),
	}

	if strings.TrimSpace(p.ID) == "" {
		return Prompt{}, errors.New("missing required key: id")
	}
	if strings.TrimSpace(string(p.Kind)) == "" {
		return Prompt{}, errors.New("missing required key: kind")
	}
	if strings.TrimSpace(p.Title) == "" {
		return Prompt{}, errors.New("missing required key: title")
	}
	return p, nil
}

// splitFrontmatter separates a leading YAML frontmatter block from the body. The
// block opens with `---` on the first line and closes with the next line that is
// exactly `---`; everything after it is the body, returned verbatim (the trailing
// newline normalization happens only on render, never on read, so a load→render
// round-trip is stable). A frontmatter that is never closed is malformed. Both LF
// and CRLF line endings are accepted so files authored on Windows load cleanly.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != frontmatterDelim {
		return "", "", fmt.Errorf("expected YAML frontmatter opening %q", frontmatterDelim)
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == frontmatterDelim {
			fm := strings.Join(lines[1:i], "\n")
			body := strings.Join(lines[i+1:], "\n")
			return fm, body, nil
		}
	}
	return "", "", fmt.Errorf("unterminated YAML frontmatter (missing closing %q)", frontmatterDelim)
}

// parseFrontmatter scans a frontmatter block into a flat key→value map for the
// scalar keys we care about (id, kind, title, description). It is a minimal,
// hand-rolled scanner — no YAML dependency, by project rule — strict about basic
// shape (a top-level line must be `key: value`) and about duplicate keys (which
// would make the source ambiguous). A quoted scalar has its quoting removed so
// the in-memory value is bare, exactly reversing the renderer's yamlScalar.
func parseFrontmatter(fm string) (map[string]string, error) {
	fields := make(map[string]string)
	for _, raw := range strings.Split(fm, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
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
		scalar, err := unquoteScalar(strings.TrimSpace(val))
		if err != nil {
			return nil, fmt.Errorf("%s: %v", key, err)
		}
		fields[key] = scalar
	}
	return fields, nil
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
