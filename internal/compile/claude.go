package compile

import (
	"errors"

	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// ErrNotImplemented is the sentinel a not-yet-completed Compiler returns from
// Compile. The orchestrator maps it to a compilation error (a non-validation,
// non-I/O failure) so the exit code and message are honest: the build did not
// fabricate artifacts, it stopped because the adapter cannot yet produce them.
// ticket-06-02 removes this from the Claude Code adapter once the real mapping
// lands.
var ErrNotImplemented = errors.New("backend adapter not yet implemented")

// claudeCompiler is the Claude Code adapter — the MVP's first and only backend
// (PRD D3). It is registered in DefaultRegistry so the build command already
// routes "claude-code" here through the registry: the surface, the routing, the
// validation-before-write and the exit codes are all live and testable today.
//
// The canonical → `.claude/` mapping itself (frontmatter agents, commands,
// settings) is ticket-06-02's deliverable. Until then Compile is a deliberate,
// clearly-marked stub that returns ErrNotImplemented rather than inventing
// artifacts: a half-real mapping would be worse than an explicit "not yet". The
// type, its registration and its Backend() are the stable parts 06-02 keeps; only
// the body of Compile is filled in.
type claudeCompiler struct{}

// newClaudeCompiler constructs the Claude Code adapter. It is a function (not a
// bare literal) so 06-02 can give the adapter configuration/dependencies without
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

// Compile is the canonical → Claude Code mapping. It is intentionally unfinished:
// it returns ErrNotImplemented (RF-6.2 territory) so the orchestration around it
// — routing, the validate-before-write gate, the differentiated exit codes — can
// be exercised today without a half-built mapping masquerading as success. The
// _ = def keeps the parameter named for the signature 06-02 implements against.
func (c *claudeCompiler) Compile(def Definition) (Artifacts, error) {
	_ = def
	return Artifacts{}, ErrNotImplemented
}
