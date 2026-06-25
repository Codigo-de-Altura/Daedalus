package backlog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrEpicNotFound is returned (wrapped) when a requested epic id has no folder/file
// under the epics root. Exposed as a sentinel so callers map "no such epic" to a usage
// error via errors.Is without string matching.
var ErrEpicNotFound = errors.New("epic not found")

// ErrTicketNotFound is returned (wrapped) when a requested ticket id has no folder/file
// under its epic. Distinct from ErrEpicNotFound so a caller can tell a missing epic from
// a missing ticket within an existing epic.
var ErrTicketNotFound = errors.New("ticket not found")

// ErrMalformed is the sentinel returned (wrapped) when an epic/ticket file on disk
// cannot be parsed against the canonical format. Distinct from the not-found sentinels
// so a caller can tell "no such artifact" from "this artifact's file is corrupt".
var ErrMalformed = errors.New("malformed backlog artifact")

// epicDir returns the absolute folder of an epic: `<epicsRoot>/<epicID>`.
func epicDir(epicsRoot, epicID string) string {
	return filepath.Join(epicsRoot, epicID)
}

// epicPath returns the absolute path of an epic's markdown file.
func epicPath(epicsRoot, epicID string) string {
	return filepath.Join(epicDir(epicsRoot, epicID), EpicFile)
}

// ticketDir returns the absolute folder of a ticket, nested under its epic:
// `<epicsRoot>/<epicID>/tickets/<ticketID>`.
func ticketDir(epicsRoot, epicID, ticketID string) string {
	return filepath.Join(epicDir(epicsRoot, epicID), TicketsSubdir, ticketID)
}

// ticketPath returns the absolute path of a ticket's markdown file.
func ticketPath(epicsRoot, epicID, ticketID string) string {
	return filepath.Join(ticketDir(epicsRoot, epicID, ticketID), TicketFile)
}

// LoadEpic reads a persisted epic by id from epicsRoot and reconstructs the model. The
// folder name is the canonical identity: we trust it over any `id:` inside the
// frontmatter, as the sibling packages trust the file name. We hand-roll the parser
// (no YAML dependency, stdlib-first). Anything that does not match the shape RenderEpic
// writes is rejected as malformed.
func LoadEpic(epicsRoot, epicID string) (Epic, error) {
	if !IsEpicID(epicID) {
		return Epic{}, fmt.Errorf("epic id %q is not a valid epic-NN-<slug> id", epicID)
	}

	raw, err := os.ReadFile(epicPath(epicsRoot, epicID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Epic{}, fmt.Errorf("%w: %q", ErrEpicNotFound, epicID)
		}
		return Epic{}, err
	}

	fields, lists, body, err := parseArtifact(string(raw))
	if err != nil {
		return Epic{}, fmt.Errorf("%w: epic %q: %v", ErrMalformed, epicID, err)
	}
	if strings.TrimSpace(fields[FieldTitle]) == "" {
		return Epic{}, fmt.Errorf("%w: epic %q: missing required key: %s", ErrMalformed, epicID, FieldTitle)
	}

	return Epic{
		// The folder name is the canonical identity; trust it over the parsed id.
		ID:              epicID,
		Title:           fields[FieldTitle],
		Status:          Status(fields[FieldStatus]),
		Priority:        Priority(fields[FieldPriority]),
		SpecRef:         fields[FieldSpec],
		ArchitectureRef: fields[FieldArchitecture],
		DependsOn:       lists[FieldDependsOn],
		Body:            body,
	}, nil
}

// LoadTicket reads a persisted ticket by id under its parent epic from epicsRoot and
// reconstructs the model. Both the epic id and the ticket id are the canonical
// identities (the nested folder names). The `epic` provenance key is required (it is
// the mandatory R5/CA5 parent link); a ticket file missing it is malformed.
func LoadTicket(epicsRoot, epicID, ticketID string) (Ticket, error) {
	if !IsEpicID(epicID) {
		return Ticket{}, fmt.Errorf("epic id %q is not a valid epic-NN-<slug> id", epicID)
	}
	if !IsTicketID(ticketID) {
		return Ticket{}, fmt.Errorf("ticket id %q is not a valid ticket-NN-MM-<slug> id", ticketID)
	}

	raw, err := os.ReadFile(ticketPath(epicsRoot, epicID, ticketID))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Ticket{}, fmt.Errorf("%w: %q", ErrTicketNotFound, ticketID)
		}
		return Ticket{}, err
	}

	fields, lists, body, err := parseArtifact(string(raw))
	if err != nil {
		return Ticket{}, fmt.Errorf("%w: ticket %q: %v", ErrMalformed, ticketID, err)
	}
	if strings.TrimSpace(fields[FieldTitle]) == "" {
		return Ticket{}, fmt.Errorf("%w: ticket %q: missing required key: %s", ErrMalformed, ticketID, FieldTitle)
	}
	if strings.TrimSpace(fields[FieldEpic]) == "" {
		return Ticket{}, fmt.Errorf("%w: ticket %q: missing required key: %s", ErrMalformed, ticketID, FieldEpic)
	}

	return Ticket{
		ID:              ticketID,
		EpicID:          fields[FieldEpic],
		Title:           fields[FieldTitle],
		Status:          Status(fields[FieldStatus]),
		Priority:        Priority(fields[FieldPriority]),
		SpecRef:         fields[FieldSpec],
		ArchitectureRef: fields[FieldArchitecture],
		DependsOn:       lists[FieldDependsOn],
		Body:            body,
	}, nil
}

// parseArtifact turns an epic/ticket file's bytes into its scalar fields, its
// list-valued fields, and the verbatim body. It accepts exactly the shape the renderers
// emit: a `---`-delimited frontmatter block of `key: value` scalars and `key: [a, b]`
// flow lists, followed by the body. The returned body has the single trailing newline
// the renderer normalizes stripped so an unedited load->render round-trips exactly.
func parseArtifact(content string) (fields map[string]string, lists map[string][]string, body string, err error) {
	fm, rawBody, err := splitFrontmatter(content)
	if err != nil {
		return nil, nil, "", err
	}
	fields, lists, err = parseFrontmatter(fm)
	if err != nil {
		return nil, nil, "", fmt.Errorf("frontmatter: %v", err)
	}
	return fields, lists, strings.TrimRight(rawBody, "\n"), nil
}

// splitFrontmatter separates a leading YAML frontmatter block from the body. The block
// opens with `---` on the first line and closes with the next exact `---`; everything
// after is the body, verbatim. An unterminated block is malformed. Both LF and CRLF are
// accepted. Mirrors the sibling helper.
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

// parseFrontmatter scans a frontmatter block into a flat scalar map and a list map. A
// top-level line must be `key: value`; a value that is a `[...]` flow sequence is parsed
// into the list map, everything else into the scalar map. It is strict about duplicate
// keys (ambiguous source). A quoted scalar has its quoting removed, reversing the
// renderer's yamlScalar.
func parseFrontmatter(fm string) (map[string]string, map[string][]string, error) {
	fields := make(map[string]string)
	lists := make(map[string][]string)
	for _, raw := range strings.Split(fm, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			return nil, nil, fmt.Errorf("line is not key: value: %q", line)
		}
		key = strings.TrimSpace(key)
		if key == "" {
			return nil, nil, fmt.Errorf("empty key in line %q", line)
		}
		if _, dup := fields[key]; dup {
			return nil, nil, fmt.Errorf("duplicate key %q", key)
		}
		if _, dup := lists[key]; dup {
			return nil, nil, fmt.Errorf("duplicate key %q", key)
		}

		val = strings.TrimSpace(val)
		if isFlowList(val) {
			items, err := parseFlowList(val)
			if err != nil {
				return nil, nil, fmt.Errorf("%s: %v", key, err)
			}
			lists[key] = items
			continue
		}

		scalar, err := unquoteScalar(val)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %v", key, err)
		}
		fields[key] = scalar
	}
	return fields, lists, nil
}

// isFlowList reports whether a frontmatter value is a YAML flow sequence `[...]`.
func isFlowList(val string) bool {
	return strings.HasPrefix(val, "[") && strings.HasSuffix(val, "]")
}

// parseFlowList parses a `[a, b, c]` flow sequence into its elements, reversing the
// renderer's writeList. An empty `[]` yields nil. Each element is unquoted exactly as a
// scalar would be. It is intentionally simple — it handles the bytes the renderer
// produces (comma-separated, each element a bare or double-quoted scalar) — and rejects
// a quoted element that spans a comma boundary only insofar as our renderer never emits
// commas inside an element's bare form; a quoted element containing a comma is split
// here, which our values (ids, kebab-case) never contain, so this stays correct for the
// data we write while remaining a small hand-rolled parser (no YAML dependency).
func parseFlowList(val string) ([]string, error) {
	inner := strings.TrimSpace(val[1 : len(val)-1])
	if inner == "" {
		return nil, nil
	}
	parts := strings.Split(inner, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		item, err := unquoteScalar(strings.TrimSpace(p))
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, nil
}

// unquoteScalar reverses yamlScalar (render.go): it strips surrounding double quotes
// and undoes the backslash/quote escaping, leaving a bare scalar untouched. Mirrors the
// sibling helper.
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
