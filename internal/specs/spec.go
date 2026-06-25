// Package specs owns the domain model and on-disk persistence of the SDD backlog's
// first canonical artifact: the *brief* and the *spec/PRD* it seeds, both living in
// a project's `.daedalus/specs/` workspace directory (ticket 05-01).
//
// # The two artifacts and the link between them (R1/R2/R8)
//
// A *brief* is the human-authored entry point of the SDD pipeline (init.md §5/§6):
// a short Markdown statement of intent that the *analyst* agent turns into a
// spec/PRD. A *spec* is that downstream artifact — a plan (what/why/requirements),
// editable by the human who refines it. This package manages the *definition* of
// both: it captures the brief, records its link to the analyst step of the default
// SDD workflow, and materializes the spec at a deterministic location that
// references its originating brief.
//
// # Phase 1: Daedalus manages the definition, it does NOT run the agent (R5)
//
// The conceptual pipeline is `brief └─► spec/PRD (analyst)`, but in phase 1 the
// agent's execution lives OUTSIDE Daedalus, on the user's backend (PRD decision
// D5). So the "link" between a brief and the analyst is purely *metadata*: this
// package never launches a process, never calls a model. It writes the provenance
// (which agent, which workflow, which phase) into stable frontmatter so that when
// the user later runs the analyst on their backend, the wiring is already recorded
// and the generated spec lands where Daedalus expects it. The spec's body is left
// for the user to fill/refine — Daedalus seeds a placeholder, never a generated
// artifact (R5/CA5).
//
// # Why a dedicated package, not a shared `backlog` one
//
// This package is intentionally self-contained and mirrors the prompts/workflows
// packages rather than coupling to them: it owns its own kebab-case rule, its own
// deterministic frontmatter renderer/parser, and its own non-destructive write
// helpers. The remaining epic-05 artifacts (architecture, epics, tickets) carry
// different models and formats; a premature shared `backlog` package would force a
// single schema onto all of them. The prompts molde shows a package-per-artifact
// scales cleanly, so `specs` follows it. In particular this package does NOT import
// internal/workflows: the provenance anchor (analyst / sdd-default / spec) is
// duplicated here as constants and pinned to the real workflow phase by a test
// (provenance_link_test.go), so the model stays backend/workflow-agnostic while the
// link can never silently drift from internal/workflows.DefaultWorkflow.
//
// # Determinism and non-destruction are first-class (R6/R7)
//
// The same brief/spec always renders byte-identical content (fixed key order,
// trailing newline). Capturing a brief never overwrites an existing brief or its
// spec, and editing a spec touches only its own file. The body is persisted
// verbatim as arbitrary Markdown; this package never reinterprets it.
package specs

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// SpecsDir is the workspace subdirectory that holds briefs and specs. It mirrors
// the canonical layout (workspace.Subdirs "specs"); kept as a constant here so
// this package does not import the workspace package just for a directory name,
// and the two stay in sync by convention rather than a build-time coupling.
const SpecsDir = "specs"

// FileExt is the on-disk extension for a persisted spec (R3): specs are stored as
// Markdown files so they are legible, editable and git-friendly.
const FileExt = ".md"

// BriefExt is the on-disk extension for a brief. A brief is co-located with its
// spec in `.daedalus/specs/` as `<slug>.brief.md` rather than in a separate
// `briefs/` directory: the workspace layout (workspace.Subdirs) deliberately has
// no `briefs/` subdir, and the brief is the spec's companion in the SDD pipeline,
// so pairing them by a shared `<slug>` keeps the two artifacts visibly adjacent
// and avoids reshaping the workspace scaffolding from this ticket (R1). The
// `.brief.md` suffix is what distinguishes a brief file from a spec file in the
// same directory; List() relies on it to never mistake one for the other.
const BriefExt = ".brief.md"

// Provenance anchor: the default SDD pipeline's `brief -> spec` step (init.md §6).
//
// These three constants name the analyst step of the factory workflow. They are
// duplicated here — not imported from internal/workflows — so this package stays
// self-contained like the prompts molde; provenance_link_test.go pins them to
// internal/workflows.DefaultWorkflow so they can never drift from the real phase.
//
// They are written into both artifacts' frontmatter as the link Daedalus manages
// in phase 1 (R2/CA2): the brief records that it is consumed by AnalystAgent at the
// DefaultPhase of DefaultWorkflowName, and the spec records that it is produced by
// the same step from its originating brief (R8/CA7). No execution is implied.
const (
	// AnalystAgent is the agent that turns a brief into a spec/PRD (init.md §5,
	// PRD RF-5.1). It matches the analyst phase's Agent in the default workflow.
	AnalystAgent = "analyst"
	// DefaultWorkflowName is the factory SDD workflow these artifacts belong to.
	// It matches workflows.DefaultWorkflowName.
	DefaultWorkflowName = "sdd-default"
	// DefaultPhase is the phase id of the `brief -> spec` step. It matches
	// workflows.DefaultPhaseSpec.
	DefaultPhase = "spec"
)

// slugPattern matches a non-empty kebab-case slug: lowercase ASCII letters and
// digits in dash-separated segments, no leading/trailing/double dashes. This is
// the same convention prompts/workflows use for ids (init.md §7); it is duplicated
// here (not imported) so specs owns its own slug rule. It is the single source of
// truth for "is this spec slug well-formed" (R3).
var slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// IsKebabCase reports whether slug is a well-formed kebab-case identifier. Exported
// because the rule is domain knowledge (a spec slug is both a path segment and a
// unique key), not something each caller should re-encode (R3).
func IsKebabCase(slug string) bool {
	return slugPattern.MatchString(slug)
}

// Brief is the in-memory canonical model of a captured brief (R1). Its fields are
// the minimum metadata the ticket requires: a unique kebab-case slug (shared with
// its spec), a human-facing title, and the Markdown body the human authored. The
// link to the analyst step is not stored as a field because it is constant for
// every brief in phase 1 (always the sdd-default analyst step); the renderer writes
// it deterministically into the frontmatter (see render.go).
type Brief struct {
	// Slug is the brief's stable identifier in kebab-case (R1/R3). It is the unique
	// key within the workspace and the base of both the brief file (`<slug>.brief.md`)
	// and its spec file (`<slug>.md`), so it must be filesystem-safe — kebab-case
	// guarantees that.
	Slug string
	// Title is the short, human-facing name of the brief (R1). Never empty.
	Title string
	// Body is the brief's Markdown content (R1/R6). It is persisted verbatim and
	// never reinterpreted by this package.
	Body string
}

// Spec is the in-memory canonical model of a materialized spec/PRD (R3). It is the
// editable artifact the human refines (R4); Daedalus only seeds and never
// regenerates it. Its provenance fields make the `brief -> spec` trace explicit and
// stable (R8/CA7).
type Spec struct {
	// Slug is the spec's stable identifier in kebab-case (R3), shared with its
	// originating brief. It is the base of the spec file (`<slug>.md`).
	Slug string
	// Title is the short, human-facing name of the spec (R3). Never empty.
	Title string
	// BriefRef is the workspace-relative file name of the originating brief
	// (`<slug>.brief.md`). It is the trace `brief -> spec` (R8/CA7): the spec always
	// references the brief it was seeded from. Never empty for a materialized spec.
	BriefRef string
	// Body is the spec's Markdown content (R3/R6). On capture this is a placeholder
	// the user replaces by running the analyst on their backend (R5); thereafter it
	// is whatever the user wrote, persisted verbatim.
	Body string
}

// briefFileName returns the on-disk file name for a brief slug: `<slug>.brief.md`.
// The slug is assumed valid kebab-case (callers validate first), so it is a safe
// single path segment.
func briefFileName(slug string) string {
	return slug + BriefExt
}

// specFileName returns the on-disk file name for a spec slug: `<slug>.md`. The slug
// is assumed valid kebab-case (callers validate first), so it is a safe single path
// segment.
func specFileName(slug string) string {
	return slug + FileExt
}

// Entry is a single listing row: the minimum a caller needs to present the
// captured briefs and their specs for selection without loading their bodies. It is
// a projection — slug, title, and whether a spec has been materialized for it.
type Entry struct {
	// Slug is the shared identity of the brief/spec pair.
	Slug string
	// Title is the brief's title (the human-facing label of the pair).
	Title string
	// HasSpec reports whether a `<slug>.md` spec exists alongside the brief, so a
	// caller can show which briefs have been carried forward to a spec.
	HasSpec bool
}

// sortEntries orders entries by slug so a listing is deterministic regardless of
// the order the filesystem returned the files in (R6). Centralized so every listing
// path shares one ordering rule.
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
