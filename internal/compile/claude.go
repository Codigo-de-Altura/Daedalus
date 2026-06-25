package compile

import (
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// claudeCompiler is the Claude Code adapter — the MVP's first backend (PRD D3).
// It is registered in DefaultRegistry so the build command routes "claude-code"
// here through the registry; the surface, the routing, the validate-before-write
// gate and the exit codes are owned by ticket-06-01, and this type owns the
// canonical → `.claude/` mapping (RF-6.2).
//
// Compile is a pure function of its Definition (no I/O): every filesystem read —
// loading agents, composing a prompt's inclusions into a command body — already
// happened in LoadDefinition, so the mapping is deterministic and the same
// canonical input yields byte-identical artifacts (RNF-5). The non-destructive
// write strategy (RF-6.3) and the diff/preview (RF-6.4) consume the pure
// Artifacts this returns; they are not this adapter's concern.
type claudeCompiler struct{}

// newClaudeCompiler constructs the Claude Code adapter. It is a function (not a
// bare literal) so the adapter can later take configuration/dependencies without
// changing its call sites in the registry.
func newClaudeCompiler() *claudeCompiler {
	return &claudeCompiler{}
}

// Backend returns the canonical backend key, anchored to the workspace package's
// default so the adapter and the supported-backend set can never disagree about
// the spelling of "claude-code".
func (c *claudeCompiler) Backend() string {
	return workspace.DefaultBackend
}

// Compile maps the validated canonical Definition to Claude Code's native
// `.claude/` artifacts (RF-6.2). It emits, in a FIXED order so the output is
// deterministic and any diff is stable (RNF-5):
//
//  1. one `.claude/agents/<id>.md` per agent (frontmatter + prompt), id-sorted;
//  2. one `.claude/commands/<id>.md` per command (optional frontmatter + composed
//     body), id-sorted;
//  3. a single `.claude/settings.json` — a minimal, honest managed marker.
//
// The Definition already delivers agents and commands in id-sorted order
// (LoadDefinition), so iterating them preserves that order. File names are
// kebab-case, derived directly from the canonical id (REQ-9), so they are stable
// across runs. Compile performs no I/O and never fails for the MVP mapping (the
// inputs are pre-validated), but it keeps the error return for adapters/extensions
// whose mapping can genuinely fail.
func (c *claudeCompiler) Compile(def Definition) (Artifacts, error) {
	arts := Artifacts{Backend: c.Backend()}

	for _, a := range def.Agents {
		arts.Files = append(arts.Files, Artifact{
			RelPath: claudeAgentsDir + "/" + a.ID + mdExt,
			Content: renderAgent(a),
		})
	}
	for _, cmd := range def.Commands {
		arts.Files = append(arts.Files, Artifact{
			RelPath: claudeCommandsDir + "/" + cmd.ID + mdExt,
			Content: renderCommand(cmd),
		})
	}
	arts.Files = append(arts.Files, Artifact{
		RelPath: claudeSettingsPath,
		Content: renderSettings(),
	})

	return arts, nil
}
