package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrManifestNotFound is the sentinel returned (wrapped) when the workspace
// manifest (`.daedalus/daedalus.yaml`) is absent. It is distinct from a parse
// error so a caller can tell "no workspace here" from "the manifest is corrupt"
// and map each to the right outcome (e.g. a build aborts with a different
// message when there is nothing to read vs. when the manifest cannot be parsed).
var ErrManifestNotFound = errors.New("workspace manifest not found")

// ErrManifestMalformed is the sentinel returned (wrapped) when the manifest
// exists but cannot be parsed against the shape this package writes. We reject a
// malformed manifest loudly rather than silently reading partial data, so a
// hand-mangled file fails with an actionable error instead of a wrong build.
var ErrManifestMalformed = errors.New("workspace manifest is malformed")

// ManifestPath returns the canonical path to the manifest under root
// (`<root>/.daedalus/daedalus.yaml`). It is the single place the manifest's
// location is encoded so readers and writers cannot drift.
func ManifestPath(root string) string {
	return filepath.Join(root, Name, RootArtifacts[0])
}

// ReadManifest reads and parses the workspace manifest under root, reconstructing
// the in-memory Manifest. It is the inverse of renderManifest (content.go): it
// reads exactly the shape we emit — the `name`, `version`, `backends` list and
// `conventions` mapping — and rejects anything else as malformed rather than
// half-reading it.
//
// As with the rest of the package we hand-roll the parser (stdlib-first, go.mod
// carries no YAML dependency) because we own the format: it is a tiny, fixed
// shape of scalars, a short string list and a flat mapping, so a purpose-built
// reader is simpler and cannot drift from the writer that lives beside it.
//
// Absence is surfaced as ErrManifestNotFound so a caller can distinguish "no
// workspace" from a parse failure; a shape mismatch is ErrManifestMalformed.
func ReadManifest(root string) (Manifest, error) {
	path := ManifestPath(root)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return Manifest{}, fmt.Errorf("%w: %s", ErrManifestNotFound, filepath.ToSlash(path))
		}
		return Manifest{}, err
	}
	m, err := parseManifest(string(b))
	if err != nil {
		return Manifest{}, fmt.Errorf("%w: %s: %v", ErrManifestMalformed, filepath.ToSlash(path), err)
	}
	return m, nil
}

// parseManifest parses the manifest body into a Manifest. It accepts the exact
// shape renderManifest emits: top-level `name`/`version` scalars, a `backends:`
// block of `- value` items, and a `conventions:` block of indented `key: value`
// pairs. Comment and blank lines are ignored. Anything that does not match the
// shape is an error so a corrupt file fails loudly.
func parseManifest(body string) (Manifest, error) {
	var m Manifest
	var sawName, sawVersion bool

	// section tracks which block we are inside so an indented line is attributed
	// to the right parent key (backends vs conventions). "" means top level.
	section := ""

	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}

		indented := line[0] == ' ' || line[0] == '\t'
		if indented {
			switch section {
			case "backends":
				item := strings.TrimSpace(line)
				if !strings.HasPrefix(item, "- ") {
					return Manifest{}, fmt.Errorf("backends: expected list item, got %q", line)
				}
				val, err := unquoteManifestScalar(strings.TrimSpace(item[2:]))
				if err != nil {
					return Manifest{}, fmt.Errorf("backends: %v", err)
				}
				m.Backends = append(m.Backends, val)
			case "conventions":
				key, val, ok := strings.Cut(strings.TrimSpace(line), ":")
				if !ok {
					return Manifest{}, fmt.Errorf("conventions: line is not key: value: %q", line)
				}
				v, err := unquoteManifestScalar(strings.TrimSpace(val))
				if err != nil {
					return Manifest{}, fmt.Errorf("conventions[%s]: %v", strings.TrimSpace(key), err)
				}
				m.Conventions = append(m.Conventions, convention{Key: strings.TrimSpace(key), Value: v})
			default:
				return Manifest{}, fmt.Errorf("unexpected indented line outside a block: %q", line)
			}
			continue
		}

		// A non-indented line is a top-level key; it also closes any open block.
		section = ""
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			return Manifest{}, fmt.Errorf("line is not key: value: %q", line)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		switch key {
		case "name":
			s, err := unquoteManifestScalar(val)
			if err != nil {
				return Manifest{}, fmt.Errorf("name: %v", err)
			}
			m.Name = s
			sawName = true
		case "version":
			s, err := unquoteManifestScalar(val)
			if err != nil {
				return Manifest{}, fmt.Errorf("version: %v", err)
			}
			m.Version = s
			sawVersion = true
		case "backends":
			if val != "" {
				return Manifest{}, fmt.Errorf("backends: expected a block, got inline %q", val)
			}
			section = "backends"
		case "conventions":
			if val != "" {
				return Manifest{}, fmt.Errorf("conventions: expected a block, got inline %q", val)
			}
			section = "conventions"
		default:
			return Manifest{}, fmt.Errorf("unknown top-level key %q", key)
		}
	}

	if !sawName {
		return Manifest{}, errors.New("missing required key: name")
	}
	if !sawVersion {
		return Manifest{}, errors.New("missing required key: version")
	}
	if len(m.Backends) == 0 {
		return Manifest{}, errors.New("missing required key: backends (at least one)")
	}
	return m, nil
}

// unquoteManifestScalar reverses yamlScalar (content.go): it strips surrounding
// double quotes and undoes the backslash/quote escaping the renderer applied,
// leaving a bare scalar untouched. It is intentionally limited to the escapes the
// renderer produces (\\ and \") because that is the entire surface of values we
// ever write; an unterminated or unexpectedly-escaped scalar is malformed.
func unquoteManifestScalar(token string) (string, error) {
	if token == "" {
		return "", errors.New("empty value")
	}
	if token[0] != '"' {
		// Bare scalar: emitted only when needsQuoting judged it safe, so it carries
		// no escaping to undo.
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
