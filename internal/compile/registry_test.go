package compile

import (
	"errors"
	"strings"
	"testing"

	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// fakeCompiler is a test double Compiler. It lets the tests drive routing and the
// orchestration without depending on the (stubbed) real Claude adapter.
type fakeCompiler struct {
	backend string
	arts    Artifacts
	err     error
}

func (f fakeCompiler) Backend() string { return f.backend }

func (f fakeCompiler) Compile(Definition) (Artifacts, error) {
	if f.err != nil {
		return Artifacts{}, f.err
	}
	return f.arts, nil
}

// TestRegistryLookupRoutesByBackend covers REQ-4/REQ-10: a registered backend
// resolves to its Compiler.
func TestRegistryLookupRoutesByBackend(t *testing.T) {
	r := NewRegistry()
	want := fakeCompiler{backend: "claude-code"}
	r.Register(want)

	got, err := r.Lookup("claude-code")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got.Backend() != "claude-code" {
		t.Errorf("routed to backend %q, want claude-code", got.Backend())
	}
}

// TestRegistryLookupMissingIsActionable covers REQ-5: a backend without an
// adapter returns ErrNoAdapter and names both the missing backend and the
// registered set.
func TestRegistryLookupMissingIsActionable(t *testing.T) {
	r := NewRegistry()
	r.Register(fakeCompiler{backend: "claude-code"})

	_, err := r.Lookup("codex")
	if !errors.Is(err, ErrNoAdapter) {
		t.Fatalf("err = %v, want ErrNoAdapter", err)
	}
	if !strings.Contains(err.Error(), "codex") {
		t.Errorf("error does not name the missing backend: %v", err)
	}
	if !strings.Contains(err.Error(), "claude-code") {
		t.Errorf("error does not list the registered backend(s): %v", err)
	}
}

// TestRegistryDuplicatePanics guards the wiring: registering two Compilers for
// the same backend is a programming error and must fail loudly.
func TestRegistryDuplicatePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("duplicate registration did not panic")
		}
	}()
	r := NewRegistry()
	r.Register(fakeCompiler{backend: "claude-code"})
	r.Register(fakeCompiler{backend: "claude-code"})
}

// TestDefaultRegistryHasClaudeAdapter covers REQ-4: the MVP backend is routable
// out of the box (even though its mapping is the 06-02 stub today).
func TestDefaultRegistryHasClaudeAdapter(t *testing.T) {
	r := DefaultRegistry()
	c, err := r.Lookup(workspace.DefaultBackend)
	if err != nil {
		t.Fatalf("default registry missing %q: %v", workspace.DefaultBackend, err)
	}
	if c.Backend() != workspace.DefaultBackend {
		t.Errorf("default adapter Backend() = %q, want %q", c.Backend(), workspace.DefaultBackend)
	}
}
