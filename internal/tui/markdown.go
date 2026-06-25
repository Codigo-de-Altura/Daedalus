package tui

import (
	"sync"

	"github.com/charmbracelet/glamour"
)

// markdown.go is the reusable markdown rendering component (ticket-07-02, R3/R4).
// Every place in the TUI that shows a markdown document — the prompt preview and
// the backlog's specs / architecture / epics — renders through renderMarkdownWidth
// so they all look identical and on-theme. It is presentation only: it formats a
// string, it never reads a file or runs domain logic.
//
// Performance (ticket-07-04): constructing a Glamour TermRenderer is relatively
// expensive (it parses the style config and builds a chroma highlighter). The
// render itself is invoked OFF the UI thread (inside loadSubCmd, see commands.go),
// so a large document never blocks the Bubble Tea loop. On top of that, the
// renderers are cached per wrap width (see rendererCache) so navigating in and out
// of documents at a stable terminal size reuses one renderer instead of
// reallocating it every time — keeping memory bounded under repeated navigation.

// renderMarkdownWidth renders markdown text to themed, terminal-formatted output
// wrapped to wrap columns. It is the single entry point for markdown rendering so
// the look and the width handling live in exactly one place. It reuses a cached
// renderer for the given width when possible. On any renderer construction or
// render failure it falls back to the raw text, so a document is always shown
// rather than the screen breaking.
//
// It is safe to call from a tea.Cmd goroutine: the renderer cache is mutex-guarded.
func (t theme) renderMarkdownWidth(text string, wrap int) string {
	if wrap < minMarkdownWrap {
		wrap = minMarkdownWrap
	}
	r := t.renderer(wrap)
	if r == nil {
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

// maxCachedRenderers bounds the renderer cache so it can never grow without limit:
// only a handful of distinct widths occur in practice (the terminal is one width at
// a time; a resize adds at most a few). When the bound is exceeded the cache is
// cleared wholesale rather than carrying an ever-growing map (requirement: any cache
// we add must be bounded). A small constant keeps memory flat under heavy use.
const maxCachedRenderers = 8

// rendererCache memoizes Glamour renderers by wrap width. The theme/style is fixed
// for the process (defaultTheme is a singleton in practice), so the width is the
// only thing that varies a renderer; keying by width is sufficient and keeps the
// cache tiny. It is package-level and mutex-guarded because renders run in tea.Cmd
// goroutines (off the UI thread), so lookups can race.
var rendererCache = struct {
	mu sync.Mutex
	m  map[int]*glamour.TermRenderer
}{m: make(map[int]*glamour.TermRenderer)}

// renderer returns a Glamour renderer for the given wrap width, building and
// caching one on first use. A construction failure returns nil (the caller falls
// back to raw text). The cache is bounded by maxCachedRenderers: exceeding it clears
// the map, so the working set stays small and memory does not grow with navigation.
func (t theme) renderer(wrap int) *glamour.TermRenderer {
	rendererCache.mu.Lock()
	defer rendererCache.mu.Unlock()

	if r, ok := rendererCache.m[wrap]; ok {
		return r
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(t.markdownStyle()),
		glamour.WithWordWrap(wrap),
	)
	if err != nil {
		return nil
	}

	if len(rendererCache.m) >= maxCachedRenderers {
		// Bounded: drop the whole map rather than let it grow unbounded. The few live
		// widths will be rebuilt lazily on next use.
		rendererCache.m = make(map[int]*glamour.TermRenderer, maxCachedRenderers)
	}
	rendererCache.m[wrap] = r
	return r
}

// markdownWrapForWidth converts a viewport width to the wrap width used for the
// sub-screen body: it subtracts the frame's horizontal border+padding (2 cols) so
// rendered content fits inside previewFrame, and clamps to a sane minimum. It is
// the single place the open-time wrap is computed, so the command (off-thread) and
// any direct caller agree.
func markdownWrapForWidth(viewportWidth int) int {
	wrap := viewportWidth - 2
	if wrap < minMarkdownWrap {
		wrap = minMarkdownWrap
	}
	return wrap
}
