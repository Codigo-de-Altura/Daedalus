package architecture

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrDocumentExists is the sentinel returned by Create/Apply when a document with the
// requested slug already has a file under the architecture root. Exposed so callers
// can map a duplicate slug to an explicit "already exists, not overwritten" error via
// errors.Is, never an overwrite (R4/R7).
var ErrDocumentExists = errors.New("architecture document already exists")

// CreatePlan is the result of planning a Create: the resolved document, the file it
// will land in and the fully rendered bytes — computed without touching the
// filesystem. Mirroring the specs/prompts Plan/Apply split, the content is captured
// at plan time so a `--preview` and the subsequent Apply describe identical bytes
// (R6), and validation happens before any I/O.
type CreatePlan struct {
	// Document is the validated document to be written.
	Document Document
	// Path is the absolute file the document will be written to (`<root>/<slug>.md`).
	Path string
	// Content is the rendered file content captured at plan time.
	Content string
}

// PlanCreate validates a document and computes the plan to persist it under archRoot,
// without writing anything (R6/R7). It rejects an invalid document here — before any
// I/O — with the rich, actionable *ValidationError, so a bad slug or empty title
// never reaches the filesystem. It does NOT check for an existing file: that is an
// Apply-time, non-destructive concern (so a preview can be shown even when the slug
// is taken, and the create-or-fail decision stays atomic). It also does NOT verify
// the optional spec link's existence: that is a friendly, filesystem-aware check the
// CLI performs (it knows the specs directory); the store stays a pure model->bytes
// transform.
//
// When the document has no body, a deterministic placeholder is seeded so a freshly
// created document is immediately legible and states that Daedalus did not run the
// architect (R5/CA5). A caller that supplies its own body keeps it verbatim.
func PlanCreate(archRoot string, d Document) (*CreatePlan, error) {
	if trimmedEmpty(d.Body) {
		d.Body = placeholderBody(d)
	}
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return &CreatePlan{
		Document: d,
		Path:     filepath.Join(archRoot, fileName(d.Slug)),
		Content:  Render(d),
	}, nil
}

// Apply writes the planned document file non-destructively: it creates the
// architecture directory if needed and the file only if it does not already exist
// (O_CREATE|O_EXCL). A slug that already has a file is reported as ErrDocumentExists
// rather than clobbered (R4/R7) — this is what protects a document the user has
// already refined: re-creating the same slug never overwrites the user's edits.
// Because the content was rendered deterministically at plan time, re-creating into a
// clean workspace yields byte-identical bytes (R6).
func (p *CreatePlan) Apply() error {
	if err := os.MkdirAll(filepath.Dir(p.Path), 0o755); err != nil {
		return err
	}
	created, err := ensureFile(p.Path, p.Content)
	if err != nil {
		return err
	}
	if !created {
		return fmt.Errorf("%w: %q", ErrDocumentExists, p.Document.Slug)
	}
	return nil
}

// Create validates and persists a new document under archRoot in one call, for
// callers that do not need to preview the content first. It is non-destructive: a
// duplicate slug returns ErrDocumentExists and the existing file is left untouched
// (R4/R7). Callers that want a preview use PlanCreate then Apply.
func Create(archRoot string, d Document) error {
	plan, err := PlanCreate(archRoot, d)
	if err != nil {
		return err
	}
	return plan.Apply()
}

// placeholderBody is the deterministic placeholder seeded into a freshly created,
// body-less document (R5/CA5). It states, in the human-readable body, that Daedalus
// did not run the architect and — when the document is linked — names the spec it
// should be derived from (reinforcing the R3 trace in the body, not only the
// frontmatter). The user replaces it with the architecture they produce by running
// the architect on their backend; thereafter Daedalus never rewrites it (R4). It is a
// pure function of the document so the seed is byte-stable (R6).
func placeholderBody(d Document) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n\n", d.Title)
	sb.WriteString("> Architecture document placeholder. Daedalus manages this artifact's definition\n")
	sb.WriteString("> but does not run the architect agent (phase 1). Generate the architecture by\n")
	if trimmedEmpty(d.SpecRef) {
		fmt.Fprintf(&sb, "> running the %q agent (workflow %q, phase %q) on your backend, then replace\n",
			ArchitectAgent, DefaultWorkflowName, DefaultPhase)
		sb.WriteString("> this placeholder with the result and refine it.\n")
	} else {
		fmt.Fprintf(&sb, "> running the %q agent (workflow %q, phase %q) on your backend, using the spec\n",
			ArchitectAgent, DefaultWorkflowName, DefaultPhase)
		sb.WriteString("> below, then replace this placeholder with the result and refine it.\n\n")
		fmt.Fprintf(&sb, "Source spec: %s\n", d.SpecRef)
	}
	return sb.String()
}

// List returns the architecture documents under archRoot as slug+title+spec entries,
// sorted by slug (R6). A non-existent architecture directory is treated as "no
// documents" (empty list), not an error, so listing a freshly initialized workspace
// is well-defined. Files that are not `*.md`, and any file that fails to parse, are
// skipped rather than aborting the whole listing — a single corrupt file must not
// hide the rest.
func List(archRoot string) ([]Entry, error) {
	dirEntries, err := os.ReadDir(archRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Entry{}, nil
		}
		return nil, err
	}

	out := make([]Entry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.HasSuffix(name, FileExt) {
			continue
		}
		slug := strings.TrimSuffix(name, FileExt)
		// The slug is the file identity; if a file's base name is not valid kebab-case
		// it is not one of ours, so skip it rather than fail the listing.
		if !IsKebabCase(slug) {
			continue
		}
		d, err := Load(archRoot, slug)
		if err != nil {
			// A malformed file is skipped from the listing; surfacing it as a hard error
			// would make `list` unusable whenever one file is hand-corrupted.
			continue
		}
		out = append(out, Entry{Slug: d.Slug, Title: d.Title, SpecRef: d.SpecRef})
	}

	sortEntries(out)
	return out, nil
}

// EditSpec describes the changes to apply to a persisted document. Every field is
// optional so a caller edits only what it names; the presence of a change is signaled
// by the *Set* booleans rather than by zero values, because "set the title to the
// empty string" (an invalid edit we must reject) and "leave the title untouched" (a
// no-op) must be distinguishable.
//
// The spec link IS editable (unlike the spec's brief reference in 05-01): R3 frames
// the link as a manageable attribute of an architecture document, so a user may
// attach a spec to a previously unlinked document, or clear the link. SetSpec with an
// empty Spec clears the link (and drops the provenance block on the next render);
// SetSpec with a value attaches/retargets it. The agent/workflow/phase are NOT
// editable: they are the constant identity of the `spec -> architecture` step, not
// free metadata.
type EditSpec struct {
	SetTitle bool
	Title    string
	SetSpec  bool
	Spec     string
	SetBody  bool
	Body     string
}

// IsEmpty reports whether the edit would change nothing. The CLI uses it to turn an
// edit with no flags into a usage error rather than rewriting a file identically.
func (e EditSpec) IsEmpty() bool {
	return !e.SetTitle && !e.SetSpec && !e.SetBody
}

// Edit applies spec to the persisted document slug under archRoot and persists the
// result, returning the edited Document. It is the read-modify-validate-write cycle
// the ticket mandates (R4/R6):
//
//  1. Load the current document from disk (Load); an absent document is
//     ErrDocumentNotFound.
//  2. Apply the requested changes to an in-memory copy.
//  3. Validate the *result* before any write — an edit that would leave the document
//     invalid (e.g. an empty title) is rejected here with an actionable error and
//     nothing is written (R4).
//  4. Persist atomically (writeAtomic: temp + rename), so a failure mid-write can
//     never leave a half-written file, and ONLY this document's file is touched —
//     other workspace files are untouched (R4/R7).
func Edit(archRoot, slug string, spec EditSpec) (Document, error) {
	d, err := Load(archRoot, slug)
	if err != nil {
		return Document{}, err
	}

	if spec.SetTitle {
		d.Title = spec.Title
	}
	if spec.SetSpec {
		d.SpecRef = spec.Spec
	}
	if spec.SetBody {
		d.Body = spec.Body
	}

	if err := d.Validate(); err != nil {
		return Document{}, err
	}

	path := filepath.Join(archRoot, fileName(slug))
	if err := writeAtomic(path, Render(d)); err != nil {
		return Document{}, err
	}
	return d, nil
}

// Remove deletes the document slug's file under archRoot and nothing else (R2/R7). An
// absent document is reported as ErrDocumentNotFound rather than a silent success, so
// a typo'd slug is surfaced explicitly. It validates the slug is kebab-case first so a
// malformed slug can never be turned into an unexpected path.
func Remove(archRoot, slug string) error {
	if !IsKebabCase(slug) {
		return fmt.Errorf("architecture slug %q is not valid kebab-case", slug)
	}
	path := filepath.Join(archRoot, fileName(slug))
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %q", ErrDocumentNotFound, slug)
		}
		return err
	}
	return nil
}

// ensureFile creates a file at path with the given deterministic content only if it
// does not already exist. O_EXCL makes the create-or-skip decision atomic and
// non-destructive: an existing file is never truncated, so a document the user has
// refined survives a re-create attempt untouched. It reports whether it created a new
// file (created=false when the path already existed). Duplicated from the specs/
// prompts/workspace helpers of the same name so this package owns its write semantics
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

// writeAtomic writes content to path atomically: it writes to a temporary file in the
// same directory and renames it over path. The rename is atomic on the same
// filesystem, so a reader sees either the old or the new content, never a partial
// write — and a crash mid-write leaves the original intact. Unlike ensureFile this
// *replaces* an existing file, which is exactly what an edit must do (the document is
// known to exist; we are updating it). On any failure the temp file is cleaned up so
// we never litter the workspace. Mirrors specs.writeAtomic.
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
