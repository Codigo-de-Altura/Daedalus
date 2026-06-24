// Package tui contains the Bubble Tea presentation layer for Daedalus.
//
// This package owns everything the user sees and touches in the terminal. It
// follows the Elm architecture (Model/Update/View) strictly: the Model holds all
// state, Update is the single place state changes (reacting to typed Msgs), and
// every side effect — listing or composing prompts — happens through a tea.Cmd
// (see commands.go) so the UI thread never blocks. No domain, composition or
// persistence logic lives here; the TUI consumes the internal/prompts core.
//
// The first product screen is the prompt browser (epic-03, ticket-03-03): a list
// of the workspace's prompts that opens a read-only, Glamour-rendered preview of
// any prompt's fully composed text. Styling is centralized in theme.go (R6).
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"

	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

// screen identifies which top-level view is active. The model is a small state
// machine: it starts on the list and toggles to the preview (for a prompt) or the
// DAG view (for a workflow) when an entry is opened, with esc returning to the
// list. Keeping the active screen explicit keeps Update's branches readable and
// the keymap unambiguous.
type screen int

const (
	// screenList shows the active section's entries (prompts or workflows) for
	// selection.
	screenList screen = iota
	// screenPreview shows the composed, Glamour-rendered preview of one prompt.
	screenPreview
	// screenWorkflowDAG shows the read-only DAG of one workflow (ticket 04-02).
	screenWorkflowDAG
)

// section identifies which area the list screen is browsing. The same list
// mechanics (cursor, up/down, enter) serve both areas; tab toggles between them
// so the user reaches workflows without a separate navigation tree (R6). The
// active section also decides what enter opens: a prompt preview or a workflow
// DAG.
type section int

const (
	// sectionPrompts browses `.daedalus/prompts/` (the original screen).
	sectionPrompts section = iota
	// sectionWorkflows browses `.daedalus/workflows/`.
	sectionWorkflows
)

// dagState captures the async lifecycle of loading one workflow into the DAG
// view, mirroring previewState: loading while the core parses the file, ready
// once the workflow is laid out into the viewport, or errored when the load
// failed (R7/CA7).
type dagState int

const (
	dagLoading dagState = iota
	dagReady
	dagErrored
)

// previewState captures the async lifecycle of a single preview: it is loading
// while the core composes the prompt, ready once the composed text is rendered
// into the viewport, or errored when composition failed (R7). The three states
// give the preview a loading, a content and an error rendering with no overlap.
type previewState int

const (
	previewLoading previewState = iota
	previewReady
	previewErrored
)

// Model is the Bubble Tea model for the Daedalus prompt browser. It holds the
// active screen plus the state of both the list and the preview. All mutation
// happens in Update; View is a pure projection of this state.
type Model struct {
	// workdir is the directory whose `.daedalus/prompts/` is browsed. It is the
	// process working directory by default (see New), so launching the TUI inside
	// a project shows that project's prompts.
	workdir string

	theme theme
	keys  keyMap
	help  help.Model

	// width/height track the terminal size so the list and the preview viewport
	// resize with the window (R4 scrolling needs an accurate viewport height).
	width  int
	height int

	screen screen

	// section is the area the list browses (prompts or workflows). It selects both
	// which entry slice the list shows and what enter opens.
	section section

	// --- prompts list state ---
	loadingList bool
	listErr     error
	entries     []prompts.Entry
	cursor      int

	// --- workflows list state ---
	// The workflows section reuses the same list mechanics with its own loading
	// flag, error, entries and cursor so toggling sections preserves each section's
	// selection independently.
	loadingWorkflows bool
	workflowsErr     error
	workflowEntries  []workflows.Entry
	workflowCursor   int

	// --- preview state ---
	previewState   previewState
	previewID      string // id of the prompt currently shown/loading in the preview
	previewTitle   string
	previewErrText string
	viewport       viewport.Model
	// viewportReady guards against using the viewport before the first WindowSizeMsg
	// has given it real dimensions.
	viewportReady bool

	// --- workflow DAG state ---
	// The DAG view reuses the same scrollable viewport as the preview (only one is
	// ever active at a time), but keeps its own loading/ready/error lifecycle and
	// the identity + title of the workflow being shown.
	dagState   dagState
	dagName    string // name of the workflow currently shown/loading in the DAG view
	dagErrText string
}

// keyMap declares every binding the TUI understands. It implements help.KeyMap
// so the contextual help footer is generated from the bindings themselves (R5):
// the help text can never drift from the actual keys. Bindings are made visible
// or hidden per screen in shortHelpFor/fullHelpFor so the help always reflects
// what the current screen accepts.
type keyMap struct {
	Up      key.Binding
	Down    key.Binding
	Open    key.Binding
	Back    key.Binding
	Section key.Binding
	Help    key.Binding
	Quit    key.Binding
	PgUp    key.Binding
	PgDn    key.Binding
	Top     key.Binding
	Botom   key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Open: key.NewBinding(
			key.WithKeys("enter", "l"),
			key.WithHelp("enter", "open"),
		),
		// esc returns from the preview or the DAG view to the list. We intentionally
		// do NOT bind "q" to "back": "q" stays the global quit so the user's muscle
		// memory for leaving the app is consistent everywhere, and esc is the
		// unambiguous "go back one level" key inside an opened entry.
		Back: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back to list"),
		),
		// tab toggles the list between the prompts and the workflows section, reusing
		// the same list/cursor mechanics so reaching workflows costs no new
		// navigation model (R6).
		Section: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch section"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		PgUp: key.NewBinding(
			key.WithKeys("pgup", "b"),
			key.WithHelp("pgup", "page up"),
		),
		PgDn: key.NewBinding(
			key.WithKeys("pgdown", "f", " "),
			key.WithHelp("pgdn", "page down"),
		),
		Top: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g", "top"),
		),
		Botom: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G", "bottom"),
		),
	}
}

// New returns an initialized prompt-browser Model rooted at workdir. workdir is
// the directory whose `.daedalus/prompts/` the browser lists; passing "" means
// the current directory, so callers that do not care about the location can omit
// it. The actual prompt loading happens in Init (async) so construction never
// touches the filesystem.
func New(workdir string) Model {
	if workdir == "" {
		workdir = "."
	}
	return Model{
		workdir:          workdir,
		theme:            defaultTheme(),
		keys:             defaultKeyMap(),
		help:             help.New(),
		screen:           screenList,
		section:          sectionPrompts,
		loadingList:      true,
		loadingWorkflows: true,
	}
}

// Init implements tea.Model. It kicks off the asynchronous prompt and workflow
// listings so both sections are populated without blocking startup; the two loads
// run concurrently via tea.Batch. The alt-screen is requested by the program
// runner (tea.WithAltScreen) so this command set stays focused on data.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		loadPromptsCmd(m.workdir),
		loadWorkflowsCmd(m.workdir),
	)
}

// Update implements tea.Model. It is the single place state changes: it routes
// window-size and async result messages, then dispatches key messages to the
// active screen's handler. Side effects are returned as tea.Cmds, never run
// inline.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case promptsLoadedMsg:
		m.loadingList = false
		m.listErr = msg.err
		m.entries = msg.entries
		if m.cursor >= len(m.entries) {
			m.cursor = 0
		}
		return m, nil

	case promptResolvedMsg:
		// Ignore a result for a prompt the user has already navigated away from, so
		// a slow compose can never overwrite a newer preview.
		if msg.id != m.previewID {
			return m, nil
		}
		if msg.err != nil {
			m.previewState = previewErrored
			m.previewErrText = composeErrorMessage(msg.id, msg.err)
			return m, nil
		}
		m.previewState = previewReady
		m.viewport.SetContent(m.renderMarkdown(msg.content))
		m.viewport.GotoTop()
		return m, nil

	case workflowsLoadedMsg:
		m.loadingWorkflows = false
		m.workflowsErr = msg.err
		m.workflowEntries = msg.entries
		if m.workflowCursor >= len(m.workflowEntries) {
			m.workflowCursor = 0
		}
		return m, nil

	case workflowLoadedMsg:
		// Ignore a result for a workflow the user has already navigated away from,
		// so a slow load can never overwrite a newer DAG view.
		if msg.name != m.dagName {
			return m, nil
		}
		if msg.err != nil {
			m.dagState = dagErrored
			m.dagErrText = workflowLoadErrorMessage(msg.name, msg.err)
			return m, nil
		}
		m.dagState = dagReady
		m.viewport.SetContent(m.dagViewportContent(msg.workflow))
		m.viewport.GotoTop()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward any other message to the viewport while previewing or showing a DAG
	// (e.g. internal viewport ticks), so scrolling stays responsive. Both screens
	// share the one viewport.
	if (m.screen == screenPreview || m.screen == screenWorkflowDAG) && m.viewportReady {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

// handleResize recomputes the layout when the terminal size changes. It sizes
// the preview viewport to the area left below the title and above the help so
// long content scrolls within a stable frame (R4).
func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	vpWidth, vpHeight := m.previewViewportSize()
	if !m.viewportReady {
		m.viewport = viewport.New(vpWidth, vpHeight)
		m.viewportReady = true
	} else {
		m.viewport.Width = vpWidth
		m.viewport.Height = vpHeight
	}
	return m, nil
}

// handleKey dispatches a key press. Quit and help-toggle are global; everything
// else is delegated to the active screen so each screen owns its own navigation
// without the other screen's keys leaking in.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case key.Matches(msg, m.keys.Quit):
		// In the list, q/ctrl+c quit. Inside an opened entry (preview or DAG),
		// ctrl+c still quits but q is reserved so the user does not accidentally exit
		// the app while reading; esc is the documented way back. This keeps quit
		// predictable without a dead-end.
		if (m.screen == screenPreview || m.screen == screenWorkflowDAG) && msg.String() == "q" {
			return m, nil
		}
		return m, tea.Quit
	}

	switch m.screen {
	case screenList:
		return m.handleListKey(msg)
	case screenPreview:
		return m.handlePreviewKey(msg)
	case screenWorkflowDAG:
		return m.handleDAGKey(msg)
	}
	return m, nil
}

// handleListKey handles navigation and selection on the list screen. It is
// section-aware: tab toggles between the prompts and workflows areas, and up/down/
// open act on whichever section is active so both areas share one set of list
// mechanics (R6).
func (m Model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Section) {
		return m.toggleSection(), nil
	}

	if m.section == sectionWorkflows {
		switch {
		case key.Matches(msg, m.keys.Up):
			if m.workflowCursor > 0 {
				m.workflowCursor--
			}
		case key.Matches(msg, m.keys.Down):
			if m.workflowCursor < len(m.workflowEntries)-1 {
				m.workflowCursor++
			}
		case key.Matches(msg, m.keys.Open):
			if len(m.workflowEntries) == 0 {
				return m, nil
			}
			return m.openDAG()
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.cursor < len(m.entries)-1 {
			m.cursor++
		}
	case key.Matches(msg, m.keys.Open):
		if len(m.entries) == 0 {
			return m, nil
		}
		return m.openPreview()
	}
	return m, nil
}

// toggleSection flips the list between the prompts and workflows areas. Each
// section keeps its own cursor, so toggling back returns the user to where they
// were. It is a pure state flip — both sections were already loaded at startup —
// so no command is needed.
func (m Model) toggleSection() Model {
	if m.section == sectionPrompts {
		m.section = sectionWorkflows
	} else {
		m.section = sectionPrompts
	}
	return m
}

// openDAG switches to the DAG screen for the currently selected workflow and
// starts loading it asynchronously. The viewport is reset to a loading state
// immediately so the user gets instant feedback while the core parses the file.
func (m Model) openDAG() (tea.Model, tea.Cmd) {
	selected := m.workflowEntries[m.workflowCursor]
	m.screen = screenWorkflowDAG
	m.dagState = dagLoading
	m.dagName = selected.Name
	m.dagErrText = ""
	if m.viewportReady {
		m.viewport.SetContent("")
		m.viewport.GotoTop()
	}
	return m, loadWorkflowCmd(m.workdir, selected.Name)
}

// openPreview switches to the preview screen for the currently selected prompt
// and starts composing it asynchronously. The viewport is reset to a loading
// state immediately so the user gets instant feedback while the core works.
func (m Model) openPreview() (tea.Model, tea.Cmd) {
	selected := m.entries[m.cursor]
	m.screen = screenPreview
	m.previewState = previewLoading
	m.previewID = selected.ID
	m.previewTitle = selected.Title
	m.previewErrText = ""
	if m.viewportReady {
		m.viewport.SetContent("")
		m.viewport.GotoTop()
	}
	return m, resolvePromptCmd(m.workdir, selected.ID)
}

// handlePreviewKey handles scrolling and returning to the list. Scrolling is
// delegated to the viewport so paging/line movement behaves like every other
// Bubbles viewport. esc returns to the list (R5); the preview never edits the
// prompt (R8) — there is simply no binding that mutates anything.
func (m Model) handlePreviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.screen = screenList
		return m, nil
	}

	// Explicit top/bottom jumps; the viewport handles the rest (line + page) via
	// its own Update below.
	switch {
	case key.Matches(msg, m.keys.Top):
		m.viewport.GotoTop()
		return m, nil
	case key.Matches(msg, m.keys.Botom):
		m.viewport.GotoBottom()
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// handleDAGKey handles scrolling and returning to the workflows list. The DAG view
// is strictly read-only (R4/CA5): the only bindings it honors are navigation
// (esc/back, top/bottom jumps) and viewport scrolling — there is deliberately no
// binding that edits a phase or runs the workflow, so the view can never mutate or
// execute anything.
func (m Model) handleDAGKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		m.screen = screenList
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Top):
		m.viewport.GotoTop()
		return m, nil
	case key.Matches(msg, m.keys.Botom):
		m.viewport.GotoBottom()
		return m, nil
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// View implements tea.Model. It is a pure projection of the model: it renders
// the active screen and the contextual help footer, never mutating state.
func (m Model) View() string {
	switch m.screen {
	case screenPreview:
		return m.viewPreview()
	case screenWorkflowDAG:
		return m.viewDAG()
	default:
		return m.viewList()
	}
}

// viewList renders the active section's browser: a section-aware title, the list
// of entries (or a loading/empty/error state), and the contextual help footer. The
// title reflects the active section so the user always knows which area they are
// in (R6).
func (m Model) viewList() string {
	if m.section == sectionWorkflows {
		return m.viewWorkflowsList()
	}

	var b strings.Builder
	b.WriteString(m.theme.title.Render("Daedalus · Prompts"))
	b.WriteString("\n\n")

	switch {
	case m.loadingList:
		b.WriteString(m.theme.subtle.Render("Loading prompts…"))
	case m.listErr != nil:
		b.WriteString(m.theme.errorBox.Render(
			fmt.Sprintf("Could not read prompts.\n\n%v", m.listErr)))
	case len(m.entries) == 0:
		b.WriteString(m.theme.emptyState.Render(
			"No prompts found.\n\n" +
				"Create one with `daedalus prompt create`, or run `daedalus init`\n" +
				"if this directory is not a Daedalus workspace yet."))
	default:
		b.WriteString(m.renderEntries())
	}

	b.WriteString("\n\n")
	b.WriteString(m.help.View(m))
	return b.String()
}

// viewWorkflowsList renders the workflows browser: title, the list of workflows
// (or a loading/empty/error state), and the help footer. It mirrors the prompts
// list exactly so the two sections feel identical to navigate (R6), with a
// workflow-specific empty state pointing at the workflow CLI (R7/CA7).
func (m Model) viewWorkflowsList() string {
	var b strings.Builder
	b.WriteString(m.theme.title.Render("Daedalus · Workflows"))
	b.WriteString("\n\n")

	switch {
	case m.loadingWorkflows:
		b.WriteString(m.theme.subtle.Render("Loading workflows…"))
	case m.workflowsErr != nil:
		b.WriteString(m.theme.errorBox.Render(
			fmt.Sprintf("Could not read workflows.\n\n%v", m.workflowsErr)))
	case len(m.workflowEntries) == 0:
		b.WriteString(m.theme.emptyState.Render(
			"No workflows found.\n\n" +
				"Create one with `daedalus workflow create`, or run `daedalus init`\n" +
				"if this directory is not a Daedalus workspace yet."))
	default:
		b.WriteString(m.renderWorkflowEntries())
	}

	b.WriteString("\n\n")
	b.WriteString(m.help.View(m))
	return b.String()
}

// renderEntries renders each prompt row as "<id>  [kind]  title", marking the
// selected row with the theme's cursor. Kept simple and theme-driven so the
// list's look is consistent with the rest of the TUI (R6).
func (m Model) renderEntries() string {
	var b strings.Builder
	for i, e := range m.entries {
		badge := m.theme.kindBadge.Render("[" + string(e.Kind) + "]")
		row := fmt.Sprintf("%s  %s  %s", e.ID, badge, m.theme.subtle.Render(e.Title))
		if i == m.cursor {
			b.WriteString(m.theme.listItemSelected.Render() + row)
		} else {
			b.WriteString(m.theme.listItem.Render(row))
		}
		if i < len(m.entries)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// renderWorkflowEntries renders each workflow row as "<name>  [N phases]", marking
// the selected row with the theme's cursor. It mirrors renderEntries so the two
// section lists are visually consistent (R6); the badge carries the cheap phase
// count the listing already provides.
func (m Model) renderWorkflowEntries() string {
	var b strings.Builder
	for i, e := range m.workflowEntries {
		badge := m.theme.kindBadge.Render(fmt.Sprintf("[%d phases]", e.Phases))
		row := fmt.Sprintf("%s  %s", e.Name, badge)
		if i == m.workflowCursor {
			b.WriteString(m.theme.listItemSelected.Render() + row)
		} else {
			b.WriteString(m.theme.listItem.Render(row))
		}
		if i < len(m.workflowEntries)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// viewDAG renders the read-only workflow DAG screen: a header naming the workflow,
// the scrollable graph (or a loading/error state), and the help footer. Like the
// preview it wraps the scrollable body in the theme's bordered frame so it reads as
// a distinct, consistent panel (R5/CA6).
func (m Model) viewDAG() string {
	var b strings.Builder
	header := fmt.Sprintf("%s  %s",
		m.theme.title.Render("Workflow"),
		m.theme.subtle.Render(m.dagName))
	b.WriteString(header)
	b.WriteString("\n\n")

	switch m.dagState {
	case dagLoading:
		b.WriteString(m.theme.subtle.Render("Loading workflow…"))
	case dagErrored:
		b.WriteString(m.theme.errorBox.Render(m.dagErrText))
	default:
		body := m.viewport.View()
		b.WriteString(m.theme.previewFrame.Render(body))
		b.WriteString("\n")
		b.WriteString(m.theme.subtle.Render(m.scrollHint()))
	}

	b.WriteString("\n\n")
	b.WriteString(m.help.View(m))
	return b.String()
}

// viewPreview renders the read-only preview screen: a header naming the prompt,
// the scrollable composed content (or a loading/error state), and the help
// footer. The viewport is wrapped in the theme's bordered frame so it reads as a
// distinct, consistent panel (R6).
func (m Model) viewPreview() string {
	var b strings.Builder
	header := fmt.Sprintf("%s  %s",
		m.theme.title.Render("Preview"),
		m.theme.subtle.Render(m.previewID))
	if m.previewTitle != "" {
		header += m.theme.subtle.Render("  ·  " + m.previewTitle)
	}
	b.WriteString(header)
	b.WriteString("\n\n")

	switch m.previewState {
	case previewLoading:
		b.WriteString(m.theme.subtle.Render("Composing prompt…"))
	case previewErrored:
		b.WriteString(m.theme.errorBox.Render(m.previewErrText))
	default:
		body := m.viewport.View()
		b.WriteString(m.theme.previewFrame.Render(body))
		b.WriteString("\n")
		b.WriteString(m.theme.subtle.Render(m.scrollHint()))
	}

	b.WriteString("\n\n")
	b.WriteString(m.help.View(m))
	return b.String()
}

// scrollHint shows how far through the content the user is, so a long prompt's
// scroll position is always discoverable (R4/R5).
func (m Model) scrollHint() string {
	if !m.viewportReady || m.viewport.TotalLineCount() <= m.viewport.Height {
		return "all content shown"
	}
	return fmt.Sprintf("%3.0f%% — scroll with ↑/↓ · pgup/pgdn · g/G", m.viewport.ScrollPercent()*100)
}

// renderMarkdown renders composed prompt text as terminal Markdown via Glamour
// (R3), word-wrapped to the viewport width so headings, lists, emphasis and code
// blocks read well and never overflow. On any render failure it falls back to
// the raw composed text so the preview always shows the content rather than
// breaking.
func (m Model) renderMarkdown(text string) string {
	width, _ := m.previewViewportSize()
	// Subtract the frame's horizontal padding/border so wrapped lines fit inside
	// the bordered panel without a horizontal scroll.
	wrap := width - 2
	if wrap < 20 {
		wrap = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(wrap),
	)
	if err != nil {
		return text
	}
	out, err := r.Render(text)
	if err != nil {
		return text
	}
	return out
}

// previewViewportSize computes the width/height available to the preview
// viewport, reserving rows for the header, the help footer and the frame so the
// scrollable area never collides with the chrome. Before the first size message
// it returns a sensible default so the model is usable in tests without a
// terminal.
func (m Model) previewViewportSize() (int, int) {
	width := m.width
	height := m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	// Reserve: 2 header lines + blank, 1 scroll hint, blank + 1 help line, and 2
	// for the frame border. Clamp so a tiny terminal still yields a usable height.
	const chrome = 9
	vpHeight := height - chrome
	if vpHeight < 3 {
		vpHeight = 3
	}
	// The frame border consumes 2 columns; keep the viewport inside it.
	vpWidth := width - 2
	if vpWidth < 20 {
		vpWidth = 20
	}
	return vpWidth, vpHeight
}

// ShortHelp implements help.KeyMap. It returns the bindings relevant to the
// active screen so the one-line help footer always matches what the current
// screen accepts (R5).
func (m Model) ShortHelp() []key.Binding {
	if m.screen == screenPreview || m.screen == screenWorkflowDAG {
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Back, m.keys.Help, m.keys.Quit}
	}
	return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Open, m.keys.Section, m.keys.Help, m.keys.Quit}
}

// FullHelp implements help.KeyMap, grouping the bindings shown when help is
// expanded (?). The preview adds the paging/jump keys it supports.
func (m Model) FullHelp() [][]key.Binding {
	if m.screen == screenPreview || m.screen == screenWorkflowDAG {
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down, m.keys.PgUp, m.keys.PgDn},
			{m.keys.Top, m.keys.Botom},
			{m.keys.Back, m.keys.Help, m.keys.Quit},
		}
	}
	return [][]key.Binding{
		{m.keys.Up, m.keys.Down},
		{m.keys.Open, m.keys.Section},
		{m.keys.Help, m.keys.Quit},
	}
}

// Ensure the model satisfies the help.KeyMap contract at compile time.
var _ help.KeyMap = Model{}

// Ensure the model satisfies tea.Model at compile time.
var _ tea.Model = Model{}
