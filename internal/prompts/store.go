package prompts

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrPromptExists is the sentinel returned by Create/Apply when a prompt with the
// requested id already has a file under the prompts root. Exposed so callers can
// map a duplicate id to an explicit "already exists, not overwritten" error via
// errors.Is, never an overwrite (R4/R8).
var ErrPromptExists = errors.New("prompt already exists")

// CreatePlan is the result of planning a Create: the resolved prompt, the file it
// will land in and the fully rendered bytes — computed without touching the
// filesystem. Mirroring the catalog's Plan/Apply split, the content is captured
// at plan time so a `--preview` and the subsequent Apply describe identical bytes
// (R5), and validation happens before any I/O.
type CreatePlan struct {
	// Prompt is the validated prompt to be written.
	Prompt Prompt
	// Path is the absolute file the prompt will be written to (`<root>/<id>.md`).
	Path string
	// Content is the rendered file content captured at plan time.
	Content string
}

// PlanCreate validates a prompt and computes the plan to persist it under
// promptsRoot, without writing anything (R5/R8). It rejects an invalid prompt
// here — before any I/O — with the rich, actionable *ValidationError, so a bad id
// or empty title never reaches the filesystem. It does NOT check for an existing
// file: that is an Apply-time, non-destructive concern (so a preview can be shown
// even when the id is taken, and the create-or-fail decision stays atomic).
func PlanCreate(promptsRoot string, p Prompt) (*CreatePlan, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &CreatePlan{
		Prompt:  p,
		Path:    filepath.Join(promptsRoot, fileName(p.ID)),
		Content: Render(p),
	}, nil
}

// Apply writes the planned prompt file non-destructively: it creates the prompts
// directory if needed and the file only if it does not already exist
// (O_CREATE|O_EXCL). A prompt id that already has a file is reported as
// ErrPromptExists rather than clobbered (R4/R8). Because the content was rendered
// deterministically at plan time, re-creating the same prompt into a clean
// workspace yields byte-identical bytes (R5).
func (p *CreatePlan) Apply() error {
	// MkdirAll is non-destructive on an existing directory; the O_EXCL write below
	// is what protects the actual file content.
	if err := os.MkdirAll(filepath.Dir(p.Path), 0o755); err != nil {
		return err
	}
	created, err := ensureFile(p.Path, p.Content)
	if err != nil {
		return err
	}
	if !created {
		return fmt.Errorf("%w: %q", ErrPromptExists, p.Prompt.ID)
	}
	return nil
}

// Create validates and persists a new prompt under promptsRoot in one call, for
// callers that do not need to preview the content first. It is non-destructive: a
// duplicate id returns ErrPromptExists and the existing file is left untouched
// (R4/R8). Callers that want a preview use PlanCreate then Apply.
func Create(promptsRoot string, p Prompt) error {
	plan, err := PlanCreate(promptsRoot, p)
	if err != nil {
		return err
	}
	return plan.Apply()
}

// List returns the prompts persisted under promptsRoot as id+kind+title entries,
// sorted by id (R3/R5). When filter is non-empty it returns only prompts of that
// kind (R3/R6); an empty filter returns all. A non-existent prompts directory is
// treated as "no prompts" (empty list), not an error, so listing a freshly
// initialized workspace is well-defined. Files that are not `*.md`, and any file
// that fails to parse, are skipped rather than aborting the whole listing — a
// single corrupt file must not hide the rest.
func List(promptsRoot string, filter Kind) ([]Entry, error) {
	entries, err := os.ReadDir(promptsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []Entry{}, nil
		}
		return nil, err
	}

	out := make([]Entry, 0, len(entries))
	for _, de := range entries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.HasSuffix(name, FileExt) {
			continue
		}
		id := strings.TrimSuffix(name, FileExt)
		// The id is the file identity; if a file's base name is not valid kebab-case
		// it is not one of ours, so skip it rather than fail the listing.
		if !IsKebabCase(id) {
			continue
		}
		p, err := Load(promptsRoot, id)
		if err != nil {
			// A malformed file is skipped from the listing; surfacing it as a hard
			// error would make `list` unusable whenever one file is hand-corrupted.
			continue
		}
		if filter != "" && p.Kind != filter {
			continue
		}
		out = append(out, Entry{ID: p.ID, Kind: p.Kind, Title: p.Title})
	}

	sortEntries(out)
	return out, nil
}

// EditSpec describes the changes to apply to a persisted prompt. Every field is
// optional so a caller edits only what it names; the presence of a change is
// signaled by the *Set* booleans rather than by zero values, because "set the
// title to the empty string" (an invalid edit we must reject) and "leave the
// title untouched" (a no-op) must be distinguishable. Kind is intentionally not
// editable: a prompt's class is part of its identity, and changing it is better
// modeled as create+remove than as an in-place mutation.
type EditSpec struct {
	// SetTitle, when true, replaces the prompt's title with Title.
	SetTitle bool
	Title    string
	// SetDescription, when true, replaces the prompt's description with
	// Description (which may be empty to clear it).
	SetDescription bool
	Description    string
	// SetBody, when true, replaces the prompt's body with Body.
	SetBody bool
	Body    string
}

// IsEmpty reports whether the spec would change nothing. The CLI uses it to turn
// an edit with no flags into a usage error rather than rewriting a file
// identically.
func (e EditSpec) IsEmpty() bool {
	return !e.SetTitle && !e.SetDescription && !e.SetBody
}

// Edit applies spec to the persisted prompt id under promptsRoot and persists the
// result, returning the edited Prompt. It is the read-modify-validate-write cycle
// the spec mandates (R3/R5):
//
//  1. Load the current prompt from disk (Load); an absent prompt is
//     ErrPromptNotFound.
//  2. Apply the requested changes to an in-memory copy.
//  3. Validate the *result* before any write — an edit that would leave the
//     prompt invalid (e.g. an empty title) is rejected here with an actionable
//     error and nothing is written (R5).
//  4. Persist atomically (writeAtomic: temp + rename), so a failure mid-write
//     can never leave a half-written file, and ONLY this prompt's file is
//     touched — other workspace files are untouched (R5).
func Edit(promptsRoot, id string, spec EditSpec) (Prompt, error) {
	p, err := Load(promptsRoot, id)
	if err != nil {
		return Prompt{}, err
	}

	if spec.SetTitle {
		p.Title = spec.Title
	}
	if spec.SetDescription {
		p.Description = spec.Description
	}
	if spec.SetBody {
		p.Body = spec.Body
	}

	// Validate the result before touching disk. A failing edit returns here with
	// the rich, actionable *ValidationError, leaving the existing file intact (R5).
	if err := p.Validate(); err != nil {
		return Prompt{}, err
	}

	path := filepath.Join(promptsRoot, fileName(id))
	if err := writeAtomic(path, Render(p)); err != nil {
		return Prompt{}, err
	}
	return p, nil
}

// Remove deletes the prompt id's file under promptsRoot and nothing else (R3/R5).
// An absent prompt is reported as ErrPromptNotFound rather than a silent success,
// so a typo'd id is surfaced explicitly (R8). It validates the id is kebab-case
// first so a malformed id can never be turned into an unexpected path.
func Remove(promptsRoot, id string) error {
	if !IsKebabCase(id) {
		return fmt.Errorf("prompt id %q is not valid kebab-case", id)
	}
	path := filepath.Join(promptsRoot, fileName(id))
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %q", ErrPromptNotFound, id)
		}
		return err
	}
	return nil
}

// ensureFile creates a file at path with the given deterministic content only if
// it does not already exist. O_EXCL makes the create-or-skip decision atomic and
// non-destructive: an existing file is never truncated, so a prompt the user has
// edited survives a re-create attempt untouched. It reports whether it created a
// new file (created=false when the path already existed). Duplicated from the
// catalog/workspace helpers of the same name so this package owns its write
// semantics independently.
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
// *replaces* an existing file, which is exactly what an edit must do (the prompt
// is known to exist; we are updating it). On any failure the temp file is cleaned
// up so we never litter the workspace.
func writeAtomic(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we bail before a successful rename; after a successful
	// rename tmpName no longer exists so the Remove is a harmless no-op.
	defer os.Remove(tmpName)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	return nil
}
