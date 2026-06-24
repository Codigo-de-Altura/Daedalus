package catalog

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrUnrecognizedSource is returned (wrapped) when a source file is neither a
// recognized Claude Code agent (Markdown with YAML frontmatter) nor a canonical
// definition we can load. It is a sentinel so a caller can distinguish "I do not
// know how to read this" from "I read it and it is invalid".
var ErrUnrecognizedSource = errors.New("unrecognized agent source")

// ImportOutcome is the per-agent result of an import, so a directory import can
// report each agent independently (imported / skipped-existing / failed) instead
// of collapsing into a single pass/fail. It mirrors the workspace package's
// preference for describing exactly what happened over a boolean.
type ImportOutcome struct {
	// SourcePath is the file the agent was read from, for traceable reporting. It
	// is set for failed sources (which have no id yet) so the error names the file.
	SourcePath string
	// AgentID is the canonical, kebab-case id the agent maps to. Set on success and
	// on a write failure (where the agent parsed but could not be persisted).
	AgentID string
	// Dir is the agent's target directory, surfaced for reporting on the write path.
	Dir string
	// Created/Skipped list the canonical files written / left in place (the
	// non-destructive case), mirroring MaterializeResult.
	Created []string
	Skipped []string
	// Err is non-nil when this source could not be imported. The other agents in a
	// directory import are unaffected (see ImportPlanFor's policy).
	Err error
}

// AlreadyExisted reports the non-destructive case: the agent's files were already
// present and left untouched.
func (o *ImportOutcome) AlreadyExisted() bool {
	return len(o.Skipped) > 0
}

// ImportPlan is the pure, side-effect-free description of what an import would do:
// one MaterializePlan per importable source agent, plus the parse/normalization
// errors for sources that could not even be planned. Holding a plan writes
// nothing, so a caller can preview it (CLI --preview) before applying.
type ImportPlan struct {
	// Agents are the successfully parsed-and-validated agents ready to materialize,
	// each as a deterministic MaterializePlan. Sorted by AgentID for stable output.
	Agents []*MaterializePlan
	// Errors are sources that failed to parse, normalize or validate, paired with
	// their path. They never abort the importable ones (see policy below); they are
	// reported alongside so nothing fails silently (R4/CA3).
	Errors []ImportError
}

// ImportError pairs a source path with the reason it could not be imported, so a
// directory import can report every failure with enough context to act on it.
type ImportError struct {
	SourcePath string
	Err        error
}

// ImportPlanFor scans a local source path — a single file or a directory — and
// builds the plan to import every agent it finds into agentsRoot, without writing
// anything. It is the detection half of import, mirroring workspace.Detect: the
// returned plan is a preview, and ImportPlan.Apply materializes it.
//
// Directory policy (R1): every regular file directly inside the directory is
// treated as a candidate source (matching a `.claude/agents/` layout, where each
// agent is one `*.md`). The scan is shallow and deterministic (entries sorted by
// name). A source that fails to parse/normalize/validate is recorded in
// Errors and does NOT abort the rest — the importable agents still plan
// successfully, so one bad file never silently blocks the good ones. Source
// recognition (R2/R3): a file with YAML frontmatter is read as Claude Code; a
// file that parses as our canonical single-file form is read as canonical;
// anything else is an unrecognized-source error.
func ImportPlanFor(agentsRoot, source string) (*ImportPlan, error) {
	info, err := os.Stat(source)
	if err != nil {
		// A missing/unreadable source path is an operational error, not a per-agent
		// one: there is nothing to iterate, so fail the whole call.
		return nil, err
	}

	var files []string
	if info.IsDir() {
		entries, err := os.ReadDir(source)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue // shallow scan: nested directories are not agent files
			}
			files = append(files, filepath.Join(source, e.Name()))
		}
		// os.ReadDir already returns entries sorted by name, but sort the resolved
		// paths explicitly so the plan order — and any aggregate output — is stable
		// regardless of the OS's directory iteration (R7/CA6).
		sort.Strings(files)
	} else {
		files = []string{source}
	}

	plan := &ImportPlan{}
	for _, f := range files {
		agent, err := importOne(f)
		if err != nil {
			plan.Errors = append(plan.Errors, ImportError{SourcePath: f, Err: err})
			continue
		}
		plan.Agents = append(plan.Agents, planMaterialize(agentsRoot, agent))
	}

	// Stable order independent of filesystem iteration: sort the materialize plans
	// by their canonical id.
	sort.Slice(plan.Agents, func(i, j int) bool { return plan.Agents[i].AgentID < plan.Agents[j].AgentID })
	return plan, nil
}

// Apply materializes every planned agent non-destructively (O_EXCL) and returns a
// per-agent outcome. It also surfaces the plan's parse/validation errors as failed
// outcomes, so a single Apply result reports the complete picture: what was
// imported, what already existed and was skipped, and what could not be imported
// and why. Applying never overwrites an existing agent (R5/CA4); a conflicting id
// comes back via the outcome's Skipped list.
func (p *ImportPlan) Apply() ([]ImportOutcome, error) {
	outcomes := make([]ImportOutcome, 0, len(p.Agents)+len(p.Errors))

	for _, mp := range p.Agents {
		res, err := mp.Apply()
		if err != nil {
			// A genuine I/O failure during one agent's write is reported per-agent so
			// the others still get their chance; the operation as a whole did not abort.
			outcomes = append(outcomes, ImportOutcome{AgentID: mp.AgentID, Dir: mp.Dir, Err: err})
			continue
		}
		outcomes = append(outcomes, ImportOutcome{
			AgentID: res.AgentID,
			Dir:     res.Dir,
			Created: res.Created,
			Skipped: res.Skipped,
		})
	}

	for _, e := range p.Errors {
		outcomes = append(outcomes, ImportOutcome{SourcePath: e.SourcePath, Err: e.Err})
	}
	return outcomes, nil
}

// Import is the convenience that plans then applies an import in one call, for
// callers that do not need to preview first. Callers that want a preview (the CLI
// --preview) use ImportPlanFor then Apply.
func Import(agentsRoot, source string) ([]ImportOutcome, error) {
	plan, err := ImportPlanFor(agentsRoot, source)
	if err != nil {
		return nil, err
	}
	return plan.Apply()
}

// importOne reads a single source file and converts it to a canonical Agent,
// choosing the reader by content (R2/R3): a leading YAML frontmatter block marks
// a Claude Code agent; otherwise we try to parse it as our canonical single-file
// form. The resulting agent is validated structurally before being returned, so
// an invalid source never produces a plan (R4/CA3). It performs no writes.
func importOne(path string) (Agent, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Agent{}, err
	}
	content := string(raw)

	var agent Agent
	switch {
	case hasFrontmatter(content):
		agent, err = fromClaudeCode(content, path)
	case looksCanonical(content):
		agent, err = fromCanonicalFile(content, path)
	default:
		return Agent{}, fmt.Errorf("%w: %s (no YAML frontmatter and not a canonical definition)",
			ErrUnrecognizedSource, filepath.Base(path))
	}
	if err != nil {
		return Agent{}, err
	}

	// Structural validation is the stand-in for the formal canonical schema
	// (ticket-02-04, not yet implemented): id kebab-case, role/prompt non-empty,
	// params valid. A failing source is rejected here with an actionable error and
	// never reaches a write (R4/CA3).
	if err := agent.Validate(); err != nil {
		return Agent{}, err
	}
	return agent, nil
}

// hasFrontmatter reports whether content opens with a YAML frontmatter block
// (`---` on the very first line). This is the discriminator for the Claude Code
// format; canonical single-file definitions never start with `---`.
func hasFrontmatter(content string) bool {
	return strings.HasPrefix(content, "---\n") || strings.HasPrefix(content, "---\r\n") || content == "---"
}

// looksCanonical reports whether content looks like our canonical single-file
// definition — it carries the top-level `id:` and `role:` keys we emit. This is a
// cheap shape check; fromCanonicalFile does the strict parsing and rejects
// anything that does not actually conform.
func looksCanonical(content string) bool {
	hasID, hasRole := false, false
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "id:") {
			hasID = true
		}
		if strings.HasPrefix(line, "role:") {
			hasRole = true
		}
	}
	return hasID && hasRole
}
