package compile

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// ErrWorkspaceNotFound is the sentinel returned (wrapped) when the target has no
// `.daedalus/` workspace to build from (REQ-2). The command maps it to a clear
// "run daedalus init first" error and a non-zero exit, with nothing written.
var ErrWorkspaceNotFound = errors.New("no .daedalus workspace found")

// Options tunes a build. A zero Options is valid: it builds (writes) using the
// DefaultRegistry. The fields are additive so new knobs never force existing
// callers to change.
type Options struct {
	// Root is the target repository directory whose `.daedalus/` is compiled.
	// Defaults to "." when empty.
	Root string
	// Preview, when true, computes the full plan but writes nothing (REQ-6). The
	// deep diff/preview rendering is RF-6.4's job; this flag is the no-write gate
	// the orchestration honors so a preview can never touch the filesystem.
	Preview bool
	// Registry is the adapter registry to route backends through. When nil the
	// DefaultRegistry (the MVP adapters) is used; tests inject their own to
	// exercise routing in isolation.
	Registry *Registry
	// Logger receives decision-point logs. When nil a no-op logger is used so the
	// core never depends on a global and a caller that wants silence gets it.
	Logger *slog.Logger
}

// BackendOutcome is the per-backend result of a build: which backend, how many
// artifacts it produced, and — once written — which were created vs. left intact.
// It is the data the command renders as the summary (REQ-7); the wording lives in
// the command so the core stays presentation-free.
type BackendOutcome struct {
	// Backend is the backend key this outcome is for.
	Backend string
	// Planned is the number of native artifacts the adapter produced.
	Planned int
	// Created and Unchanged partition the written artifacts. In Preview mode both
	// are zero (nothing was written); Planned still reflects what would be written.
	Created   []string
	Unchanged []string
}

// Outcome is the whole build's result: the resolved target backends and a
// per-backend outcome, in the deterministic order the manifest listed them.
type Outcome struct {
	// Root is the resolved target repository directory.
	Root string
	// Preview reports whether the run was a dry run (nothing was written).
	Preview bool
	// Backends are the per-backend outcomes, in manifest order.
	Backends []BackendOutcome
}

// Build is the orchestrator behind `daedalus build`/`sync` (RF-6.1). It runs the
// pipeline in the order that guarantees safety and clear failure modes:
//
//  1. Locate the workspace (REQ-2) — absent ⇒ ErrWorkspaceNotFound, no write.
//  2. Resolve the target backend(s) from the manifest (REQ-4).
//  3. Load and validate the canonical definition (REQ-3) — invalid ⇒
//     *DefinitionError, no write (validate-before-write).
//  4. For each backend: look up its adapter via the Registry (REQ-5) and compile
//     (a pure mapping). A backend with no adapter ⇒ ErrNoAdapter, no write.
//  5. Write the artifacts — unless Preview, which stops here (REQ-6).
//
// Because every gate that can reject the build runs before any write, an invalid
// definition or an unroutable backend never leaves a partial result behind. The
// error types let the command compute differentiated exit codes (REQ-8): a
// *DefinitionError is a validation error; everything else (no adapter, a compile
// failure, an I/O failure) is a compilation/write error.
//
// Build is deterministic (REQ-9): backends are processed in manifest order and
// the definition is loaded id-sorted, so the same workspace yields the same
// Outcome and the same artifacts.
func Build(opts Options) (*Outcome, error) {
	root := opts.Root
	if root == "" {
		root = "."
	}
	logger := opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(noopWriter{}, nil))
	}
	reg := opts.Registry
	if reg == nil {
		reg = DefaultRegistry()
	}

	// 1. Locate the workspace. We require the manifest to exist: it is both the
	// proof a workspace is here (REQ-2) and the source of the target backends
	// (REQ-4), so a single read settles both.
	manifest, err := workspace.ReadManifest(root)
	if err != nil {
		if errors.Is(err, workspace.ErrManifestNotFound) {
			logger.Error("build aborted", "phase", "locate", "root", filepath.ToSlash(root), "reason", "no workspace")
			return nil, fmt.Errorf("%w at %s", ErrWorkspaceNotFound, filepath.ToSlash(filepath.Join(root, workspace.Name)))
		}
		// A malformed manifest is a real, actionable failure (not a missing
		// workspace); surface it as-is so the command reports it as a build error.
		logger.Error("build aborted", "phase", "locate", "root", filepath.ToSlash(root), "err", err)
		return nil, err
	}
	logger.Info("workspace located", "root", filepath.ToSlash(root), "backends", manifest.Backends)

	// 3. Validate the canonical definition BEFORE any adapter or write (REQ-3).
	// Loading here, before the per-backend loop, means an invalid definition aborts
	// the whole build once, with nothing written.
	def, err := LoadDefinition(root)
	if err != nil {
		if IsDefinitionInvalid(err) {
			logger.Error("build aborted", "phase", "validate", "root", filepath.ToSlash(root), "err", err)
		} else {
			logger.Error("build failed", "phase", "load", "root", filepath.ToSlash(root), "err", err)
		}
		return nil, err
	}
	logger.Info("definition validated", "agents", len(def.Agents))

	// 4. Resolve and compile each backend through the registry. We compile ALL
	// backends before writing ANY, so a single unroutable or failing backend
	// aborts the whole build with nothing on disk — the same validate-before-write
	// guarantee, extended across backends.
	type compiled struct {
		artifacts Artifacts
	}
	plans := make([]compiled, 0, len(manifest.Backends))
	for _, backend := range manifest.Backends {
		comp, err := reg.Lookup(backend)
		if err != nil {
			logger.Error("build aborted", "phase", "route", "backend", backend, "err", err)
			return nil, err
		}
		arts, err := comp.Compile(def)
		if err != nil {
			logger.Error("build failed", "phase", "compile", "backend", backend, "err", err)
			return nil, fmt.Errorf("compiling backend %q: %w", backend, err)
		}
		// Defend the contract: a Compiler must label its output with its own key.
		if arts.Backend == "" {
			arts.Backend = backend
		}
		plans = append(plans, compiled{artifacts: arts})
		logger.Info("backend compiled", "backend", backend, "artifacts", len(arts.Files))
	}

	// 5. Write (or, in Preview, do not). Every gate has passed; only now do we
	// touch the filesystem.
	out := &Outcome{Root: root, Preview: opts.Preview}
	for _, p := range plans {
		bo := BackendOutcome{Backend: p.artifacts.Backend, Planned: len(p.artifacts.Files)}
		if !opts.Preview {
			created, unchanged, err := writeArtifacts(root, p.artifacts.Files)
			if err != nil {
				logger.Error("build failed", "phase", "write", "backend", p.artifacts.Backend, "err", err)
				return nil, fmt.Errorf("writing backend %q: %w", p.artifacts.Backend, err)
			}
			bo.Created = created
			bo.Unchanged = unchanged
			logger.Info("backend written", "backend", p.artifacts.Backend,
				"created", len(created), "unchanged", len(unchanged))
		}
		out.Backends = append(out.Backends, bo)
	}

	logger.Info("build done", "preview", opts.Preview, "backends", len(out.Backends))
	return out, nil
}

// writeArtifacts materializes a backend's artifacts under root and reports which
// were created vs. left intact.
//
// SCOPE BOUNDARY (RF-6.3): this is a minimal, non-destructive writer — it creates
// parent directories and writes each file with O_CREATE|O_EXCL, so an existing
// file is never overwritten (a manual change survives). The full idempotent,
// non-destructive-of-the-managed-area strategy (re-writing managed artifacts that
// changed while preserving manual edits outside the managed area, with the diff
// that drives it) is ticket-06-03's deliverable and will replace this body. The
// signature — (created, unchanged, err) — is shaped for that successor so the
// orchestration above does not change.
func writeArtifacts(root string, files []Artifact) (created, unchanged []string, err error) {
	for _, f := range files {
		abs := filepath.Join(root, filepath.FromSlash(f.RelPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return nil, nil, err
		}
		wrote, err := ensureFile(abs, f.Content)
		if err != nil {
			return nil, nil, err
		}
		if wrote {
			created = append(created, f.RelPath)
		} else {
			unchanged = append(unchanged, f.RelPath)
		}
	}
	return created, unchanged, nil
}

// ensureFile creates a file at path with content only if it does not already
// exist. O_EXCL makes the create-or-skip atomic and non-destructive: an existing
// file is never truncated. It reports whether it created a new file. This mirrors
// the workspace/catalog helpers of the same name; it is duplicated rather than
// shared because those are unexported and this package's write semantics will
// evolve independently (ticket-06-03).
func ensureFile(path, content string) (bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, err
	}
	_, writeErr := f.WriteString(content)
	closeErr := f.Close()
	if writeErr != nil {
		return false, writeErr
	}
	if closeErr != nil {
		return false, closeErr
	}
	return true, nil
}

// noopWriter discards everything, backing the default no-op logger so a caller
// that supplies no Logger gets silence without the core reaching for a global.
type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }
