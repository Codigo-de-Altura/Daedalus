package compile

import (
	"errors"
	"fmt"
	"sort"
)

// ErrNoAdapter is the sentinel returned (wrapped) when a requested backend has no
// registered Compiler. The command distinguishes it via errors.Is so it can map
// "backend without an adapter" to a clear, dedicated error and — crucially —
// abort before writing anything (REQ-5). It is separate from a validation error
// because the cause and the fix differ: here the definition may be perfectly
// valid; what is missing is an adapter for the chosen backend.
var ErrNoAdapter = errors.New("no adapter registered for backend")

// Registry maps a backend key to the Compiler that targets it. It is the seam
// that makes the orchestration backend-agnostic (REQ-10/RNF-7): the build command
// looks a backend up here and routes to whatever Compiler is registered, so
// adding a backend is a registration, never a change to the command. A Registry
// is not safe for concurrent mutation; it is built once at startup (see
// DefaultRegistry) and then read.
type Registry struct {
	compilers map[string]Compiler
}

// NewRegistry returns an empty Registry ready to accept Register calls. Most
// callers want DefaultRegistry, which is pre-seeded with the MVP adapters; this
// constructor exists so tests can build an isolated registry with exactly the
// adapters they want to exercise.
func NewRegistry() *Registry {
	return &Registry{compilers: make(map[string]Compiler)}
}

// Register adds c under its Backend() key. It panics on a duplicate registration
// or an empty backend key, because both are programming errors in the wiring
// (registration happens at startup over a fixed set, never on user input) and a
// silent overwrite would make which adapter wins depend on registration order.
func (r *Registry) Register(c Compiler) {
	key := c.Backend()
	if key == "" {
		panic("compile: Compiler with empty Backend() cannot be registered")
	}
	if _, dup := r.compilers[key]; dup {
		panic(fmt.Sprintf("compile: duplicate Compiler registration for backend %q", key))
	}
	r.compilers[key] = c
}

// Lookup returns the Compiler registered for backend, or an error wrapping
// ErrNoAdapter naming the backend and listing the backends that do have an
// adapter — an actionable message the command surfaces verbatim (REQ-5). It
// never returns a partial result: a miss is a hard, no-write error.
func (r *Registry) Lookup(backend string) (Compiler, error) {
	c, ok := r.compilers[backend]
	if !ok {
		return nil, fmt.Errorf("%w: %q (registered: %s)",
			ErrNoAdapter, backend, joinOrNone(r.Backends()))
	}
	return c, nil
}

// Backends returns the registered backend keys in sorted order, so error
// messages and any listing are deterministic regardless of registration order.
func (r *Registry) Backends() []string {
	keys := make([]string, 0, len(r.compilers))
	for k := range r.compilers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// joinOrNone renders a backend list for an error message, using a clear "none"
// when the registry is empty so the message never reads "registered: ".
func joinOrNone(keys []string) string {
	if len(keys) == 0 {
		return "none"
	}
	out := keys[0]
	for _, k := range keys[1:] {
		out += ", " + k
	}
	return out
}

// DefaultRegistry returns a Registry pre-seeded with the adapters the MVP ships.
// It is the registry the build command uses, and it is the single place new
// backends are wired in (RNF-7): registering an adapter here — and nowhere else —
// is what makes a new backend buildable. A fresh registry is returned per call so
// callers (and tests) cannot mutate shared state.
func DefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(newClaudeCompiler())
	return r
}
