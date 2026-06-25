package architecture

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrDocumentNotFound is returned (wrapped) when a requested slug has no document
// file under the architecture root. Exposed as a sentinel so callers (the CLI/TUI)
// can map "no such document" to a usage error via errors.Is without string matching.
var ErrDocumentNotFound = errors.New("architecture document not found")

// ErrMalformed is the sentinel returned (wrapped) when a document file on disk
// cannot be parsed against the canonical format. It is distinct from
// ErrDocumentNotFound (the file is absent) so a caller can tell "no such document"
// from "this document's file is corrupt".
var ErrMalformed = errors.New("malformed architecture document")

// Load reads a persisted architecture document by slug from archRoot and
// reconstructs the in-memory model. The file's slug component (the `<slug>` of
// `<slug>.md`) is the canonical identity: we trust it over any `slug:` written inside
// the frontmatter, exactly as specs.LoadSpec trusts the file name. So a file renamed
// on disk cannot disagree with its contents.
//
// We hand-roll the parser instead of pulling in a YAML library: go.mod carries none
// and stdlib-first is the rule, and we own this tiny, fixed format. Anything that
// does not match the shape Render writes is rejected as malformed rather than
// silently half-read. The `spec:` provenance key is OPTIONAL — an unlinked document
// simply has no spec reference — so its absence is not an error.
func Load(archRoot, slug string) (Document, error) {
	if !IsKebabCase(slug) {
		return Document{}, fmt.Errorf("architecture slug %q is not valid kebab-case", slug)
	}

	path := filepath.Join(archRoot, fileName(slug))
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Document{}, fmt.Errorf("%w: %q", ErrDocumentNotFound, slug)
		}
		return Document{}, err
	}

	fields, body, err := parseArtifact(string(raw))
	if err != nil {
		return Document{}, fmt.Errorf("%w: document %q: %v", ErrMalformed, slug, err)
	}

	// title is required so a hand-written document without one is flagged rather than
	// loaded with an empty title.
	if strings.TrimSpace(fields[FieldTitle]) == "" {
		return Document{}, fmt.Errorf("%w: document %q: missing required key: %s", ErrMalformed, slug, FieldTitle)
	}

	return Document{
		// The file name is the canonical identity; trust it over the parsed slug.
		Slug:  slug,
		Title: fields[FieldTitle],
		// spec is optional; an absent key yields an empty SpecRef (an unlinked document).
		SpecRef: fields[FieldSpec],
		Body:    body,
	}, nil
}

// parseArtifact turns a document file's bytes into its frontmatter fields and body.
// It accepts exactly the shape Render emits: a `---`-delimited frontmatter block of
// simple `key: value` scalars followed by the verbatim body. The returned body has
// had the single trailing newline the renderer normalizes stripped so an unedited
// load->render round-trips exactly (the renderer re-appends exactly one). The body is
// otherwise preserved verbatim (R6).
func parseArtifact(content string) (fields map[string]string, body string, err error) {
	fm, rawBody, err := splitFrontmatter(content)
	if err != nil {
		return nil, "", err
	}
	fields, err = parseFrontmatter(fm)
	if err != nil {
		return nil, "", fmt.Errorf("frontmatter: %v", err)
	}
	return fields, strings.TrimRight(rawBody, "\n"), nil
}

// splitFrontmatter separates a leading YAML frontmatter block from the body. The
// block opens with `---` on the first line and closes with the next line that is
// exactly `---`; everything after it is the body, returned verbatim. A frontmatter
// that is never closed is malformed. Both LF and CRLF line endings are accepted so
// files authored on Windows load cleanly. Mirrors specs.splitFrontmatter.
func splitFrontmatter(content string) (frontmatter, body string, err error) {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != frontmatterDelim {
		return "", "", fmt.Errorf("expected YAML frontmatter opening %q", frontmatterDelim)
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == frontmatterDelim {
			fm := strings.Join(lines[1:i], "\n")
			b := strings.Join(lines[i+1:], "\n")
			return fm, b, nil
		}
	}
	return "", "", fmt.Errorf("unterminated YAML frontmatter (missing closing %q)", frontmatterDelim)
}

// parseFrontmatter scans a frontmatter block into a flat key->value map. It is a
// minimal, hand-rolled scanner — no YAML dependency, by project rule — strict about
// basic shape (a top-level line must be `key: value`) and about duplicate keys (which
// would make the source ambiguous). A quoted scalar has its quoting removed so the
// in-memory value is bare, exactly reversing the renderer's yamlScalar. Mirrors
// specs.parseFrontmatter.
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

// unquoteScalar reverses yamlScalar (render.go): it strips surrounding double quotes
// and undoes the backslash/quote escaping the renderer applied, leaving a bare scalar
// untouched. It is limited to the escapes the renderer produces (\\ and \") because
// that is the entire surface of values we ever write. Mirrors specs.unquoteScalar.
func unquoteScalar(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	if token[0] != '"' {
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
