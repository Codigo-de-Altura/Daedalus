package linters

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// lintWorkflows loads and validates every workflow under the workspace's workflows
// root, reusing workflows.Load (parse), workflows.Workflow.Validate (structural
// schema: unique kebab phase ids, non-empty agent/gate) and
// workflows.Workflow.ValidateGraph (semantic DAG: cycles, missing artifacts,
// unknown agents, dangling depends_on). knownAgents is the agent-existence
// predicate the semantic validator uses to catch references to non-existent agents
// (R3).
//
// A malformed source (workflows.Load failure) becomes one controlled finding (R6);
// structural and semantic problems each yield one finding apiece, named to the
// offending phase/field (R3/R4/R5). No path panics. An absent workflows directory
// is not an error.
func (w Workspace) lintWorkflows(knownAgents func(string) bool) ([]Finding, error) {
	names, err := workflowNames(w.WorkflowsRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil // no workflows directory ⇒ nothing to lint
		}
		return nil, err
	}

	var findings []Finding
	for _, name := range names {
		location := w.workflowLocation(name)

		wf, loadErr := workflows.Load(w.WorkflowsRoot, name)
		if loadErr != nil {
			findings = append(findings, Finding{
				Family:     FamilyWorkflow,
				Severity:   SeverityError,
				Location:   location,
				Definition: name,
				Rule:       "malformed",
				Reason:     "workflow could not be loaded: " + loadErr.Error(),
			})
			continue
		}

		findings = append(findings, workflowSchemaFindings(name, location, wf.Validate())...)
		findings = append(findings, workflowGraphFindings(name, location, wf.ValidateGraph(knownAgents))...)
	}
	return findings, nil
}

// workflowSchemaFindings maps a workflow's structural *ValidationError to one
// actionable Finding per schema error, each scoped to the offending phase and
// field. A nil error (valid structure) yields nothing. The structural validator
// already stable-sorts its findings, so iteration order is deterministic.
func workflowSchemaFindings(name, location string, err error) []Finding {
	if err == nil {
		return nil
	}
	var ve *workflows.ValidationError
	if !errors.As(err, &ve) {
		return []Finding{{
			Family:     FamilyWorkflow,
			Severity:   SeverityError,
			Location:   location,
			Definition: name,
			Rule:       "schema",
			Reason:     err.Error(),
		}}
	}

	findings := make([]Finding, 0, len(ve.Errors))
	for _, se := range ve.Errors {
		findings = append(findings, Finding{
			Family:     FamilyWorkflow,
			Severity:   SeverityError,
			Location:   location,
			Definition: name,
			Spot:       workflowSpot(se.Phase, se.Field),
			Rule:       "schema",
			Reason:     "observed " + se.Observed + "; expected " + se.Expected,
		})
	}
	return findings
}

// workflowGraphFindings maps a workflow's semantic *GraphReport to one actionable
// Finding per graph finding, each scoped to the affected phase and carrying the
// finding kind (cycle / missing-artifact / unknown-agent / unknown-dependency) as
// the rule. A valid report yields nothing. The semantic validator already
// stable-sorts its findings, so iteration order is deterministic. A dangling
// depends_on is reported as a warning (it is an additive diagnostic in the owning
// package); the three mandatory classes are errors.
func workflowGraphFindings(name, location string, report *workflows.GraphReport) []Finding {
	if report == nil || report.Valid() {
		return nil
	}
	findings := make([]Finding, 0, len(report.Findings))
	for _, f := range report.Findings {
		findings = append(findings, Finding{
			Family:     FamilyWorkflow,
			Severity:   graphSeverity(f.Kind),
			Location:   location,
			Definition: name,
			Spot:       "phase " + f.Phase,
			Rule:       string(f.Kind),
			Reason:     fmt.Sprintf("observed %q; %s", f.Observed, f.Reason),
		})
	}
	return findings
}

// graphSeverity maps a semantic finding kind to a severity: the three mandatory
// classes (cycle, missing-artifact, unknown-agent) are hard errors; a dangling
// depends_on is an advisory warning, matching the owning package's framing of it as
// an additive diagnostic rather than one of the mandatory classes.
func graphSeverity(kind workflows.FindingKind) Severity {
	if kind == workflows.KindUnknownDependency {
		return SeverityWarning
	}
	return SeverityError
}

// workflowSpot renders the precise location of a structural finding: the phase plus
// field when both are present, the field alone for a workflow-level finding.
func workflowSpot(phase, field string) string {
	if strings.TrimSpace(phase) == "" {
		return field
	}
	return fmt.Sprintf("phase %s: %s", phase, field)
}

// workflowLocation renders a workflow's workspace-relative file path as the finding
// location (e.g. ".daedalus/workflows/sdd-default.yaml"), slash-form for
// cross-platform stability.
func (w Workspace) workflowLocation(name string) string {
	return filepath.ToSlash(filepath.Join(workspace.Name, workflows.WorkflowsDir, name+workflows.FileExt))
}

// workflowNames lists the workflow names under workflowsRoot — the base names of
// the `<name>.yaml` files, sorted — so linting is deterministic. Directories and
// non-`.yaml` files are ignored. Unlike workflows.List (which skips a file whose
// base name is not kebab-case as "not one of ours"), the linter still surfaces a
// kebab-invalid name through workflows.Load, which reports it as a malformed source
// — a misnamed workflow must be seen, not hidden. The not-exist case is propagated
// so the caller can treat "no workflows dir" as "no workflows".
func workflowNames(workflowsRoot string) ([]string, error) {
	entries, err := os.ReadDir(workflowsRoot)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		fname := e.Name()
		if !strings.HasSuffix(fname, workflows.FileExt) {
			continue
		}
		names = append(names, strings.TrimSuffix(fname, workflows.FileExt))
	}
	sort.Strings(names)
	return names, nil
}
