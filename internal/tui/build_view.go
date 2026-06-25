package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/Codigo-de-Altura/Daedalus/internal/compile"
)

// build_view.go is the Bubble Tea screen for `daedalus build` (RF-6.4): it shows
// what a build WOULD change before anything is written, lets the user navigate the
// per-artifact diff, and gates the write behind an explicit confirmation. It is a
// self-contained model (buildModel) the command runs with its own program, kept
// separate from the prompt/workflow browser Model so the two flows never tangle.
//
// Presentation/core split: the model never plans or writes itself. compile.Plan
// (read-only) and compile.Build (write) are injected as planFn/buildFn and invoked
// only through tea.Cmds (build_commands.go). State lives in the model; Update is
// the single place it changes; View is a pure projection.
//
// Two modes, one model:
//   - confirmable (daedalus build in a TTY): the gate offers confirm (writes) /
//     cancel (writes nothing). buildFn is wired.
//   - readOnly (daedalus build --preview in a TTY): the same diff, navigation and
//     help, but NO confirm — only view + quit. buildFn is nil, so a write is
//     structurally impossible.

// buildState is the async/decision lifecycle of the preview screen. It is a small
// state machine: planning → (planErrored | empty | ready); from ready a
// confirmation moves to writing → (written | writeErrored); a cancel moves to
// cancelled. Each state has exactly one rendering, so there is never an ambiguous
// or blank screen (REQ-7, loading/empty/error coverage).
type buildState int

const (
	// buildPlanning: compile.Plan is running; nothing has been classified yet.
	buildPlanning buildState = iota
	// buildPlanErrored: the plan could not be computed (missing workspace, invalid
	// definition, unroutable backend); an actionable message is shown.
	buildPlanErrored
	// buildEmpty: the plan succeeded and there is nothing to do — every artifact is
	// unchanged and there are no orphans. Clearly communicated (REQ-7/Check-3).
	buildEmpty
	// buildReady: the plan has changes; the artifact list + diff + gate are shown.
	buildReady
	// buildWriting: a confirmed write (compile.Build) is in flight.
	buildWriting
	// buildWritten: the confirmed write finished; the result summary is shown.
	buildWritten
	// buildWriteErrored: the confirmed write failed (an I/O error after confirm).
	buildWriteErrored
	// buildCancelled: the user declined the gate; nothing was written.
	buildCancelled
)

// buildKeyMap declares the bindings the build preview understands. Like keyMap it
// implements help.KeyMap so the footer is generated from the bindings and can
// never drift. Navigation/scroll keys mirror the rest of the TUI for muscle
// memory; the confirm/cancel pair is the only addition this screen introduces.
type buildKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	PgUp    key.Binding
	PgDn    key.Binding
	Top     key.Binding
	Botom   key.Binding
	Confirm key.Binding
	Cancel  key.Binding
	Help    key.Binding
	Quit    key.Binding
}

// defaultBuildKeyMap derives the build preview's bindings. The build preview is a
// standalone program (`daedalus build`), not part of the six-area shell, but it must
// still honor the project-wide rule "same action ⇒ same key" (07-03/R2). So its
// shared actions (scroll/jump/help/quit) reuse the KEYS from the central registry
// (keybindings.go) — only the context-specific HELP wording differs (e.g. "prev
// artifact" vs "up"). The confirm/cancel gate is the build-specific pair this screen
// adds; it is intentionally NOT in the shell registry because confirm-write is unique
// to this flow. A test (TestBuildPreviewSharesRegistryKeys) locks the shared keys to
// the registry so they can never drift.
func defaultBuildKeyMap() buildKeyMap {
	reg := defaultKeymap()
	// withDesc clones a registry binding's KEYS but gives it build-context help text,
	// so the key is the registry's (consistent) while the wording fits this screen.
	withDesc := func(a keyAction, helpKey, helpDesc string) key.Binding {
		return key.NewBinding(
			key.WithKeys(reg.binding(a).Keys()...),
			key.WithHelp(helpKey, helpDesc),
		)
	}
	return buildKeyMap{
		Up:    withDesc(actionUp, "↑/k", "prev artifact"),
		Down:  withDesc(actionDown, "↓/j", "next artifact"),
		PgUp:  withDesc(actionPageUp, "pgup", "scroll diff up"),
		PgDn:  withDesc(actionPageDown, "pgdn", "scroll diff down"),
		Top:   withDesc(actionTop, "g", "diff top"),
		Botom: withDesc(actionBottom, "G", "diff bottom"),
		// y/enter confirm the write; n/esc cancel it. This build-specific gate is the
		// conventional yes/no pair; esc here "backs out" of the build (cancel),
		// consistent with esc-goes-back elsewhere.
		Confirm: key.NewBinding(
			key.WithKeys("y", "enter"),
			key.WithHelp("y", "confirm & write"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("n/esc", "cancel"),
		),
		Help: reg.binding(actionHelp),
		Quit: reg.binding(actionQuit),
	}
}

// buildModel is the Bubble Tea model for the build preview. It holds the plan
// lifecycle, the flattened artifact list (across backends) the user navigates, the
// scrollable diff viewport, and the mode (whether a write may be confirmed).
type buildModel struct {
	theme theme
	keys  buildKeyMap
	help  help.Model

	width  int
	height int

	// readOnly is true for `--preview`: the confirm gate is hidden and buildFn is
	// nil, so the screen can only show the diff and quit — never write.
	readOnly bool

	// planFn / buildFn are the injected core entry points (see build_commands.go).
	// planFn is always set; buildFn is nil in readOnly mode.
	planFn  planFunc
	buildFn buildFunc

	state buildState

	// root is the resolved target dir, echoed in the header so the user knows what
	// they are about to compile.
	root string

	// plan holds the computed plan once loaded (nil until then). artifacts is the
	// flattened, backend-tagged view of plan.Backends the cursor walks; orphanCount
	// and the per-backend summary are derived from plan.
	plan      *compile.PlanResult
	artifacts []artifactRow
	cursor    int

	// errText holds the rendered plan/write error message (state-dependent).
	errText string
	// planErr holds the raw plan failure (kept alongside errText so the command can
	// map the original error to an exit code, while the view shows errText).
	planErr error
	// outcome holds the confirmed-write result for the "written" summary.
	outcome *compile.Outcome

	viewport      viewport.Model
	viewportReady bool
}

// artifactRow is one navigable row in the flattened artifact list: which backend
// it belongs to plus the planned artifact (status + Current/Target for the diff).
// Flattening keeps the cursor a single index across all backends while the header
// still groups counts per backend.
type artifactRow struct {
	backend  string
	artifact compile.PlannedArtifact
}

// newBuildModel builds a preview model. readOnly selects `--preview` (no confirm,
// no write): in that mode buildFn is left nil so a write is impossible even if the
// keymap were misused. planFn must always be provided.
func newBuildModel(root string, readOnly bool, planFn planFunc, buildFn buildFunc) buildModel {
	if readOnly {
		buildFn = nil
	}
	return buildModel{
		theme:    defaultTheme(),
		keys:     defaultBuildKeyMap(),
		help:     help.New(),
		readOnly: readOnly,
		planFn:   planFn,
		buildFn:  buildFn,
		state:    buildPlanning,
		root:     root,
	}
}

// newBuildPreviewModel is the production constructor: it wires planFn/buildFn to
// compile.Plan / compile.Build over root. In readOnly (--preview) mode buildFn is
// supplied but discarded by newBuildModel, so the write path stays unreachable.
func newBuildPreviewModel(root string, readOnly bool) buildModel {
	return newBuildModel(root, readOnly, planFnFor(root), buildFnFor(root))
}

// Init kicks off the read-only plan asynchronously so the screen shows a loading
// state immediately and never blocks startup on compilation.
func (m buildModel) Init() tea.Cmd {
	return planCmd(m.planFn)
}

// Update is the single place state changes. It routes resize and the two async
// result messages, then dispatches keys per state.
func (m buildModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case planLoadedMsg:
		return m.handlePlanLoaded(msg)

	case buildDoneMsg:
		if msg.err != nil {
			m.state = buildWriteErrored
			m.errText = fmt.Sprintf("The write failed after confirmation; the workspace may be partially written.\n\n%v", msg.err)
			return m, nil
		}
		m.state = buildWritten
		m.outcome = msg.outcome
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward other messages (viewport ticks) to the diff viewport while it is live.
	if m.state == buildReady && m.viewportReady {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handlePlanLoaded turns a computed plan into the right post-plan state: an error
// becomes buildPlanErrored with an actionable message; an all-unchanged,
// orphan-free plan becomes buildEmpty; anything else becomes buildReady with the
// flattened artifact list and the first artifact's diff loaded.
func (m buildModel) handlePlanLoaded(msg planLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.state = buildPlanErrored
		m.errText = planErrorMessage(msg.err)
		m.planErr = msg.err
		return m, nil
	}

	m.plan = msg.result
	m.artifacts = flattenArtifacts(msg.result)

	if !planHasChanges(msg.result) {
		m.state = buildEmpty
		return m, nil
	}

	m.state = buildReady
	m.cursor = 0
	m.loadDiffForCursor()
	return m, nil
}

// handleResize recomputes the diff viewport size and (re)loads the current diff so
// the content rewraps to the new width.
func (m buildModel) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	vpWidth, vpHeight := m.diffViewportSize()
	if !m.viewportReady {
		m.viewport = viewport.New(vpWidth, vpHeight)
		m.viewportReady = true
	} else {
		m.viewport.Width = vpWidth
		m.viewport.Height = vpHeight
	}
	if m.state == buildReady {
		m.loadDiffForCursor()
	}
	return m, nil
}

// handleKey dispatches a key press. Help and ctrl+c are global; everything else is
// state-aware so, for example, the confirm/cancel keys only act on the ready
// screen and navigation only moves the cursor when there is a list.
func (m buildModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Help) {
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	}
	// ctrl+c always quits, from any state, so the user is never trapped.
	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	switch m.state {
	case buildReady:
		return m.handleReadyKey(msg)
	default:
		// In every terminal state (empty, errored, written, cancelled) and while
		// planning/writing, any quit-ish key exits. q and esc both quit here because
		// there is no sub-screen to back out of.
		if key.Matches(msg, m.keys.Quit) || key.Matches(msg, m.keys.Cancel) {
			return m, tea.Quit
		}
		return m, nil
	}
}

// handleReadyKey handles the interactive ready screen: artifact navigation, diff
// scrolling, and the confirm/cancel gate. In readOnly mode confirm is inert and
// cancel/quit simply leave; in confirmable mode confirm triggers the write and
// cancel ends the program without writing (REQ-4/Check-6).
func (m buildModel) handleReadyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
			m.loadDiffForCursor()
		}
		return m, nil
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.artifacts)-1 {
			m.cursor++
			m.loadDiffForCursor()
		}
		return m, nil
	case key.Matches(msg, m.keys.Top):
		m.viewport.GotoTop()
		return m, nil
	case key.Matches(msg, m.keys.Botom):
		m.viewport.GotoBottom()
		return m, nil
	}

	// Confirm: only meaningful (and only wired) in confirmable mode. y/enter trigger
	// the actual write via compile.Build. Read-only mode has no buildFn, so confirm
	// can never write — it is simply ignored there.
	if key.Matches(msg, m.keys.Confirm) {
		if m.readOnly || m.buildFn == nil {
			return m, nil
		}
		m.state = buildWriting
		return m, buildCmd(m.buildFn)
	}

	// Cancel: in confirmable mode this is the explicit "do not write" decision; in
	// read-only mode it is just "leave". Either way nothing is written.
	if key.Matches(msg, m.keys.Cancel) {
		if m.readOnly {
			return m, tea.Quit
		}
		m.state = buildCancelled
		return m, tea.Quit
	}

	// q quits without confirming in read-only mode; in confirmable mode q is
	// reserved so the user does not exit without an explicit decision (use n/esc to
	// cancel, y/enter to confirm) — mirroring the browser's q-reserved-inside-entry
	// rule. ctrl+c (handled above) always works as the escape hatch.
	if key.Matches(msg, m.keys.Quit) && m.readOnly {
		return m, tea.Quit
	}

	// Any remaining keys go to the viewport so line/page scrolling stays responsive.
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// loadDiffForCursor renders the selected artifact's diff into the viewport. It is
// called on open and whenever the cursor moves so the diff panel always matches
// the highlighted artifact. Created/updated show a content diff; unchanged shows a
// clear "no changes" note (REQ-2/REQ-3).
func (m *buildModel) loadDiffForCursor() {
	if !m.viewportReady || len(m.artifacts) == 0 {
		return
	}
	row := m.artifacts[m.cursor]
	m.viewport.SetContent(m.renderDiffBody(row.artifact))
	m.viewport.GotoTop()
}

// --- helpers shared with the textual renderer live in build_text.go ---

// planErrorMessage maps a plan failure to an actionable, user-facing message,
// mirroring writeBuildError's wording so the TUI and the CLI agree on the cause.
func planErrorMessage(err error) string {
	switch {
	case compile.IsDefinitionInvalid(err):
		return fmt.Sprintf("The canonical definition failed validation; fix the reported sources and try again.\n\n%v", err)
	case isWorkspaceMissing(err):
		return fmt.Sprintf("No .daedalus workspace was found here.\n\nRun 'daedalus init' to create one first.\n\n%v", err)
	default:
		return fmt.Sprintf("The build could not be planned.\n\n%v", err)
	}
}

// flattenArtifacts collapses the per-backend plans into a single navigable list,
// tagging each row with its backend, in the plan's deterministic order.
func flattenArtifacts(res *compile.PlanResult) []artifactRow {
	if res == nil {
		return nil
	}
	var rows []artifactRow
	for _, b := range res.Backends {
		for _, a := range b.Artifacts {
			rows = append(rows, artifactRow{backend: b.Backend, artifact: a})
		}
	}
	return rows
}

// planHasChanges reports whether the plan would actually change anything: any
// created/updated artifact, or any detected orphan. An all-unchanged, orphan-free
// plan is the "nothing to do" case (Check-3).
func planHasChanges(res *compile.PlanResult) bool {
	if res == nil {
		return false
	}
	for _, b := range res.Backends {
		if len(b.Orphans) > 0 {
			return true
		}
		for _, a := range b.Artifacts {
			if a.Status != compile.StatusUnchanged {
				return true
			}
		}
	}
	return false
}

// diffViewportSize computes the diff panel's dimensions, reserving rows for the
// header/summary, the artifact list, the scroll hint, the gate and the help so the
// scrollable diff never collides with the chrome. Defaults keep the model usable
// in tests without a terminal.
func (m buildModel) diffViewportSize() (int, int) {
	width := m.width
	height := m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	// Reserve roughly: title + summary box (4), artifact list up to a cap, scroll
	// hint, gate, help. Clamp so a small terminal still yields a usable diff height.
	listRows := len(m.artifacts)
	if listRows > 8 {
		listRows = 8
	}
	chrome := 12 + listRows
	vpHeight := height - chrome
	if vpHeight < 3 {
		vpHeight = 3
	}
	vpWidth := width - 2
	if vpWidth < 20 {
		vpWidth = 20
	}
	return vpWidth, vpHeight
}

// View is a pure projection of the model: one rendering per state, always with the
// contextual help footer, never a blank or ambiguous screen.
func (m buildModel) View() string {
	switch m.state {
	case buildPlanning:
		return m.frame(m.theme.subtle.Render("Planning build… (computing the diff; nothing has been written)"))
	case buildPlanErrored:
		return m.frame(m.theme.errorBox.Render(m.errText))
	case buildWriteErrored:
		return m.frame(m.theme.errorBox.Render(m.errText))
	case buildEmpty:
		return m.frame(m.viewEmpty())
	case buildWriting:
		return m.frame(m.theme.subtle.Render("Writing artifacts…"))
	case buildWritten:
		return m.frame(m.viewWritten())
	case buildCancelled:
		return m.frame(m.theme.subtle.Render("Cancelled. Nothing was written."))
	default:
		return m.frame(m.viewReady())
	}
}

// frame wraps a screen body with the shared title and the contextual help footer
// so every state shares one chrome and the help always reflects the current keys.
func (m buildModel) frame(body string) string {
	var b strings.Builder
	title := "Daedalus · Build preview"
	if m.readOnly {
		title = "Daedalus · Build preview (read-only)"
	}
	b.WriteString(m.theme.title.Render(title))
	b.WriteString(m.theme.subtle.Render("  ·  " + m.root))
	b.WriteString("\n\n")
	b.WriteString(body)
	b.WriteString("\n\n")
	b.WriteString(m.help.View(m))
	return b.String()
}

// viewEmpty renders the no-changes state clearly: the user must be able to tell at
// a glance that there is nothing to write (REQ-7/Check-3).
func (m buildModel) viewEmpty() string {
	return m.theme.emptyState.Render(
		"No changes — every artifact is already up to date.\n\n" +
			"There is nothing to write. Press q to exit.")
}

// viewWritten renders the post-write summary so the confirmed result matches what
// the preview promised (Check-5).
func (m buildModel) viewWritten() string {
	var b strings.Builder
	b.WriteString(m.theme.statusCreated.Render("Done — artifacts written."))
	b.WriteString("\n\n")
	if m.outcome != nil {
		b.WriteString(summaryText(m.outcome))
	}
	b.WriteString("\n")
	b.WriteString(m.theme.subtle.Render("Press q to exit."))
	return b.String()
}

// viewReady renders the main preview: the per-backend summary, the navigable
// artifact list, the selected artifact's diff, the scroll hint and the gate.
func (m buildModel) viewReady() string {
	var b strings.Builder
	b.WriteString(m.summaryBoxView())
	b.WriteString("\n\n")
	b.WriteString(m.renderArtifactList())
	b.WriteString("\n\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	b.WriteString(m.theme.subtle.Render(m.diffScrollHint()))
	b.WriteString("\n\n")
	b.WriteString(m.renderGate())
	return b.String()
}

// summaryBoxView wraps the summary content in the bordered box, normalizing every
// content line to the same VISIBLE width (measured with lipgloss.Width, which
// ignores ANSI color sequences) before handing it to the bordered style. Lines
// carrying colored counts have different visible widths; padding them to a common
// width up front makes the right border close evenly on every line, instead of a
// character short on the lines whose color sequences threw off the measurement
// (minor fix #1). We pad the lines ourselves rather than calling Style.Width so
// the style never word-wraps the single-line summary.
func (m buildModel) summaryBoxView() string {
	lines := strings.Split(m.renderSummary(), "\n")
	width := 0
	for _, line := range lines {
		if w := lipgloss.Width(line); w > width {
			width = w
		}
	}
	for i, line := range lines {
		if pad := width - lipgloss.Width(line); pad > 0 {
			lines[i] = line + strings.Repeat(" ", pad)
		}
	}
	return m.theme.summaryBox.Render(strings.Join(lines, "\n"))
}

// renderSummary renders the per-backend counts (created/updated/unchanged/orphans)
// plus a total, so the scope of the change is clear before reading any diff
// (REQ-6). It reuses the same counting the textual renderer uses.
func (m buildModel) renderSummary() string {
	var b strings.Builder
	total := 0
	for i, bp := range m.plan.Backends {
		c, u, n := countStatuses(bp)
		total += len(bp.Artifacts)
		fmt.Fprintf(&b, "%s: %s, %s, %s",
			m.theme.title.Render(bp.Backend),
			m.theme.statusCreated.Render(fmt.Sprintf("%d new", c)),
			m.theme.statusUpdated.Render(fmt.Sprintf("%d modified", u)),
			m.theme.statusUnchanged.Render(fmt.Sprintf("%d unchanged", n)))
		if len(bp.Orphans) > 0 {
			fmt.Fprintf(&b, ", %s", m.theme.statusOrphan.Render(fmt.Sprintf("%d orphan%s", len(bp.Orphans), plural2(len(bp.Orphans)))))
		}
		if i < len(m.plan.Backends)-1 {
			b.WriteString("\n")
		}
	}
	if len(m.plan.Backends) > 1 {
		fmt.Fprintf(&b, "\n%s", m.theme.subtle.Render(fmt.Sprintf("%d artifact%s total", total, plural2(total))))
	}
	return b.String()
}

// badgeWidth is the fixed column width every status badge is padded to so the
// artifact paths align in one grid. "[unchanged]" is the widest label (11), so
// every badge — including "[orphan]" — pads to it (minor fix #2).
const badgeWidth = 11

// renderArtifactList renders the navigable artifact rows with a status badge,
// marking the cursor row, then — clearly separated — the read-only orphan section.
// Each navigable row reads "<badge> <relpath>" so the classification is
// unmistakable (REQ-2/Check-2/Check-4).
func (m buildModel) renderArtifactList() string {
	var b strings.Builder
	for i, row := range m.artifacts {
		badge := m.statusBadge(row.artifact.Status)
		text := fmt.Sprintf("%s  %s", badge, row.artifact.RelPath)
		if i == m.cursor {
			b.WriteString(m.theme.listItemSelected.Render() + text)
		} else {
			b.WriteString(m.theme.listItem.Render(text))
		}
		b.WriteString("\n")
	}
	b.WriteString(m.renderOrphanSection())
	return strings.TrimRight(b.String(), "\n")
}

// renderOrphanSection renders detected orphans as a clearly separated, read-only
// info block under its own heading, so the user never mistakes it for part of the
// navigable list (no cursor dead-end). Orphans are detected, never deleted, and
// left for the user to handle manually (Check-8). The whole block is styled subtle
// and the heading spells out "not selectable" so its non-interactive nature reads
// at a glance (minor fix #3). Returns "" when there are no orphans.
func (m buildModel) renderOrphanSection() string {
	var lines []string
	for _, bp := range m.plan.Backends {
		for _, o := range bp.Orphans {
			badge := padBadge("[orphan]")
			lines = append(lines, fmt.Sprintf("%s  %s", badge, o))
		}
	}
	if len(lines) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(m.theme.statusOrphan.Render("Orphans — left untouched · not selectable"))
	b.WriteString("\n")
	for _, line := range lines {
		// The whole orphan block is rendered subtle (non-emphasized) so it never
		// competes with the navigable rows for the user's attention.
		b.WriteString(m.theme.statusOrphan.Render(line))
		b.WriteString("\n")
	}
	return b.String()
}

// statusBadge maps a status to its themed, fixed-width badge so the list columns
// line up and each class reads at a glance. The label is padded to badgeWidth so
// every badge occupies one grid column regardless of label length.
func (m buildModel) statusBadge(s compile.ArtifactStatus) string {
	switch s {
	case compile.StatusCreated:
		return m.theme.statusCreated.Render(padBadge("[new]"))
	case compile.StatusUpdated:
		return m.theme.statusUpdated.Render(padBadge("[modified]"))
	default:
		return m.theme.statusUnchanged.Render(padBadge("[unchanged]"))
	}
}

// padBadge right-pads a badge label to the shared badgeWidth column so all badges
// — status and orphan alike — align on one grid (minor fix #2). Padding the plain
// label (before any color is applied) keeps the visible width exact.
func padBadge(label string) string {
	if len(label) >= badgeWidth {
		return label
	}
	return label + strings.Repeat(" ", badgeWidth-len(label))
}

// renderDiffBody renders the body shown in the diff viewport for one artifact.
// Created → the full new content as added lines; updated → a line-level diff;
// unchanged → a clear note (there is nothing to diff). The wrapping happens in the
// viewport; here we only color and prefix the lines (REQ-3).
func (m buildModel) renderDiffBody(a compile.PlannedArtifact) string {
	switch a.Status {
	case compile.StatusUnchanged:
		return m.theme.subtle.Render("No changes — this artifact is already up to date.")
	case compile.StatusCreated:
		var b strings.Builder
		b.WriteString(m.theme.subtle.Render("New file — full content shown below:"))
		b.WriteString("\n\n")
		for _, line := range splitLines(a.Target) {
			b.WriteString(m.theme.diffAdd.Render("+ " + line))
			b.WriteString("\n")
		}
		return strings.TrimRight(b.String(), "\n")
	default:
		var b strings.Builder
		for _, dl := range lineDiff(a.Current, a.Target) {
			switch dl.op {
			case diffAdd:
				b.WriteString(m.theme.diffAdd.Render("+ " + dl.text))
			case diffRemove:
				b.WriteString(m.theme.diffRemove.Render("- " + dl.text))
			default:
				b.WriteString(m.theme.diffContext.Render("  " + dl.text))
			}
			b.WriteString("\n")
		}
		return strings.TrimRight(b.String(), "\n")
	}
}

// diffScrollHint shows the diff scroll position so a long diff's extent is
// discoverable, mirroring the browser's scrollHint.
func (m buildModel) diffScrollHint() string {
	if !m.viewportReady || m.viewport.TotalLineCount() <= m.viewport.Height {
		return "diff fully shown"
	}
	return fmt.Sprintf("%3.0f%% of diff — scroll with pgup/pgdn · g/G", m.viewport.ScrollPercent()*100)
}

// renderGate renders the confirmation call to action: in confirmable mode it
// spells out confirm-writes / cancel-discards; in read-only mode it states plainly
// that nothing will be written (REQ-4/REQ-5/Check-7).
func (m buildModel) renderGate() string {
	if m.readOnly {
		return m.theme.subtle.Render("Read-only preview — nothing will be written. Press q to exit.")
	}
	return m.theme.confirmPrompt.Render("Write these changes? ") +
		m.theme.subtle.Render("y/enter to confirm · n/esc to cancel (nothing is written)")
}

// ShortHelp implements help.KeyMap for the one-line footer, reflecting the keys
// the current state actually accepts so the help never advertises a dead key.
func (m buildModel) ShortHelp() []key.Binding {
	switch m.state {
	case buildReady:
		if m.readOnly {
			return []key.Binding{m.keys.Up, m.keys.Down, m.keys.PgDn, m.keys.Help, m.keys.Quit}
		}
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Confirm, m.keys.Cancel, m.keys.Help, m.keys.Quit}
	default:
		return []key.Binding{m.keys.Help, m.keys.Quit}
	}
}

// FullHelp implements help.KeyMap for the expanded footer.
func (m buildModel) FullHelp() [][]key.Binding {
	switch m.state {
	case buildReady:
		nav := []key.Binding{m.keys.Up, m.keys.Down, m.keys.PgUp, m.keys.PgDn, m.keys.Top, m.keys.Botom}
		if m.readOnly {
			return [][]key.Binding{nav, {m.keys.Help, m.keys.Quit}}
		}
		return [][]key.Binding{nav, {m.keys.Confirm, m.keys.Cancel}, {m.keys.Help, m.keys.Quit}}
	default:
		return [][]key.Binding{{m.keys.Help, m.keys.Quit}}
	}
}

// Compile-time guarantees that buildModel satisfies the Bubble Tea and help
// contracts, exactly like the browser Model.
var (
	_ tea.Model   = buildModel{}
	_ help.KeyMap = buildModel{}
)

// BuildResult is the outcome of running the interactive build preview, in the
// exported vocabulary the command needs to choose an exit code WITHOUT importing
// the model's private state. The command maps Confirmed/Wrote/Err to the existing
// build exit codes and renders Outcome/ErrText as needed.
type BuildResult struct {
	// Wrote is true iff the confirmed write actually ran and succeeded (the only
	// case in which the filesystem changed). It is always false in read-only mode.
	Wrote bool
	// Cancelled is true if the user declined the gate (confirmable mode) or simply
	// quit a read-only preview — either way nothing was written.
	Cancelled bool
	// NoChanges is true if the plan had nothing to do (all unchanged, no orphans).
	NoChanges bool
	// Outcome is the confirmed-write result (nil unless Wrote).
	Outcome *compile.Outcome
	// PlanErr is the plan failure, if the preview could not be computed; the command
	// maps it to the same exit codes as the non-interactive path.
	PlanErr error
	// WriteErr is a write failure that happened after confirmation.
	WriteErr error
}

// RunBuildPreview launches the interactive build preview over an alt-screen and
// blocks until the user exits, returning a BuildResult the command turns into an
// exit code. readOnly selects `--preview` (no confirm, no write). It is the single
// public entry point the command calls; everything below it (the model, its keys,
// the diff) stays internal to the package, preserving the presentation/core split
// the command already relies on for the browser.
func RunBuildPreview(root string, readOnly bool) (BuildResult, error) {
	m := newBuildPreviewModel(root, readOnly)
	final, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	if err != nil {
		return BuildResult{}, err
	}
	bm, ok := final.(buildModel)
	if !ok {
		return BuildResult{}, fmt.Errorf("unexpected final model type %T", final)
	}
	return bm.result(), nil
}

// result projects the model's terminal state into the exported BuildResult so the
// command never reaches into private fields.
func (m buildModel) result() BuildResult {
	switch m.state {
	case buildWritten:
		return BuildResult{Wrote: true, Outcome: m.outcome}
	case buildWriteErrored:
		return BuildResult{WriteErr: errors.New(m.errText)}
	case buildPlanErrored:
		return BuildResult{PlanErr: m.planErr}
	case buildEmpty:
		return BuildResult{NoChanges: true}
	case buildCancelled:
		return BuildResult{Cancelled: true}
	default:
		// Quitting from the ready/read-only screen without confirming counts as a
		// cancel: nothing was written.
		return BuildResult{Cancelled: true}
	}
}
