package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

// newHelpModel builds the shared bubbles/help model, themed from the palette and
// given clear separators so the short bar and the expanded columns are legible
// (R6). Centralizing it here keeps the help look consistent and on-theme with the
// rest of the TUI, and gives the expanded view a roomy gutter so adjacent key
// groups never visually collide.
func newHelpModel(t theme) help.Model {
	h := help.New()
	h.ShortSeparator = "  •  "
	h.FullSeparator = "     "
	h.Styles.ShortKey = lipgloss.NewStyle().Foreground(t.palette.accent)
	h.Styles.ShortDesc = lipgloss.NewStyle().Foreground(t.palette.muted)
	h.Styles.ShortSeparator = lipgloss.NewStyle().Foreground(t.palette.muted)
	h.Styles.FullKey = lipgloss.NewStyle().Foreground(t.palette.accent)
	h.Styles.FullDesc = lipgloss.NewStyle().Foreground(t.palette.muted)
	h.Styles.FullSeparator = lipgloss.NewStyle().Foreground(t.palette.muted)
	return h
}

// help.go turns the central keybinding registry (keybindings.go) into contextual
// help (ticket-07-03, R3–R7). A screen declares WHICH actions it exposes — never
// which keys — as a helpContext; the renderer resolves those actions to their one
// binding in the registry and feeds them to bubbles/help. Because the keys and the
// help text come from the same key.Binding, what the help announces is exactly what
// runs (announced == real, Check-5): there is no parallel hardcoded help string to
// drift.
//
// Two views, one source (R6): the short help bar (always visible) and the expanded
// help view (toggled with ?) are produced from the same context by bubbles/help's
// ShortHelpView / FullHelpView, so they can never disagree about what is available.

// helpContext is a screen's declared help: the actions shown in the always-visible
// short bar, and the grouped actions shown in the expanded view. Both are lists of
// keyAction (not keys), so a context is a pure declaration of "what applies here"
// that the registry resolves to concrete keys.
type helpContext struct {
	// short is the compact, single-line subset — the few actions most worth knowing
	// at a glance.
	short []keyAction
	// full is the grouped, complete subset shown when help is expanded; each inner
	// slice is one column/group in the expanded layout.
	full [][]keyAction
}

// shortBindings / fullBindings resolve the context's declared actions against the
// registry, so the help renderer always gets the live bindings (and thus the live
// keys + help text). This is the single bridge from "declared actions" to "real
// keys" that guarantees announced == real.
func (c helpContext) shortBindings(k keymap) []key.Binding {
	return k.resolve(c.short)
}

func (c helpContext) fullBindings(k keymap) [][]key.Binding {
	out := make([][]key.Binding, 0, len(c.full))
	for _, group := range c.full {
		out = append(out, k.resolve(group))
	}
	return out
}

// renderFullHelp lays out the expanded help columns from the registry-resolved
// groups. We render it ourselves rather than via bubbles/help's FullHelpView because
// that version (v0.20.0) drops the separator between some columns (it compares the
// group index against the current group's length instead of the number of groups),
// which made adjacent key groups visually collide. Laying out the columns here — from
// the SAME bindings the registry provides — keeps a clear gutter between every group
// while preserving announced == real (the keys/descs still come from the registry).
func renderFullHelp(t theme, groups [][]key.Binding) string {
	keyStyle := lipgloss.NewStyle().Foreground(t.palette.accent)
	descStyle := lipgloss.NewStyle().Foreground(t.palette.muted)
	gutter := lipgloss.NewStyle().Foreground(t.palette.muted).Render("     ")

	var cols []string
	for _, group := range groups {
		var keys, descs []string
		for _, b := range group {
			if !b.Enabled() {
				continue
			}
			keys = append(keys, b.Help().Key)
			descs = append(descs, b.Help().Desc)
		}
		if len(keys) == 0 {
			continue
		}
		col := lipgloss.JoinHorizontal(lipgloss.Top,
			keyStyle.Render(strings.Join(keys, "\n")),
			keyStyle.Render(" "),
			descStyle.Render(strings.Join(descs, "\n")),
		)
		cols = append(cols, col)
	}
	if len(cols) == 0 {
		return ""
	}

	// Join the columns with an explicit gutter between every pair.
	withGutters := make([]string, 0, len(cols)*2-1)
	for i, c := range cols {
		if i > 0 {
			withGutters = append(withGutters, gutter)
		}
		withGutters = append(withGutters, c)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, withGutters...)
}

// The active context is resolved by Model.helpContextFor and exposed through the
// Model's own ShortHelp/FullHelp (app.go), which the shell footer renders. There is
// no separate adapter type: the Model is the single help.KeyMap, so there is exactly
// one path from "declared actions" to rendered help.

// --- per-context declarations -----------------------------------------------
//
// Each screen's help is declared here as the subset of actions it exposes. The keys
// are NOT repeated — only the actions — so consistency (same action ⇒ same key) is
// automatic and a screen can never accidentally advertise a key it does not handle,
// nor handle a key it does not advertise, as long as its handler matches the same
// actions it declares (the tests assert this announced == real property).

// rootHelp: the area menu — move the selection, open an area, help, quit. No back
// (the root is the top of the stack) so esc is intentionally absent here.
var rootHelp = helpContext{
	short: []keyAction{actionUp, actionDown, actionEnter, actionHelp, actionQuit},
	full: [][]keyAction{
		{actionUp, actionDown},
		{actionEnter, actionHelp, actionQuit},
	},
}

// areaHelp: an area's list — navigate, open an entry, filter, go back/home, help,
// quit. This is the richest context.
var areaHelp = helpContext{
	short: []keyAction{actionUp, actionDown, actionEnter, actionFilter, actionBack, actionHome, actionHelp, actionQuit},
	full: [][]keyAction{
		{actionUp, actionDown, actionEnter, actionFilter},
		{actionBack, actionHome},
		{actionHelp, actionQuit},
	},
}

// areaErrorHelp: an area whose load failed — retry plus the always-present back/
// home/help/quit, so an error state still announces (and offers) a way forward and a
// way out (Check-8).
var areaErrorHelp = helpContext{
	short: []keyAction{actionRetry, actionBack, actionHome, actionHelp, actionQuit},
	full: [][]keyAction{
		{actionRetry},
		{actionBack, actionHome},
		{actionHelp, actionQuit},
	},
}

// subHelp: a scrollable read-only sub-screen — scroll (line/page/jump), go back/
// home, help, quit. enter/filter do not apply here and are absent.
var subHelp = helpContext{
	short: []keyAction{actionUp, actionDown, actionBack, actionHome, actionHelp, actionQuit},
	full: [][]keyAction{
		{actionUp, actionDown, actionPageUp, actionPageDown},
		{actionTop, actionBottom},
		{actionBack, actionHome, actionHelp, actionQuit},
	},
}

// formHelp: a form — submit, cancel, move between fields, plus help and the hard
// quit. This is the context that makes Check-4 pass: a form's help lists submit/
// cancel/move-between-fields, not just generic navigation. It is rendered by the
// SAME shell footer as every other screen (unifying the old double help line), so
// there is one help source for forms too.
var formHelp = helpContext{
	short: []keyAction{actionFormSubmit, actionFormCancel, actionHelp},
	full: [][]keyAction{
		{actionFormSubmit, actionFormCancel},
		{actionFormNextField, actionFormPrevField},
		{actionHelp, actionQuit},
	},
}
