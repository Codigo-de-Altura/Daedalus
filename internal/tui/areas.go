package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// areas.go defines the six functional areas of the TUI and the generic machinery
// that drives any of them: an areaState (loading/empty/error + a list of items +
// a cursor), the key handling shared by every area's list, and the per-area views.
//
// The areas map one-to-one to the product domains and consume the core ONLY
// through the per-area commands in commands.go (R6): init → internal/workspace,
// agents → internal/catalog, prompts → internal/prompts, workflows →
// internal/workflows, backlog → internal/{backlog,specs,architecture}, build →
// internal/compile. No area reads the disk or runs domain logic itself; it asks a
// command to do it and renders the resulting items, empty state or error.
//
// Keeping the area behavior generic is what makes navigation consistent: every
// area uses the same cursor mechanics, the same enter/back/home keys, and the same
// loading/empty/error rendering, so a user who learns one area knows them all
// (R4/CA6).

// areaID identifies one of the six top-level areas.
type areaID int

const (
	areaInit areaID = iota
	areaAgents
	areaPrompts
	areaWorkflows
	areaBacklog
	areaBuild
)

// areaOrder is the fixed display order of the areas on the root menu, matching the
// product's domain order (init first, build last). It is the single source of
// truth for both the menu and the cursor bounds, so the menu can never drift from
// the set of real areas (R1).
var areaOrder = []areaID{
	areaInit,
	areaAgents,
	areaPrompts,
	areaWorkflows,
	areaBacklog,
	areaBuild,
}

// areaDef is the static description of an area: its menu title, its one-line
// purpose (shown on the menu so the user can tell the areas apart), and the empty
// state shown when the area loaded successfully but has nothing to list.
type areaDef struct {
	title   string
	summary string
	empty   string
}

// areaDefs holds every area's static description. The empty-state text points the
// user at the CLI verb that would populate the area, so an empty area is an
// actionable dead-endless state rather than a blank wall (R7/CA6).
var areaDefs = map[areaID]areaDef{
	areaInit: {
		title:   "Init",
		summary: "workspace status and what init would create",
		empty:   "No .daedalus workspace here yet.\n\nRun `daedalus init` to create one.",
	},
	areaAgents: {
		title:   "Agents",
		summary: "the built-in agent catalog",
		empty:   "The agent catalog is empty.",
	},
	areaPrompts: {
		title:   "Prompts",
		summary: "global and shared prompts",
		empty:   "No prompts found.\n\nCreate one with `daedalus prompt create`, or run `daedalus init` first.",
	},
	areaWorkflows: {
		title:   "Workflows",
		summary: "declarative DAG workflows",
		empty:   "No workflows found.\n\nCreate one with `daedalus workflow create`, or run `daedalus init` first.",
	},
	areaBacklog: {
		title:   "Backlog",
		summary: "specs, architecture, epics and tickets",
		empty:   "The backlog is empty.\n\nAdd specs with `daedalus spec`, epics with `daedalus epic`.",
	},
	areaBuild: {
		title:   "Build",
		summary: "preview compiling to the configured backend",
		empty:   "Nothing to compile — the build plan has no changes.",
	},
}

// areaState is one area's mutable state. It captures the async lifecycle (loading
// until the core load returns, then loaded with either items or an err) plus the
// list cursor, so each area remembers its own selection across navigation. The
// three render paths — loading, error, empty/list — are mutually exclusive, so an
// area never shows a blank or ambiguous screen.
type areaState struct {
	// loading is true while the core load is in flight (after enter, before the
	// areaLoadedMsg arrives). loaded becomes true once the load has returned at
	// least once, so a re-entered area is not reloaded.
	loading bool
	loaded  bool

	// err holds a load failure; when non-nil the area renders an actionable error
	// state with a retry that re-triggers the load (R7/CA6). items is the loaded
	// rows (empty slice = a valid empty state, not an error).
	err   error
	items []areaItem

	// cursor is the selected row within the CURRENTLY VISIBLE rows (items filtered
	// by filter). It indexes visibleItems(), not items, so filtering and selection
	// stay consistent.
	cursor int

	// filter is the active in-memory list filter (a case-insensitive substring set
	// via the filter form, R5). Empty means "show everything". It filters the
	// already-loaded items; it never touches the core.
	filter string

	// --- sub-screen state (set when a row is opened into a detail view) ---
	// sub captures the currently open sub-screen's lifecycle so the shared viewport
	// renders the right loading/ready/error state. key/title identify which item is
	// shown so a late-arriving load for an item the user navigated away from is
	// ignored.
	sub      subState
	subKey   string
	subTitle string
	subErr   string
}

// areaItem is one navigable row in an area's list. It carries a stable key (used
// to load the row's detail and to discard stale loads), a display label, and an
// optional badge (kind, phase count, status…) rendered next to the label so the
// list reads at a glance. opens reports whether enter opens a sub-screen for this
// row (some areas, like a status summary, have purely informational rows).
type areaItem struct {
	key   string
	label string
	badge string
	opens bool
}

// subState is the async lifecycle of an opened sub-screen, mirroring areaState's
// loading/loaded split: loading while the core composes/parses the detail, ready
// once it is rendered into the viewport, or errored when the load failed (R7/CA6).
type subState int

const (
	subLoading subState = iota
	subReady
	subErrored
)

// newAreaStates builds a fresh, empty state for every area. Each starts unloaded
// so its data is fetched lazily on first entry (RNF-2), not at startup.
func newAreaStates() map[areaID]*areaState {
	states := make(map[areaID]*areaState, len(areaOrder))
	for _, id := range areaOrder {
		states[id] = &areaState{}
	}
	return states
}

// visibleItems returns the items matching the active filter (case-insensitive
// substring on the row's label or badge). An empty filter returns every item. This
// is the single place the filter is applied, so the list view, the cursor bounds
// and "open" all agree on what is currently shown.
func (st *areaState) visibleItems() []areaItem {
	if strings.TrimSpace(st.filter) == "" {
		return st.items
	}
	needle := strings.ToLower(st.filter)
	var out []areaItem
	for _, it := range st.items {
		if strings.Contains(strings.ToLower(it.label), needle) ||
			strings.Contains(strings.ToLower(it.badge), needle) {
			out = append(out, it)
		}
	}
	return out
}

// handleAreaKey handles an area's list screen: cursor movement, opening the
// selected row into a sub-screen, going back to the root, and retrying a failed
// load. Back (esc) pops to the root and Home (h, handled in app.go) jumps there;
// either way the area is never a dead end (R3/CA4). The keys are the shared nav
// keys, so this behaves identically for every area (R4/CA6).
func (m Model) handleAreaKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	st := m.areas[m.active]

	if key.Matches(msg, m.keys.Back) {
		return m.pop(), nil
	}

	// In the error state, retry re-triggers the core load so the user can recover
	// in place instead of being stuck (R7/CA6). Other navigation still works.
	if st.err != nil && key.Matches(msg, m.keys.Retry) {
		st.loading = true
		st.loaded = false
		st.err = nil
		return m, loadAreaCmd(m.workdir, m.active)
	}

	// While loading or in an error state there is no list to move through, so only
	// Back/Home/Retry (above) apply; ignore list keys.
	if st.loading || st.err != nil {
		return m, nil
	}

	// "/" opens the filter form over the loaded list (R5). It is available whenever
	// there are items to filter, including over an active filter (to refine it).
	if key.Matches(msg, m.keys.Filter) && len(st.items) > 0 {
		return m.openFilterForm()
	}

	// With no items at all (a genuinely empty area) there is nothing to move through.
	if len(st.items) == 0 {
		return m, nil
	}

	visible := st.visibleItems()
	switch {
	case key.Matches(msg, m.keys.Up):
		if st.cursor > 0 {
			st.cursor--
		}
	case key.Matches(msg, m.keys.Down):
		if st.cursor < len(visible)-1 {
			st.cursor++
		}
	case key.Matches(msg, m.keys.Enter):
		if len(visible) == 0 {
			return m, nil
		}
		return m.openSub()
	}
	return m, nil
}

// openSub opens the selected row into a read-only sub-screen and starts loading
// its detail asynchronously. The viewport is reset to a loading state immediately
// so the user gets instant feedback while the core works. Rows that carry no
// detail (opens == false) are a no-op, so enter never leads to a blank sub-screen.
func (m Model) openSub() (tea.Model, tea.Cmd) {
	st := m.areas[m.active]
	visible := st.visibleItems()
	if st.cursor >= len(visible) {
		return m, nil
	}
	item := visible[st.cursor]
	if !item.opens {
		return m, nil
	}

	m.stack = append(m.stack, routeSub)
	st.sub = subLoading
	st.subKey = item.key
	st.subTitle = item.label
	st.subErr = ""
	if m.viewportReady {
		m.viewport.SetContent("")
		m.viewport.GotoTop()
	}
	return m, loadSubCmd(m.workdir, m.active, item.key)
}

// handleSubLoaded stores an opened sub-screen's rendered content (or its load
// error) for the area that requested it. A result whose area/key no longer matches
// the active sub-screen is discarded, so a slow load can never overwrite a newer
// view. A failed load becomes an error state the sub-screen renders with a way
// back intact (R7/CA6).
func (m Model) handleSubLoaded(msg subLoadedMsg) (tea.Model, tea.Cmd) {
	st := m.areas[msg.id]
	if st.subKey != msg.key {
		return m, nil
	}
	if msg.err != nil {
		st.sub = subErrored
		st.subErr = msg.err.Error()
		return m, nil
	}
	st.sub = subReady
	if m.viewportReady {
		content := msg.content
		if msg.markdown {
			content = m.renderMarkdown(content)
		}
		m.viewport.SetContent(content)
		m.viewport.GotoTop()
	}
	return m, nil
}

// handleSubKey handles a sub-screen: scrolling and returning to the area. Back
// (esc) pops to the area list (R3/CA4); the sub-screens are strictly read-only, so
// there is deliberately no binding that mutates or runs anything. Top/bottom jumps
// are explicit; the viewport handles line/page scrolling via its own Update.
func (m Model) handleSubKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, m.keys.Back) {
		return m.pop(), nil
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

// viewArea renders the active area's screen: its loading, error or empty state, or
// its navigable list of items. Every path keeps the way back visible via the
// shared chrome (breadcrumb + help), so no area state traps the user (R7/CA6).
func (m Model) viewArea() string {
	st := m.areas[m.active]
	def := areaDefs[m.active]

	switch {
	case st.loading:
		return m.theme.loading.Render("Loading " + strings.ToLower(def.title) + "…")
	case st.err != nil:
		return m.errorPanel(
			fmt.Sprintf("Could not load %s.\n\n%v\n\nPress r to retry, or esc to go back.",
				strings.ToLower(def.title), st.err))
	case len(st.items) == 0:
		return m.emptyPanel(def.empty)
	default:
		return m.renderItems(st)
	}
}

// errorPanel renders an error message inside the themed, alignment-corrected error
// box (R8/Check-9). It is the single error-panel renderer so every error state —
// area load, sub-screen load, form failure — shares one look and the border closes
// evenly regardless of the message's uneven line widths (fixes the 07-01 errorBox
// finding via theme.box).
func (m Model) errorPanel(msg string) string {
	return m.theme.box(m.theme.errorBox, m.theme.errorText.Render(msg))
}

// emptyPanel renders an empty-state message inside the themed empty box, so an
// empty area reads as a deliberate, on-theme panel (R8/Check-8) rather than a stray
// italic line. It uses the same width-normalizing box helper as errorPanel.
func (m Model) emptyPanel(msg string) string {
	return m.theme.box(m.theme.emptyBox, m.theme.emptyState.Render(msg))
}

// renderItems renders an area's list: each row as "<label>  <badge>", marking the
// selected row with the theme's cursor. It is shared by every area so the lists
// look and behave identically (R4/CA6). It renders the FILTERED view, shows the
// active filter, and degrades gracefully when a filter matches nothing — without
// trapping the user (they can clear the filter or go back). A footer hint advertises
// open/filter/back so the user knows what each key does (Check-10).
func (m Model) renderItems(st *areaState) string {
	var b strings.Builder

	// Show the active filter as a removable banner so the user is never confused by a
	// short list that is actually filtered (and knows "/" refines it).
	if strings.TrimSpace(st.filter) != "" {
		b.WriteString(m.theme.subtle.Render(fmt.Sprintf("Filter: %q  ·  press / to change", st.filter)))
		b.WriteString("\n\n")
	}

	visible := st.visibleItems()
	if len(visible) == 0 {
		// A filter that matches nothing is not a dead end: tell the user and how to
		// recover.
		b.WriteString(m.theme.emptyState.Render(
			fmt.Sprintf("No matches for %q.\n\nPress / to change the filter, or esc to go back.", st.filter)))
		return b.String()
	}

	for i, it := range visible {
		row := it.label
		if it.badge != "" {
			row += "  " + m.theme.kindBadge.Render(it.badge)
		}
		if i == st.cursor {
			b.WriteString(m.theme.listItemSelected.Render() + row)
		} else {
			b.WriteString(m.theme.listItem.Render(row))
		}
		if i < len(visible)-1 {
			b.WriteString("\n")
		}
	}

	b.WriteString("\n\n")
	if st.cursor < len(visible) && visible[st.cursor].opens {
		b.WriteString(m.theme.subtle.Render("Press enter to open · / filter · esc to go back."))
	} else {
		b.WriteString(m.theme.subtle.Render("Press / to filter · esc to go back."))
	}
	return b.String()
}

// viewSub renders the active sub-screen: a sub-header naming what is shown, the
// scrollable, read-only body (or its loading/error state), and a scroll hint. The
// body is wrapped in the theme's bordered frame so it reads as a distinct panel,
// consistent with the prompt preview and build views it reconciles.
func (m Model) viewSub() string {
	st := m.areas[m.active]

	var b strings.Builder
	// The sub-screen title is a section heading (a level below the breadcrumb title),
	// so it uses the heading token for a clear visual hierarchy.
	header := m.theme.heading.Render(st.subTitle)
	b.WriteString(header)
	b.WriteString("\n\n")

	switch st.sub {
	case subLoading:
		b.WriteString(m.theme.loading.Render("Loading…"))
	case subErrored:
		b.WriteString(m.errorPanel(st.subErr + "\n\nPress esc to go back."))
	default:
		b.WriteString(m.theme.previewFrame.Render(m.viewport.View()))
		b.WriteString("\n")
		b.WriteString(m.theme.subtle.Render(m.scrollHint()))
	}
	return b.String()
}

// subBreadcrumb returns the breadcrumb label for the active sub-screen (the opened
// item's title), so the trail reads "Daedalus › Area › <item>". Empty when no
// sub-screen is active.
func (m Model) subBreadcrumb() string {
	st := m.areas[m.active]
	return st.subTitle
}

// formatScrollPercent renders a viewport scroll fraction as a discoverable hint,
// kept here so the shell's scrollHint and any future scrollable share one phrasing.
func formatScrollPercent(frac float64) string {
	return fmt.Sprintf("%3.0f%% — scroll with ↑/↓ · pgup/pgdn · g/G", frac*100)
}
