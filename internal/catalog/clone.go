package catalog

import "fmt"

// ClonePlanFor computes the plan to clone a built-in catalog agent (sourceID)
// into the workspace under a new identifier (destID), without touching the
// filesystem. The clone is a *copy* of the canonical definition: the resulting
// MaterializePlan renders destID's own files, so applying it writes an
// independent definition the user can edit freely without affecting the built-in
// (which is an immutable Go literal) or any other agent (R1/R2/CA1/CA2).
//
// It reuses the same Plan/Apply machinery as MaterializePlanFor, so cloning is
// non-destructive (O_EXCL) and deterministic for free (R4/R6/CA4): a clone over
// an existing destID reports the conflict via Skipped rather than overwriting.
// destID is validated as kebab-case here (R1/CA6) before any rendering.
func (c *Catalog) ClonePlanFor(agentsRoot, sourceID, destID string) (*MaterializePlan, error) {
	if !IsKebabCase(destID) {
		return nil, fmt.Errorf("destination agent id %q is not valid kebab-case", destID)
	}

	// Get returns a defensive copy of the source agent, so re-stamping its ID
	// cannot mutate the catalog's source of truth — the foundation of clone
	// independence (R2/CA2).
	src, err := c.Get(sourceID)
	if err != nil {
		return nil, err
	}
	src.ID = destID

	// Re-validate under the new identity before planning any write, so an invalid
	// destination never produces files.
	if err := src.Validate(); err != nil {
		return nil, err
	}

	return planMaterialize(agentsRoot, src), nil
}

// Clone is the convenience that plans then applies a clone in one call, for
// callers that do not need to preview the content first. Callers that want a
// preview (the CLI --preview, a TUI diff) use ClonePlanFor then Apply.
func (c *Catalog) Clone(agentsRoot, sourceID, destID string) (*MaterializeResult, error) {
	plan, err := c.ClonePlanFor(agentsRoot, sourceID, destID)
	if err != nil {
		return nil, err
	}
	return plan.Apply()
}
