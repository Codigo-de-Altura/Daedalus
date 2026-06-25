package compile

import (
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// TestPipelineDeterministicFromWorkspace is the discoverable, pipeline-level
// determinism statement the testing strategy requires (CA3 / RNF-5): loading the
// canonical definition from a real on-disk workspace and compiling it TWICE must
// yield byte-identical Artifacts, in the same order.
//
// Unlike TestClaudeCompileDeterministic (which compiles a hand-built Definition)
// and the on-disk idempotent tests (which compare written trees), this drives the
// full load → validate → compile path from the filesystem, so it proves the output
// is independent of directory-read order and map iteration order — the concrete
// hazards the determinism contract forbids.
func TestPipelineDeterministicFromWorkspace(t *testing.T) {
	root := initWorkspace(t)
	// Multiple agents (and the built-in prompts the workspace seeds) make the
	// id-sort and prompt-resolve paths meaningful: a non-deterministic order would
	// surface as a difference between the two compiles.
	addAgent(t, root, "analyst")
	addAgent(t, root, "architect")
	addAgent(t, root, "planner")

	compileOnce := func(t *testing.T) Artifacts {
		t.Helper()
		def, err := LoadDefinition(root)
		if err != nil {
			t.Fatalf("LoadDefinition: %v", err)
		}
		arts, err := newClaudeCompiler().Compile(def)
		if err != nil {
			t.Fatalf("Compile: %v", err)
		}
		return arts
	}

	first := compileOnce(t)
	second := compileOnce(t)

	if first.Backend != second.Backend {
		t.Errorf("backend differs between compiles: %q vs %q", first.Backend, second.Backend)
	}
	if len(first.Files) != len(second.Files) {
		t.Fatalf("artifact count differs between compiles: %d vs %d", len(first.Files), len(second.Files))
	}
	for i := range first.Files {
		if first.Files[i] != second.Files[i] {
			t.Errorf("artifact %d differs between compiles:\n--- first ---\n%+v\n--- second ---\n%+v",
				i, first.Files[i], second.Files[i])
		}
	}
}

// TestPipelineArtifactPathsAreRelativeAndSlashed guards two determinism invariants
// the golden compare relies on (RNF-5/RNF-6): every emitted artifact path is
// repository-relative (never absolute) and uses forward slashes, so the output is
// byte-identical across operating systems regardless of where the workspace lives.
func TestPipelineArtifactPathsAreRelativeAndSlashed(t *testing.T) {
	root := initWorkspace(t)
	addAgent(t, root, "analyst")

	def, err := LoadDefinition(root)
	if err != nil {
		t.Fatalf("LoadDefinition: %v", err)
	}
	arts, err := newClaudeCompiler().Compile(def)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}

	for _, f := range arts.Files {
		if filepathIsAbs(f.RelPath) {
			t.Errorf("artifact path is absolute, breaking portability: %q", f.RelPath)
		}
		if containsByte(f.RelPath, '\\') {
			t.Errorf("artifact path uses a backslash, breaking cross-platform stability: %q", f.RelPath)
		}
	}

	// Sanity: the workspace name never leaks into a produced path — artifacts are
	// the backend's native tree, not the canonical source tree.
	for _, f := range arts.Files {
		if containsSub(f.RelPath, workspace.Name+"/") {
			t.Errorf("artifact path leaks the canonical workspace dir: %q", f.RelPath)
		}
	}
}

// filepathIsAbs is a tiny local wrapper so the assertion above reads as one
// expression; it intentionally treats both Unix and Windows absolute shapes as
// absolute even when the test runs on the other OS.
func filepathIsAbs(p string) bool {
	if p == "" {
		return false
	}
	if p[0] == '/' {
		return true
	}
	// Windows drive-letter form (e.g. "C:\..." or "C:/...").
	return len(p) >= 2 && p[1] == ':'
}

func containsByte(s string, b byte) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return true
		}
	}
	return false
}

func containsSub(s, sub string) bool {
	if len(sub) == 0 {
		return true
	}
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
