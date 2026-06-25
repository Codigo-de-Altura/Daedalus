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
	"github.com/charmbracelet/glamour"
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
	keys  navKeyMap
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
}

// navKeyMap declares every navigation binding the shell understands. It is the
// SINGLE keymap used by every area and every sub-screen, which is what makes the
// shortcuts consistent everywhere (R4/CA6): enter/back/quit behave identically no
// matter where the user is. It implements help.KeyMap so the contextual help
// footer is generated from the bindings themselves and can never drift from the
// real keys.
//
// 07-03 will formalize a central keybinding registry; this map is intentionally
// small and uniform so it can be lifted into that registry without changing any
// area's behavior.
type navKeyMap struct {
	Up    key.Binding
	Down  key.Binding
	Enter key.Binding
	Back  key.Binding
	Home  key.Binding
	Retry key.Binding
	Help  key.Binding
	Quit  key.Binding
	PgUp  key.Binding
	PgDn  key.Binding
	Top   key.Binding
	Botom key.Binding
}

func defaultNavKeyMap() navKeyMap {
	return navKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		// enter (and l, vim-style) is the universal "go in" key: it enters the
		// selected area from the root and opens the selected entry inside an area.
		Enter: key.NewBinding(
			key.WithKeys("enter", "l"),
			key.WithHelp("enter", "enter"),
		),
		// esc (and backspace) is the universal "go back one level" key. It is the
		// same everywhere — from a sub-screen back to its area, and from an area back
		// to the root — so the way out is always identical (R4/CA6). On the root it is
		// inert (there is nothing above the menu); q quits from there.
		Back: key.NewBinding(
			key.WithKeys("esc", "backspace"),
			key.WithHelp("esc", "back"),
		),
		// "h" (Home) jumps straight to the root from anywhere, so a user deep inside
		// an area can return to the menu in one keystroke instead of popping levels.
		Home: key.NewBinding(
			key.WithKeys("h"),
			key.WithHelp("h", "home"),
		),
		// "r" retries an area that failed to load, so an error state is never a dead
		// end: the user can re-trigger the core load without leaving and re-entering.
		Retry: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "retry"),
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

// New returns an initialized area-shell Model rooted at workdir. workdir is the
// directory whose `.daedalus/` workspace the areas read; passing "" means the
// current directory, so callers that do not care about the location can omit it.
// No filesystem access happens here — each area loads lazily, off the UI thread,
// the first time it is entered (see Update/enterArea).
func New(workdir string) Model {
	if workdir == "" {
		workdir = "."
	}
	return Model{
		workdir: workdir,
		theme:   defaultTheme(),
		keys:    defaultNavKeyMap(),
		help:    help.New(),
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
	case key.Matches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil
	case key.Matches(msg, m.keys.Quit):
		// q/ctrl+c quit from the root and from any area list, where leaving the app is
		// the natural action. Inside a scrollable sub-screen, q is reserved (esc is the
		// documented way back) so the user does not accidentally exit while reading;
		// ctrl+c still quits everywhere as the escape hatch.
		if m.current() == routeSub && msg.String() == "q" {
			return m, nil
		}
		return m, tea.Quit
	}

	// Home jumps to the root from anywhere except the root itself (where it would be
	// a no-op and "h" is free for future use). It is a one-key shortcut for popping
	// every level, complementing esc's one-level-at-a-time back.
	if key.Matches(msg, m.keys.Home) && m.current() != routeRoot {
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
	case key.Matches(msg, m.keys.Up):
		if m.rootCursor > 0 {
			m.rootCursor--
		}
	case key.Matches(msg, m.keys.Down):
		if m.rootCursor < len(areaOrder)-1 {
			m.rootCursor++
		}
	case key.Matches(msg, m.keys.Enter):
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

// View implements tea.Model. It is a pure projection of the model: it renders the
// active screen (root menu, an area, or a sub-screen) wrapped in the shared chrome
// — a breadcrumb header and the contextual help footer — and never mutates state.
func (m Model) View() string {
	switch m.current() {
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
	b.WriteString("\n\n")
	b.WriteString(m.help.View(m))
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
	if m.current() == routeSub {
		if label := m.subBreadcrumb(); label != "" {
			crumbs = append(crumbs, m.theme.subtle.Render(label))
		}
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

// renderMarkdown renders text as terminal Markdown via Glamour (R3), word-wrapped
// to the sub-screen viewport width so headings, lists, emphasis and code blocks
// read well and never overflow. On any render failure it falls back to the raw
// text so a sub-screen always shows its content rather than breaking.
func (m Model) renderMarkdown(text string) string {
	width, _ := m.subViewportSize()
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

// ShortHelp implements help.KeyMap. It returns the bindings relevant to the
// active screen so the one-line help footer always matches what the current
// screen accepts (R4/Check-10). The keys themselves are identical across areas —
// only which subset is advertised changes per route.
func (m Model) ShortHelp() []key.Binding {
	switch m.current() {
	case routeRoot:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Help, m.keys.Quit}
	case routeSub:
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Back, m.keys.Home, m.keys.Help, m.keys.Quit}
	default:
		st := m.areas[m.active]
		if st != nil && st.err != nil {
			return []key.Binding{m.keys.Retry, m.keys.Back, m.keys.Home, m.keys.Help, m.keys.Quit}
		}
		return []key.Binding{m.keys.Up, m.keys.Down, m.keys.Enter, m.keys.Back, m.keys.Home, m.keys.Help, m.keys.Quit}
	}
}

// FullHelp implements help.KeyMap, grouping the bindings shown when help is
// expanded (?). It mirrors ShortHelp's per-route subsets and adds the paging/jump
// keys a sub-screen supports.
func (m Model) FullHelp() [][]key.Binding {
	switch m.current() {
	case routeRoot:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down},
			{m.keys.Enter, m.keys.Help, m.keys.Quit},
		}
	case routeSub:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down, m.keys.PgUp, m.keys.PgDn},
			{m.keys.Top, m.keys.Botom},
			{m.keys.Back, m.keys.Home, m.keys.Help, m.keys.Quit},
		}
	default:
		return [][]key.Binding{
			{m.keys.Up, m.keys.Down, m.keys.Enter},
			{m.keys.Retry, m.keys.Back, m.keys.Home},
			{m.keys.Help, m.keys.Quit},
		}
	}
}

// Ensure the model satisfies the help.KeyMap and tea.Model contracts at compile
// time.
var (
	_ help.KeyMap = Model{}
	_ tea.Model   = Model{}
)
