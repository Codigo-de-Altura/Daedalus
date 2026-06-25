package tui

import (
	"github.com/charmbracelet/glamour"
)

// markdown.go is the reusable markdown rendering component (ticket-07-02, R3/R4).
// Every place in the TUI that shows a markdown document — the prompt preview and
// the backlog's specs / architecture / epics — renders through renderMarkdownWidth
// so they all look identical and on-theme. It is presentation only: it formats a
// string, it never reads a file or runs domain logic.
//
// Two things the 07-01 review asked us to get right are handled here:
//   - The render is themed (headings, lists, tables, code, emphasis use the
//     palette via theme.markdownStyle), not Glamour's stock colors.
//   - The render is width-bounded to the viewport, so Glamour's H1 background
//     banner and long lines wrap inside the frame instead of overflowing it
//     (the "wide colored block" finding).

// renderMarkdownWidth renders markdown text to themed, terminal-formatted output
// wrapped to wrap columns. It is the single entry point for markdown rendering so
// the look and the width handling live in exactly one place. On any renderer
// construction or render failure it falls back to the raw text, so a document is
// always shown rather than the screen breaking — a malformed style or an exotic
// document can never blank the preview.
func (t theme) renderMarkdownWidth(text string, wrap int) string {
	if wrap < minMarkdownWrap {
		wrap = minMarkdownWrap
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(t.markdownStyle()),
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

// minMarkdownWrap is the floor for the markdown wrap width so a tiny terminal still
// produces readable (if narrow) output instead of a degenerate one-character
// column.
const minMarkdownWrap = 20

// renderMarkdown renders markdown for the shared sub-screen viewport, wrapping to
// the viewport width minus the frame padding so wrapped lines — and Glamour's
// background-colored headings — fit inside the bordered panel without a horizontal
// scrollbar. It delegates the actual rendering to the theme component so the model
// holds no markdown logic of its own.
func (m Model) renderMarkdown(text string) string {
	width, _ := m.subViewportSize()
	// Subtract the frame's horizontal border+padding (2 cols) so content fits inside
	// previewFrame; the component clamps to a sane minimum.
	return m.theme.renderMarkdownWidth(text, width-2)
}
