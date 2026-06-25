package tui

import (
	"strings"

	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// theme is the single source of visual truth for the TUI (ticket-07-02, RF-7.2):
// one palette and one set of role-based Lipgloss style tokens that every view,
// area and shared component pulls from, so the whole application speaks one visual
// language (RNF-4). No view constructs ad-hoc lipgloss styles or hardcodes a color
// — it reaches for a token here. The Glamour markdown style and the Huh form theme
// are derived from the SAME palette (see markdownStyle / formTheme) so rendered
// documents and forms match the rest of the chrome rather than drifting.
type theme struct {
	// palette is the small, named set of colors every token is built from. Keeping
	// the raw colors in one place means a re-theme touches the palette, not the
	// dozens of styles below.
	palette palette

	// Accent is the primary brand color, reused by titles and the selection cursor.
	// Kept as a field (not only in the palette) because several callers reference
	// m.theme.accent directly for inline measurements.
	accent lipgloss.Color

	// --- chrome: titles, breadcrumb, help, secondary text ---
	// title styles a screen's heading (bold, accent-colored).
	title lipgloss.Style
	// heading styles an in-content section heading (one notch below title).
	heading lipgloss.Style
	// subtle styles secondary, lower-emphasis text (help hints, metadata).
	subtle lipgloss.Style
	// breadcrumbActive styles the active area's name in the navigation breadcrumb
	// so the user can always see which of the six areas they are in.
	breadcrumbActive lipgloss.Style
	// breadcrumbSep styles the separator drawn between breadcrumb segments.
	breadcrumbSep lipgloss.Style

	// --- lists ---
	// listItem styles a non-selected row in a list.
	listItem lipgloss.Style
	// listItemSelected styles the currently highlighted row (the active selection),
	// shared by every list so the cursor reads the same everywhere (R2/Check-1).
	listItemSelected lipgloss.Style
	// kindBadge styles the small kind/metadata tag shown next to a list row.
	kindBadge lipgloss.Style

	// --- transversal states (loading / empty / error) ---
	// loading styles the in-progress indicator text so every async wait looks the
	// same (R8/Check-7).
	loading lipgloss.Style
	// emptyState styles the message shown when there is nothing to list (R8/Check-8).
	emptyState lipgloss.Style
	// emptyBox is the bordered frame the empty state is rendered inside, so empty
	// reads as a deliberate panel, not a stray line.
	emptyBox lipgloss.Style
	// errorText styles the message inside an error panel; errorBox is the bordered
	// frame around it (R8/Check-9).
	errorText lipgloss.Style
	errorBox  lipgloss.Style

	// previewFrame styles the bordered area that wraps a scrollable detail body.
	previewFrame lipgloss.Style

	// --- forms (Huh) ---
	// formTitle styles the heading drawn above an embedded form. The form's own help
	// is no longer styled here: the shell draws one contextual help footer for forms
	// (from the central registry) like every other screen (07-03), and the fields and
	// their validation errors are drawn by Huh using the derived form theme (formTheme).
	formTitle lipgloss.Style

	// --- workflow DAG view (ticket 04-02) ---
	dagNode   lipgloss.Style
	dagNodeID lipgloss.Style
	dagAgent  lipgloss.Style
	dagMeta   lipgloss.Style
	dagEdge   lipgloss.Style

	// --- build preview / diff view (ticket 06-04) ---
	summaryBox      lipgloss.Style
	statusCreated   lipgloss.Style
	statusUpdated   lipgloss.Style
	statusUnchanged lipgloss.Style
	statusOrphan    lipgloss.Style
	diffAdd         lipgloss.Style
	diffRemove      lipgloss.Style
	diffContext     lipgloss.Style
	confirmPrompt   lipgloss.Style
}

// palette is the named color set the theme is built from. Colors use 256-color
// codes so the look is stable across terminals; the accent (63, a soft indigo)
// matches the original skeleton title color so the visual identity is unbroken.
type palette struct {
	accent  lipgloss.Color // primary brand / selection / titles
	text    lipgloss.Color // default foreground for prominent content
	muted   lipgloss.Color // secondary text, metadata, hints
	info    lipgloss.Color // badges, links, agent labels
	success lipgloss.Color // created/added/success
	warning lipgloss.Color // updated/changed/caution
	danger  lipgloss.Color // errors/removed
}

// defaultPalette is the one palette the whole TUI is themed from.
func defaultPalette() palette {
	return palette{
		accent:  lipgloss.Color("63"),
		text:    lipgloss.Color("252"),
		muted:   lipgloss.Color("245"),
		info:    lipgloss.Color("39"),
		success: lipgloss.Color("42"),
		warning: lipgloss.Color("214"),
		danger:  lipgloss.Color("203"),
	}
}

// defaultTheme builds the shared theme from the default palette. Every token below
// is derived from a palette color — there are no literal color codes here, so the
// palette is genuinely the single knob for the application's look (R1).
func defaultTheme() theme {
	p := defaultPalette()

	return theme{
		palette: p,
		accent:  p.accent,

		title:   lipgloss.NewStyle().Bold(true).Foreground(p.accent),
		heading: lipgloss.NewStyle().Bold(true).Foreground(p.text),
		subtle:  lipgloss.NewStyle().Foreground(p.muted),
		breadcrumbActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.text),
		breadcrumbSep: lipgloss.NewStyle().Foreground(p.muted),

		listItem: lipgloss.NewStyle().
			Foreground(p.text).
			PaddingLeft(2),
		listItemSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.accent).
			PaddingLeft(0).
			SetString("> "),
		kindBadge: lipgloss.NewStyle().Foreground(p.info),

		loading: lipgloss.NewStyle().Foreground(p.muted).Italic(true),
		emptyState: lipgloss.NewStyle().
			Foreground(p.muted).
			Italic(true),
		emptyBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.muted).
			Padding(1, 2),
		errorText: lipgloss.NewStyle().Foreground(p.danger),
		errorBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.danger).
			Padding(1, 2),

		previewFrame: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.accent).
			Padding(0, 1),

		formTitle: lipgloss.NewStyle().Bold(true).Foreground(p.accent),

		dagNode: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.accent).
			Padding(0, 1),
		dagNodeID: lipgloss.NewStyle().Bold(true).Foreground(p.text),
		dagAgent:  lipgloss.NewStyle().Foreground(p.info),
		dagMeta:   lipgloss.NewStyle().Foreground(p.muted),
		dagEdge:   lipgloss.NewStyle().Foreground(p.accent),

		summaryBox: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.accent).
			Padding(0, 1),
		statusCreated:   lipgloss.NewStyle().Bold(true).Foreground(p.success),
		statusUpdated:   lipgloss.NewStyle().Bold(true).Foreground(p.warning),
		statusUnchanged: lipgloss.NewStyle().Foreground(p.muted),
		statusOrphan:    lipgloss.NewStyle().Foreground(p.muted).Italic(true),
		diffAdd:         lipgloss.NewStyle().Foreground(p.success),
		diffRemove:      lipgloss.NewStyle().Foreground(p.danger),
		diffContext:     lipgloss.NewStyle().Foreground(p.muted),
		confirmPrompt:   lipgloss.NewStyle().Bold(true).Foreground(p.accent),
	}
}

// box renders content inside a bordered style with EVERY line padded to the same
// visible width first, so the border closes evenly on multi-line content of
// unequal widths. Lipgloss measures each line independently when a style has no
// fixed width, which leaves the right border ragged on a block whose lines differ
// in length (and worse when ANSI color sequences throw off naive measurement).
// Padding to a common visible width up front — measured with lipgloss.Width, which
// ignores color codes — makes the frame a clean rectangle (fixes the 07-01
// errorBox alignment finding). It deliberately pads rather than setting Style.Width
// so the style never re-wraps the already-laid-out content.
func (t theme) box(style lipgloss.Style, content string) string {
	lines := strings.Split(content, "\n")
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
	return style.Render(strings.Join(lines, "\n"))
}

// markdownStyle returns the Glamour style config derived from the theme palette so
// rendered markdown (headings, lists, tables, code, emphasis) matches the rest of
// the TUI instead of Glamour's stock palette (R3/R4). It starts from Glamour's
// well-tuned dark base — which already handles tables, code blocks, lists and the
// rest readably — and overrides ONLY the color-bearing elements with palette
// colors, so the result stays correct as Glamour evolves while looking like
// Daedalus. The H1 background block (Glamour's wide highlighted banner) is the
// element the 07-01 review flagged as a "wide colored block"; here it is recolored
// to the palette and, crucially, the renderer is width-bounded at call sites
// (see markdown.go) so it never overflows the viewport.
func (t theme) markdownStyle() ansi.StyleConfig {
	accent := string(t.palette.accent)
	text := string(t.palette.text)
	info := string(t.palette.info)

	s := styles.DarkStyleConfig

	// Drop the document's default left margin so rendered content fits within the
	// requested wrap width (the dark base indents every line by 2, which pushed the
	// output to wrap+2 and produced the overflowing "wide block" the 07-01 review
	// flagged). With a zero margin the visible width equals the wrap we ask for, so
	// content sits cleanly inside the viewport frame.
	zero := uint(0)
	s.Document.Margin = &zero

	// Headings and emphasis carry the brand accent.
	s.Heading.Color = strptr(accent)
	s.H1.Color = strptr(text)
	s.H1.BackgroundColor = strptr(accent)
	s.H2.Color = strptr(accent)
	s.H3.Color = strptr(accent)
	s.H4.Color = strptr(accent)
	s.H5.Color = strptr(accent)
	s.H6.Color = strptr(accent)
	s.Strong.Color = strptr(text)
	s.Enumeration.Color = strptr(accent)

	// Links and inline code use the info color so they stand out without clashing.
	s.Link.Color = strptr(info)
	s.LinkText.Color = strptr(info)
	s.Code.Color = strptr(accent)

	return s
}

// formTheme returns the Huh theme derived from the palette so embedded forms match
// the rest of the TUI (R5). It starts from Huh's Charm base — a well-balanced
// layout — and recolors the focused field, the help, and the error text with
// palette colors so the form is unmistakably part of Daedalus.
func (t theme) formTheme() *huh.Theme {
	h := huh.ThemeBase()

	h.Focused.Title = h.Focused.Title.Foreground(t.palette.accent).Bold(true)
	h.Focused.Description = h.Focused.Description.Foreground(t.palette.muted)
	h.Focused.TextInput.Cursor = h.Focused.TextInput.Cursor.Foreground(t.palette.accent)
	h.Focused.TextInput.Prompt = h.Focused.TextInput.Prompt.Foreground(t.palette.accent)
	h.Focused.TextInput.Text = h.Focused.TextInput.Text.Foreground(t.palette.text)
	h.Focused.TextInput.Placeholder = h.Focused.TextInput.Placeholder.Foreground(t.palette.muted)
	h.Focused.ErrorIndicator = h.Focused.ErrorIndicator.Foreground(t.palette.danger)
	h.Focused.ErrorMessage = h.Focused.ErrorMessage.Foreground(t.palette.danger)
	h.Focused.SelectSelector = h.Focused.SelectSelector.Foreground(t.palette.accent)
	h.Focused.SelectedOption = h.Focused.SelectedOption.Foreground(t.palette.accent)
	h.Help.ShortKey = h.Help.ShortKey.Foreground(t.palette.muted)
	h.Help.ShortDesc = h.Help.ShortDesc.Foreground(t.palette.muted)

	// Mirror the focused recoloring onto the blurred state so an unfocused field is
	// muted but still on-palette.
	h.Blurred = h.Focused
	h.Blurred.Title = h.Blurred.Title.Foreground(t.palette.muted)
	h.Blurred.TextInput.Prompt = h.Blurred.TextInput.Prompt.Foreground(t.palette.muted)

	return h
}

// DefaultFormTheme is the package-level Huh theme used by any standalone form (kept
// for API stability with earlier epics). It returns the themed form look.
func DefaultFormTheme() *huh.Theme {
	return defaultTheme().formTheme()
}

// strptr is a tiny helper for the *string color fields Glamour's ansi.StyleConfig
// uses for optional overrides.
func strptr(s string) *string { return &s }
