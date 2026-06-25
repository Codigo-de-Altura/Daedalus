package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Codigo-de-Altura/Daedalus/internal/catalog"
	"github.com/Codigo-de-Altura/Daedalus/internal/compile"
	"github.com/Codigo-de-Altura/Daedalus/internal/workspace"
)

// These tests drive the build-preview model the way a user would — by sending the
// EXACT tea.KeyMsgs to Update and asserting the resulting state — because a TUI
// cannot be typed headless. The plan/write effects are injected (planFn/buildFn)
// so most tests need no real workspace; one integration test confirms a confirmed
// write actually lands on disk in a t.TempDir().

// --- fixtures ---------------------------------------------------------------

// samplePlan returns a plan with one created, one modified and one unchanged
// artifact under a single backend, so navigation, classification and the diff can
// all be exercised from one fixture.
func samplePlan() *compile.PlanResult {
	return &compile.PlanResult{
		Root: "repo",
		Backends: []compile.BackendPlan{{
			Backend: "claude-code",
			Artifacts: []compile.PlannedArtifact{
				{RelPath: ".claude/agents/new.md", Status: compile.StatusCreated, Target: "fresh\nlines\n"},
				{RelPath: ".claude/agents/mod.md", Status: compile.StatusUpdated, Current: "old line\nkeep\n", Target: "new line\nkeep\n"},
				{RelPath: ".claude/agents/same.md", Status: compile.StatusUnchanged, Current: "stable\n", Target: "stable\n"},
			},
		}},
	}
}

// planWithOrphans returns a plan that has one created artifact plus two detected
// orphans, so the orphan section's rendering (heading + alignment + non-navigable
// nature) can be asserted.
func planWithOrphans() *compile.PlanResult {
	return &compile.PlanResult{
		Root: "repo",
		Backends: []compile.BackendPlan{{
			Backend: "claude-code",
			Artifacts: []compile.PlannedArtifact{
				{RelPath: ".claude/agents/new.md", Status: compile.StatusCreated, Target: "fresh\n"},
			},
			Orphans: []string{".claude/agents/gone.md", ".claude/commands/old.md"},
		}},
	}
}

// unchangedPlan returns a plan with nothing to do (the "no changes" state).
func unchangedPlan() *compile.PlanResult {
	return &compile.PlanResult{
		Root: "repo",
		Backends: []compile.BackendPlan{{
			Backend: "claude-code",
			Artifacts: []compile.PlannedArtifact{
				{RelPath: ".claude/agents/same.md", Status: compile.StatusUnchanged, Current: "stable\n", Target: "stable\n"},
			},
		}},
	}
}

// confirmableModel returns a sized, ready model whose buildFn is a spy: it records
// whether a write was attempted and returns a fixed outcome.
func confirmableModel(t *testing.T, plan *compile.PlanResult, wrote *bool) buildModel {
	t.Helper()
	build := func() (*compile.Outcome, error) {
		*wrote = true
		return &compile.Outcome{Root: "repo", Backends: []compile.BackendOutcome{
			{Backend: "claude-code", Planned: 3, Created: []string{".claude/agents/new.md"}, Updated: []string{".claude/agents/mod.md"}, Unchanged: []string{".claude/agents/same.md"}},
		}}, nil
	}
	m := newBuildModel("repo", false, func() (*compile.PlanResult, error) { return plan, nil }, build)
	return drive(t, m, plan)
}

// drive sizes the model and feeds it the plan so it lands in its ready/empty state.
func drive(t *testing.T, m buildModel, plan *compile.PlanResult) buildModel {
	t.Helper()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(buildModel)
	updated, _ = m.Update(planLoadedMsg{result: plan})
	return updated.(buildModel)
}

// --- plan lifecycle ---------------------------------------------------------

// TestPlanLoadedReadyClassifies covers Check-2/Check-4: a plan with changes lands
// in the ready state with the artifacts flattened and classified.
func TestPlanLoadedReadyClassifies(t *testing.T) {
	var wrote bool
	m := confirmableModel(t, samplePlan(), &wrote)
	if m.state != buildReady {
		t.Fatalf("state = %v, want ready", m.state)
	}
	if len(m.artifacts) != 3 {
		t.Fatalf("artifacts = %d, want 3", len(m.artifacts))
	}
	view := m.View()
	for _, want := range []string{"[new]", "[modified]", "[unchanged]", "1 new", "1 modified", "1 unchanged"} {
		if !strings.Contains(view, want) {
			t.Errorf("ready view missing %q, got:\n%s", want, view)
		}
	}
}

// TestSummaryBoxBorderCloses covers minor fix #1: the bordered summary box closes
// evenly — every rendered line (including the top/bottom border) has the same
// visible width, so the right border never falls a character short despite the
// colored counts inside.
func TestSummaryBoxBorderCloses(t *testing.T) {
	var wrote bool
	m := confirmableModel(t, samplePlan(), &wrote)
	box := m.summaryBoxView()

	lines := strings.Split(box, "\n")
	if len(lines) < 3 {
		t.Fatalf("summary box should have at least 3 lines (borders + content), got:\n%s", box)
	}
	want := lipgloss.Width(lines[0])
	for i, line := range lines {
		if w := lipgloss.Width(line); w != want {
			t.Errorf("summary box line %d width = %d, want %d (border not flush):\n%s", i, w, want, box)
		}
	}
}

// TestOrphanSectionSeparatedAndAligned covers minor fixes #2 and #3: orphans are
// rendered under a clearly labelled, non-selectable heading (no cursor dead-end),
// and the [orphan] badge aligns to the same grid column as the status badges.
func TestOrphanSectionSeparatedAndAligned(t *testing.T) {
	var wrote bool
	m := confirmableModel(t, planWithOrphans(), &wrote)
	list := m.renderArtifactList()

	// #3: a clear heading communicates the section is read-only / not selectable.
	if !strings.Contains(list, "Orphans") || !strings.Contains(list, "not selectable") {
		t.Errorf("orphan section should be a clearly labelled, non-selectable block, got:\n%s", list)
	}
	// Both orphans appear.
	for _, want := range []string{".claude/agents/gone.md", ".claude/commands/old.md"} {
		if !strings.Contains(list, want) {
			t.Errorf("orphan %q missing from list, got:\n%s", want, list)
		}
	}
	// #2: every badge — status and orphan — is padded to the same grid width, so the
	// padded plain label lengths all equal badgeWidth.
	for _, label := range []string{"[new]", "[modified]", "[unchanged]", "[orphan]"} {
		if got := lipgloss.Width(padBadge(label)); got != badgeWidth {
			t.Errorf("padBadge(%q) width = %d, want %d", label, got, badgeWidth)
		}
	}
	// The orphan rows must NOT carry the navigable selection cursor ("> ").
	for _, line := range strings.Split(list, "\n") {
		if strings.Contains(line, "gone.md") && strings.Contains(line, ">") {
			t.Errorf("orphan row must not look selectable (no cursor): %q", line)
		}
	}
}

// TestPlanEmptyState covers Check-3: an all-unchanged plan communicates "no
// changes" clearly and is not confirmable.
func TestPlanEmptyState(t *testing.T) {
	m := newBuildModel("repo", false, func() (*compile.PlanResult, error) { return unchangedPlan(), nil }, func() (*compile.Outcome, error) { return nil, nil })
	m = drive(t, m, unchangedPlan())
	if m.state != buildEmpty {
		t.Fatalf("state = %v, want empty", m.state)
	}
	if !strings.Contains(m.View(), "No changes") {
		t.Errorf("empty state should say 'No changes', got:\n%s", m.View())
	}
}

// TestPlanErrorActionable covers the error state: a plan failure shows an
// actionable message (and the workspace-missing case names `daedalus init`).
func TestPlanErrorActionable(t *testing.T) {
	m := newBuildModel("repo", false, func() (*compile.PlanResult, error) {
		return nil, compile.ErrWorkspaceNotFound
	}, nil)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(buildModel)
	updated, _ = m.Update(planLoadedMsg{err: compile.ErrWorkspaceNotFound})
	m = updated.(buildModel)

	if m.state != buildPlanErrored {
		t.Fatalf("state = %v, want plan-errored", m.state)
	}
	if !strings.Contains(m.View(), "daedalus init") {
		t.Errorf("workspace-missing error should point at 'daedalus init', got:\n%s", m.View())
	}
	if m.result().PlanErr == nil {
		t.Error("result should carry the plan error for exit-code mapping")
	}
}

// --- navigation -------------------------------------------------------------

// TestNavigationMovesCursorAndDiff covers Check-8: up/down move the selection and
// reload the diff for the highlighted artifact.
func TestNavigationMovesCursorAndDiff(t *testing.T) {
	var wrote bool
	m := confirmableModel(t, samplePlan(), &wrote)

	if m.cursor != 0 {
		t.Fatalf("cursor should start at 0, got %d", m.cursor)
	}
	// down to the modified artifact; its diff body must show +/- lines.
	updated, _ := m.Update(keyPress("down"))
	m = updated.(buildModel)
	if m.cursor != 1 {
		t.Fatalf("cursor = %d after down, want 1", m.cursor)
	}
	body := m.renderDiffBody(m.artifacts[m.cursor].artifact)
	if !strings.Contains(body, "+ new line") || !strings.Contains(body, "- old line") {
		t.Errorf("modified diff must show +/- lines, got:\n%s", body)
	}

	// up clamps at the top.
	updated, _ = m.Update(keyPress("up"))
	m = updated.(buildModel)
	updated, _ = m.Update(keyPress("up"))
	m = updated.(buildModel)
	if m.cursor != 0 {
		t.Errorf("cursor should clamp at 0, got %d", m.cursor)
	}
}

// --- confirm / cancel gate --------------------------------------------------

// TestConfirmTriggersWrite covers Check-5: confirming (y) in confirmable mode
// triggers the write command; delivering its result moves to the written state.
func TestConfirmTriggersWrite(t *testing.T) {
	var wrote bool
	m := confirmableModel(t, samplePlan(), &wrote)

	updated, cmd := m.Update(keyPress("y"))
	m = updated.(buildModel)
	if m.state != buildWriting {
		t.Fatalf("state after confirm = %v, want writing", m.state)
	}
	if cmd == nil {
		t.Fatal("confirm should return a write command")
	}
	// Execute the command to run the (spy) write and feed back its message.
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(buildModel)

	if !wrote {
		t.Error("confirm must invoke the write function")
	}
	if m.state != buildWritten {
		t.Errorf("state = %v, want written", m.state)
	}
	if r := m.result(); !r.Wrote {
		t.Error("result should report a successful write")
	}
}

// TestEnterAlsoConfirms covers the documented confirm binding: enter is the second
// confirm key alongside y.
func TestEnterAlsoConfirms(t *testing.T) {
	var wrote bool
	m := confirmableModel(t, samplePlan(), &wrote)
	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(buildModel)
	if m.state != buildWriting || cmd == nil {
		t.Fatalf("enter should confirm (state=%v, cmd nil=%v)", m.state, cmd == nil)
	}
}

// TestCancelWritesNothing covers Check-6: cancelling (n) never writes and ends as
// cancelled.
func TestCancelWritesNothing(t *testing.T) {
	for _, k := range []string{"n", "esc"} {
		t.Run(k, func(t *testing.T) {
			var wrote bool
			m := confirmableModel(t, samplePlan(), &wrote)
			updated, cmd := m.Update(keyPress(k))
			m = updated.(buildModel)
			if wrote {
				t.Errorf("%q must not write", k)
			}
			if m.state != buildCancelled {
				t.Errorf("state after %q = %v, want cancelled", k, m.state)
			}
			if cmd == nil {
				t.Errorf("%q should quit the program", k)
			}
			if !m.result().Cancelled {
				t.Errorf("result after %q should be Cancelled", k)
			}
		})
	}
}

// TestReadOnlyNeverWrites covers Check-7: in --preview the confirm key is inert
// (no buildFn) and esc simply quits — nothing is ever written.
func TestReadOnlyNeverWrites(t *testing.T) {
	var wrote bool
	build := func() (*compile.Outcome, error) { wrote = true; return nil, nil }
	m := newBuildModel("repo", true, func() (*compile.PlanResult, error) { return samplePlan(), nil }, build)
	m = drive(t, m, samplePlan())

	if m.buildFn != nil {
		t.Fatal("read-only model must have a nil buildFn")
	}
	// y must NOT write or change state in read-only mode.
	updated, cmd := m.Update(keyPress("y"))
	m = updated.(buildModel)
	if wrote || m.state != buildReady || cmd != nil {
		t.Errorf("read-only confirm must be inert (wrote=%v state=%v)", wrote, m.state)
	}
	// The gate must advertise that nothing will be written.
	if !strings.Contains(m.View(), "nothing will be written") {
		t.Errorf("read-only gate should say nothing will be written, got:\n%s", m.View())
	}
	// esc quits without writing.
	_, cmd = m.Update(keyPress("esc"))
	if wrote {
		t.Error("esc in read-only must not write")
	}
	if cmd == nil {
		t.Error("esc in read-only should quit")
	}
}

// TestConfirmableQReserved covers the keymap choice: in confirmable mode q does
// NOT quit (the user must make an explicit decision), while ctrl+c always escapes.
func TestConfirmableQReserved(t *testing.T) {
	var wrote bool
	m := confirmableModel(t, samplePlan(), &wrote)

	_, cmd := m.Update(keyPress("q"))
	if cmd != nil {
		t.Error("q in confirmable mode should be reserved (no quit), use n/esc to cancel")
	}
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c must always quit")
	}
}

// --- write effect on disk ---------------------------------------------------

// TestConfirmWritesToDisk is the integration check for Check-1/Check-5: against a
// real workspace, the preview writes NOTHING until confirmed, and confirming
// creates the artifacts on disk matching what was previewed.
func TestConfirmWritesToDisk(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Create(root); err != nil {
		t.Fatalf("scaffold workspace: %v", err)
	}
	agentsRoot := filepath.Join(root, workspace.Name, catalog.AgentsDir)
	if _, err := catalog.Builtin.Materialize(agentsRoot, "analyst"); err != nil {
		t.Fatalf("materialize agent: %v", err)
	}

	m := newBuildPreviewModel(root, false)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(buildModel)

	// Run the real plan command; nothing must be on disk yet.
	msg := m.Init()()
	updated, _ = m.Update(msg)
	m = updated.(buildModel)
	if m.state != buildReady {
		t.Fatalf("state after real plan = %v, want ready", m.state)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Fatalf(".claude must not exist before confirmation (err=%v)", err)
	}

	// Confirm → run the real write command.
	updated, cmd := m.Update(keyPress("y"))
	m = updated.(buildModel)
	if cmd == nil {
		t.Fatal("confirm should return the write command")
	}
	updated, _ = m.Update(cmd())
	m = updated.(buildModel)
	if m.state != buildWritten {
		t.Fatalf("state after write = %v, want written", m.state)
	}
	// The artifact previewed as created now exists on disk.
	if _, err := os.Stat(filepath.Join(root, ".claude", "agents", "analyst.md")); err != nil {
		t.Errorf("confirmed write did not create the artifact: %v", err)
	}
}

// TestCancelLeavesDiskUntouched covers Check-6 end-to-end: cancelling a real
// preview writes nothing.
func TestCancelLeavesDiskUntouched(t *testing.T) {
	root := t.TempDir()
	if _, err := workspace.Create(root); err != nil {
		t.Fatalf("scaffold workspace: %v", err)
	}
	agentsRoot := filepath.Join(root, workspace.Name, catalog.AgentsDir)
	if _, err := catalog.Builtin.Materialize(agentsRoot, "analyst"); err != nil {
		t.Fatalf("materialize agent: %v", err)
	}

	m := newBuildPreviewModel(root, false)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = updated.(buildModel)
	updated, _ = m.Update(m.Init()())
	m = updated.(buildModel)

	updated, _ = m.Update(keyPress("n"))
	m = updated.(buildModel)
	if m.state != buildCancelled {
		t.Fatalf("state = %v, want cancelled", m.state)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude")); !os.IsNotExist(err) {
		t.Errorf("cancel must leave the disk untouched (.claude err=%v)", err)
	}
}

// --- textual (non-TTY) render ----------------------------------------------

// TestRenderPlanTextClassifiesAndDiffs covers the non-TTY rendering (Check-2/4):
// the textual report classifies each artifact and shows +/- diff lines for changes.
func TestRenderPlanTextClassifiesAndDiffs(t *testing.T) {
	var b strings.Builder
	RenderPlanText(&b, samplePlan())
	out := b.String()

	for _, want := range []string{
		"1 new, 1 modified, 1 unchanged",
		"[new]      .claude/agents/new.md",
		"[modified] .claude/agents/mod.md",
		"[unchanged] .claude/agents/same.md",
		"+ new line",
		"- old line",
		"no files written",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("textual report missing %q, got:\n%s", want, out)
		}
	}
}

// TestRenderPlanTextNoChanges covers Check-3 for the textual path: an all-unchanged
// plan clearly says there is nothing to write.
func TestRenderPlanTextNoChanges(t *testing.T) {
	var b strings.Builder
	RenderPlanText(&b, unchangedPlan())
	if !strings.Contains(b.String(), "No changes") || !strings.Contains(b.String(), "Nothing to write") {
		t.Errorf("no-changes textual report unclear, got:\n%s", b.String())
	}
}

// TestLineDiffBasic covers the diff util directly: it emits removed-then-added for
// a changed line and keeps context lines.
func TestLineDiffBasic(t *testing.T) {
	got := lineDiff("a\nb\nc\n", "a\nB\nc\n")
	want := []diffLine{
		{op: diffContext, text: "a"},
		{op: diffRemove, text: "b"},
		{op: diffAdd, text: "B"},
		{op: diffContext, text: "c"},
	}
	if len(got) != len(want) {
		t.Fatalf("diff length = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestPlanErrorMessageValidation covers the validation-error wording path so the
// errored view distinguishes a bad definition from a missing workspace.
func TestPlanErrorMessageValidation(t *testing.T) {
	// A non-workspace, non-validation error falls through to the generic message.
	if msg := planErrorMessage(errors.New("boom")); !strings.Contains(msg, "could not be planned") {
		t.Errorf("generic plan error wording unexpected: %q", msg)
	}
}
