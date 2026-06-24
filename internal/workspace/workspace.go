// Package workspace creates and describes the canonical Daedalus workspace,
// the `.daedalus/` directory that is the backend-agnostic source of truth for a
// project's AI structure (agents, prompts, workflows, SDD backlog).
//
// This package owns the *scaffolding* of the workspace: the deterministic
// directory tree and the root artifact files. The deterministic *content* of
// those artifacts (daedalus.yaml, init.md) belongs to a later ticket; here they
// are created empty when missing so the structure is complete. All operations
// are non-destructive: existing files are never truncated or removed.
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

// RootArtifacts are the files placed at the root of the workspace. Their
// deterministic content is produced by a later ticket; this package only
// guarantees their existence as part of the scaffolding.
var RootArtifacts = []string{
	"daedalus.yaml",
	"init.md",
}

// Result describes what Create produced, so callers can report it to the user.
type Result struct {
	// Path is the workspace path (`<root>/.daedalus`).
	Path string
	// AlreadyExisted is true when the workspace directory was present before the
	// call. Detection and upgrade/merge of an existing workspace is a separate
	// concern (ticket 01-02); Create only reports the fact here.
	AlreadyExisted bool
	// CreatedDirs and CreatedFiles list the entries Create actually created
	// (relative to the workspace), in deterministic order.
	CreatedDirs  []string
	CreatedFiles []string
}

// Create scaffolds the canonical `.daedalus/` workspace under root. It is
// idempotent and non-destructive: existing directories are reused and existing
// files are left untouched. The traversal order is fixed so repeated runs over
// identical inputs yield identical structures.
func Create(root string) (*Result, error) {
	wsPath := filepath.Join(root, Name)
	res := &Result{Path: wsPath}

	switch info, err := os.Stat(wsPath); {
	case err == nil:
		if !info.IsDir() {
			return nil, fmt.Errorf("%s exists but is not a directory", wsPath)
		}
		res.AlreadyExisted = true
	case errors.Is(err, fs.ErrNotExist):
		// fresh workspace; nothing to note
	default:
		return nil, err
	}

	if err := os.MkdirAll(wsPath, 0o755); err != nil {
		return nil, err
	}

	for _, sub := range Subdirs {
		path := filepath.Join(wsPath, sub)
		existed, err := isDir(path)
		if err != nil {
			return nil, err
		}
		if err := os.MkdirAll(path, 0o755); err != nil {
			return nil, err
		}
		if !existed {
			res.CreatedDirs = append(res.CreatedDirs, sub)
		}
	}

	for _, name := range RootArtifacts {
		created, err := ensureFile(filepath.Join(wsPath, name))
		if err != nil {
			return nil, err
		}
		if created {
			res.CreatedFiles = append(res.CreatedFiles, name)
		}
	}

	return res, nil
}

// isDir reports whether path exists and is a directory, treating absence as a
// non-error (false).
func isDir(path string) (bool, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		return info.IsDir(), nil
	case errors.Is(err, fs.ErrNotExist):
		return false, nil
	default:
		return false, err
	}
}

// ensureFile creates an empty file at path only if it does not already exist.
// It never truncates an existing file (non-destructive) and reports whether it
// created a new one.
func ensureFile(path string) (bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, err
	}
	return true, f.Close()
}
