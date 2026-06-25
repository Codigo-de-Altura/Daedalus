// Package tui contains the Bubble Tea presentation layer for Daedalus.
//
// This package owns everything the user sees and touches in the terminal. It
// follows the Elm architecture (Model/Update/View) strictly: the Model holds all
// state, Update is the single place state changes (reacting to typed Msgs), and
// every side effect — listing or composing the workspace's artifacts — happens
// through a tea.Cmd (see commands.go) so the UI thread never blocks. No domain,
// composition or persistence logic lives here; the TUI consumes the internal/*
// core packages through clean interfaces only.
//
// The shell (this file) is a six-area navigation frame: a root screen that lists
// the areas (init, agents, prompts, workflows, backlog, build), each reachable by
// keyboard, each with a consistent way back to the previous screen and to the
// root, and each rendering a loading/empty/error state that never traps the user
// (epic-07, ticket-07-01). The areas consume the core through the per-area
// commands in commands.go; the sub-screens (prompt preview, workflow DAG, build
// plan) are reconciled inside this same frame so navigation stays uniform.
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// route identifies one screen in the navigation stack. The stack is the single
// source of truth for "where am I and how do I get back": pushing a route enters
// a screen, popping returns to the previous one, and clearing back to the root
// returns home. Because every screen is reached by a push, there is always a pop
// that undoes it — structurally there are no dead ends (R3/CA4).
type route int

const (
	// routeRoot is the area menu: the six areas, one of which is selected.
	routeRoot route = iota
	// routeArea is an area's own screen (its list / loading / empty / error).
	routeArea
	// routeSub is a sub-screen reached from within an area (prompt preview,
	// workflow DAG, build plan): a scrollable, read-only detail view.
	routeSub
	// routeForm is a form screen reached from within an area (e.g. the list filter):
	// an embedded Huh form the user submits or cancels, returning to the area either
	// way (no dead end).
	routeForm
)

// Model is the Bubble Tea model for the Daedalus area shell. It holds the
// navigation stack (which screen is active and the breadcrumb back to the root),
// the cursor on the root menu, and the per-area state. All mutation happens in
// Update; View is a pure projection of this state.
type Model struct {
	// workdir is the directory whose `.daedalus/` workspace the areas read. It is
	// the process working directory by default (see New), so launching the TUI
	// inside a project shows that project's workspace.
	workdir string

	theme theme
	keys  keymap
	help  help.Model

	// width/height track the terminal size so the lists and the sub-screen viewport
	// resize with the window.
	width  int
	height int

	// stack is the navigation stack. It always has at least one entry (routeRoot);
	// the last entry is the active screen and the entries below it are the way back.
	// Pushing enters a screen, popping (Back) returns to the previous one, and
	// resetting to just the root (Home) returns to the menu — so the user is never
	// trapped (R3/CA4).
	stack []route

	// rootCursor is the selected area on the root menu.
	rootCursor int

	// active is the area currently entered (meaningful when the active route is
	// routeArea or a routeSub reached from it). It selects which per-area state the
	// area screen renders and which sub-screen a sub route shows.
	active areaID

	// areas holds every area's independent state (loading/empty/error + items +
	// cursor), keyed by areaID so each area remembers its own selection across
	// navigation.
	areas map[areaID]*areaState

	// --- shared sub-screen viewport ---
	// The prompt preview, the workflow DAG and the build plan are all scrollable,
	// read-only detail views; only one is ever active at a time, so they share one
	// viewport. The active sub-screen's identity/title/state lives on the relevant
	// areaState.
	viewport      viewport.Model
	viewportReady bool

	// --- active form (routeForm) ---
	// form is the embedded form component while the active route is routeForm; it is
	// the reusable themed wrapper around a Huh form (see form.go). The submitted value
	// is read back from the form itself (form.StringValue), not a model pointer, since
	// the model is copied on every Update. Only one form is ever active at a time, so a
	// single slot suffices.
	form formComponent
}

// New returns an initialized area-shell Model rooted at workdir. workdir is the
// directory whose `.daedalus/` workspace the areas read; passing "" means the
// current directory, so callers that do not care about the location can omit it.
// No filesystem access happens here — each area loads lazily, off the UI thread,
// the first time it is entered (see Update/enterArea).
func New(workdir string) Model {
	if workdir == "" {
		workdir = "."
	}
	th := defaultTheme()
	return Model{
		workdir: workdir,
		theme:   th,
		keys:    defaultKeymap(),
		help:    newHelpModel(th),
		stack:   []route{routeRoot},
		areas:   newAreaStates(),
	}
}

// Init implements tea.Model. The shell starts on the root menu, which needs no
// data, so there is nothing to load up front: each area's data is fetched lazily
// when first entered. This keeps startup instant and avoids reading the workspace
// for areas the user may never open (RNF-2).
func (m Model) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model. It is the single place state changes: it routes
// window-size and the per-area async result messages, then dispatches key
// messages to the active screen's handler. Side effects are returned as tea.Cmds,
// never run inline.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// While a form is active it owns the keyboard: it must receive every message
	// (keys AND its own cursor-blink ticks) so typing and validation work. The one
	// exception is the dedicated help key (?), which toggles the expanded help in
	// EVERY context — including forms (R4/Check-4) — so help is always reachable with
	// the same key; everything else (including a literal q or letters) goes to the
	// form so typing is never swallowed.
	if m.current() == routeForm {
		switch msg := msg.(type) {
		case tea.WindowSizeMsg:
			// Still track size so other screens are laid out correctly after the form.
			m.width = msg.Width
			m.height = msg.Height
		case tea.KeyMsg:
			if key.Matches(msg, m.keys.binding(actionHelp)) {
				m.help.ShowAll = !m.help.ShowAll
				return m, nil
			}
		}
		return m.updateForm(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m.handleResize(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)

	case areaLoadedMsg:
		return m.handleAreaLoaded(msg)

	case subLoadedMsg:
		return m.handleSubLoaded(msg)
	}

	// Forward any other message to the viewport while a sub-screen is active (e.g.
	// internal viewport ticks), so scrolling stays responsive.
	if m.current() == routeSub && m.viewportReady {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

// current returns the active route — the top of the navigation stack. The stack
// is never empty (New seeds it with routeRoot), so this is always defined.
func (m Model) current() route {
	return m.stack[len(m.stack)-1]
}

// handleResize recomputes the layout when the terminal size changes. It sizes the
// shared sub-screen viewport to the area left below the header and above the help
// so long content scrolls within a stable frame.
func (m Model) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	// Let the help renderer know the available width so the short bar truncates and
	// the expanded columns lay out within the terminal instead of overflowing.
	m.help.Width = msg.Width

	vpWidth, vpHeight := m.subViewportSize()
	if !m.viewportReady {
		m.viewport = viewport.New(vpWidth, vpHeight)
		m.viewportReady = true
	} else {
		m.viewport.Width = vpWidth
		m.viewport.Height = vpHeight
	}
	return m, nil
}

// handleKey dispatches a key press. Quit and help-toggle are global so they
// behave the same on every screen; everything else is delegated to the active
// route's handler, which uses the one shared navKeyMap so entering/going back is
// uniform across areas (R4/CA6).
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.binding(actionHelp)):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case key.Matches(msg, m.keys.binding(actionQuit)):
		// q/ctrl+c quit from the root and from any area list, where leaving the app is
		// the natural action. Inside a scrollable sub-screen, q is reserved (esc is the
		// documented way back) so the user does not accidentally exit while reading;
		// ctrl+c still quits everywhere as the escape hatch.
		//
		// A form is NEVER reached here — Update routes routeForm to updateForm before
		// handleKey runs, so a typed "q" reaches the text field instead of quitting.
		// We still guard explicitly so a future reorder of Update cannot let the
		// global "q" quit swallow a character the user is typing into a form input.
		if msg.String() == "q" && (m.current() == routeSub || m.current() == routeForm) {
			return m, nil
		}
		return m, tea.Quit
	}

	// Home jumps to the root from anywhere except the root itself (where it would be
	// a no-op and "h" is free for future use). It is a one-key shortcut for popping
	// every level, complementing esc's one-level-at-a-time back.
	if key.Matches(msg, m.keys.binding(actionHome)) && m.current() != routeRoot {
		return m.goHome(), nil
	}

	switch m.current() {
	case routeRoot:
		return m.handleRootKey(msg)
	case routeArea:
		return m.handleAreaKey(msg)
	case routeSub:
		return m.handleSubKey(msg)
	}
	return m, nil
}

// handleRootKey handles the area menu: move the selection and enter the selected
// area. esc is inert here (the root is the top of the stack — there is nowhere
// above it to go back to), so the only way out of the app from the root is q,
// keeping quit predictable.
func (m Model) handleRootKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.binding(actionUp)):
		if m.rootCursor > 0 {
			m.rootCursor--
		}
	case key.Matches(msg, m.keys.binding(actionDown)):
		if m.rootCursor < len(areaOrder)-1 {
			m.rootCursor++
		}
	case key.Matches(msg, m.keys.binding(actionEnter)):
		return m.enterArea(areaOrder[m.rootCursor])
	}
	return m, nil
}

// goHome resets the navigation stack to just the root, returning the user to the
// area menu from anywhere in one step. It does not discard per-area state, so
// re-entering an area shows it exactly as the user left it.
func (m Model) goHome() Model {
	m.stack = []route{routeRoot}
	return m
}

// pop removes the top screen, returning to the previous one. It never pops the
// root (the stack always keeps at least routeRoot), so esc on the root is a safe
// no-op rather than emptying the stack.
func (m Model) pop() Model {
	if len(m.stack) > 1 {
		m.stack = m.stack[:len(m.stack)-1]
	}
	return m
}

// enterArea pushes the selected area onto the stack and, if that area has not
// loaded its data yet, kicks off the load off the UI thread. The area is marked
// loading immediately so entering it gives instant feedback; the data arrives
// later as an areaLoadedMsg. An area already loaded (the user is re-entering) is
// shown straight away with its remembered selection, with no redundant reload.
func (m Model) enterArea(id areaID) (tea.Model, tea.Cmd) {
	m.active = id
	m.stack = append(m.stack, routeArea)

	st := m.areas[id]
	if st.loaded || st.loading {
		return m, nil
	}
	st.loading = true
	return m, loadAreaCmd(m.workdir, id)
}

// handleAreaLoaded stores an area's loaded data (or its load error) on that
// area's state. A load that fails becomes an error state the area renders without
// blocking navigation — the user can still go back or retry (R7/CA6).
func (m Model) handleAreaLoaded(msg areaLoadedMsg) (tea.Model, tea.Cmd) {
	st := m.areas[msg.id]
	st.loading = false
	st.loaded = true
	st.err = msg.err
	st.items = msg.items
	if st.cursor >= len(st.items) {
		st.cursor = 0
	}
	return m, nil
}

// openFilterForm pushes the list-filter form for the active area onto the stack
// and initializes it, seeding the input with the area's current filter so opening
// it again refines rather than resets. It returns the form's Init command so the
// input focuses immediately. Filtering is purely in-memory (it shapes a string the
// area matches against already-loaded items), so this needs no core seam.
func (m Model) openFilterForm() (tea.Model, tea.Cmd) {
	st := m.areas[m.active]
	m.form = newFilterForm(m.theme, areaDefs[m.active].title, st.filter)
	m.stack = append(m.stack, routeForm)
	return m, m.form.Init()
}

// updateForm drives the active form and acts on its lifecycle. On submit it applies
// the captured value (here: the list filter) and returns to the area; on cancel it
// returns to the area unchanged. Either outcome pops routeForm, so a form is never
// a dead end (R7/Check-6). While the form is pending it just keeps rendering.
func (m Model) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	form, result, cmd := m.form.Update(msg)
	m.form = form

	switch result {
	case formSubmitted:
		// Apply the filter and reset the cursor so the selection is valid against the
		// newly filtered view. The value was validated by the form before submit and is
		// read back from the form itself (not a stale model pointer).
		st := m.areas[m.active]
		st.filter = strings.TrimSpace(m.form.StringValue())
		st.cursor = 0
		return m.pop(), nil
	case formCancelled:
		return m.pop(), nil
	default:
		return m, cmd
	}
}

// View implements tea.Model. It is a pure projection of the model: it renders the
// active screen (root menu, an area, or a sub-screen) wrapped in the shared chrome
// — a breadcrumb header and the contextual help footer — and never mutates state.
func (m Model) View() string {
	switch m.current() {
	case routeForm:
		return m.frame(m.form.View())
	case routeSub:
		return m.frame(m.viewSub())
	case routeArea:
		return m.frame(m.viewArea())
	default:
		return m.frame(m.viewRoot())
	}
}

// frame wraps a screen body with the shared chrome: a breadcrumb that names the
// active area (and sub-screen) so the user always knows where they are (R5/CA5),
// and the contextual help footer generated from the active screen's bindings so
// the way to navigate is always visible (Check-10).
func (m Model) frame(body string) string {
	var b strings.Builder
	b.WriteString(m.breadcrumb())
	b.WriteString("\n\n")
	b.WriteString(body)
	// EVERY screen — including forms — gets exactly one contextual help footer,
	// rendered here from the central registry. Forms used to draw their own help line
	// (plus Huh's), producing a redundant double footer (07-02 minor); now the form
	// body carries no help and Huh's built-in help is off, so this single footer is
	// the sole, consistent help source across the whole TUI (R6/R7).
	//
	// Collapsed: the short bar (bubbles/help). Expanded (?): our own column layout
	// (renderFullHelp) which keeps a clear gutter between groups; both draw from the
	// same registry-resolved bindings, so announced == real either way.
	b.WriteString("\n\n")
	if m.help.ShowAll {
		b.WriteString(renderFullHelp(m.theme, m.FullHelp()))
	} else {
		b.WriteString(m.help.ShortHelpView(m.ShortHelp()))
	}
	return b.String()
}

// breadcrumb renders the navigation trail "Daedalus › Area › Sub" so the active
// area is always identifiable and the path back is visible (R5/CA5). The leading
// "Daedalus" is the root; the user can read it as "press esc to walk back up this
// trail, or h to jump home".
func (m Model) breadcrumb() string {
	crumbs := []string{m.theme.title.Render("Daedalus")}
	if m.current() != routeRoot {
		crumbs = append(crumbs, m.theme.breadcrumbActive.Render(areaDefs[m.active].title))
	}
	switch m.current() {
	case routeSub:
		if label := m.subBreadcrumb(); label != "" {
			crumbs = append(crumbs, m.theme.subtle.Render(label))
		}
	case routeForm:
		crumbs = append(crumbs, m.theme.subtle.Render("Filter"))
	}
	return strings.Join(crumbs, m.theme.breadcrumbSep.Render(" › "))
}

// viewRoot renders the area menu: the six areas as a navigable list, each marked
// with its one-line purpose so the user can tell them apart, with the selected
// area highlighted (R1/R5).
func (m Model) viewRoot() string {
	var b strings.Builder
	b.WriteString(m.theme.subtle.Render("Choose an area:"))
	b.WriteString("\n\n")
	for i, id := range areaOrder {
		def := areaDefs[id]
		row := def.title + "  " + m.theme.subtle.Render(def.summary)
		if i == m.rootCursor {
			b.WriteString(m.theme.listItemSelected.Render() + row)
		} else {
			b.WriteString(m.theme.listItem.Render(row))
		}
		if i < len(areaOrder)-1 {
			b.WriteString("\n")
		}
	}
	return b.String()
}

// subViewportSize computes the width/height available to the shared sub-screen
// viewport, reserving rows for the breadcrumb header, the help footer and the
// frame border so the scrollable area never collides with the chrome. Before the
// first size message it returns a sensible default so the model is usable in tests
// without a terminal.
func (m Model) subViewportSize() (int, int) {
	width := m.width
	height := m.height
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	// Reserve: breadcrumb + blank, a sub-header line + blank, 1 scroll hint,
	// blank + 1 help line, and 2 for the frame border.
	const chrome = 9
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

// scrollHint shows how far through a sub-screen's content the user is, so a long
// document's scroll position is always discoverable.
func (m Model) scrollHint() string {
	if !m.viewportReady || m.viewport.TotalLineCount() <= m.viewport.Height {
		return "all content shown"
	}
	return strings.TrimSpace(
		// Reuse the same vocabulary the build/preview screens use for consistency.
		formatScrollPercent(m.viewport.ScrollPercent()))
}

// helpContextFor returns the declared help context for the active screen — the
// subset of actions that apply right here (R5). It is the single switch that maps a
// route (and an area's error state) to its declared context; the contexts
// themselves live in help.go and the keys behind them in keybindings.go, so this
// function chooses WHAT applies, never WHICH key it is.
func (m Model) helpContextFor() helpContext {
	switch m.current() {
	case routeRoot:
		return rootHelp
	case routeForm:
		return formHelp
	case routeSub:
		return subHelp
	default:
		st := m.areas[m.active]
		if st != nil && st.err != nil {
			return areaErrorHelp
		}
		return areaHelp
	}
}

// ShortHelp / FullHelp implement bubbles/help's KeyMap by delegating to the active
// context resolved against the central registry. Because both views come from the
// same context and the same bindings, the short bar and the expanded view agree
// with each other and with the real keys (announced == real, R7/Check-5).
func (m Model) ShortHelp() []key.Binding {
	return m.helpContextFor().shortBindings(m.keys)
}

func (m Model) FullHelp() [][]key.Binding {
	return m.helpContextFor().fullBindings(m.keys)
}

// Ensure the model satisfies the help.KeyMap and tea.Model contracts at compile
// time.
var (
	_ help.KeyMap = Model{}
	_ tea.Model   = Model{}
)
