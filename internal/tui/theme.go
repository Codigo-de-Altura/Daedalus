package tui

import (
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// theme centralizes every Lipgloss style token the TUI uses so that views share
// one consistent visual language (colors, spacing, borders) rather than each
// view inventing ad-hoc styles (R6). New views must pull their styles from here
// instead of constructing lipgloss styles inline.
//
// The palette is deliberately small and anchored on the same accent (color 63,
// the existing skeleton title color) so the prompt list, the preview and any
// future screen feel like one application.
type theme struct {
	// Accent is the primary brand color, reused by titles and the selection cursor.
	accent lipgloss.Color

	// Title styles a screen's heading (bold, accent-colored).
	title lipgloss.Style
	// subtle styles secondary, lower-emphasis text (help hints, metadata).
	subtle lipgloss.Style
	// listItem styles a non-selected row in the prompt list.
	listItem lipgloss.Style
	// listItemSelected styles the currently highlighted row in the prompt list.
	listItemSelected lipgloss.Style
	// kindBadge styles the small kind tag (global/shared) shown next to a prompt.
	kindBadge lipgloss.Style
	// emptyState styles the message shown when there is nothing to list.
	emptyState lipgloss.Style
	// errorBox styles a composition/error message inside the preview (R7).
	errorBox lipgloss.Style
	// previewFrame styles the bordered area that wraps the scrollable preview body.
	previewFrame lipgloss.Style
}

// defaultTheme builds the shared theme. Colors use 256-color codes so the look
// is stable across terminals; the accent (63, a soft indigo) matches the
// skeleton's original title color so this epic does not visually drift from the
// foundations.
func defaultTheme() theme {
	accent := lipgloss.Color("63")
	muted := lipgloss.Color("245")
	danger := lipgloss.Color("203")

	return theme{
		accent: accent,
		title:  lipgloss.NewStyle().Bold(true).Foreground(accent),
		subtle: lipgloss.NewStyle().Foreground(muted),
		listItem: lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			PaddingLeft(2),
		listItemSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			PaddingLeft(0).
			SetString("> "),
		kindBadge: lipgloss.NewStyle().
			Foreground(lipgloss.Color("39")),
		emptyState: lipgloss.NewStyle().
			Foreground(muted).
			Italic(true).
			Padding(1, 2),
		errorBox: lipgloss.NewStyle().
			Foreground(danger).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(danger).
			Padding(1, 2),
		previewFrame: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).
			Padding(0, 1),
	}
}

// DefaultFormTheme returns the Huh theme reserved for the interactive forms
// that later epics (agent and prompt editors, the init/build wizards) build on
// top of this skeleton. It lives in the foundations so the Huh dependency is
// fixed from the start and every form shares one consistent look.
func DefaultFormTheme() *huh.Theme {
	return huh.ThemeCharm()
}
