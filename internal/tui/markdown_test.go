package tui

import (
	"regexp"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

// ansiSeq matches ANSI escape sequences so tests can assert on the visible text
// Glamour produced, independent of the color codes it interleaves.
var ansiSeq = regexp.MustCompile("\x1b\\[[0-9;]*m")

// visibleText strips ANSI sequences and collapses runs of whitespace to single
// spaces, so a substring assertion is robust against Glamour's per-cell coloring
// and soft-wrap padding (which interleaves spaces and color codes between words).
func visibleText(s string) string {
	clean := ansiSeq.ReplaceAllString(s, "")
	return strings.Join(strings.Fields(clean), " ")
}

// markdown_test.go covers the reusable themed markdown renderer (ticket-07-02,
// R3/R4): it must format the markdown the product actually uses — headings, lists,
// tables, code blocks, emphasis — and keep every line within the requested width so
// it fits inside the viewport frame (the 07-01 "wide colored block" finding).

const sampleMarkdown = "# Title\n\n" +
	"Some **bold** and *italic* text.\n\n" +
	"## Section\n\n" +
	"- first item\n" +
	"- second item\n\n" +
	"```go\nfmt.Println(\"hi\")\n```\n\n" +
	"| Col A | Col B |\n|-------|-------|\n| a1 | b1 |\n| a2 | b2 |\n"

// TestMarkdownRendersFormatted verifies the renderer formats (does not echo raw
// markdown) and includes the document's textual content.
func TestMarkdownRendersFormatted(t *testing.T) {
	th := defaultTheme()
	out := th.renderMarkdownWidth(sampleMarkdown, 72)

	if strings.TrimSpace(out) == "" {
		t.Fatal("renderer produced empty output")
	}
	// Content survives rendering (asserted on the visible text, ignoring the color
	// codes and soft-wrap padding Glamour interleaves between words).
	visible := visibleText(out)
	for _, want := range []string{"Title", "Section", "first item", "second item", "Col A", "Col B"} {
		if !strings.Contains(visible, want) {
			t.Errorf("rendered markdown missing %q, got visible text:\n%s", want, visible)
		}
	}
	// The table separators render (Glamour draws box-drawing/aligned columns), so the
	// output is not the raw pipe-table source.
	if strings.Contains(visible, "|-------|") {
		t.Errorf("table should be rendered, not raw pipes, got:\n%s", visible)
	}
	// Emphasis is applied via ANSI styling, so styled output differs from plain text.
	if !strings.Contains(out, "\x1b[") {
		t.Errorf("rendered markdown should carry ANSI styling, got:\n%s", out)
	}
}

// TestMarkdownRespectsWidth verifies no rendered line exceeds the requested width,
// so the content always fits inside the viewport frame without horizontal overflow
// (the fix for the wide-block finding).
func TestMarkdownRespectsWidth(t *testing.T) {
	th := defaultTheme()
	const wrap = 60
	out := th.renderMarkdownWidth(sampleMarkdown, wrap)

	for _, line := range strings.Split(out, "\n") {
		// lipgloss.Width measures visible width, ignoring ANSI sequences.
		if w := lipgloss.Width(line); w > wrap {
			t.Errorf("line exceeds wrap width %d (got %d): %q", wrap, w, line)
		}
	}
}

// TestMarkdownTinyWidthClamped verifies an absurdly small width is clamped to the
// minimum rather than producing a degenerate column or panicking.
func TestMarkdownTinyWidthClamped(t *testing.T) {
	th := defaultTheme()
	out := th.renderMarkdownWidth("# Hi\n\ntext", 1)
	if strings.TrimSpace(out) == "" {
		t.Fatal("tiny width should still render something")
	}
}

// TestPromptPreviewRendersMarkdown verifies the prompt preview sub-screen renders
// its content as markdown through the shared renderer (R3 applied where documents
// are shown).
func TestPromptPreviewRendersMarkdown(t *testing.T) {
	m := sizedModel(t)
	m, _ = enterAreaByIndex(t, m, indexOf(areaPrompts))
	m = deliverAreaLoaded(m, areaPrompts, []areaItem{
		{key: "intro", label: "intro", opens: true},
	})
	m = update(m, "enter") // open the preview sub-screen
	m = deliverSubLoaded(m, areaPrompts, "intro", "# Heading\n\n- a\n- b")

	view := visibleText(m.View())
	if !strings.Contains(view, "Heading") {
		t.Errorf("prompt preview should render the markdown heading, got:\n%s", view)
	}
}
