// Package catalog owns the built-in catalog of canonical agent definitions and
// the act of materializing one into a project's `.daedalus/agents/` workspace.
//
// An *agent* is the canonical, backend-agnostic unit of "an AI role": an
// identifier, a human-facing role/description, a prompt and a set of parameters
// (init.md §3, §8). The catalog ships the five canonical SDD agents — analyst,
// architect, planner, validator, documenter — embedded in the binary so a user
// can scaffold a working pipeline offline, with no network and no external
// files (D6; a remote/marketplace catalog is post-MVP).
//
// This package is the *source* of agent definitions; it deliberately does not
// know how to compile them to a backend's native format (epic-06) nor how to
// formally validate them against the canonical schema (ticket-02-04, not yet
// implemented). It guarantees only that every built-in agent is structurally
// well-formed — non-empty kebab-case id, non-empty role and prompt, valid
// parameters — so it will pass that schema once it lands.
//
// Determinism and non-destruction are first-class, mirroring the workspace
// package: the same agent always renders byte-identical content (stable key
// order), and materializing an agent that already exists never overwrites it.
package catalog

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ParamType is the canonical type of an agent parameter value. The set is
// intentionally small — the MVP only needs scalar knobs (e.g. a model name or a
// temperature) — and closed, so the renderer can map every value to a stable
// YAML scalar without a YAML dependency. New types are an additive change here.
type ParamType string

const (
	// ParamString is a free-form text parameter (rendered as a YAML scalar,
	// quoted only when needed).
	ParamString ParamType = "string"
	// ParamNumber is a numeric parameter carried as its canonical string form so
	// rendering stays byte-stable (no float formatting drift across platforms).
	ParamNumber ParamType = "number"
	// ParamBool is a boolean parameter, rendered as the literal `true`/`false`.
	ParamBool ParamType = "bool"
)

// Param is a single, ordered agent parameter. Parameters are modeled as an
// ordered slice rather than a map because a Go map cannot guarantee iteration
// order, and order stability is what keeps the rendered YAML byte-for-byte
// reproducible (R8/CA6). The Value is always the canonical *string* form of the
// typed value; Type tells the renderer how to emit it so a number/bool is not
// accidentally quoted like a string.
type Param struct {
	// Key is the parameter name. It is the YAML key under the `parameters` block.
	Key string
	// Type is the canonical type used to render Value safely.
	Type ParamType
	// Value is the canonical string form of the parameter's value.
	Value string
	// Description documents the parameter's intent for a human editing the
	// materialized definition. It is optional and never rendered into the YAML
	// (kept out so the canonical file stays minimal); it exists so the catalog
	// and future TUI can surface help text.
	Description string
}

// Agent is the in-memory canonical model of a built-in agent. Its fields are the
// canonical contract every agent definition must satisfy (init.md §5): an
// identifier, a role/description, a prompt and parameters. This is the model
// ticket-02-04 will formally validate against; here we only guarantee structural
// well-formedness via Validate so the built-ins are ready for that schema.
type Agent struct {
	// ID is the agent's stable identifier in kebab-case (R7/CA5). It is both the
	// catalog key and the on-disk directory name when materialized, so it must be
	// filesystem-safe — kebab-case guarantees that.
	ID string
	// Role is the short, human-facing description of what the agent does,
	// coherent with the built-in catalog vision (init.md §8). Never empty.
	Role string
	// Prompt is the agent's system prompt in Markdown — the body of its behavior.
	// Never empty. Materialized verbatim as the agent's prompt.md.
	Prompt string
	// Params are the agent's parameters in declaration (render) order.
	Params []Param
}

// kebabCase matches a non-empty kebab-case identifier: lowercase ASCII letters
// and digits in dash-separated segments, no leading/trailing/double dashes. This
// is the convention from init.md §7 applied to agent ids (R7/CA5). It is the
// single source of truth for "is this id well-formed", used by Validate and
// reused by callers that need to check an id before trusting it as a path
// segment.
var kebabCase = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IsKebabCase reports whether id is a well-formed kebab-case identifier. It is
// exported because the kebab-case rule is domain knowledge (an agent id is a
// path segment and a backend key), not something each caller should re-encode.
func IsKebabCase(id string) bool {
	return kebabCase.MatchString(id)
}

// NormalizeID converts an arbitrary source identifier (e.g. a Claude Code agent
// `name` or a file base name) into a canonical kebab-case id (R6/CA5). The
// transformation is deterministic and lossy-by-design: it lowercases ASCII,
// turns any run of non-alphanumeric characters into a single dash, and trims
// leading/trailing dashes. A source that already is kebab-case is returned
// unchanged, so importing a well-formed id never mangles it.
//
// It returns an error when the source cannot be normalized into a *non-empty*
// kebab-case id (e.g. it is empty or contains no ASCII alphanumerics at all),
// because silently inventing an id would violate "report, don't guess" — the
// caller surfaces this as an actionable import error rather than writing an agent
// under a fabricated name.
func NormalizeID(source string) (string, error) {
	var b strings.Builder
	prevDash := false
	for _, r := range source {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r - 'A' + 'a') // fold to lowercase
			prevDash = false
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			prevDash = false
		default:
			// Any other character (space, '_', '.', punctuation, non-ASCII) becomes a
			// separator; collapse consecutive separators into a single dash.
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	id := strings.Trim(b.String(), "-")
	if id == "" {
		return "", fmt.Errorf("cannot derive a kebab-case id from %q", source)
	}
	// The construction above can only produce a valid kebab-case id, but assert it
	// so a future change to the rule cannot let a malformed id through unnoticed.
	if !IsKebabCase(id) {
		return "", fmt.Errorf("derived id %q from %q is not valid kebab-case", id, source)
	}
	return id, nil
}

// Validate checks that an agent is structurally well-formed against the canonical
// contract: a kebab-case id, a non-empty role, a non-empty prompt, and parameters
// whose keys are present, unique and of a known type. It is a *structural*
// pre-check, not the formal schema validator (ticket-02-04, out of scope here);
// its job is to guarantee every built-in is ready to pass that schema and that a
// malformed agent can never be silently materialized. Errors name the offending
// agent and field so they are actionable.
func (a Agent) Validate() error {
	if !IsKebabCase(a.ID) {
		// We surface the raw id even when empty so the message is unambiguous
		// (an empty id is a common authoring mistake).
		return fmt.Errorf("agent id %q is not valid kebab-case", a.ID)
	}
	if strings.TrimSpace(a.Role) == "" {
		return fmt.Errorf("agent %q has an empty role", a.ID)
	}
	if strings.TrimSpace(a.Prompt) == "" {
		return fmt.Errorf("agent %q has an empty prompt", a.ID)
	}

	seen := make(map[string]struct{}, len(a.Params))
	for _, p := range a.Params {
		if strings.TrimSpace(p.Key) == "" {
			return fmt.Errorf("agent %q has a parameter with an empty key", a.ID)
		}
		if _, dup := seen[p.Key]; dup {
			return fmt.Errorf("agent %q has a duplicate parameter key %q", a.ID, p.Key)
		}
		seen[p.Key] = struct{}{}
		switch p.Type {
		case ParamString, ParamNumber, ParamBool:
			// known type
		default:
			return fmt.Errorf("agent %q parameter %q has unknown type %q", a.ID, p.Key, p.Type)
		}
	}
	return nil
}

// Entry is a single listing row of the catalog: the minimum a caller needs to
// present the available agents for selection (R4/CA1) without materializing them.
// It is a projection of an Agent — id plus role — so listing never exposes the
// full prompt/parameters until the caller actually materializes one.
type Entry struct {
	ID   string
	Role string
}

// sortEntries orders entries by id so a listing is deterministic regardless of
// the catalog's internal storage order (R8). Centralized so every listing path
// shares one ordering rule.
func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
}
