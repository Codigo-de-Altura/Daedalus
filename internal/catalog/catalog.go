package catalog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// AgentsDir is the workspace subdirectory that holds materialized agent
// definitions. It matches the canonical layout (init.md §4.2 / workspace.Subdirs
// "agents"); kept as a constant here so the catalog does not have to import the
// workspace package just for a directory name, and the two stay in sync by
// convention rather than a build-time coupling.
const AgentsDir = "agents"

// DefinitionFileName and PromptFileName are the two files that make up a
// materialized agent on disk. We lay each agent out in its own subdirectory —
// `.daedalus/agents/<id>/agent.yaml` + `.daedalus/agents/<id>/prompt.md` —
// rather than flat `<id>.yaml` + `<id>.md` files. Rationale: a per-agent
// directory keeps the pair self-contained, makes the id the single namespacing
// unit (no risk of a `.yaml`/`.md` pair drifting apart or colliding with another
// agent), and leaves room for future per-agent assets (examples, fixtures)
// without changing the layout. The structure map in init.md describes
// `agents/  # ... (yaml + prompt md)` without prescribing flat vs. nested, so a
// subdir is a conforming, more durable choice.
const (
	DefinitionFileName = "agent.yaml"
	PromptFileName     = "prompt.md"
)

// ErrAgentNotFound is returned by Get/MaterializePlan when a requested id is not
// in the built-in catalog. Exposed as a sentinel so callers (the CLI/TUI) can map
// "no such agent" to a usage error via errors.Is without string matching.
var ErrAgentNotFound = errors.New("agent not found in catalog")

// Catalog is a read-only collection of canonical agents. It is the entry point
// for listing the built-ins and materializing one into a workspace. A Catalog
// holds no mutable state, so the package-level Builtin is safe to share.
type Catalog struct {
	// agents is keyed by id for O(1) lookup; the source order lives in the
	// builtin slice and listings are sorted by id, so the map's iteration order
	// never leaks into output.
	agents map[string]Agent
	// ids preserves the insertion order of the source agents, used only as the
	// stable basis for building listings (which are then sorted by id).
	ids []string
}

// Builtin is the catalog of agents embedded in the binary (R1): the five
// canonical SDD agents and any future built-ins. It is built once at package
// init from the validated builtinAgents literals.
var Builtin = newBuiltin()

// newBuiltin constructs the built-in catalog from the literals, validating each
// agent so a malformed built-in is caught at startup (and in tests) rather than
// silently materializing a bad definition. A duplicate id is also a programming
// error in the literals, so we panic: this runs at package init over a fixed,
// in-binary set, never on user input.
func newBuiltin() *Catalog {
	c := &Catalog{agents: make(map[string]Agent, len(builtinAgents))}
	for _, a := range builtinAgents {
		if err := a.Validate(); err != nil {
			panic(fmt.Sprintf("built-in agent is malformed: %v", err))
		}
		if _, dup := c.agents[a.ID]; dup {
			panic(fmt.Sprintf("duplicate built-in agent id %q", a.ID))
		}
		c.agents[a.ID] = a
		c.ids = append(c.ids, a.ID)
	}
	return c
}

// List returns the available agents as id+role entries, sorted by id (R4/CA1).
// It is a projection that never exposes the full prompt/parameters, so a caller
// can present a selection menu cheaply and deterministically.
func (c *Catalog) List() []Entry {
	entries := make([]Entry, 0, len(c.ids))
	for _, id := range c.ids {
		a := c.agents[id]
		entries = append(entries, Entry{ID: a.ID, Role: a.Role})
	}
	sortEntries(entries)
	return entries
}

// Get returns the full canonical agent for an id, or ErrAgentNotFound. It
// returns a copy of the Agent value (params slice included via copy below) so a
// caller cannot mutate the catalog's source of truth.
func (c *Catalog) Get(id string) (Agent, error) {
	a, ok := c.agents[id]
	if !ok {
		return Agent{}, fmt.Errorf("%w: %q", ErrAgentNotFound, id)
	}
	// Defensive copy of the params slice: the Agent struct is copied by value,
	// but its slice header would otherwise alias the catalog's backing array.
	if len(a.Params) > 0 {
		params := make([]Param, len(a.Params))
		copy(params, a.Params)
		a.Params = params
	}
	return a, nil
}

// MaterializePlan is the result of planning a materialization: the resolved,
// rendered files for an agent and where they will live, computed without
// touching the filesystem. Like workspace.Plan it is a pure description so a
// caller can preview the exact content (and detect conflicts) before committing
// to writing. The rendered bytes are captured at plan time so a preview and the
// subsequent Apply describe identical content (R8/CA6).
type MaterializePlan struct {
	// AgentID is the id being materialized (kebab-case, validated).
	AgentID string
	// AgentsRoot is the absolute `.daedalus/agents` directory the agent lands in.
	AgentsRoot string
	// Dir is the agent's own directory (`<AgentsRoot>/<id>`).
	Dir string
	// Files are the canonical files to create, in deterministic order, each with
	// its workspace-relative-ish name (DefinitionFileName, PromptFileName) and
	// fully rendered content.
	Files []PlannedFile
}

// PlannedFile is one file of a MaterializePlan: its name within the agent dir,
// its absolute path and its rendered content.
type PlannedFile struct {
	Name    string
	Path    string
	Content string
}

// MaterializeResult reports what Apply produced so a caller can tell the user
// whether the agent was created or already present (the non-destructive case).
type MaterializeResult struct {
	// AgentID is the materialized agent's id.
	AgentID string
	// Dir is the agent's directory.
	Dir string
	// Created lists the files actually written (relative names), in order.
	Created []string
	// Skipped lists the files left untouched because they already existed —
	// the signal that the operation was non-destructive (R6/CA4). When Skipped is
	// non-empty the caller should surface a conflict rather than claim success.
	Skipped []string
}

// AlreadyExisted reports whether any target file was already present, i.e. the
// materialization was (fully or partially) a non-destructive no-op. Callers use
// it to decide between "created" and "already exists, not overwritten" messaging.
func (r *MaterializeResult) AlreadyExisted() bool {
	return len(r.Skipped) > 0
}

// MaterializePlanFor computes the plan to materialize agent id under the given
// `.daedalus/agents` directory, without writing anything. It validates the agent
// (so a malformed definition is rejected before any I/O) and renders both files
// deterministically. agentsRoot is typically `<root>/.daedalus/agents`; the
// caller owns where the workspace lives so this package does not duplicate the
// workspace-location logic.
func (c *Catalog) MaterializePlanFor(agentsRoot, id string) (*MaterializePlan, error) {
	a, err := c.Get(id)
	if err != nil {
		return nil, err
	}
	// Re-validate the resolved agent: Get returns built-ins that newBuiltin
	// already validated, but this keeps Materialize correct if the catalog ever
	// serves non-built-in agents (ticket-02-02/03) without a separate guard.
	if err := a.Validate(); err != nil {
		return nil, err
	}

	dir := filepath.Join(agentsRoot, a.ID)
	plan := &MaterializePlan{
		AgentID:    a.ID,
		AgentsRoot: agentsRoot,
		Dir:        dir,
		// Fixed order: definition first, then prompt. Deterministic and matches
		// the way a reader thinks about an agent (metadata, then body).
		Files: []PlannedFile{
			{Name: DefinitionFileName, Path: filepath.Join(dir, DefinitionFileName), Content: renderDefinition(a)},
			{Name: PromptFileName, Path: filepath.Join(dir, PromptFileName), Content: renderPrompt(a)},
		},
	}
	return plan, nil
}

// Apply writes the planned files non-destructively: it creates the agent
// directory and each file only if it does not already exist (O_CREATE|O_EXCL),
// never truncating or overwriting an existing file. An agent already present in
// the workspace is therefore reported via Result.Skipped rather than clobbered
// (R6/CA4). Because every file's content was rendered deterministically at plan
// time, re-materializing the same agent into a clean workspace yields byte-
// identical files (R8/CA6).
func (p *MaterializePlan) Apply() (*MaterializeResult, error) {
	res := &MaterializeResult{AgentID: p.AgentID, Dir: p.Dir}

	// MkdirAll is non-destructive on an existing directory, so a re-run over an
	// already-materialized agent reuses its directory and the O_EXCL writes below
	// are what protect the actual content.
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		return nil, err
	}

	for _, f := range p.Files {
		created, err := ensureFile(f.Path, f.Content)
		if err != nil {
			return nil, err
		}
		if created {
			res.Created = append(res.Created, f.Name)
		} else {
			res.Skipped = append(res.Skipped, f.Name)
		}
	}
	return res, nil
}

// Materialize is the convenience that plans then applies in one call, for callers
// that do not need to preview the content first. New callers that want a preview
// (e.g. the TUI showing a diff) should use MaterializePlanFor then Apply.
func (c *Catalog) Materialize(agentsRoot, id string) (*MaterializeResult, error) {
	plan, err := c.MaterializePlanFor(agentsRoot, id)
	if err != nil {
		return nil, err
	}
	return plan.Apply()
}

// ensureFile creates a file at path with the given deterministic content only if
// it does not already exist. O_EXCL makes the create-or-skip decision atomic and
// non-destructive: an existing file is never truncated, so a materialized agent
// the user has edited survives a re-run untouched. It reports whether it created
// a new file (created=false when the path already existed). This mirrors the
// workspace package's helper of the same name; it is duplicated rather than
// shared because that one is unexported and the two packages must be able to
// evolve their write semantics independently.
func ensureFile(path, content string) (bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, err
	}
	// Write then close without defer: a deferred Close would mask a Write error.
	// Write first, capture its error, then close and surface whichever occurred
	// (Write takes precedence — a failed Write leaves the file in a bad state).
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
