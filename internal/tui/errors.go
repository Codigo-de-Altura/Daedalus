package tui

import (
	"errors"
	"fmt"

	"github.com/Codigo-de-Altura/Daedalus/internal/prompts"
	"github.com/Codigo-de-Altura/Daedalus/internal/workflows"
)

// composeErrorMessage turns a prompt resolution failure into a human-readable,
// actionable message for the preview's error state (R7). It distinguishes the
// two composition failures the core reports as typed errors — a missing
// include reference and an inclusion cycle — and names the offending ids, so the
// user knows exactly what to fix instead of seeing broken or corrupt content.
//
// It branches with errors.As/errors.Is on the core's typed errors rather than on
// message text, so it stays correct if the core rewords its messages. Any other
// failure (e.g. a malformed prompt file, or the prompt not existing) falls back
// to a clear generic line so the preview never shows a raw, confusing dump.
func composeErrorMessage(id string, err error) string {
	if err == nil {
		return ""
	}

	var cycle *prompts.IncludeCycleError
	if errors.As(err, &cycle) {
		return fmt.Sprintf(
			"Cannot compose %q: inclusion cycle detected.\n\n%s\n\nA prompt cannot include itself, directly or through a chain. "+
				"Break the loop in one of the listed prompts.",
			id, joinChain(cycle.Chain),
		)
	}

	var notFound *prompts.IncludeNotFoundError
	if errors.As(err, &notFound) {
		return fmt.Sprintf(
			"Cannot compose %q: missing include reference.\n\nPrompt %q includes %q, but no such prompt exists.\n\n"+
				"Create the missing prompt or fix the {{include: ...}} reference.",
			id, notFound.ReferencedBy, notFound.MissingID,
		)
	}

	if errors.Is(err, prompts.ErrPromptNotFound) {
		return fmt.Sprintf("Cannot open %q: the prompt no longer exists.", id)
	}

	// Fallback: surface the error plainly rather than rendering broken content.
	return fmt.Sprintf("Cannot compose %q: %v", id, err)
}

// workflowLoadErrorMessage turns a workflow load failure into a human-readable,
// actionable message for the DAG view's error state (R7/CA7). It distinguishes the
// two typed load failures the core reports — a workflow that does not exist and a
// malformed (unparseable) file — so the user knows whether to create the workflow
// or fix its YAML, instead of seeing a raw error. It branches with errors.Is on
// the core's sentinels rather than on message text, so it stays correct if the
// core rewords its messages. Any other failure falls back to a clear generic line.
func workflowLoadErrorMessage(name string, err error) string {
	if err == nil {
		return ""
	}

	if errors.Is(err, workflows.ErrWorkflowNotFound) {
		return fmt.Sprintf("Cannot open %q: the workflow no longer exists.", name)
	}

	if errors.Is(err, workflows.ErrMalformedWorkflow) {
		return fmt.Sprintf(
			"Cannot read %q: the workflow file is malformed.\n\n%v\n\n"+
				"Fix the YAML in .daedalus/workflows/%s.yaml and try again.",
			name, err, name)
	}

	// Fallback: surface the error plainly rather than rendering broken content.
	return fmt.Sprintf("Cannot load %q: %v", name, err)
}

// joinChain renders an inclusion cycle chain (e.g. ["a","b","a"]) as an arrowed
// path so the loop is visible at a glance.
func joinChain(chain []string) string {
	out := ""
	for i, id := range chain {
		if i > 0 {
			out += " -> "
		}
		out += id
	}
	return out
}
