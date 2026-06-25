package compile

import (
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// DefinitionError aggregates everything wrong with the canonical definition that
// must be fixed before it can be compiled (REQ-3). It is the build's "validation
// error" type: the command maps it to the validation exit code, distinct from a
// compilation/write failure. It collects problems across every agent in a single
// pass so the user fixes them in one cycle rather than one re-run per problem.
//
// A DefinitionError carries two kinds of finding, both actionable:
//   - Malformed sources: files that could not be parsed/loaded at all (the path
//     and the reason).
//   - Invalid agents: agents that parsed but fail the canonical schema, each
//     keeping its rich *catalog.ValidationError so the command can render every
//     field/observed/expected finding.
type DefinitionError struct {
	// Malformed lists sources that could not be loaded, in deterministic order.
	Malformed []SourceError
	// Invalid lists agents that loaded but failed schema validation, in
	// deterministic (id-sorted) order. Each error is a *catalog.ValidationError.
	Invalid []error
}

// SourceError pairs a source path with the reason it could not be loaded, so the
// command can name the exact file the user must fix.
type SourceError struct {
	// ID is the agent id (directory name) the source belongs to, for context.
	ID string
	// Err is the load/parse failure.
	Err error
}

// Error renders the aggregate as actionable lines, malformed sources first then
// invalid agents, so the message is self-contained. The order is the
// deterministic order the loader produced, so the rendered text is stable.
func (e *DefinitionError) Error() string {
	total := len(e.Malformed) + len(e.Invalid)
	msg := fmt.Sprintf("canonical definition is invalid (%d problem%s)", total, plural(total))
	for _, s := range e.Malformed {
		msg += fmt.Sprintf("\n  - %s: %v", s.ID, s.Err)
	}
	for _, v := range e.Invalid {
		msg += "\n  - " + v.Error()
	}
	return msg
}

// plural is a tiny local pluralizer; the core must not depend on the cmd
// package's helper.
func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// LoadDefinition reads the canonical definition from the workspace under root and
// validates it, returning a ready-to-compile Definition or a *DefinitionError
// listing every problem (REQ-3). It performs no compilation and no writes — it is
// the read+validate half of the pipeline, kept separate so the command can abort
// on an invalid definition before any adapter or write is involved.
//
// It is deterministic: agents are returned id-sorted and findings are collected
// in that same order, so the same workspace always yields the same Definition or
// the same error text (RNF-5).
//
// An empty agents directory is not an error here: a workspace with no agents is a
// valid (if trivial) definition. Whether a backend needs at least one agent is a
// backend concern, surfaced by its Compiler, not a property of the canonical
// definition.
//
// LoadDefinition is the public, logger-free entry point preserved for callers that
// only want the result; the build pipeline uses loadDefinition to also emit a
// per-definition validation event at each decision point (RF-9.1/CA3).
func LoadDefinition(root string) (Definition, error) {
	return loadDefinition(root, discardLogger())
}

// loadDefinition is LoadDefinition with decision-point instrumentation: it emits
// one structured event per canonical definition (agent/command) reporting whether
// it was accepted, rejected (invalid schema) or malformed, and — on rejection —
// the reason, so a build's trace reconstructs exactly which definition was
// evaluated and why it passed or failed (RF-9.1/CA3). The logging is purely
// observational: the returned Definition/error is identical with or without it, so
// the validate-before-write contract and determinism (RNF-5/CA6) are untouched.
func loadDefinition(root string, logger *slog.Logger) (Definition, error) {
	var (
		def    Definition
		defErr DefinitionError
	)

	if err := loadAgents(root, &def, &defErr, logger); err != nil {
		return Definition{}, err
	}
	if err := loadCommands(root, &def, &defErr, logger); err != nil {
		return Definition{}, err
	}

	if len(defErr.Malformed) > 0 || len(defErr.Invalid) > 0 {
		return Definition{}, &defErr
	}
	return def, nil
}

// discardLogger returns a logger that drops every record, so the logger-free
// LoadDefinition and any test path can run the instrumented loader without a sink.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(noopWriter{}, nil))
}

// loadAgents reads and validates the canonical agents into def, appending any
// per-source problem to defErr. A real I/O error (other than a missing agents
// directory, which simply means "no agents") is returned so the caller surfaces
// it rather than treating it as a validation problem.
//
// It emits one decision-point event per agent (CA3): a malformed source or a
// schema-invalid agent is logged at warn with its workspace-relative path and the
// reason for rejection; an accepted agent is logged at info. The reason is taken
// from the structured error text (field/observed/expected for invalid agents),
// which never carries the prompt body or any secret (R5/CA5).
func loadAgents(root string, def *Definition, defErr *DefinitionError, logger *slog.Logger) error {
	agentsRoot := filepath.Join(root, workspace.Name, catalog.AgentsDir)
	ids, err := subdirIDs(agentsRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // no agents directory ⇒ nothing to load
		}
		return err
	}
	for _, id := range ids {
		rel := definitionRel(catalog.AgentsDir, id)
		a, err := catalog.Load(agentsRoot, id)
		if err != nil {
			defErr.Malformed = append(defErr.Malformed, SourceError{ID: id, Err: err})
			logger.Warn("definition rejected", "phase", "validate", "kind", "agent",
				"id", id, "definition", rel, "result", "malformed", "reason", err.Error())
			continue
		}
		if err := a.Validate(); err != nil {
			defErr.Invalid = append(defErr.Invalid, err)
			logger.Warn("definition rejected", "phase", "validate", "kind", "agent",
				"id", id, "definition", rel, "result", "invalid", "reason", err.Error())
			continue
		}
		def.Agents = append(def.Agents, a)
		logger.Info("definition accepted", "phase", "validate", "kind", "agent",
			"id", id, "definition", rel, "result", "valid")
	}
	return nil
}

// loadCommands reads the canonical prompts and turns each into a Command whose
// body is fully composed (inclusions expanded via prompts.Resolve), so the
// Compiler sees self-contained, deterministic data. A prompt that fails to load
// or whose inclusions cannot be resolved is recorded as a malformed source so the
// build aborts before writing (validate-before-write), naming the exact prompt.
//
// Unlike prompts.List — which silently skips a corrupt file to keep `list`
// usable — the build must be honest about every problem, so we enumerate the ids
// ourselves and surface each failure instead of hiding it.
func loadCommands(root string, def *Definition, defErr *DefinitionError, logger *slog.Logger) error {
	promptsRoot := filepath.Join(root, workspace.Name, prompts.PromptsDir)
	ids, err := promptIDs(promptsRoot)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil // no prompts directory ⇒ no commands
		}
		return err
	}
	for _, id := range ids {
		rel := definitionRel(prompts.PromptsDir, id+prompts.FileExt)
		p, err := prompts.Load(promptsRoot, id)
		if err != nil {
			defErr.Malformed = append(defErr.Malformed, SourceError{ID: id, Err: err})
			logger.Warn("definition rejected", "phase", "validate", "kind", "command",
				"id", id, "definition", rel, "result", "malformed", "reason", err.Error())
			continue
		}
		// Compose the body so the emitted command is self-contained: an unresolved
		// inclusion (missing reference or cycle) is a definition problem, not a
		// silent partial command.
		body, err := prompts.Resolve(promptsRoot, id)
		if err != nil {
			defErr.Malformed = append(defErr.Malformed, SourceError{ID: id, Err: err})
			logger.Warn("definition rejected", "phase", "validate", "kind", "command",
				"id", id, "definition", rel, "result", "malformed", "reason", err.Error())
			continue
		}
		def.Commands = append(def.Commands, Command{
			ID:          p.ID,
			Description: p.Title,
			Body:        body,
		})
		logger.Info("definition accepted", "phase", "validate", "kind", "command",
			"id", id, "definition", rel, "result", "valid")
	}
	return nil
}

// definitionRel renders a canonical definition's location as a workspace-relative,
// slash-separated path (e.g. ".daedalus/agents/my-agent"), the form every
// per-definition log event uses so the trace points at the exact source without
// leaking an absolute path (R5/CA5) and reads identically across operating
// systems (RNF-5).
func definitionRel(domainDir, leaf string) string {
	return filepath.ToSlash(filepath.Join(workspace.Name, domainDir, leaf))
}

// subdirIDs lists the agent ids materialized under agentsRoot — the
// subdirectories, in sorted order — so loading is deterministic. Non-directory
// entries are ignored (the layout puts each agent in its own directory,
// catalog.AgentsDir), so a stray loose file never derails the build. A directory
// whose name is not valid kebab-case is skipped here and would be caught by
// catalog.Load if it were a real agent; we do not invent ids.
func subdirIDs(agentsRoot string) ([]string, error) {
	entries, err := os.ReadDir(agentsRoot)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		ids = append(ids, e.Name())
	}
	sort.Strings(ids)
	return ids, nil
}

// promptIDs lists the prompt ids under promptsRoot — the base names of the flat
// `<id>.md` files, in sorted order — so command loading is deterministic. Unlike
// agents (one directory each), prompts live flat (prompts.PromptsDir), keyed by
// id. Directories and non-`.md` files are ignored; a base name that is not valid
// kebab-case is skipped (it is not one of ours), matching prompts.List's identity
// rule without importing its body.
func promptIDs(promptsRoot string) ([]string, error) {
	entries, err := os.ReadDir(promptsRoot)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, prompts.FileExt) {
			continue
		}
		id := strings.TrimSuffix(name, prompts.FileExt)
		if !prompts.IsKebabCase(id) {
			continue
		}
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids, nil
}

// IsDefinitionInvalid reports whether err is (or wraps) a *DefinitionError — the
// build's canonical-validation failure. The command uses it to map a validation
// error to its dedicated exit code via errors.As rather than matching a message,
// mirroring how the catalog flows detect *catalog.ValidationError.
func IsDefinitionInvalid(err error) bool {
	var de *DefinitionError
	return errors.As(err, &de)
}
