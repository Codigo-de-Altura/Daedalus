// Package compile orchestrates compiling the canonical Daedalus workspace
// (`.daedalus/`) into a backend's native format. It is the backend-agnostic core
// behind the `daedalus build`/`sync` command (RF-6.1): it locates the workspace,
// loads and validates the canonical definition, resolves the target backend(s)
// from the manifest, routes each to its registered adapter, and reports the
// outcome — all without knowing any single backend's format.
//
// The package draws the seams the rest of epic-06 fills in:
//
//   - Compiler is the adapter contract (RF-6.2): one implementation per backend,
//     selected through the Registry. Adding a backend is registering a Compiler;
//     the orchestration never changes (RNF-7).
//   - A Compiler produces the desired native Artifacts as a pure value (no I/O),
//     so the write strategy and the preview are decoupled from the mapping. The
//     idempotent, non-destructive write strategy is RF-6.3's concern; the
//     diff/preview rendering is RF-6.4's. This package keeps Compile pure so both
//     can be layered on without reshaping the contract.
//
// Determinism (RNF-5) is a property of the whole pipeline: the same workspace
// must yield the same Artifacts, so Compilers must be pure functions of their
// Definition input and emit artifacts in a fixed order.
package compile

import "github.com/Codigo-de-Altura/Daedalus/internal/catalog"

// Definition is the loaded, validated canonical definition a Compiler maps to a
// backend's native format. It is the backend-agnostic source of truth, read from
// `.daedalus/` and validated before any Compiler sees it (so a Compiler never has
// to defend against an invalid definition).
//
// Today it carries the agents; prompts, workflows and the SDD backlog are
// additive fields a later ticket can add without breaking the contract, because
// a Compiler reads only the fields it maps.
type Definition struct {
	// Agents are the canonical agents in deterministic, id-sorted order so a
	// Compiler that iterates them emits artifacts in a stable order (RNF-5).
	Agents []catalog.Agent
}

// Artifact is a single native file a Compiler wants written: its path relative
// to the target repository root (slash-separated, backend-owned, e.g.
// ".claude/agents/foo.md") and its fully rendered content. A Compiler returns
// Artifacts as a pure description; this package owns turning them into files, so
// the non-destructive write strategy (RF-6.3) and the preview (RF-6.4) operate
// on the same value.
type Artifact struct {
	// RelPath is the artifact's path relative to the target repository root, in
	// slash form, so the value is identical on every OS (determinism, RNF-5). The
	// orchestrator joins it onto the target root with the OS separator at write
	// time.
	RelPath string
	// Content is the artifact's fully rendered bytes, captured at compile time so
	// a preview and the eventual write describe identical content.
	Content string
}

// Artifacts is a Compiler's complete output for one backend: the native files it
// wants to exist. The slice is in the Compiler's fixed emission order so the
// summary and any diff are deterministic.
type Artifacts struct {
	// Backend is the backend key these artifacts target (mirrors the Compiler's
	// Backend()), carried so a caller assembling multiple backends' output can
	// attribute each artifact.
	Backend string
	// Files are the native files to write, in deterministic order.
	Files []Artifact
}

// Compiler is the adapter contract: one implementation per agent backend maps
// the canonical Definition to that backend's native Artifacts (RF-6.2). It is
// the single extension point — registering a Compiler is all it takes to support
// a new backend (RNF-7) — so the interface is deliberately minimal and free of
// any I/O: Compile is a pure function of the Definition, which is what makes the
// pipeline deterministic (RNF-5) and lets the write strategy (RF-6.3) and the
// preview (RF-6.4) be layered on independently.
type Compiler interface {
	// Backend returns the canonical backend key this Compiler targets (e.g.
	// "claude-code"). It must match a workspace.SupportedBackends entry and is the
	// key the Registry stores it under.
	Backend() string

	// Compile maps the validated canonical Definition to the backend's native
	// Artifacts without performing any I/O. It must be deterministic: the same
	// Definition yields byte-identical Artifacts in a fixed order. It returns an
	// error only when the mapping itself cannot be produced (a backend-level
	// failure), not for I/O — there is none here.
	Compile(def Definition) (Artifacts, error)
}
