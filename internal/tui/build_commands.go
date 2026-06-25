package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Codigo-de-Altura/Daedalus/internal/compile"
)

// This file owns the bridge between the build-preview TUI and the compile core
// (internal/compile). Like commands.go, every call into the core happens inside a
// tea.Cmd so the UI thread never blocks on filesystem I/O or compilation, and the
// result is delivered back to Update as a typed Msg. The presentation layer never
// computes a plan or writes a file itself — it only triggers these commands and
// renders their results.
//
// The two core entry points are injected as function fields on buildModel (planFn
// / buildFn) rather than called directly, so tests can drive the model's state
// machine and its write effect without a real workspace or a real terminal. In
// production New wires them to compile.Plan / compile.Build over a fixed Options.

// planFunc computes the read-only build plan (compile.Plan) — what a build WOULD
// do, with nothing written. It is the source of every classification and diff the
// preview renders.
type planFunc func() (*compile.PlanResult, error)

// buildFunc performs the actual write (compile.Build, non-preview). It is invoked
// ONLY when the user explicitly confirms the gate; recompiling at confirm time is
// the core's deliberate, TOCTOU-safe design (see compile.PlanResult docs).
type buildFunc func() (*compile.Outcome, error)

// planLoadedMsg reports the result of computing the build plan. Exactly one of
// result / err is meaningful: a non-nil err means the plan could not be computed
// (missing workspace, invalid definition, unroutable backend) and drives the
// preview into its actionable error state; a nil err carries the full plan to
// render (which may legitimately be all-unchanged — a valid "no changes" state).
type planLoadedMsg struct {
	result *compile.PlanResult
	err    error
}

// buildDoneMsg reports the result of the confirmed write (compile.Build). A
// non-nil err means the write failed after confirmation (an I/O problem); a nil
// err carries the outcome whose counts the "written" screen reports.
type buildDoneMsg struct {
	outcome *compile.Outcome
	err     error
}

// planCmd runs the read-only plan off the UI thread and delivers a planLoadedMsg.
func planCmd(plan planFunc) tea.Cmd {
	return func() tea.Msg {
		res, err := plan()
		return planLoadedMsg{result: res, err: err}
	}
}

// buildCmd performs the confirmed write off the UI thread and delivers a
// buildDoneMsg. It is only ever returned from Update after an explicit
// confirmation, so a write can never happen without the user's go-ahead (REQ-4).
func buildCmd(build buildFunc) tea.Cmd {
	return func() tea.Msg {
		out, err := build()
		return buildDoneMsg{outcome: out, err: err}
	}
}

// planFnFor returns a planFunc bound to compile.Plan over root. Plan never writes
// regardless of Options.Preview, so this is safe in every mode.
func planFnFor(root string) planFunc {
	return func() (*compile.PlanResult, error) {
		return compile.Plan(compile.Options{Root: root})
	}
}

// buildFnFor returns a buildFunc bound to compile.Build over root in WRITE mode
// (Preview: false). It is only wired into a confirmable model, never a read-only
// (--preview) one, so the no-write guarantee is structural.
func buildFnFor(root string) buildFunc {
	return func() (*compile.Outcome, error) {
		return compile.Build(compile.Options{Root: root, Preview: false})
	}
}
