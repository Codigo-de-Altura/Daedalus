package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// form.go is the reusable form layer (ticket-07-02, R5/R6/R7). It wraps a Huh form
// in a small component the shell can drive through Update/View, applies the shared
// theme so forms look like the rest of Daedalus, and reports a clear lifecycle
// (submitted with a valid value, or cancelled) so the caller can act and return to
// navigation WITHOUT a dead end (R7/Check-6).
//
// It is presentation only: a form captures and validates input, it never reads or
// writes the workspace. The one flow wired in this ticket — a list filter — operates
// purely on items already in memory, so it needs no new core seam (the deliberate
// choice from the ticket brief). Domain-mutating forms (create/edit) come later,
// each behind a core write interface Obi-Wan owns.

// formField identifies a reusable field kind the component knows how to build. The
// ticket asks for text, selection and confirmation; the filter flow uses text, and
// the select/confirm builders are provided so later flows reuse the same component
// rather than reinventing forms.
//
// (select and confirm builders live in newSelectForm / newConfirmForm below.)

// formComponent is a themed, navigable wrapper around a huh.Form. It owns the
// form's lifecycle and the title drawn above it. It deliberately renders NO help of
// its own: the shell draws a single contextual help footer (from the central
// registry, formHelp context) for forms just like every other screen, so there is
// one help source and no duplicated footer (07-03 consolidation of the 07-02 double
// help line). The bound value is read back from the form on submit (StringValue).
type formComponent struct {
	theme theme
	form  *huh.Form
	title string
	// valueKey is the Huh field key the caller reads on submit via StringValue. We
	// read the value back from the form (not a bound *string) because the model is a
	// value type copied on every Update — a pointer into a model field would go stale
	// across copies, so the form-owned value is the single reliable source.
	valueKey string
}

// StringValue returns the current value of the form's primary field, read from the
// form itself so it is always the live value regardless of model copying.
func (c formComponent) StringValue() string {
	return c.form.GetString(c.valueKey)
}

// formResultKind is the outcome of running a form, so the caller can branch without
// reaching into Huh's state enum.
type formResultKind int

const (
	// formPending: the form is still being edited.
	formPending formResultKind = iota
	// formSubmitted: the user submitted a valid form (validation passed).
	formSubmitted
	// formCancelled: the user cancelled (esc/ctrl+c); nothing should be applied.
	formCancelled
)

// newFormComponent builds a themed form component from an already-constructed
// huh.Form. It wires the shared form theme and a keymap where esc cancels (matching
// actionFormCancel in the central registry) so cancelling a form uses the same "esc
// goes back" muscle memory as the rest of the shell. Huh's built-in help is turned
// OFF (WithShowHelp(false)) because the shell now draws the single contextual help
// footer for forms from the central registry — leaving Huh's on would reproduce the
// 07-02 double help line. Validation errors stay on so they render in-place.
func newFormComponent(th theme, form *huh.Form, title, valueKey string) formComponent {
	km := huh.NewDefaultKeyMap()
	// esc cancels the form everywhere; ctrl+c stays as the hard quit. These mirror
	// actionFormCancel / actionQuit so the form's real keys match what the shell's
	// formHelp context announces (announced == real).
	km.Quit = key.NewBinding(
		key.WithKeys("esc", "ctrl+c"),
		key.WithHelp("esc", "cancel"),
	)

	form = form.
		WithTheme(th.formTheme()).
		WithKeyMap(km).
		WithShowHelp(false).
		WithShowErrors(true)

	return formComponent{
		theme:    th,
		form:     form,
		title:    title,
		valueKey: valueKey,
	}
}

// Init starts the underlying form (focuses the first field, etc.).
func (c formComponent) Init() tea.Cmd {
	return c.form.Init()
}

// Update forwards a message to the form and reports the resulting lifecycle. The
// caller uses the returned kind to decide whether to apply the submitted value or
// simply return — the component itself applies nothing, keeping domain decisions
// out of the presentation wrapper.
func (c formComponent) Update(msg tea.Msg) (formComponent, formResultKind, tea.Cmd) {
	model, cmd := c.form.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		c.form = f
	}

	switch c.form.State {
	case huh.StateCompleted:
		return c, formSubmitted, cmd
	case huh.StateAborted:
		return c, formCancelled, cmd
	default:
		return c, formPending, cmd
	}
}

// View renders the form inside the themed chrome: a title and the form's own fields
// (which Huh draws with our theme). It deliberately renders NO help line — the shell
// adds the single contextual help footer (formHelp) below the form body, so help is
// unified across the TUI. When the form has quit Huh returns an empty body; the
// caller stops rendering the form route before that, but we guard anyway so a stray
// frame is never blank.
func (c formComponent) View() string {
	var b strings.Builder
	b.WriteString(c.theme.formTitle.Render(c.title))
	b.WriteString("\n\n")
	body := c.form.View()
	if strings.TrimSpace(body) == "" {
		body = c.theme.subtle.Render("…")
	}
	b.WriteString(body)
	return b.String()
}

// --- concrete reusable form builders ---------------------------------------

// newFilterForm builds the list-filter form: a single themed text input, bound to
// the given *string, that validates the query and, on submit, lets the caller
// filter an in-memory list. The validation rejects a whitespace-only query with a
// clear message (the testable invalid case for Check-5) while allowing an empty
// query to mean "clear the filter". It needs no core seam: it only shapes a string
// the caller matches against items already loaded in memory.
func newFilterForm(th theme, area, initial string) formComponent {
	value := initial
	input := huh.NewInput().
		Key("filter").
		Title(fmt.Sprintf("Filter %s", area)).
		Description("Type to narrow the list. Leave empty to show everything.").
		Placeholder("substring to match").
		Value(&value).
		Validate(validateFilterQuery)

	form := huh.NewForm(huh.NewGroup(input))
	return newFormComponent(th, form, fmt.Sprintf("Filter · %s", area), "filter")
}

// validateFilterQuery is the filter input's validation rule. An empty string is
// valid (clears the filter); a value that is only whitespace is invalid, because it
// can never be a meaningful filter and is almost always an accident — surfacing it
// is the clear-error path the validation exercises (Check-5). The message tells the
// user exactly what to do.
func validateFilterQuery(s string) error {
	if s == "" {
		return nil
	}
	if strings.TrimSpace(s) == "" {
		return fmt.Errorf("enter a non-blank filter, or clear the field to show all")
	}
	return nil
}

// newSelectForm builds a reusable single-choice selection form bound to the given
// *string, for later flows (e.g. choosing a backend or a kind). It is part of the
// reusable form toolkit this ticket delivers even though the wired flow uses the
// text input; keeping it here means future tickets reuse the themed component
// instead of constructing Huh forms ad hoc.
func newSelectForm(th theme, title string, options []string, value *string) formComponent {
	opts := make([]huh.Option[string], 0, len(options))
	for _, o := range options {
		opts = append(opts, huh.NewOption(o, o))
	}
	sel := huh.NewSelect[string]().
		Key("choice").
		Title(title).
		Options(opts...).
		Value(value)
	form := huh.NewForm(huh.NewGroup(sel))
	return newFormComponent(th, form, title, "choice")
}

// newConfirmForm builds a reusable yes/no confirmation form bound to the given
// *bool, for later flows that gate an action behind an explicit confirmation. Like
// newSelectForm it rounds out the reusable toolkit (R5) without being the wired
// flow.
func newConfirmForm(th theme, title, affirmative, negative string, value *bool) formComponent {
	confirm := huh.NewConfirm().
		Key("confirm").
		Title(title).
		Affirmative(affirmative).
		Negative(negative).
		Value(value)
	form := huh.NewForm(huh.NewGroup(confirm))
	return newFormComponent(th, form, title, "confirm")
}
