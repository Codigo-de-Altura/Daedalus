// Package workspace creates and describes the canonical Daedalus workspace,
// the `.daedalus/` directory that is the backend-agnostic source of truth for a
// project's AI structure (agents, prompts, workflows, SDD backlog).
//
// This package owns the *scaffolding* of the workspace: the deterministic
// directory tree and the root artifact files, including their deterministic
// *content* (the daedalus.yaml manifest and the base init.md, see content.go).
// All operations are non-destructive: existing files are never truncated or
// removed, so manually edited artifacts survive a re-run untouched.
//
// Detection is separated from writing. Plan inspects a target and computes what
// the canonical structure is missing without touching the filesystem, so a
// caller can preview the changes before committing to them. Apply (and the
// backwards-compatible Create) then materialize exactly that plan and nothing
// more — turning a re-run over an existing workspace into a non-destructive
// upgrade rather than a blind recreation.
package workspace

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Name is the canonical workspace directory name created inside a target repo.
const Name = ".daedalus"

// Subdirs are the canonical workspace subdirectories, in a fixed, deterministic
// order matching init.md §4.2 / PRD §8.2.
var Subdirs = []string{
	"agents",
	"prompts",
	"workflows",
	"specs",
	"architecture",
	"epics",
	"tickets",
	"docs",
	".state",
}

// RootArtifacts are the files placed at the root of the workspace, in fixed
// deterministic order. Their content is generated deterministically from the
// target (see content.go / artifactContent); this package guarantees both their
// existence and their initial content as part of the scaffolding.
var RootArtifacts = []string{
	"daedalus.yaml",
	"init.md",
}

// Plan is the result of detecting an existing (or absent) workspace against the
// canonical structure. It is a pure description of the intended changes: holding
// a Plan performs no writes, so callers can present it as a preview before
// deciding to Apply it. The Missing* slices are in canonical (deterministic)
// order and contain only what is absent — never anything that already exists,
// which is what makes the eventual upgrade non-destructive.
type Plan struct {
	// Root is the target repository directory the plan was computed for.
	Root string
	// Path is the workspace path (`<root>/.daedalus`).
	Path string
	// ProjectName is the deterministic project name derived from Root (the base
	// name of its absolute path). It is the input to the manifest/init.md content
	// generators, captured at detection time so Apply renders the same content a
	// preview would describe.
	ProjectName string
	// WorkspaceExisted is true when the workspace directory was already present
	// at detection time. It is the signal that distinguishes a from-scratch
	// creation from an upgrade over an existing workspace.
	WorkspaceExisted bool
	// MissingDirs and MissingFiles are the canonical subdirectories and root
	// artifacts that are absent and would be created by Apply, in deterministic
	// order. Paths are workspace-relative (e.g. "agents", "init.md").
	MissingDirs  []string
	MissingFiles []string
}

// IsEmpty reports whether the plan would create nothing, i.e. the existing
// structure is already complete. A re-run that yields an empty plan is the
// idempotent case: there is nothing to upgrade.
func (p *Plan) IsEmpty() bool {
	return len(p.MissingDirs) == 0 && len(p.MissingFiles) == 0
}

// Result describes what Apply (or Create) produced, so callers can report it to
// the user.
type Result struct {
	// Path is the workspace path (`<root>/.daedalus`).
	Path string
	// AlreadyExisted is true when the workspace directory was present before the
	// call. When true and the created slices are empty, the run was a no-op
	// upgrade (idempotent re-run); when true with created entries, it was a
	// non-destructive upgrade that completed the structure.
	AlreadyExisted bool
	// CreatedDirs and CreatedFiles list the entries actually created (relative to
	// the workspace), in deterministic order.
	CreatedDirs  []string
	CreatedFiles []string
}

// Detect inspects root and computes what the canonical `.daedalus/` structure is
// missing, without writing anything. It is the detection half of the workflow
// (ticket 01-02): callers use it to preview an upgrade before applying it, and
// Apply/Create reuse it so detection logic is never duplicated. The returned
// *Plan is a pure description; nothing is created until Plan.Apply runs.
//
// A subdirectory or root artifact is "missing" when it is absent. An entry that
// exists but has the wrong type (e.g. a file where a directory is expected) is
// reported as an error rather than silently planned over, because materializing
// it would require a destructive overwrite.
func Detect(root string) (*Plan, error) {
	wsPath := filepath.Join(root, Name)
	p := &Plan{Root: root, Path: wsPath, ProjectName: deriveProjectName(root)}

	switch info, err := os.Stat(wsPath); {
	case err == nil:
		if !info.IsDir() {
			return nil, fmt.Errorf("%s exists but is not a directory", wsPath)
		}
		p.WorkspaceExisted = true
	case errors.Is(err, fs.ErrNotExist):
		// Fresh target: everything below will be reported as missing.
	default:
		return nil, err
	}

	for _, sub := range Subdirs {
		exists, err := dirExists(filepath.Join(wsPath, sub))
		if err != nil {
			return nil, err
		}
		if !exists {
			p.MissingDirs = append(p.MissingDirs, sub)
		}
	}

	for _, name := range RootArtifacts {
		exists, err := fileExists(filepath.Join(wsPath, name))
		if err != nil {
			return nil, err
		}
		if !exists {
			p.MissingFiles = append(p.MissingFiles, name)
		}
	}

	return p, nil
}

// Apply materializes the plan: it creates exactly the missing directories and
// root artifacts it describes and nothing else. It is non-destructive — existing
// directories are reused and existing files are never truncated or removed — and
// idempotent: applying an empty plan is a no-op. Because Apply only acts on the
// entries Detect flagged as absent, any manually edited content is preserved.
func (p *Plan) Apply() (*Result, error) {
	res := &Result{Path: p.Path, AlreadyExisted: p.WorkspaceExisted}

	if err := os.MkdirAll(p.Path, 0o755); err != nil {
		return nil, err
	}

	for _, sub := range p.MissingDirs {
		if err := os.MkdirAll(filepath.Join(p.Path, sub), 0o755); err != nil {
			return nil, err
		}
		res.CreatedDirs = append(res.CreatedDirs, sub)
	}

	for _, name := range p.MissingFiles {
		// Generate deterministic content for the artifact; unknown artifacts fall
		// back to empty so adding a new root file never silently breaks.
		content, _ := artifactContent(name, p.ProjectName)
		// O_EXCL guards against a TOCTOU race: if the file appeared between Detect
		// and Apply we skip it rather than truncate, staying non-destructive.
		created, err := ensureFile(filepath.Join(p.Path, name), content)
		if err != nil {
			return nil, err
		}
		if created {
			res.CreatedFiles = append(res.CreatedFiles, name)
		}
	}

	return res, nil
}

// Create scaffolds the canonical `.daedalus/` workspace under root. It is
// idempotent and non-destructive: existing directories are reused and existing
// files are left untouched. The traversal order is fixed so repeated runs over
// identical inputs yield identical structures.
//
// Create is a convenience that detects then applies in one call, kept for
// callers that do not need to preview the changes first. New callers that want a
// preview should use Detect followed by Plan.Apply.
func Create(root string) (*Result, error) {
	plan, err := Detect(root)
	if err != nil {
		return nil, err
	}
	return plan.Apply()
}

// dirExists reports whether path exists and is a directory. Absence is not an
// error (false). An entry that exists but is not a directory is an error,
// because creating the directory there would clash with existing content.
func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if !info.IsDir() {
			return false, fmt.Errorf("%s exists but is not a directory", path)
		}
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

// fileExists reports whether path exists (as a non-directory). Absence is not an
// error (false). A directory where a root artifact is expected is an error,
// because the structures are incompatible.
func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if info.IsDir() {
			return false, fmt.Errorf("%s exists but is a directory, expected a file", path)
		}
		return true, nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

// ensureFile creates a file at path with the given deterministic content only if
// it does not already exist. O_EXCL makes the create-or-skip decision atomic and
// non-destructive: an existing file is never truncated or overwritten, so manual
// edits survive a re-run untouched. It reports whether it created a new file
// (created=false when the path already existed).
func ensureFile(path, content string) (bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, err
	}
	// Write the content, then close. We must not defer Close here: a deferred
	// Close would mask a Write error and we would lose the close error on the
	// happy path. Write first, capture its error, then close and surface
	// whichever error occurred (Write takes precedence, since a failed Write
	// leaves the file in a bad state regardless of how Close goes).
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
