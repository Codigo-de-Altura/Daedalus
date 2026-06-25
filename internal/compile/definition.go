package compile

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
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
func LoadDefinition(root string) (Definition, error) {
	agentsRoot := filepath.Join(root, workspace.Name, catalog.AgentsDir)

	ids, err := agentIDs(agentsRoot)
	if err != nil {
		// A missing agents directory means there is simply nothing to load (the
		// workspace presence check is the command's job, REQ-2); any other I/O error
		// is a real failure the caller surfaces.
		if errors.Is(err, fs.ErrNotExist) {
			return Definition{}, nil
		}
		return Definition{}, err
	}

	var (
		def    Definition
		defErr DefinitionError
	)
	for _, id := range ids {
		a, err := catalog.Load(agentsRoot, id)
		if err != nil {
			defErr.Malformed = append(defErr.Malformed, SourceError{ID: id, Err: err})
			continue
		}
		if err := a.Validate(); err != nil {
			defErr.Invalid = append(defErr.Invalid, err)
			continue
		}
		def.Agents = append(def.Agents, a)
	}

	if len(defErr.Malformed) > 0 || len(defErr.Invalid) > 0 {
		return Definition{}, &defErr
	}
	return def, nil
}

// agentIDs lists the agent ids materialized under agentsRoot — the subdirectories,
// in sorted order — so loading is deterministic. Non-directory entries are
// ignored (the layout puts each agent in its own directory, catalog.AgentsDir),
// so a stray loose file never derails the build. A directory whose name is not
// valid kebab-case is skipped here and would be caught by catalog.Load if it were
// a real agent; we do not invent ids.
func agentIDs(agentsRoot string) ([]string, error) {
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

// IsDefinitionInvalid reports whether err is (or wraps) a *DefinitionError — the
// build's canonical-validation failure. The command uses it to map a validation
// error to its dedicated exit code via errors.As rather than matching a message,
// mirroring how the catalog flows detect *catalog.ValidationError.
func IsDefinitionInvalid(err error) bool {
	var de *DefinitionError
	return errors.As(err, &de)
}
