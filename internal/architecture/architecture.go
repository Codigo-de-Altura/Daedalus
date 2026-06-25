// Package architecture owns the domain model and on-disk persistence of the SDD
// backlog's architecture documents, living in a project's `.daedalus/architecture/`
// workspace directory (ticket 05-02).
//
// # The artifact and its link to the originating spec (R1/R2/R3)
//
// An *architecture document* is a plan (init.md §5/§6): a high-level description of
// structure, components and decisions — NOT an implementation recipe. In the SDD
// pipeline it sits after the spec/PRD: the *architect* agent turns a spec into an
// architecture document, which the *planner* and the external developer/agent then
// consume. This package manages the *definition* of that document: its canonical
// location, its diff-friendly form, and its optional link back to the spec it was
// derived from (the `spec -> architecture` trace).
//
// # Phase 1: Daedalus manages the definition, it does NOT run the agent (R5)
//
// As in ticket 05-01, the architect's execution lives OUTSIDE Daedalus, on the
// user's backend (PRD decision D5). So the "link" to the spec is purely *metadata*:
// this package never launches a process, never calls a model. When a document is
// linked to a spec it records the provenance (which spec, which agent, which
// workflow, which phase) into stable frontmatter so that when the user later runs
// the architect on their backend, the wiring is already recorded and the generated
// document lands where Daedalus expects it. The body is the user's to write/refine —
// Daedalus seeds a placeholder, never a generated artifact (R5/CA5).
//
// # Why a dedicated package, mirroring internal/specs
//
// This package is intentionally self-contained and mirrors internal/specs (and the
// prompts/workflows packages) rather than coupling to them: it owns its own
// kebab-case rule, its own deterministic frontmatter renderer/parser, and its own
// non-destructive write helpers. The epic-05 artifacts (spec, architecture, epics,
// tickets) carry DIFFERENT schemas — a spec references one brief, an architecture
// document references one spec, epics/tickets will carry status/priority/deps — so a
// shared `backlog` package would force one schema onto all of them and reintroduce
// exactly the build-time coupling the prompts package documents avoiding. The molde
// is proven (prompts, workflows, specs); architecture follows it for consistency,
// and the small shared helpers (frontmatter split, yamlScalar, ensureFile,
// writeAtomic) are duplicated on purpose so each package can evolve its own canonical
// format. In particular this package does NOT import internal/workflows or
// internal/specs: the provenance anchor (architect / sdd-default / architecture) is
// duplicated here as constants and pinned to the real workflow phase by a test
// (provenance_link_test.go).
//
// # Determinism and non-destruction are first-class (R6/R7)
//
// The same document always renders byte-identical content (fixed key order, trailing
// newline). Creating a document never overwrites an existing one, and editing a
// document touches only its own file. The body is persisted verbatim as arbitrary
// Markdown; this package never reinterprets it.
package architecture

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ArchitectureDir is the workspace subdirectory that holds architecture documents.
// It mirrors the canonical layout (workspace.Subdirs "architecture"); kept as a
// constant here so this package does not import the workspace package just for a
// directory name, and the two stay in sync by convention rather than a build-time
// coupling.
const ArchitectureDir = "architecture"

// FileExt is the on-disk extension for a persisted architecture document (R1):
// documents are stored as Markdown files so they are legible, editable and
// git-friendly.
const FileExt = ".md"

// Provenance anchor: the default SDD pipeline's `spec -> architecture` step
// (init.md §6).
//
// These constants name the architect step of the factory workflow. They are
// duplicated here — not imported from internal/workflows — so this package stays
// self-contained like the internal/specs molde; provenance_link_test.go pins them to
// internal/workflows.DefaultWorkflow so they can never drift from the real phase.
//
// When a document is linked to a spec they are written into its frontmatter as the
// trace Daedalus manages in phase 1 (R3/CA3): the document records that it is
// produced by ArchitectAgent at the DefaultPhase of DefaultWorkflowName from its
// originating SpecRef. No execution is implied.
const (
	// ArchitectAgent is the agent that turns a spec into an architecture document
	// (init.md §5/§6). It matches the architecture phase's Agent in the default
	// workflow.
	ArchitectAgent = "architect"
	// DefaultWorkflowName is the factory SDD workflow these documents belong to. It
	// matches workflows.DefaultWorkflowName.
	DefaultWorkflowName = "sdd-default"
	// DefaultPhase is the phase id of the `spec -> architecture` step. It matches
	// workflows.DefaultPhaseArchitecture.
	DefaultPhase = "architecture"
)

// slugPattern matches a non-empty kebab-case slug: lowercase ASCII letters and
// digits in dash-separated segments, no leading/trailing/double dashes. This is the
// same convention specs/prompts/workflows use for ids (init.md §7); it is duplicated
// here (not imported) so architecture owns its own slug rule. It is the single
// source of truth for "is this architecture slug well-formed" (R1).
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IsKebabCase reports whether slug is a well-formed kebab-case identifier. Exported
// because the rule is domain knowledge (an architecture slug is both a path segment
// and a unique key), not something each caller should re-encode (R1).
func IsKebabCase(slug string) bool {
	return slugPattern.MatchString(slug)
}

// Document is the in-memory canonical model of an architecture document (R2). Its
// fields are the minimum metadata the ticket requires: a unique kebab-case slug, a
// human-facing title, an OPTIONAL reference to the originating spec (the R3/CA3
// trace), and the Markdown body the user authors/refines.
type Document struct {
	// Slug is the document's stable identifier in kebab-case (R1). It is the unique
	// key within the workspace and the on-disk file base name, so it must be
	// filesystem-safe — kebab-case guarantees that.
	Slug string
	// Title is the short, human-facing name of the document (R2). Never empty.
	Title string
	// SpecRef is the OPTIONAL workspace-relative file name of the originating spec
	// (`<spec-slug>.md`). R3 says a document "can be linked" to its spec, so this may
	// be empty (an unlinked document). When non-empty it is the `spec -> architecture`
	// trace (R3/CA3) and triggers the architect-step provenance in the frontmatter
	// (see render.go). When empty those provenance keys are omitted entirely so an
	// unlinked document carries no misleading wiring.
	SpecRef string
	// Body is the document's Markdown content (R2/R6). On create this is a placeholder
	// the user replaces by running the architect on their backend (R5); thereafter it
	// is whatever the user wrote, persisted verbatim.
	Body string
}

// fileName returns the on-disk file name for a document slug: `<slug>.md` (R1). The
// slug is assumed valid kebab-case (callers validate first), so it is a safe single
// path segment.
func fileName(slug string) string {
	return slug + FileExt
}

// Entry is a single listing row: the minimum a caller needs to present the
// architecture documents for selection without loading their bodies. It is a
// projection — slug, title, and the optional spec link so a caller can show which
// documents are wired to a spec.
type Entry struct {
	// Slug is the document's identity.
	Slug string
	// Title is the document's human-facing label.
	Title string
	// SpecRef is the originating spec reference, or "" for an unlinked document.
	SpecRef string
}

// sortEntries orders entries by slug so a listing is deterministic regardless of the
// order the filesystem returned the files in (R6). Centralized so every listing path
// shares one ordering rule.
func sortEntries(entries []Entry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].Slug < entries[j].Slug })
}

// fmtQuote renders an observed value with quotes consistently for findings.
func fmtQuote(s string) string {
	return fmt.Sprintf("%q", s)
}

// trimmedEmpty reports whether s is empty after trimming surrounding whitespace.
// Shared by the validator and the renderer so they agree on what "empty" means.
func trimmedEmpty(s string) bool {
	return strings.TrimSpace(s) == ""
}
