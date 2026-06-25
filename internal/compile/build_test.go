package compile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// initWorkspace scaffolds a `.daedalus/` workspace (manifest included) under a
// fresh temp dir and returns its root. It is the precondition for a buildable
// workspace; tests then add or corrupt agents as needed.
func initWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := workspace.Create(root); err != nil {
		t.Fatalf("scaffold workspace: %v", err)
	}
	return root
}

// addAgent materializes a built-in catalog agent into the workspace so the
// definition has something valid to load.
func addAgent(t *testing.T, root, id string) {
	t.Helper()
	agentsRoot := filepath.Join(root, workspace.Name, catalog.AgentsDir)
	if _, err := catalog.Builtin.Materialize(agentsRoot, id); err != nil {
		t.Fatalf("materialize %q: %v", id, err)
	}
}

// registryWith builds a registry whose claude-code adapter is the given fake, so
// a test can exercise the success path the real (stubbed) adapter cannot yet.
func registryWith(c Compiler) *Registry {
	r := NewRegistry()
	r.Register(c)
	return r
}

// TestBuildMissingWorkspaceAborts covers REQ-2/REQ-8: with no workspace the build
// returns ErrWorkspaceNotFound and writes nothing.
func TestBuildMissingWorkspaceAborts(t *testing.T) {
	root := t.TempDir() // no .daedalus/ here

	_, err := Build(Options{Root: root})
	if !errors.Is(err, ErrWorkspaceNotFound) {
		t.Fatalf("err = %v, want ErrWorkspaceNotFound", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, workspace.Name)); !os.IsNotExist(statErr) {
		t.Errorf("build created a workspace despite there being none (stat err=%v)", statErr)
	}
}

// TestBuildInvalidDefinitionAbortsWithoutWriting covers REQ-3/REQ-8: an invalid
// canonical definition aborts with a *DefinitionError and no artifact is written.
func TestBuildInvalidDefinitionAbortsWithoutWriting(t *testing.T) {
	root := initWorkspace(t)
	// Corrupt an agent: a directory with a malformed agent.yaml (missing role).
	agentDir := filepath.Join(root, workspace.Name, catalog.AgentsDir, "broken")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, catalog.DefinitionFileName), []byte("id: broken\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(agentDir, catalog.PromptFileName), []byte("body\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A fake adapter that would write a file if it were ever reached; it must not be.
	fake := fakeCompiler{backend: workspace.DefaultBackend, arts: Artifacts{
		Backend: workspace.DefaultBackend,
		Files:   []Artifact{{RelPath: ".claude/agents/x.md", Content: "x"}},
	}}

	_, err := Build(Options{Root: root, Registry: registryWith(fake)})
	if !IsDefinitionInvalid(err) {
		t.Fatalf("err = %v, want a *DefinitionError", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(statErr) {
		t.Errorf("an artifact was written despite an invalid definition (stat err=%v)", statErr)
	}
}

// TestBuildNoAdapterAbortsWithoutWriting covers REQ-5/REQ-8: a configured backend
// with no registered adapter aborts with ErrNoAdapter and writes nothing.
func TestBuildNoAdapterAbortsWithoutWriting(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")

	// An empty registry has no adapter for the manifest's claude-code backend.
	_, err := Build(Options{Root: root, Registry: NewRegistry()})
	if !errors.Is(err, ErrNoAdapter) {
		t.Fatalf("err = %v, want ErrNoAdapter", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(statErr) {
		t.Errorf("an artifact was written despite no adapter (stat err=%v)", statErr)
	}
}

// TestBuildCompileFailureIsNotValidation covers REQ-8: a backend whose adapter
// fails to compile is a compile error, distinct from a validation error.
func TestBuildCompileFailureIsNotValidation(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")

	fake := fakeCompiler{backend: workspace.DefaultBackend, err: errors.New("boom")}
	_, err := Build(Options{Root: root, Registry: registryWith(fake)})
	if err == nil {
		t.Fatal("want a compile error, got nil")
	}
	if IsDefinitionInvalid(err) {
		t.Errorf("a compile failure was misclassified as a validation error: %v", err)
	}
}

// TestBuildWritesArtifacts covers REQ-7: a successful build writes the adapter's
// artifacts and the outcome reports them.
func TestBuildWritesArtifacts(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")

	fake := fakeCompiler{backend: workspace.DefaultBackend, arts: Artifacts{
		Backend: workspace.DefaultBackend,
		Files: []Artifact{
			{RelPath: ".claude/agents/analyst.md", Content: "agent\n"},
		},
	}}

	out, err := Build(Options{Root: root, Registry: registryWith(fake)})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if out.Preview {
		t.Error("non-preview build reported as preview")
	}
	if len(out.Backends) != 1 || out.Backends[0].Backend != workspace.DefaultBackend {
		t.Fatalf("unexpected outcome backends: %+v", out.Backends)
	}
	if got := len(out.Backends[0].Created); got != 1 {
		t.Errorf("created = %d, want 1", got)
	}
	written := filepath.Join(root, ".claude", "agents", "analyst.md")
	if b, err := os.ReadFile(written); err != nil || string(b) != "agent\n" {
		t.Errorf("artifact not written as expected: content=%q err=%v", string(b), err)
	}
}

// TestBuildPreviewWritesNothing covers REQ-6: --preview computes the plan but
// touches no files.
func TestBuildPreviewWritesNothing(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")

	fake := fakeCompiler{backend: workspace.DefaultBackend, arts: Artifacts{
		Backend: workspace.DefaultBackend,
		Files:   []Artifact{{RelPath: ".claude/agents/analyst.md", Content: "agent\n"}},
	}}

	out, err := Build(Options{Root: root, Preview: true, Registry: registryWith(fake)})
	if err != nil {
		t.Fatalf("Build preview: %v", err)
	}
	if !out.Preview {
		t.Error("preview build not reported as preview")
	}
	if got := out.Backends[0].Planned; got != 1 {
		t.Errorf("planned = %d, want 1 (preview still reports what would be written)", got)
	}
	if len(out.Backends[0].Created) != 0 {
		t.Errorf("preview reported created files: %v", out.Backends[0].Created)
	}
	if _, statErr := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(statErr) {
		t.Errorf("preview wrote to the filesystem (stat err=%v)", statErr)
	}
}

// TestBuildIsDeterministic covers REQ-9: the same workspace yields the same
// outcome (same backends, same artifact set) across runs. We use preview so the
// second run is not perturbed by the first run's writes.
func TestBuildIsDeterministic(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")
	addAgent(t, root, "architect")

	fake := fakeCompiler{backend: workspace.DefaultBackend, arts: Artifacts{
		Backend: workspace.DefaultBackend,
		Files: []Artifact{
			{RelPath: ".claude/agents/analyst.md", Content: "a\n"},
			{RelPath: ".claude/agents/architect.md", Content: "b\n"},
		},
	}}

	first, err := Build(Options{Root: root, Preview: true, Registry: registryWith(fake)})
	if err != nil {
		t.Fatalf("first build: %v", err)
	}
	second, err := Build(Options{Root: root, Preview: true, Registry: registryWith(fake)})
	if err != nil {
		t.Fatalf("second build: %v", err)
	}
	if first.Backends[0].Planned != second.Backends[0].Planned {
		t.Errorf("non-deterministic planned count: %d vs %d",
			first.Backends[0].Planned, second.Backends[0].Planned)
	}
}

// TestClaudeStubReportsNotImplemented documents the 06-01/06-02 boundary: the
// real default registry routes claude-code to a stub that honestly reports it is
// not implemented yet (a compile error, never a fake success).
func TestClaudeStubReportsNotImplemented(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")

	_, err := Build(Options{Root: root}) // DefaultRegistry → claude stub
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("err = %v, want ErrNotImplemented from the claude stub", err)
	}
	if IsDefinitionInvalid(err) {
		t.Errorf("the not-implemented stub was misclassified as a validation error: %v", err)
	}
}
