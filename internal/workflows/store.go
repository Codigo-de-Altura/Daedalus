package workflows

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrWorkflowExists is the sentinel returned by Create/Apply when a workflow with
// the requested name already has a file under the workflows root. Exposed so
// callers can map a duplicate name to an explicit "already exists, not
// overwritten" error via errors.Is, never an overwrite (R4).
var ErrWorkflowExists = errors.New("workflow already exists")

// ErrPhaseExists is returned by the in-memory edit operations when an add or a
// rename would introduce a second phase with an id already in use. A phase id is
// the DAG node key and must be unique within the workflow.
var ErrPhaseExists = errors.New("phase already exists")

// ErrPhaseNotFound is returned by the in-memory edit operations when an edit or
// remove names a phase id that the workflow does not contain.
var ErrPhaseNotFound = errors.New("phase not found")

// CreatePlan is the result of planning a Create: the resolved workflow, the file
// it will land in and the fully rendered bytes — computed without touching the
// filesystem. Mirroring the prompts Plan/Apply split, the content is captured at
// plan time so a `--preview` and the subsequent Apply describe identical bytes
// (R4), and validation happens before any I/O.
type CreatePlan struct {
	// Workflow is the validated workflow to be written.
	Workflow Workflow
	// Path is the absolute file the workflow will be written to
	// (`<root>/<name>.yaml`).
	Path string
	// Content is the rendered file content captured at plan time.
	Content string
}

// PlanCreate validates a workflow and computes the plan to persist it under
// workflowsRoot, without writing anything (R4/R7). It rejects an invalid name or
// an invalid workflow here — before any I/O — with the rich, actionable
// *ValidationError (or a plain name error), so a bad name or a malformed phase
// never reaches the filesystem. It does NOT check for an existing file: that is
// an Apply-time, non-destructive concern (so a preview can be shown even when the
// name is taken, and the create-or-fail decision stays atomic).
func PlanCreate(workflowsRoot string, w Workflow) (*CreatePlan, error) {
	if !IsKebabCase(w.Name) {
		return nil, fmt.Errorf("workflow name %q is not valid kebab-case", w.Name)
	}
	if err := w.Validate(); err != nil {
		return nil, err
	}
	return &CreatePlan{
		Workflow: w,
		Path:     filepath.Join(workflowsRoot, fileName(w.Name)),
		Content:  Render(w),
	}, nil
}

// Apply writes the planned workflow file non-destructively: it creates the
// workflows directory if needed and the file only if it does not already exist
// (O_CREATE|O_EXCL). A workflow name that already has a file is reported as
// ErrWorkflowExists rather than clobbered (R4). Because the content was rendered
// deterministically at plan time, re-creating the same workflow into a clean
// workspace yields byte-identical bytes (R4).
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
		return fmt.Errorf("%w: %q", ErrWorkflowExists, p.Workflow.Name)
	}
	return nil
}

// Create validates and persists a new workflow under workflowsRoot in one call,
// for callers that do not need to preview the content first. It is
// non-destructive: a duplicate name returns ErrWorkflowExists and the existing
// file is left untouched (R4). Callers that want a preview use PlanCreate then
// Apply.
func Create(workflowsRoot string, w Workflow) error {
	plan, err := PlanCreate(workflowsRoot, w)
	if err != nil {
		return err
	}
	return plan.Apply()
}

// List returns the workflows persisted under workflowsRoot as name+phase-count
// entries, sorted by name (R4). A non-existent workflows directory is treated as
// "no workflows" (empty list), not an error, so listing a freshly initialized
// workspace is well-defined. Files that are not `*.yaml`, and any file that fails
// to parse, are skipped rather than aborting the whole listing — a single corrupt
// file must not hide the rest.
func List(workflowsRoot string) ([]Entry, error) {
	entries, err := os.ReadDir(workflowsRoot)
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
		fname := de.Name()
		if !strings.HasSuffix(fname, FileExt) {
			continue
		}
		name := strings.TrimSuffix(fname, FileExt)
		// The name is the file identity; if a file's base name is not valid
		// kebab-case it is not one of ours, so skip it rather than fail the listing.
		if !IsKebabCase(name) {
			continue
		}
		w, err := Load(workflowsRoot, name)
		if err != nil {
			// A malformed file is skipped from the listing; surfacing it as a hard
			// error would make `list` unusable whenever one file is hand-corrupted.
			continue
		}
		out = append(out, Entry{Name: w.Name, Phases: len(w.Phases)})
	}

	sortEntries(out)
	return out, nil
}

// EditFunc mutates a loaded workflow in place. It is the unit of a persisted
// edit: Edit loads the workflow, applies the func, validates the result and
// writes it back atomically. Callers compose the in-memory edit operations
// (AddPhase/EditPhase/RemovePhase) inside it; returning an error from the func
// (e.g. ErrPhaseNotFound) aborts the edit before any write.
type EditFunc func(w *Workflow) error

// Edit applies mutate to the persisted workflow name under workflowsRoot and
// persists the result, returning the edited Workflow. It is the
// read-modify-validate-write cycle the spec mandates (R3/R4/R6):
//
//  1. Load the current workflow from disk (Load); an absent workflow is
//     ErrWorkflowNotFound, a corrupt one is ErrMalformedWorkflow.
//  2. Apply mutate to the in-memory model; an error from mutate (e.g. a missing
//     phase) aborts here with nothing written.
//  3. Validate the *result* before any write — an edit that would leave the
//     workflow structurally invalid is rejected here with the actionable
//     *ValidationError and nothing is written (R4).
//  4. Persist atomically (writeAtomic: temp + rename), so a failure mid-write can
//     never leave a half-written file, and ONLY this workflow's file is touched —
//     other workspace files are untouched (R4).
func Edit(workflowsRoot, name string, mutate EditFunc) (Workflow, error) {
	w, err := Load(workflowsRoot, name)
	if err != nil {
		return Workflow{}, err
	}

	if err := mutate(&w); err != nil {
		return Workflow{}, err
	}

	// Validate the result before touching disk. A failing edit returns here with
	// the rich, actionable *ValidationError, leaving the existing file intact (R4).
	if err := w.Validate(); err != nil {
		return Workflow{}, err
	}

	path := filepath.Join(workflowsRoot, fileName(name))
	if err := writeAtomic(path, Render(w)); err != nil {
		return Workflow{}, err
	}
	return w, nil
}

// Remove deletes the workflow name's file under workflowsRoot and nothing else
// (R3). An absent workflow is reported as ErrWorkflowNotFound rather than a silent
// success, so a typo'd name is surfaced explicitly. It validates the name is
// kebab-case first so a malformed name can never be turned into an unexpected path.
func Remove(workflowsRoot, name string) error {
	if !IsKebabCase(name) {
		return fmt.Errorf("workflow name %q is not valid kebab-case", name)
	}
	path := filepath.Join(workflowsRoot, fileName(name))
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %q", ErrWorkflowNotFound, name)
		}
		return err
	}
	return nil
}

// ensureFile creates a file at path with the given deterministic content only if
// it does not already exist. O_EXCL makes the create-or-skip decision atomic and
// non-destructive: an existing file is never truncated, so a workflow the user
// has edited survives a re-create attempt untouched. It reports whether it created
// a new file (created=false when the path already existed). Duplicated from the
// prompts/catalog/workspace helpers of the same name so this package owns its
// write semantics independently.
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
// *replaces* an existing file, which is exactly what an edit must do (the workflow
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
