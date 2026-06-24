// Package prompts owns the domain model and on-disk persistence of reusable
// prompts in a project's `.daedalus/prompts/` workspace directory.
//
// A *prompt* is a reusable fragment of Markdown text that feeds agents and
// project conventions (style, glossary, guidelines). Daedalus distinguishes two
// kinds (init.md / PRD §EPIC-3): *global* prompts (guidelines that apply to the
// whole project) and *shared* prompts (referenceable fragments such as a
// glossary or a commit policy). Both coexist flat in `.daedalus/prompts/` keyed
// by a unique kebab-case id, so the id is the single namespacing unit and there
// is never a collision between a global and a shared prompt.
//
// This package is intentionally self-contained and mirrors the catalog/workspace
// packages rather than coupling to them: it owns its own kebab-case rule, its own
// deterministic frontmatter renderer/parser, and its own non-destructive write
// helpers. The project duplicates these small helpers on purpose so each package
// can evolve its own canonical format without a build-time dependency reshaping a
// sibling's output. In particular this package does NOT import internal/catalog —
// the catalog's types are agent-specific.
//
// Determinism and non-destruction are first-class (R5): the same prompt always
// renders byte-identical content (fixed key order, trailing newline), creating a
// prompt never overwrites an existing one, and editing a prompt touches only its
// own file. The body is persisted verbatim as arbitrary Markdown; this package
// never reinterprets it or resolves inclusions — composition is ticket-03-02 (R7).
package prompts

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// PromptsDir is the workspace subdirectory that holds persisted prompts. It
// mirrors catalog.AgentsDir and matches the canonical layout
// (workspace.Subdirs "prompts"); kept as a constant here so this package does
// not import the workspace package just for a directory name, and the two stay
// in sync by convention rather than a build-time coupling.
const PromptsDir = "prompts"

// FileExt is the on-disk extension for a persisted prompt (R1): prompts are
// stored as Markdown files so they are legible and git-friendly.
const FileExt = ".md"

// Kind is the class of a prompt: a project-wide guideline (KindGlobal) or a
// reusable, referenceable fragment (KindShared). The set is closed and small
// (R6); a value outside it is a validation error.
type Kind string

const (
	// KindGlobal marks a prompt that applies to the whole project (e.g. style,
	// language, SDD conventions).
	KindGlobal Kind = "global"
	// KindShared marks a reusable fragment referenceable by agents or other
	// prompts (e.g. glossary, role definitions, commit policy).
	KindShared Kind = "shared"
)

// IsValidKind reports whether k is one of the two canonical kinds. Centralized
// so the renderer, parser and validator share one definition of "known kind".
func (k Kind) IsValidKind() bool {
	return k == KindGlobal || k == KindShared
}

// Prompt is the in-memory canonical model of a persisted prompt. Its fields are
// the minimum metadata the epic requires (R2): a unique kebab-case id, a kind, a
// human-facing title and an optional description, plus the Markdown body.
type Prompt struct {
	// ID is the prompt's stable identifier in kebab-case (R1/R2). It is both the
	// unique key within the workspace and the on-disk file base name, so it must
	// be filesystem-safe — kebab-case guarantees that.
	ID string
	// Kind is global or shared (R2/R6).
	Kind Kind
	// Title is the short, human-facing name of the prompt (R2). Never empty.
	Title string
	// Description is an optional one-line summary (R2). It may be empty; when empty
	// it is omitted from the persisted frontmatter entirely (see render.go).
	Description string
	// Body is the prompt's Markdown content (R7). It is persisted verbatim and
	// never reinterpreted by this package.
	Body string
}

// kebabCase matches a non-empty kebab-case identifier: lowercase ASCII letters
// and digits in dash-separated segments, no leading/trailing/double dashes. This
// is the same convention the catalog uses for agent ids (init.md §7); it is
// duplicated here (not imported) so prompts owns its own id rule. It is the
// single source of truth for "is this prompt id well-formed".
var kebabCase = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IsKebabCase reports whether id is a well-formed kebab-case identifier. Exported
// because the rule is domain knowledge (a prompt id is both a path segment and a
// unique key), not something each caller should re-encode.
func IsKebabCase(id string) bool {
	return kebabCase.MatchString(id)
}

// Entry is a single listing row: the minimum a caller needs to present the
// persisted prompts for selection (R3) without loading their bodies. It is a
// projection of a Prompt — id, kind and title — so listing stays cheap.
type Entry struct {
	ID    string
	Kind  Kind
	Title string
}

// sortEntries orders entries by id so a listing is deterministic regardless of
// the order the filesystem returned the files in (R5). Centralized so every
// listing path shares one ordering rule.
func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
}

// fmtQuote is a tiny helper so findings render an observed value with quotes
// consistently without pulling fmt into every call site inline.
func fmtQuote(s string) string {
	return fmt.Sprintf("%q", s)
}

// trimmedEmpty reports whether s is empty after trimming surrounding whitespace.
// Used by both the validator and the renderer's "omit empty description" rule so
// the two agree on what "empty" means.
func trimmedEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}
