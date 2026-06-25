package linters

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// lintAgents loads and validates every agent under the workspace's agents root,
// reusing catalog.Load (parse) and catalog.Agent.Validate (schema). It returns the
// set of agent ids that loaded AND validated cleanly — the basis for the workflow
// linter's knownAgents predicate — alongside the findings.
//
// A malformed source (catalog.Load failure) becomes one controlled finding and the
// agent is excluded from the valid set; a schema-invalid agent yields one finding
// per schema error, each naming the field, what was observed and what was expected
// (R2/R5). Neither path panics (R6). An absent agents directory is not an error: a
// workspace with no agents simply has no agent findings.
func (w Workspace) lintAgents() (validIDs map[string]struct{}, findings []Finding, err error) {
	validIDs = make(map[string]struct{})

	ids, err := w.agentIDs()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return validIDs, nil, nil // no agents directory ⇒ nothing to lint
		}
		return nil, nil, err
	}

	for _, id := range ids {
		location := w.agentLocation(id)

		a, loadErr := catalog.Load(w.AgentsRoot, id)
		if loadErr != nil {
			findings = append(findings, Finding{
				Family:     FamilyAgent,
				Severity:   SeverityError,
				Location:   location,
				Definition: id,
				Rule:       "malformed",
				Reason:     "agent definition could not be loaded: " + loadErr.Error(),
			})
			continue
		}

		if vErr := a.Validate(); vErr != nil {
			findings = append(findings, agentSchemaFindings(id, location, vErr)...)
			continue
		}

		validIDs[id] = struct{}{}
	}

	return validIDs, findings, nil
}

// agentSchemaFindings maps a catalog.ValidationError to one actionable Finding per
// schema error. The catalog validator already collects and stable-sorts its
// findings, so iterating them preserves a deterministic order. If the error is not
// a *catalog.ValidationError (it always is for Validate, but we never assume), a
// single controlled finding carries its text so nothing is lost and nothing panics.
func agentSchemaFindings(id, location string, err error) []Finding {
	var ve *catalog.ValidationError
	if !errors.As(err, &ve) {
		return []Finding{{
			Family:     FamilyAgent,
			Severity:   SeverityError,
			Location:   location,
			Definition: id,
			Rule:       "schema",
			Reason:     err.Error(),
		}}
	}

	findings := make([]Finding, 0, len(ve.Errors))
	for _, se := range ve.Errors {
		findings = append(findings, Finding{
			Family:     FamilyAgent,
			Severity:   SeverityError,
			Location:   location,
			Definition: id,
			Spot:       se.Field,
			Rule:       "schema",
			Reason:     "observed " + se.Observed + "; expected " + se.Expected,
		})
	}
	return findings
}

// agentLocation renders an agent's workspace-relative directory as the finding
// location (e.g. ".daedalus/agents/analyst"), slash-form so it is identical across
// operating systems.
func (w Workspace) agentLocation(id string) string {
	return filepath.ToSlash(filepath.Join(workspace.Name, catalog.AgentsDir, id))
}

// catalogAgentIDs lists the agent ids materialized under agentsRoot — the
// subdirectory names, sorted — so linting is deterministic. Non-directory entries
// are ignored (each agent lives in its own directory). A directory whose name is
// not valid kebab-case is still surfaced to catalog.Load, which reports it as a
// malformed source rather than silently dropping it, so a misnamed agent is not
// hidden. The not-exist case is propagated so the caller can treat "no agents dir"
// as "no agents".
func catalogAgentIDs(agentsRoot string) ([]string, error) {
	entries, err := os.ReadDir(agentsRoot)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ids = append(ids, e.Name())
	}
	sort.Strings(ids)
	return ids, nil
}
