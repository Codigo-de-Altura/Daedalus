package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ErrMalformedDefinition is the sentinel returned (wrapped) when a materialized
// agent definition on disk cannot be parsed against the canonical format. It is
// distinct from ErrAgentNotFound (the directory/files are absent) so a caller can
// tell "no such agent" from "this agent's files are corrupt".
var ErrMalformedDefinition = errors.New("malformed agent definition")

// Load reads a materialized agent from its directory under agentsRoot and
// reconstructs the in-memory Agent model. It is the inverse of the renderer
// (render.go): Load(render(a)) yields an equivalent Agent, and re-rendering a
// freshly loaded definition produces byte-identical output (the round-trip
// determinism edit relies on). It reads both canonical files — agent.yaml for
// the metadata/parameters and prompt.md for the prompt body — because the
// definition deliberately stores the prompt out-of-line (render.go).
//
// We hand-roll the parser instead of pulling in a YAML library: go.mod carries
// none and stdlib-first is the rule, and — crucially — we *own* this format. It
// is a fixed, tiny shape (a handful of top-level scalar keys plus a flat
// `parameters` mapping of scalars), so a purpose-built reader is both simpler and
// safer than a general YAML parser, and it cannot drift from what we emit because
// the two live side by side. Anything that does not match the shape we write is
// rejected as malformed rather than silently half-read.
func Load(agentsRoot, id string) (Agent, error) {
	if !IsKebabCase(id) {
		return Agent{}, fmt.Errorf("agent id %q is not valid kebab-case", id)
	}

	dir := filepath.Join(agentsRoot, id)
	defPath := filepath.Join(dir, DefinitionFileName)
	promptPath := filepath.Join(dir, PromptFileName)

	defBytes, err := os.ReadFile(defPath)
	if err != nil {
		// Surface absence as not-found so a caller can distinguish it from a parse
		// failure; other I/O errors propagate as-is.
		if errors.Is(err, os.ErrNotExist) {
			return Agent{}, fmt.Errorf("%w: %q (%s)", ErrAgentNotFound, id, DefinitionFileName)
		}
		return Agent{}, err
	}
	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Agent{}, fmt.Errorf("%w: %q (%s)", ErrAgentNotFound, id, PromptFileName)
		}
		return Agent{}, err
	}

	a, err := parseDefinition(string(defBytes))
	if err != nil {
		return Agent{}, fmt.Errorf("%w: agent %q: %v", ErrMalformedDefinition, id, err)
	}

	// The prompt body is the source of truth for the prompt; the `prompt:` key in
	// the YAML is only a pointer to this file (render.go), so we overwrite whatever
	// the pointer said with the actual file content. We strip the single trailing
	// newline the renderer added so an unedited load→render round-trips exactly
	// (renderPrompt re-appends exactly one).
	a.Prompt = strings.TrimRight(string(promptBytes), "\n")

	// The directory name is the canonical identity of a materialized agent; trust
	// it over any `id:` written inside the file so a renamed directory cannot
	// disagree with its contents.
	a.ID = id

	return a, nil
}

// parseDefinition parses the canonical agent.yaml body into an Agent (prompt left
// empty; Load fills it from prompt.md). It accepts exactly the shape the renderer
// emits and rejects anything else as an error, so a corrupt or hand-mangled file
// fails loudly instead of loading partial data. The prompt pointer line is read
// and discarded (the body comes from prompt.md).
func parseDefinition(body string) (Agent, error) {
	var a Agent
	var (
		sawID, sawRole bool
		inParams       bool
	)

	for _, raw := range strings.Split(body, "\n") {
		// Comment and blank lines carry no data; the renderer emits a comment
		// header and a trailing newline, both ignorable here.
		if strings.TrimSpace(raw) == "" || strings.HasPrefix(raw, "#") {
			continue
		}

		// A line indented under `parameters:` is a parameter entry. We detect
		// indentation explicitly (the renderer uses two spaces) so a top-level key
		// can never be misread as a parameter and vice versa.
		if inParams && (strings.HasPrefix(raw, "  ") || raw[0] == '\t') {
			key, val, ok := strings.Cut(strings.TrimSpace(raw), ":")
			if !ok {
				return Agent{}, fmt.Errorf("parameter line is not key: value: %q", raw)
			}
			key = strings.TrimSpace(key)
			if key == "" {
				return Agent{}, fmt.Errorf("parameter with empty key: %q", raw)
			}
			pType, pVal, err := parseParamValue(strings.TrimSpace(val))
			if err != nil {
				return Agent{}, fmt.Errorf("parameter %q: %v", key, err)
			}
			a.Params = append(a.Params, Param{Key: key, Type: pType, Value: pVal})
			continue
		}

		// Any non-indented line ends the parameters block and is a top-level key.
		inParams = false

		key, val, ok := strings.Cut(raw, ":")
		if !ok {
			return Agent{}, fmt.Errorf("line is not key: value: %q", raw)
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		switch key {
		case "id":
			// Parsed for completeness but not trusted: Load overrides ID with the
			// directory name (the canonical identity).
			sawID = true
		case "version":
			// Accepted and ignored: the on-disk version is informational for this
			// MVP; a future schema migration (ticket-02-04+) is where it gains
			// meaning. Re-rendering always stamps DefinitionVersion.
		case "role":
			s, err := unquoteScalar(val)
			if err != nil {
				return Agent{}, fmt.Errorf("role: %v", err)
			}
			a.Role = s
			sawRole = true
		case "prompt":
			// Pointer to prompt.md; the value is discarded (Load reads the file).
		case "parameters":
			// `parameters: {}` is the explicit empty mapping; anything else opens an
			// indented block whose entries are read on subsequent iterations.
			if val == "{}" {
				continue
			}
			if val != "" {
				return Agent{}, fmt.Errorf("parameters: expected block or {}, got %q", val)
			}
			inParams = true
		default:
			return Agent{}, fmt.Errorf("unknown top-level key %q", key)
		}
	}

	if !sawID {
		return Agent{}, errors.New("missing required key: id")
	}
	if !sawRole {
		return Agent{}, errors.New("missing required key: role")
	}
	return a, nil
}

// parseParamValue infers a parameter's canonical type and value from its rendered
// YAML scalar, mirroring renderParamValue: a bare `true`/`false` is a bool, a bare
// numeric literal is a number, and anything quoted (or otherwise) is a string with
// its quoting removed. This re-inference is what makes load→render byte-stable:
// the renderer emits numbers/bools bare and strings quoted-when-needed, so reading
// the same syntactic cues back reconstructs the exact Type the value was written
// with.
func parseParamValue(token string) (ParamType, string, error) {
	if token == "" {
		return "", "", errors.New("empty value")
	}
	// Quoted => string. Unquote and we are done; a value the renderer quoted (e.g.
	// the literal "true" or "0.2" as text) round-trips back to a string, exactly
	// as it was authored.
	if token[0] == '"' {
		s, err := unquoteScalar(token)
		if err != nil {
			return "", "", err
		}
		return ParamString, s, nil
	}
	// Bare boolean literals.
	if token == "true" || token == "false" {
		return ParamBool, token, nil
	}
	// Bare numeric literal: anything strconv accepts as an integer or float. We
	// keep the original textual form as the canonical Value so re-rendering emits
	// the identical token (no float reformatting).
	if _, err := strconv.ParseFloat(token, 64); err == nil {
		return ParamNumber, token, nil
	}
	// A bare, unquoted, non-numeric, non-boolean token. The renderer only emits
	// bare strings when yamlScalar deemed them safe (e.g. "default"), so treat it
	// as a plain string.
	return ParamString, token, nil
}

// unquoteScalar reverses yamlScalar (render.go): it strips surrounding double
// quotes and undoes the backslash/quote escaping the renderer applied, leaving a
// bare scalar untouched. It is intentionally limited to the escapes the renderer
// produces (\\ and \") because that is the entire surface of values we ever write;
// an unterminated or unexpectedly-escaped quoted scalar is reported as malformed.
func unquoteScalar(token string) (string, error) {
	if token == "" {
		return "", nil
	}
	if token[0] != '"' {
		// Bare scalar: emitted only when yamlScalar judged it safe, so it carries no
		// escaping to undo.
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
		// Escape sequence: consume the next byte. The renderer only ever escapes a
		// backslash or a double quote.
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
