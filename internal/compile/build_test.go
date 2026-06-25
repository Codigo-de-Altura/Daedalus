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
	// Preview classifies what WOULD happen: a fresh workspace ⇒ the artifact would
	// be created. The classification is reported, but NOTHING is written.
	if len(out.Backends[0].Created) != 1 {
		t.Errorf("preview should classify the artifact as would-create; got created=%v", out.Backends[0].Created)
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

// TestBuildDefaultRegistryWritesClaudeArtifacts verifies the real Claude Code
// adapter (DefaultRegistry) produces and writes `.claude/` artifacts for a valid
// workspace end-to-end (no fake compiler).
func TestBuildDefaultRegistryWritesClaudeArtifacts(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")

	out, err := Build(Options{Root: root}) // DefaultRegistry → real claude adapter
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(out.Backends) != 1 {
		t.Fatalf("backends = %d, want 1", len(out.Backends))
	}
	// At least the agent file and settings.json must exist.
	for _, rel := range []string{".claude/agents/analyst.md", ".claude/settings.json"} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected artifact %s not written: %v", rel, err)
		}
	}
}
