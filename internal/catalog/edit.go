package catalog

import (
	"fmt"
	"os"
	"path/filepath"
)

// EditSpec describes the changes to apply to a workspace agent's canonical
// definition. Every field is optional so a caller edits only what it names; the
// presence of a change is signaled by the *Set* booleans rather than by zero
// values, because "set the role to the empty string" and "leave the role
// untouched" must be distinguishable (the former is an invalid edit we must
// reject, not a silent no-op).
type EditSpec struct {
	// SetRole, when true, replaces the agent's role with Role.
	SetRole bool
	Role    string
	// SetPrompt, when true, replaces the agent's prompt with Prompt.
	SetPrompt bool
	Prompt    string
	// SetParams are parameters to add or overwrite, in caller order. A key already
	// present is updated in place (preserving its position) so a re-set does not
	// reorder the file; a new key is appended. Values are typed by the caller.
	SetParams []Param
	// RemoveParams are parameter keys to delete. Removing an absent key is a no-op
	// (idempotent), not an error, so repeated edits are safe.
	RemoveParams []string
}

// IsEmpty reports whether the spec would change nothing. The CLI uses it to turn
// an edit with no flags into a usage error rather than rewriting a file
// identically (which would still be a confusing no-op to the user).
func (e EditSpec) IsEmpty() bool {
	return !e.SetRole && !e.SetPrompt && len(e.SetParams) == 0 && len(e.RemoveParams) == 0
}

// Edit applies spec to the workspace agent id under agentsRoot and persists the
// result, returning the edited Agent. It is the read-modify-validate-write cycle
// the spec mandates (R3/R5/CA3/CA5):
//
//  1. Load the current canonical definition from disk (Load).
//  2. Apply the requested changes to an in-memory copy.
//  3. Validate the *result* structurally before any write — an edit that would
//     leave the definition invalid (e.g. an empty role) is rejected here with an
//     actionable error and nothing is written (R5/CA5).
//  4. Persist atomically: each file is written to a temp sibling and renamed over
//     the original, so a failure mid-write can never leave a half-written,
//     corrupt definition on disk.
//
// Because the agent is re-rendered with the deterministic renderer, the on-disk
// result is diff-friendly and stable (R6).
func (c *Catalog) Edit(agentsRoot, id string, spec EditSpec) (Agent, error) {
	a, err := Load(agentsRoot, id)
	if err != nil {
		return Agent{}, err
	}

	applyEdit(&a, spec)

	// Validate the resulting definition before touching disk. This is the
	// structural stand-in for the formal canonical schema (ticket-02-04, not yet
	// implemented): id/role/prompt non-empty, kebab-case id, valid params. A
	// failing edit returns here, leaving the existing files untouched (R5/CA5).
	if err := a.Validate(); err != nil {
		return Agent{}, fmt.Errorf("invalid edit to agent %q: %w", id, err)
	}

	dir := filepath.Join(agentsRoot, id)
	if err := writeAtomic(filepath.Join(dir, DefinitionFileName), renderDefinition(a)); err != nil {
		return Agent{}, err
	}
	if err := writeAtomic(filepath.Join(dir, PromptFileName), renderPrompt(a)); err != nil {
		return Agent{}, err
	}

	return a, nil
}

// applyEdit mutates a in place according to spec. Parameter edits preserve the
// existing order — an updated key keeps its slot, a new key is appended, a
// removed key is filtered out — so an edit produces a minimal, readable diff
// rather than reshuffling the whole `parameters` block.
func applyEdit(a *Agent, spec EditSpec) {
	if spec.SetRole {
		a.Role = spec.Role
	}
	if spec.SetPrompt {
		a.Prompt = spec.Prompt
	}

	for _, set := range spec.SetParams {
		updated := false
		for i := range a.Params {
			if a.Params[i].Key == set.Key {
				a.Params[i] = set
				updated = true
				break
			}
		}
		if !updated {
			a.Params = append(a.Params, set)
		}
	}

	if len(spec.RemoveParams) > 0 {
		remove := make(map[string]struct{}, len(spec.RemoveParams))
		for _, k := range spec.RemoveParams {
			remove[k] = struct{}{}
		}
		kept := a.Params[:0]
		for _, p := range a.Params {
			if _, drop := remove[p.Key]; drop {
				continue
			}
			kept = append(kept, p)
		}
		a.Params = kept
	}
}

// writeAtomic writes content to path atomically: it writes to a temporary file in
// the same directory and renames it over path. The rename is atomic on the same
// filesystem, so a concurrent reader sees either the old or the new content, never
// a partial write — and a crash mid-write leaves the original intact. Unlike
// ensureFile this *replaces* an existing file, which is exactly what an edit must
// do (the agent is known to exist; we are updating it, not creating it). On any
// failure the temp file is cleaned up so we never litter the workspace.
func writeAtomic(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we bail before a successful rename; after a
	// successful rename tmpName no longer exists so the Remove is a harmless no-op.
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
