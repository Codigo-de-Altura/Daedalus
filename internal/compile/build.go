package compile

import (
	"errors"
	"fmt"
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
	// Created, Updated and Unchanged partition the produced artifacts by what the
	// run did with each (RF-6.3): Created were newly written, Updated were rewritten
	// because their canonical source changed, Unchanged were byte-identical and left
	// completely untouched. In Preview mode these reflect what WOULD happen (the plan
	// classification), computed without writing.
	Created   []string
	Updated   []string
	Unchanged []string
	// Orphans are files inside the managed directories the current build no longer
	// produces. They are detected and reported but NEVER deleted (safe default,
	// Check-8); the preview (RF-6.4) surfaces them for the user to act on manually.
	Orphans []string
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
	root, logger, _ := resolveOptions(opts)

	// Stages 1–4 (locate, validate, route, compile) are shared with Plan: an
	// invalid definition or an unroutable backend aborts here, before any write.
	compiled, err := resolveAndCompile(opts)
	if err != nil {
		return nil, err
	}

	// 5. Plan against the current on-disk state, then write (or, in Preview, do
	// not). We always compute the pure plan first — it classifies every produced
	// artifact (created/updated/unchanged) and detects orphans without writing — so
	// a preview reports exactly what a real run would do, and a real run reuses the
	// same classification for its compare-and-skip writes (RF-6.3/RF-6.4).
	out := &Outcome{Root: root, Preview: opts.Preview}
	for _, arts := range compiled {
		plan, err := PlanArtifacts(root, arts)
		if err != nil {
			logger.Error("build failed", "phase", "plan", "backend", arts.Backend, "err", err)
			return nil, fmt.Errorf("planning backend %q: %w", arts.Backend, err)
		}

		bo := BackendOutcome{
			Backend: arts.Backend,
			Planned: len(arts.Files),
			Orphans: plan.Orphans,
		}

		if opts.Preview {
			// Report what WOULD happen, computed from the plan, with no writes.
			for _, pa := range plan.Artifacts {
				bo.append(pa.Status, pa.RelPath)
			}
			logger.Info("backend planned", "backend", arts.Backend,
				"created", len(bo.Created), "updated", len(bo.Updated),
				"unchanged", len(bo.Unchanged), "orphans", len(plan.Orphans))
		} else {
			created, updated, unchanged, err := writeArtifacts(root, arts.Files)
			if err != nil {
				logger.Error("build failed", "phase", "write", "backend", arts.Backend, "err", err)
				return nil, fmt.Errorf("writing backend %q: %w", arts.Backend, err)
			}
			bo.Created, bo.Updated, bo.Unchanged = created, updated, unchanged
			logger.Info("backend written", "backend", arts.Backend,
				"created", len(created), "updated", len(updated),
				"unchanged", len(unchanged), "orphans", len(plan.Orphans))
		}
		out.Backends = append(out.Backends, bo)
	}

	logger.Info("build done", "preview", opts.Preview, "backends", len(out.Backends))
	return out, nil
}

// append records relPath under the slice matching status, so the preview path and
// the write path produce the same partitioning of the produced artifacts.
func (bo *BackendOutcome) append(status ArtifactStatus, relPath string) {
	switch status {
	case StatusCreated:
		bo.Created = append(bo.Created, relPath)
	case StatusUpdated:
		bo.Updated = append(bo.Updated, relPath)
	default:
		bo.Unchanged = append(bo.Unchanged, relPath)
	}
}

// PlanResult is the whole plan of a build computed WITHOUT writing anything: the
// resolved target root and, per backend (in manifest order), the enriched
// BackendPlan that carries every artifact's status and its Current/Target content
// for diffing, plus the detected orphans. It is what the interactive preview
// (RF-6.4) consumes to render the diff and the confirm/cancel gate; on confirm the
// caller invokes Build (non-preview), which recompiles deterministically and
// writes — recompiling at confirm time avoids any TOCTOU between preview and write
// and is cheap because Compile is pure.
type PlanResult struct {
	// Root is the resolved target repository directory the plan was computed for.
	Root string
	// Backends are the per-backend plans, in manifest order, each enriched with
	// per-artifact Current/Target content and the backend's orphans.
	Backends []BackendPlan
}

// Plan computes the full build plan for every backend in the manifest WITHOUT
// writing anything (RF-6.4). It runs the same locate → validate → route → compile
// pipeline as Build — so it fails identically on a missing workspace, an invalid
// definition or an unroutable backend, before producing any plan — then classifies
// each produced artifact against disk and returns the enriched BackendPlan (status
// + Current/Target for the diff) plus orphans.
//
// It is the read-only half the presentation layer calls: the TUI shows what Plan
// returns and, on confirmation, calls Build. Plan never writes, so a preview can
// never mutate the workspace (the --preview / no-TTY-no-yes guarantee). It is
// deterministic: backends in manifest order, artifacts in emission order, orphans
// sorted. The Preview field of Options is irrelevant here (Plan never writes
// regardless); the field is honored by Build.
func Plan(opts Options) (*PlanResult, error) {
	root, logger, _ := resolveOptions(opts)

	compiled, err := resolveAndCompile(opts)
	if err != nil {
		return nil, err
	}

	res := &PlanResult{Root: root}
	for _, arts := range compiled {
		plan, err := PlanArtifacts(root, arts)
		if err != nil {
			logger.Error("plan failed", "phase", "plan", "backend", arts.Backend, "err", err)
			return nil, fmt.Errorf("planning backend %q: %w", arts.Backend, err)
		}
		res.Backends = append(res.Backends, plan)
		logger.Info("backend planned", "backend", arts.Backend,
			"artifacts", len(plan.Artifacts), "orphans", len(plan.Orphans))
	}

	logger.Info("plan done", "backends", len(res.Backends))
	return res, nil
}

// resolveOptions applies the Options defaults shared by Build and Plan: a "."
// root, a no-op logger, and the DefaultRegistry. Centralizing it keeps the two
// entry points in lockstep so they can never drift on defaulting.
func resolveOptions(opts Options) (root string, logger *slog.Logger, reg *Registry) {
	root = opts.Root
	if root == "" {
		root = "."
	}
	logger = opts.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(noopWriter{}, nil))
	}
	reg = opts.Registry
	if reg == nil {
		reg = DefaultRegistry()
	}
	return root, logger, reg
}

// resolveAndCompile runs the safe, write-free front half of the pipeline shared by
// Build and Plan: locate the workspace (REQ-2), validate the canonical definition
// before anything else (REQ-3), then route each manifest backend through the
// registry (REQ-5) and compile it (a pure mapping). It compiles ALL backends
// before returning, so a single unroutable or failing backend aborts the whole
// operation with nothing produced — the validate-before-write guarantee, extended
// across backends. The returned artifacts are in manifest order (determinism).
func resolveAndCompile(opts Options) ([]Artifacts, error) {
	root, logger, reg := resolveOptions(opts)

	// 1. Locate the workspace. The manifest is both the proof a workspace is here
	// (REQ-2) and the source of the target backends (REQ-4), so one read settles
	// both.
	manifest, err := workspace.ReadManifest(root)
	if err != nil {
		if errors.Is(err, workspace.ErrManifestNotFound) {
			logger.Error("build aborted", "phase", "locate", "root", filepath.ToSlash(root), "reason", "no workspace")
			return nil, fmt.Errorf("%w at %s", ErrWorkspaceNotFound, filepath.ToSlash(filepath.Join(root, workspace.Name)))
		}
		logger.Error("build aborted", "phase", "locate", "root", filepath.ToSlash(root), "err", err)
		return nil, err
	}
	logger.Info("workspace located", "root", filepath.ToSlash(root), "backends", manifest.Backends)

	// 2. Validate the canonical definition before any adapter (REQ-3).
	def, err := LoadDefinition(root)
	if err != nil {
		if IsDefinitionInvalid(err) {
			logger.Error("build aborted", "phase", "validate", "root", filepath.ToSlash(root), "err", err)
		} else {
			logger.Error("build failed", "phase", "load", "root", filepath.ToSlash(root), "err", err)
		}
		return nil, err
	}
	logger.Info("definition validated", "agents", len(def.Agents), "commands", len(def.Commands))

	// 3. Route and compile each backend through the registry.
	compiled := make([]Artifacts, 0, len(manifest.Backends))
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
		compiled = append(compiled, arts)
		logger.Info("backend compiled", "backend", backend, "artifacts", len(arts.Files))
	}
	return compiled, nil
}

// writeArtifacts applies a backend's artifacts under root with the idempotent,
// non-destructive write strategy (RF-6.3), and reports what it did per file.
//
// Managed area: the set of paths in `files` — and ONLY those — is what this build
// touches (REQ-1/REQ-7). Anything else on disk, inside or outside `.claude/`, is
// never read for writing, never overwritten and never deleted (REQ-3/REQ-5): a
// user's manual file the build does not produce survives untouched. Orphans (files
// in the managed directories the current build no longer produces) are detected by
// PlanArtifacts and exposed for the preview, but this writer leaves them in place
// — the safe default (Check-8).
//
// Per artifact it does compare-and-skip: it classifies the target against disk and
//   - StatusUnchanged ⇒ writes NOTHING (no truncate, no rename, no mtime churn), so
//     a re-run over an unchanged definition leaves the tree byte-identical with zero
//     spurious writes (REQ-2/Check-1/Check-2);
//   - StatusCreated/StatusUpdated ⇒ writes atomically (temp + rename), so a reader
//     never sees a partial file and a crash leaves the previous content intact.
//
// The change is always bounded to the managed area and reproducible (REQ-6): only
// artifacts whose canonical source changed end up updated.
func writeArtifacts(root string, files []Artifact) (created, updated, unchanged []string, err error) {
	for _, f := range files {
		status, _, cerr := classify(root, f)
		if cerr != nil {
			return nil, nil, nil, cerr
		}
		if status == StatusUnchanged {
			unchanged = append(unchanged, f.RelPath)
			continue
		}

		abs := filepath.Join(root, filepath.FromSlash(f.RelPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return nil, nil, nil, err
		}
		if err := writeAtomic(abs, f.Content); err != nil {
			return nil, nil, nil, err
		}
		if status == StatusCreated {
			created = append(created, f.RelPath)
		} else {
			updated = append(updated, f.RelPath)
		}
	}
	return created, updated, unchanged, nil
}

// writeAtomic writes content to path atomically: it writes to a temporary file in
// the same directory and renames it over path. The rename is atomic on the same
// filesystem, so a reader sees either the old or the new content, never a partial
// write — and a crash mid-write leaves the original intact. On any failure the
// temp file is cleaned up so we never litter the managed directory. This mirrors
// the workspace/prompts/catalog helpers of the same name; it is duplicated rather
// than shared because those are unexported and this package owns its own write
// semantics. The temp name shape (".<base>.tmp-*") is what detectOrphans skips.
func writeAtomic(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Best-effort cleanup if we bail before a successful rename; after a successful
	// rename tmpName no longer exists so the Remove is a harmless no-op.
	defer os.Remove(tmpName)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// noopWriter discards everything, backing the default no-op logger so a caller
// that supplies no Logger gets silence without the core reaching for a global.
type noopWriter struct{}

func (noopWriter) Write(p []byte) (int, error) { return len(p), nil }
