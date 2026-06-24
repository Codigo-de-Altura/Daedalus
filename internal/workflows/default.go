package workflows

// The factory-default SDD workflow (ticket 04-04).
//
// Daedalus ships a ready-to-use workflow, `sdd-default.yaml`, so a user gets the
// default SDD pipeline without authoring it by hand (R1). This file is the single
// source of truth for that pipeline: DefaultWorkflow builds the canonical model
// and the existing deterministic renderer (render.go) turns it into the on-disk
// bytes. There is deliberately NO loose `sdd-default.yaml` checked into the repo —
// the content is generated from this function so it can never drift from the model
// and is byte-stable across builds (R8/CA7). The CLI seeds it into a workspace at
// `daedalus init` time (see cmd/daedalus), non-destructively.
//
// # The pipeline (R3/R4/R5)
//
// The default SDD pipeline is brief -> spec -> architecture -> epics -> tickets ->
// ⟨external implementation⟩ -> validation -> docs, each phase run by a built-in
// catalog agent and crossing a validation gate. As a linear DAG:
//
//	spec          analyst     [brief]        -> [spec]          depends_on []
//	architecture  architect   [spec]         -> [architecture]  depends_on [spec]
//	epics         planner     [architecture] -> [epics]         depends_on [architecture]
//	tickets       planner     [epics]        -> [tickets]       depends_on [epics]
//	validation    validator   [tickets]      -> [validation]    depends_on [tickets]
//	docs          documenter  [validation]   -> [docs]          depends_on [validation]
//
// # Why these agents
//
// The five built-in agents are analyst, architect, planner, validator and
// documenter (catalog/builtin.go). There is intentionally no "developer"/
// "implementer" agent, because the implementation step is NOT executed by Daedalus
// (see below). Each phase references exactly one of those five, so the workflow
// passes the unknown-agent check of the semantic validator (04-03) with the
// built-ins as the known set (R7/CA6).
//
// # How the external implementation step is reflected (R6)
//
// The conceptual pipeline has an *external implementation* step between `tickets`
// and `validation`: a developer (or an agent on the backend) turns the tickets
// into an implementation, which `validation` then checks. Daedalus does not run
// that step in phase 1 (PRD §4.2, D5).
//
// We reflect it *structurally*, not as a data artifact. The conceptual diagram in
// the spec shows `validation inputs: [tickets, implementation]`, but our semantic
// validator (04-03) treats ONLY `brief` as an external/initial artifact: any input
// that is neither `brief` nor an output of a transitive predecessor is reported as
// a missing artifact. Introducing an `implementation` input that no phase produces
// would therefore make this very workflow fail R7/CA6. So instead:
//
//   - `validation` depends on `tickets` and consumes `[tickets]`.
//   - The external implementation is the (un-modeled) gap on the edge tickets ->
//     validation: Daedalus hands the tickets off, an external actor implements,
//     and validation resumes from the tickets the implementation was built from.
//
// This keeps the workflow a clean, valid linear DAG while still faithfully placing
// validation *after* the implementation gap. The spec's diagram is explicitly
// "conceptual"; the hard requirement R7/CA6 (pass the validator) governs, and it
// is satisfied without modifying the already-shipped 04-03 validator.
//
// # depends_on names phase ids, not artifacts
//
// `depends_on` lists predecessor *phase ids* (the DAG edges), never artifacts.
// `spec` is the root: it consumes the external `brief` via `inputs`, not via
// `depends_on`, so its depends_on is empty — listing `brief` there would be flagged
// as an unknown-dependency (brief is not a phase). Every other phase depends on the
// single phase before it, giving a clean linear chain with no cycles.

// DefaultWorkflowName is the canonical name (and file base name) of the factory
// SDD workflow: it is persisted as `<DefaultWorkflowName>.yaml` in the workspace's
// workflows directory.
const DefaultWorkflowName = "sdd-default"

// Phase ids of the default pipeline, exported so callers (the CLI, tests) refer to
// them by a stable identifier rather than hardcoding strings.
const (
	DefaultPhaseSpec         = "spec"
	DefaultPhaseArchitecture = "architecture"
	DefaultPhaseEpics        = "epics"
	DefaultPhaseTickets      = "tickets"
	DefaultPhaseValidation   = "validation"
	DefaultPhaseDocs         = "docs"
)

// DefaultWorkflow returns the canonical model of the factory SDD workflow (R3/R4/
// R5). It is a pure constructor — no I/O — so it is safe to call anywhere and
// always yields an identical model, which Render then serializes to byte-stable
// bytes (R8). The model passes the semantic validator (04-03) with the built-in
// agents as the known set (R7/CA6); see this file's doc-comment for the why.
//
// Each phase carries a per-phase gate named `<id>-gate`. The gate names are part
// of the canonical default: a stable, predictable convention a user can rename
// later, and concrete enough that the file is immediately usable.
func DefaultWorkflow() Workflow {
	return Workflow{
		Name: DefaultWorkflowName,
		Phases: []Phase{
			{
				ID:        DefaultPhaseSpec,
				Agent:     "analyst",
				Inputs:    []string{"brief"},
				Outputs:   []string{"spec"},
				Gate:      "spec-gate",
				DependsOn: []string{},
			},
			{
				ID:        DefaultPhaseArchitecture,
				Agent:     "architect",
				Inputs:    []string{"spec"},
				Outputs:   []string{"architecture"},
				Gate:      "architecture-gate",
				DependsOn: []string{DefaultPhaseSpec},
			},
			{
				ID:        DefaultPhaseEpics,
				Agent:     "planner",
				Inputs:    []string{"architecture"},
				Outputs:   []string{"epics"},
				Gate:      "epics-gate",
				DependsOn: []string{DefaultPhaseArchitecture},
			},
			{
				ID:        DefaultPhaseTickets,
				Agent:     "planner",
				Inputs:    []string{"epics"},
				Outputs:   []string{"tickets"},
				Gate:      "tickets-gate",
				DependsOn: []string{DefaultPhaseEpics},
			},
			{
				ID:        DefaultPhaseValidation,
				Agent:     "validator",
				Inputs:    []string{"tickets"},
				Outputs:   []string{"validation"},
				Gate:      "validation-gate",
				DependsOn: []string{DefaultPhaseTickets},
			},
			{
				ID:        DefaultPhaseDocs,
				Agent:     "documenter",
				Inputs:    []string{"validation"},
				Outputs:   []string{"docs"},
				Gate:      "docs-gate",
				DependsOn: []string{DefaultPhaseValidation},
			},
		},
	}
}
