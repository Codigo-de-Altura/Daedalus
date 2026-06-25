package linters

import (
	"errors"
	"path/filepath"

	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// lintManifest reads and validates the workspace manifest (`daedalus.yaml`),
// reusing workspace.ReadManifest (parse) and workspace.ValidateManifest (schema:
// name/version/backends/conventions). An absent manifest and a malformed (unparsable)
// manifest each become one controlled finding (R4/R5/R6) rather than aborting the
// whole report or panicking; a parsed-but-invalid manifest yields one finding per
// schema problem, each naming the field, what was observed and what was expected.
func (w Workspace) lintManifest() ([]Finding, error) {
	location := w.manifestLocation()

	m, err := workspace.ReadManifest(w.Repo)
	if err != nil {
		// Both "not found" and "malformed" are reported as findings: linting a
		// workspace whose manifest is missing or corrupt should say so actionably, not
		// fail with an opaque error. A genuinely unexpected I/O error (neither sentinel)
		// is propagated so the caller can surface it.
		switch {
		case errors.Is(err, workspace.ErrManifestNotFound):
			return []Finding{{
				Family:     FamilyManifest,
				Severity:   SeverityError,
				Location:   location,
				Definition: manifestDefinitionName(),
				Rule:       "missing",
				Reason:     "no workspace manifest found; run 'daedalus init' to scaffold it",
			}}, nil
		case errors.Is(err, workspace.ErrManifestMalformed):
			return []Finding{{
				Family:     FamilyManifest,
				Severity:   SeverityError,
				Location:   location,
				Definition: manifestDefinitionName(),
				Rule:       "malformed",
				Reason:     "manifest could not be parsed: " + err.Error(),
			}}, nil
		default:
			return nil, err
		}
	}

	vErr := workspace.ValidateManifest(m)
	if vErr == nil {
		return nil, nil
	}

	findings := make([]Finding, 0, len(vErr.Findings))
	for _, f := range vErr.Findings {
		findings = append(findings, Finding{
			Family:     FamilyManifest,
			Severity:   manifestSeverity(f.Severity),
			Location:   location,
			Definition: manifestDefinitionName(),
			Spot:       f.Field,
			Rule:       "schema",
			Reason:     "observed " + f.Observed + "; expected " + f.Expected,
		})
	}
	return findings, nil
}

// manifestSeverity maps a workspace manifest severity onto a linter severity. The
// two share the same error/warning split, so the mapping is direct; keeping it
// explicit means a future divergence in either package is a deliberate edit here,
// not a silent mismatch.
func manifestSeverity(s workspace.ManifestSeverity) Severity {
	if s == workspace.ManifestError {
		return SeverityError
	}
	return SeverityWarning
}

// manifestDefinitionName is the human identity used for manifest findings: the
// manifest's file name, so a report names the concrete file the user edits.
func manifestDefinitionName() string {
	return workspace.RootArtifacts[0]
}

// manifestLocation renders the manifest's workspace-relative path as the finding
// location (e.g. ".daedalus/daedalus.yaml"), slash-form for cross-platform stability.
func (w Workspace) manifestLocation() string {
	return filepath.ToSlash(filepath.Join(workspace.Name, workspace.RootArtifacts[0]))
}
