package linters

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// newWorkspace scaffolds a fresh `.daedalus/` workspace (manifest + canonical
// layout) under a temp dir and returns the repo root. It is the precondition for a
// lintable workspace; tests then add valid or broken definitions as needed.
func newWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := workspace.Create(root); err != nil {
		t.Fatalf("scaffold workspace: %v", err)
	}
	return root
}

// addBuiltinAgent materializes a valid built-in catalog agent into the workspace.
func addBuiltinAgent(t *testing.T, root, id string) {
	t.Helper()
	agentsRoot := filepath.Join(root, workspace.Name, catalog.AgentsDir)
	if _, err := catalog.Builtin.Materialize(agentsRoot, id); err != nil {
		t.Fatalf("materialize agent %q: %v", id, err)
	}
}

// writeRawAgent materializes an agent directly from raw file bytes so a test can
// craft a malformed (unparsable) or schema-invalid definition the built-in catalog
// never emits.
func writeRawAgent(t *testing.T, root, id, agentYAML, prompt string) {
	t.Helper()
	dir := filepath.Join(root, workspace.Name, catalog.AgentsDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, catalog.DefinitionFileName), []byte(agentYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, catalog.PromptFileName), []byte(prompt), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeWorkflow renders and writes a valid-shaped workflow via the canonical
// renderer (Create validates structure first, so this is only for structurally
// valid workflows whose semantics a test wants to exercise).
func writeWorkflow(t *testing.T, root string, w workflows.Workflow) {
	t.Helper()
	workflowsRoot := filepath.Join(root, workspace.Name, workflows.WorkflowsDir)
	if err := workflows.Create(workflowsRoot, w); err != nil {
		t.Fatalf("create workflow %q: %v", w.Name, err)
	}
}

// writeRawWorkflow writes raw workflow bytes, bypassing validation, so a test can
// craft a malformed file or a structurally invalid one (e.g. duplicate phase ids)
// that Create would reject.
func writeRawWorkflow(t *testing.T, root, name, content string) {
	t.Helper()
	workflowsRoot := filepath.Join(root, workspace.Name, workflows.WorkflowsDir)
	if err := os.MkdirAll(workflowsRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(workflowsRoot, name+workflows.FileExt)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// lint runs the linters over the workspace root, failing on an unexpected I/O error.
func lint(t *testing.T, root string) *Report {
	t.Helper()
	report, err := WorkspaceUnder(root).Lint()
	if err != nil {
		t.Fatalf("Lint: %v", err)
	}
	return report
}

// findOf returns the first finding matching family and rule (and, when non-empty, a
// definition substring), or fails the test. It is how a test pins a specific class
// of finding without depending on exact ordering.
func findOf(t *testing.T, r *Report, family Family, rule, defContains string) Finding {
	t.Helper()
	for _, f := range r.Findings {
		if f.Family != family || f.Rule != rule {
			continue
		}
		if defContains != "" && !strings.Contains(f.Definition, defContains) {
			continue
		}
		return f
	}
	t.Fatalf("no %s finding with rule %q (def~%q); findings=%s", family, rule, defContains, r)
	return Finding{}
}

// TestLintCleanWorkspaceHasNoErrors covers CA6: a workspace with a valid manifest,
// valid agents and a valid workflow that references existing agents passes the
// linters with no error-level findings (no false positives).
func TestLintCleanWorkspaceHasNoErrors(t *testing.T) {
	root := newWorkspace(t)
	addBuiltinAgent(t, root, "analyst")
	addBuiltinAgent(t, root, "architect")

	// A valid 2-phase DAG: architect depends on analyst, consumes its output.
	writeWorkflow(t, root, workflows.Workflow{
		Name: "sdd",
		Phases: []workflows.Phase{
			{ID: "spec", Agent: "analyst", Inputs: []string{"brief"}, Outputs: []string{"spec"}, Gate: "spec-gate", DependsOn: []string{"brief"}},
			{ID: "design", Agent: "architect", Inputs: []string{"spec"}, Outputs: []string{"design"}, Gate: "design-gate", DependsOn: []string{"spec"}},
		},
	})

	report := lint(t, root)
	if report.HasErrors() {
		t.Fatalf("clean workspace reported errors: %s", report)
	}
}

// TestLintInvalidAgentDetected covers CA1: an agent with a missing required field
// (empty prompt) is detected and reported with file, field and expectation.
func TestLintInvalidAgentDetected(t *testing.T) {
	root := newWorkspace(t)
	// Parses (id + role present) but the prompt body is empty ⇒ schema-invalid.
	writeRawAgent(t, root, "hollow", "id: hollow\nrole: tester\nprompt: prompt.md\n", "")

	report := lint(t, root)
	f := findOf(t, report, FamilyAgent, "schema", "hollow")
	if f.Severity != SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
	if !strings.Contains(f.Location, ".daedalus/agents/hollow") {
		t.Errorf("location does not point at the agent: %q", f.Location)
	}
	if f.Spot != catalog.FieldPrompt {
		t.Errorf("spot = %q, want the prompt field", f.Spot)
	}
	if !strings.Contains(f.Reason, "expected") {
		t.Errorf("reason is not actionable (no expectation): %q", f.Reason)
	}
}

// TestLintWorkflowCycleDetected covers CA2: a DAG with a cycle is detected and
// reported actionably, without panicking.
func TestLintWorkflowCycleDetected(t *testing.T) {
	root := newWorkspace(t)
	addBuiltinAgent(t, root, "analyst")
	// a -> b -> a is a cycle. Written raw because Create's structural validation
	// would pass (the cycle is semantic), but we want full control over the edges.
	writeWorkflow(t, root, workflows.Workflow{
		Name: "looped",
		Phases: []workflows.Phase{
			{ID: "a", Agent: "analyst", Inputs: []string{}, Outputs: []string{}, Gate: "g", DependsOn: []string{"b"}},
			{ID: "b", Agent: "analyst", Inputs: []string{}, Outputs: []string{}, Gate: "g", DependsOn: []string{"a"}},
		},
	})

	report := lint(t, root)
	f := findOf(t, report, FamilyWorkflow, "cycle", "looped")
	if f.Severity != SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
	if !strings.Contains(f.Reason, "cycle") {
		t.Errorf("reason does not explain the cycle: %q", f.Reason)
	}
}

// TestLintWorkflowMissingArtifactDetected covers CA3 (artifact half): a phase input
// that no predecessor produces is detected and reported with its phase.
func TestLintWorkflowMissingArtifactDetected(t *testing.T) {
	root := newWorkspace(t)
	addBuiltinAgent(t, root, "analyst")
	writeWorkflow(t, root, workflows.Workflow{
		Name: "gap",
		Phases: []workflows.Phase{
			// Consumes "ghost", which no predecessor outputs and is not the initial artifact.
			{ID: "only", Agent: "analyst", Inputs: []string{"ghost"}, Outputs: []string{}, Gate: "g", DependsOn: []string{}},
		},
	})

	report := lint(t, root)
	f := findOf(t, report, FamilyWorkflow, "missing-artifact", "gap")
	if !strings.Contains(f.Reason, "ghost") {
		t.Errorf("reason does not name the missing artifact: %q", f.Reason)
	}
	if !strings.Contains(f.Spot, "only") {
		t.Errorf("spot does not name the phase: %q", f.Spot)
	}
}

// TestLintWorkflowUnknownAgentDetected covers CA3 (agent half): a phase referencing
// an agent that does not exist in the workspace catalog is detected. The predicate
// is built from the agents actually loaded, so a reference to a never-created agent
// is unknown.
func TestLintWorkflowUnknownAgentDetected(t *testing.T) {
	root := newWorkspace(t)
	addBuiltinAgent(t, root, "analyst")
	writeWorkflow(t, root, workflows.Workflow{
		Name: "stranger",
		Phases: []workflows.Phase{
			{ID: "p1", Agent: "analyst", Inputs: []string{"brief"}, Outputs: []string{"x"}, Gate: "g", DependsOn: []string{"brief"}},
			{ID: "p2", Agent: "ghost-agent", Inputs: []string{"x"}, Outputs: []string{}, Gate: "g", DependsOn: []string{"p1"}},
		},
	})

	report := lint(t, root)
	f := findOf(t, report, FamilyWorkflow, "unknown-agent", "stranger")
	if !strings.Contains(f.Reason, "ghost-agent") {
		t.Errorf("reason does not name the unknown agent: %q", f.Reason)
	}
}

// TestLintWorkflowDuplicatePhaseIDsDetected covers CA4 (duplicate ids): two phases
// sharing an id is a structural violation, detected and reported. Written raw
// because Create would reject it before writing.
func TestLintWorkflowDuplicatePhaseIDsDetected(t *testing.T) {
	root := newWorkspace(t)
	addBuiltinAgent(t, root, "analyst")
	writeRawWorkflow(t, root, "dupes", strings.Join([]string{
		"phases:",
		"  - id: same",
		"    agent: analyst",
		"    inputs: []",
		"    outputs: []",
		"    gate: g",
		"    depends_on: []",
		"  - id: same",
		"    agent: analyst",
		"    inputs: []",
		"    outputs: []",
		"    gate: g",
		"    depends_on: []",
		"",
	}, "\n"))

	report := lint(t, root)
	f := findOf(t, report, FamilyWorkflow, "schema", "dupes")
	if f.Severity != SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
	if !strings.Contains(strings.ToLower(f.Reason), "unique") {
		t.Errorf("reason does not explain the duplicate-id rule: %q", f.Reason)
	}
}

// TestLintWorkflowMalformedDepsDetected covers CA4 (malformed deps): a depends_on
// that is not a flow list makes the whole file unparsable, surfaced as a controlled
// malformed finding (no panic — CA7 too).
func TestLintWorkflowMalformedDepsDetected(t *testing.T) {
	root := newWorkspace(t)
	writeRawWorkflow(t, root, "broken-deps", strings.Join([]string{
		"phases:",
		"  - id: p1",
		"    agent: analyst",
		"    inputs: []",
		"    outputs: []",
		"    gate: g",
		"    depends_on: not-a-list", // must be [a, b]; this is malformed
		"",
	}, "\n"))

	report := lint(t, root)
	f := findOf(t, report, FamilyWorkflow, "malformed", "broken-deps")
	if f.Severity != SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
}

// TestLintManifestInvalidDetected covers CA5: a manifest with an unsupported backend
// is detected and reported actionably. We rewrite the scaffolded manifest in place.
func TestLintManifestInvalidDetected(t *testing.T) {
	root := newWorkspace(t)
	manifestPath := filepath.Join(root, workspace.Name, workspace.RootArtifacts[0])
	bad := strings.Join([]string{
		"name: proj",
		"version: " + workspace.SchemaVersion,
		"backends:",
		"  - not-a-real-backend",
		"conventions:",
		"  naming: kebab-case",
		"  markdown: hierarchical-headings",
		"  yaml: ordered-deterministic",
		"",
	}, "\n")
	if err := os.WriteFile(manifestPath, []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	report := lint(t, root)
	f := findOf(t, report, FamilyManifest, "schema", "daedalus.yaml")
	if f.Severity != SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
	if !strings.Contains(f.Reason, "not-a-real-backend") {
		t.Errorf("reason does not name the offending backend: %q", f.Reason)
	}
}

// TestLintMalformedAgentNoPanic covers CA7: a severely malformed agent.yaml yields a
// controlled malformed finding rather than crashing.
func TestLintMalformedAgentNoPanic(t *testing.T) {
	root := newWorkspace(t)
	// Garbage that the agent loader cannot parse at all.
	writeRawAgent(t, root, "garbage", "::: not yaml at all :::\n\t\x00bad", "body")

	report := lint(t, root) // must not panic
	f := findOf(t, report, FamilyAgent, "malformed", "garbage")
	if f.Severity != SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
}

// TestLintMalformedManifestNoPanic covers CA7: an unparsable manifest yields a
// controlled malformed finding, not a panic or an opaque error.
func TestLintMalformedManifestNoPanic(t *testing.T) {
	root := newWorkspace(t)
	manifestPath := filepath.Join(root, workspace.Name, workspace.RootArtifacts[0])
	if err := os.WriteFile(manifestPath, []byte("this is : not : a : valid manifest\n@@@\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := lint(t, root) // must not panic
	f := findOf(t, report, FamilyManifest, "malformed", "daedalus.yaml")
	if f.Severity != SeverityError {
		t.Errorf("severity = %q, want error", f.Severity)
	}
}

// TestLintMissingManifestDetected covers CA5/CA7: a workspace with no manifest at
// all is reported as a controlled "missing" finding.
func TestLintMissingManifestDetected(t *testing.T) {
	root := t.TempDir() // no .daedalus/ scaffolded
	report := lint(t, root)
	findOf(t, report, FamilyManifest, "missing", "daedalus.yaml")
}

// TestLintDeterministicAndBackendAgnostic covers CA8: linting the same invalid
// workspace twice yields identical, stably-ordered findings, and no finding text
// names a concrete backend literal (a backend only ever appears as a manifest value,
// which a clean/invalid-by-other-means workspace does not surface).
func TestLintDeterministicAndBackendAgnostic(t *testing.T) {
	build := func(t *testing.T) string {
		t.Helper()
		root := newWorkspace(t)
		addBuiltinAgent(t, root, "analyst")
		writeRawAgent(t, root, "hollow", "id: hollow\nrole: tester\nprompt: prompt.md\n", "")
		writeWorkflow(t, root, workflows.Workflow{
			Name: "looped",
			Phases: []workflows.Phase{
				{ID: "a", Agent: "analyst", Inputs: []string{}, Outputs: []string{}, Gate: "g", DependsOn: []string{"b"}},
				{ID: "b", Agent: "ghost", Inputs: []string{"nope"}, Outputs: []string{}, Gate: "g", DependsOn: []string{"a"}},
			},
		})
		return root
	}

	first := lint(t, build(t))
	second := lint(t, build(t))

	if first.String() != second.String() {
		t.Errorf("non-deterministic linter output:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}

	// Backend-agnosticism: no finding message mentions the concrete MVP backend name.
	// (This workspace has a valid manifest, so the backend name never legitimately
	// appears; any occurrence would indicate a hardcoded backend literal in a rule.)
	for _, f := range first.Findings {
		if strings.Contains(strings.ToLower(f.Error()), workspace.DefaultBackend) {
			t.Errorf("finding leaks a concrete backend literal: %q", f.Error())
		}
		if strings.Contains(strings.ToLower(f.Error()), "claude") {
			t.Errorf("finding references a concrete backend: %q", f.Error())
		}
	}
}
