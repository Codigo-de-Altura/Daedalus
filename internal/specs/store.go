package specs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrBriefExists is the sentinel returned by Apply when a brief with the requested
// slug already has a file under the specs root. Exposed so callers can map a
// duplicate slug to an explicit "already exists, not overwritten" error via
// errors.Is, never an overwrite (R4/R7).
var ErrBriefExists = errors.New("brief already exists")

// ErrSpecExists is the sentinel returned by Apply when a spec with the requested
// slug already has a file. Distinct from ErrBriefExists so a caller can report
// precisely which artifact of the pair was preserved.
var ErrSpecExists = errors.New("spec already exists")

// CapturePlan is the result of planning a brief capture: the resolved brief, the
// seeded spec, the two files they will land in, and the fully rendered bytes —
// computed without touching the filesystem. Mirroring the prompts Plan/Apply split,
// the content is captured at plan time so a `--preview` and the subsequent Apply
// describe identical bytes (R6), and validation happens before any I/O.
//
// The plan deliberately holds BOTH artifacts: capturing a brief in phase 1 means
// (R1) persisting the brief and (R3) materializing the canonical spec destination
// that references it (R8/CA7) — so a reader can run the analyst on their backend and
// drop the result into a spec file Daedalus already wired up. Daedalus does not
// generate the spec body (R5/CA5); it seeds a placeholder the user replaces.
type CapturePlan struct {
	// Brief is the validated brief to be written.
	Brief Brief
	// Spec is the validated spec destination to be seeded (placeholder body).
	Spec Spec
	// BriefPath and SpecPath are the absolute files the artifacts will be written to.
	BriefPath string
	SpecPath  string
	// BriefContent and SpecContent are the rendered file contents captured at plan
	// time, so a preview and the apply are byte-identical.
	BriefContent string
	SpecContent  string
}

// CaptureResult describes what Apply produced, so callers can report it. Because
// the operation is non-destructive, either artifact may have already existed and
// been left intact; the *Created flags distinguish a fresh write from a preserved
// existing file (R4/R7).
type CaptureResult struct {
	Slug string
	// BriefPath and SpecPath are the absolute artifact paths (for reporting).
	BriefPath string
	SpecPath  string
	// BriefCreated and SpecCreated report whether each file was newly written. A
	// false value means the file already existed and was preserved untouched.
	BriefCreated bool
	SpecCreated  bool
}

// PlanCapture validates a brief and computes the plan to persist it together with
// its seeded spec under specsRoot, without writing anything (R6/R7). It rejects an
// invalid brief here — before any I/O — with the rich, actionable *ValidationError,
// so a bad slug or empty title never reaches the filesystem. It does NOT check for
// existing files: that is an Apply-time, non-destructive concern (so a preview can
// be shown even when the slug is taken, and the create-or-skip decision stays
// atomic per file).
//
// The seeded spec's body is a deterministic placeholder that (a) tells the user
// Daedalus did not run the analyst (R5) and (b) names the brief it must be generated
// from (reinforcing the R8 trace in the human-readable body, not only the
// frontmatter). The frontmatter carries the machine-readable trace and provenance.
func PlanCapture(specsRoot string, b Brief) (*CapturePlan, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}

	spec := Spec{
		Slug:     b.Slug,
		Title:    b.Title,
		BriefRef: briefFileName(b.Slug),
		Body:     specPlaceholderBody(b),
	}
	// The seeded spec is constructed from a valid brief, so it is valid by
	// construction; validating it keeps the invariant explicit and guards against a
	// future change to specPlaceholderBody/seed shape.
	if err := spec.Validate(); err != nil {
		return nil, err
	}

	return &CapturePlan{
		Brief:        b,
		Spec:         spec,
		BriefPath:    filepath.Join(specsRoot, briefFileName(b.Slug)),
		SpecPath:     filepath.Join(specsRoot, specFileName(b.Slug)),
		BriefContent: RenderBrief(b),
		SpecContent:  RenderSpec(spec),
	}, nil
}

// Apply writes the planned brief and spec files non-destructively: it creates the
// specs directory if needed and each file only if it does not already exist
// (O_CREATE|O_EXCL). An existing brief or spec is preserved untouched and reported
// via the result flags rather than clobbered (R4/R7) — this is what protects a spec
// the user has already refined: re-capturing the same brief never overwrites the
// user's edits.
//
// Both files are attempted independently so a partially-present pair (e.g. the brief
// exists but its spec was deleted) is completed rather than aborted: the missing
// artifact is created, the present one is preserved. Because the content was
// rendered deterministically at plan time, re-creating into a clean workspace yields
// byte-identical bytes (R6).
func (p *CapturePlan) Apply() (*CaptureResult, error) {
	if err := os.MkdirAll(filepath.Dir(p.BriefPath), 0o755); err != nil {
		return nil, err
	}

	briefCreated, err := ensureFile(p.BriefPath, p.BriefContent)
	if err != nil {
		return nil, err
	}
	specCreated, err := ensureFile(p.SpecPath, p.SpecContent)
	if err != nil {
		return nil, err
	}

	return &CaptureResult{
		Slug:         p.Brief.Slug,
		BriefPath:    p.BriefPath,
		SpecPath:     p.SpecPath,
		BriefCreated: briefCreated,
		SpecCreated:  specCreated,
	}, nil
}

// Capture validates and persists a brief together with its seeded spec under
// specsRoot in one call, for callers that do not need to preview the content first.
// It is non-destructive: existing files are preserved and reported via the result
// (R4/R7). Callers that want a preview use PlanCapture then Apply.
func Capture(specsRoot string, b Brief) (*CaptureResult, error) {
	plan, err := PlanCapture(specsRoot, b)
	if err != nil {
		return nil, err
	}
	return plan.Apply()
}

// specPlaceholderBody is the deterministic placeholder seeded into a freshly
// captured spec (R5/CA5). It states, in the human-readable body, that Daedalus did
// not run the analyst and names the brief the spec must be generated from (R8). The
// user replaces it with the spec/PRD they produce by running the analyst on their
// backend; thereafter Daedalus never rewrites it (R4). It is a pure function of the
// brief so the seed is byte-stable (R6).
func specPlaceholderBody(b Brief) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n\n", b.Title)
	sb.WriteString("> Spec/PRD placeholder. Daedalus manages this artifact's definition but does\n")
	sb.WriteString("> not run the analyst agent (phase 1). Generate the spec by running the\n")
	fmt.Fprintf(&sb, "> %q agent (workflow %q, phase %q) on your backend, using the brief below,\n",
		AnalystAgent, DefaultWorkflowName, DefaultPhase)
	sb.WriteString("> then replace this placeholder with the result and refine it.\n\n")
	fmt.Fprintf(&sb, "Source brief: %s\n", briefFileName(b.Slug))
	return sb.String()
}

// List returns the captured briefs under specsRoot as slug+title entries, each
// flagged with whether its spec has been materialized, sorted by slug (R6). A
// non-existent specs directory is treated as "no briefs" (empty list), not an error,
// so listing a freshly initialized workspace is well-defined. A brief is the unit of
// listing (the pipeline entry point); a `<slug>.md` without a matching brief is not
// listed here. Files that fail to parse are skipped rather than aborting the whole
// listing — a single corrupt file must not hide the rest.
func List(specsRoot string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(specsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Entry{}, nil
		}
		return nil, err
	}

	// Collect the spec slugs first so each brief can report whether its spec exists
	// without a second directory scan.
	specSlugs := make(map[string]bool)
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		// A spec is `<slug>.md` but NOT `<slug>.brief.md`; exclude briefs explicitly.
		if strings.HasSuffix(name, BriefExt) || !strings.HasSuffix(name, FileExt) {
			continue
		}
		if slug := strings.TrimSuffix(name, FileExt); IsKebabCase(slug) {
			specSlugs[slug] = true
		}
	}

	out := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.HasSuffix(name, BriefExt) {
			continue
		}
		slug := strings.TrimSuffix(name, BriefExt)
		if !IsKebabCase(slug) {
			continue
		}
		brief, err := LoadBrief(specsRoot, slug)
		if err != nil {
			// A malformed brief is skipped from the listing; surfacing it as a hard
			// error would make `list` unusable whenever one file is hand-corrupted.
			continue
		}
		out = append(out, Entry{Slug: brief.Slug, Title: brief.Title, HasSpec: specSlugs[slug]})
	}

	sortEntries(out)
	return out, nil
}

// SpecEditSpec describes the changes to apply to a materialized spec. Every field is
// optional so a caller edits only what it names; the presence of a change is
// signaled by the *Set* booleans rather than by zero values, because "set the title
// to the empty string" (an invalid edit we must reject) and "leave the title
// untouched" (a no-op) must be distinguishable. The brief reference and the
// provenance are intentionally NOT editable: the brief -> spec trace (R8) and the
// analyst-step provenance (R2) are part of the artifact's identity, not free
// metadata; rewiring them is better modeled as a fresh capture than an in-place
// mutation.
type SpecEditSpec struct {
	SetTitle bool
	Title    string
	SetBody  bool
	Body     string
}

// IsEmpty reports whether the spec edit would change nothing. The CLI uses it to
// turn an edit with no flags into a usage error rather than rewriting a file
// identically.
func (e SpecEditSpec) IsEmpty() bool {
	return !e.SetTitle && !e.SetBody
}

// EditSpec applies spec to the persisted spec slug under specsRoot and persists the
// result, returning the edited Spec. It is the read-modify-validate-write cycle the
// ticket mandates (R4/R6):
//
//  1. Load the current spec from disk (LoadSpec); an absent spec is ErrSpecNotFound.
//  2. Apply the requested changes to an in-memory copy, preserving the brief trace
//     and provenance.
//  3. Validate the *result* before any write — an edit that would leave the spec
//     invalid (e.g. an empty title) is rejected here with an actionable error and
//     nothing is written (R4).
//  4. Persist atomically (writeAtomic: temp + rename), so a failure mid-write can
//     never leave a half-written file, and ONLY this spec's file is touched — the
//     brief and other workspace files are untouched (R4/R7).
func EditSpecArtifact(specsRoot, slug string, edit SpecEditSpec) (Spec, error) {
	s, err := LoadSpec(specsRoot, slug)
	if err != nil {
		return Spec{}, err
	}

	if edit.SetTitle {
		s.Title = edit.Title
	}
	if edit.SetBody {
		s.Body = edit.Body
	}

	if err := s.Validate(); err != nil {
		return Spec{}, err
	}

	path := filepath.Join(specsRoot, specFileName(slug))
	if err := writeAtomic(path, RenderSpec(s)); err != nil {
		return Spec{}, err
	}
	return s, nil
}

// ensureFile creates a file at path with the given deterministic content only if it
// does not already exist. O_EXCL makes the create-or-skip decision atomic and
// non-destructive: an existing file is never truncated, so a spec the user has
// refined survives a re-capture untouched. It reports whether it created a new file
// (created=false when the path already existed). Duplicated from the prompts/
// workspace helpers of the same name so this package owns its write semantics
// independently.
func ensureFile(path, content string) (bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, err
	}
	// Write then close without defer: a deferred Close would mask a Write error.
	_, writeErr := f.WriteString(content)
	closeErr := f.Close()
	if writeErr != nil {
		return false, writeErr
	}
	if closeErr != nil {
		return false, closeErr
	}
	return true, nil
}

// writeAtomic writes content to path atomically: it writes to a temporary file in
// the same directory and renames it over path. The rename is atomic on the same
// filesystem, so a reader sees either the old or the new content, never a partial
// write — and a crash mid-write leaves the original intact. Unlike ensureFile this
// *replaces* an existing file, which is exactly what an edit must do (the spec is
// known to exist; we are updating it). On any failure the temp file is cleaned up so
// we never litter the workspace. Mirrors prompts.writeAtomic.
func writeAtomic(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
