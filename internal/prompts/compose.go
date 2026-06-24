package prompts

import (
	"errors"
	"fmt"
	"strings"
)

// Prompt composition / inclusion (R1..R9).
//
// A prompt body may pull in another prompt's content so a shared fragment
// (glossary, style, role definition) lives in exactly one file and is reused by
// reference rather than copied (DRY, R7). Resolve expands those references into a
// single composed text, recursively and deterministically, without ever touching
// the source files (R8).
//
// # Inclusion syntax (R1/R9)
//
// An inclusion is a directive on its OWN line that, after trimming surrounding
// whitespace, reads exactly:
//
//	{{include: <id>}}
//
//   - The directive must be the only content on its line. A line that carries
//     other text alongside the `{{include: ...}}` token is NOT a directive — it is
//     left verbatim — so the syntax is unambiguous and a body can mention the
//     literal token in prose by surrounding it with other text.
//   - <id> is trimmed of surrounding whitespace and must be a valid kebab-case
//     prompt id (the same id used as the file base name). An id that is not
//     kebab-case is reported as a not-found reference (it can never name a real
//     prompt), naming the offending token.
//   - Leading/trailing whitespace and indentation on the directive line are not
//     significant: the whole line is replaced by the included content, so the
//     included fragment is never indented under the directive's column.
//
// # Whitespace / newline semantics (R4 determinism)
//
// Resolution is line-oriented and byte-stable for a given set of prompts:
//
//   - Each prompt's body is taken verbatim (as persisted) and split into lines.
//   - A directive line is replaced, in place, by the fully-resolved body of the
//     referenced prompt. The included body is itself resolved first (recursion,
//     R3), and its own trailing newlines are trimmed so an inclusion never injects
//     blank padding around itself; the surrounding lines of the including body are
//     preserved exactly. The composed text is then re-joined with "\n".
//   - Non-directive lines are emitted unchanged. The result therefore depends only
//     on the bodies of the involved prompts, never on map iteration or filesystem
//     order, so resolving the same prompt twice yields byte-identical output (R4).
//
// # Errors (R5/R6)
//
// Resolution reports two composition failures as typed, actionable errors:
//
//   - *IncludeNotFoundError (sentinel ErrIncludeNotFound): a directive references
//     an id with no prompt file. The error names the missing id and the prompt
//     that referenced it.
//   - *IncludeCycleError (sentinel ErrIncludeCycle): a prompt is reached again
//     while already being resolved (A→B→A, or self-inclusion A→A). The error names
//     the full cycle chain. Detection uses the active resolution stack, so it never
//     loops infinitely or overflows the stack.

// ErrIncludeNotFound is the sentinel wrapped by *IncludeNotFoundError so callers
// (the CLI and the 03-03 TUI preview) can branch on a missing-reference failure
// via errors.Is without string matching.
var ErrIncludeNotFound = errors.New("included prompt not found")

// ErrIncludeCycle is the sentinel wrapped by *IncludeCycleError so callers can
// branch on a cycle failure distinctly from a missing reference.
var ErrIncludeCycle = errors.New("include cycle detected")

// IncludeNotFoundError reports a directive that references a prompt id with no
// file. It names both the missing id and the prompt whose body referenced it, so
// the user knows exactly where to look (R6).
type IncludeNotFoundError struct {
	// MissingID is the id the directive tried to include.
	MissingID string
	// ReferencedBy is the id of the prompt whose body held the directive.
	ReferencedBy string
}

func (e *IncludeNotFoundError) Error() string {
	return fmt.Sprintf("included prompt %q not found (referenced by %q)", e.MissingID, e.ReferencedBy)
}

// Unwrap exposes the sentinel for errors.Is.
func (e *IncludeNotFoundError) Unwrap() error { return ErrIncludeNotFound }

// IncludeCycleError reports an inclusion cycle. Chain is the resolution path that
// closes the loop, in order, with the repeated id appearing at both ends so the
// cycle is self-evident (e.g. ["a", "b", "a"]) (R5).
type IncludeCycleError struct {
	Chain []string
}

func (e *IncludeCycleError) Error() string {
	return fmt.Sprintf("include cycle detected: %s", strings.Join(e.Chain, " -> "))
}

// Unwrap exposes the sentinel for errors.Is.
func (e *IncludeCycleError) Unwrap() error { return ErrIncludeCycle }

// Resolve loads the prompt id from promptsRoot and returns its fully composed
// text: every inclusion directive recursively expanded into a single text, with
// deterministic, byte-stable output (R2/R3/R4). It never writes to disk and never
// mutates the source prompts (R8) — it only reads each referenced prompt's body
// and assembles the result in memory.
//
// It is the entry point the CLI's `render` verb and the 03-03 TUI preview consume.
// On a composition failure it returns a typed error (*IncludeNotFoundError or
// *IncludeCycleError, each wrapping its sentinel) so a UI can distinguish "missing
// reference" from "cycle" and show a precise message; a load/parse failure of the
// root or an included prompt surfaces the underlying prompts error
// (ErrPromptNotFound / ErrMalformedPrompt).
func Resolve(promptsRoot, id string) (string, error) {
	// stack tracks the active resolution path for cycle detection; seen mirrors it
	// as a set for O(1) membership tests. They are kept in lockstep by resolve.
	return resolve(promptsRoot, id, nil, map[string]bool{})
}

// resolve is the recursive worker. stack is the active inclusion path (root-first)
// used to build a cycle's chain; seen is the same ids as a set for fast lookup.
// The two are pushed/popped together around each recursion so a prompt included
// twice on *different* branches (a diamond: A includes B and C, both include D) is
// allowed and expanded each time (DRY is about disk, not about how many times a
// fragment appears in the composed output), while a prompt that includes itself
// transitively (a true cycle on the active path) is rejected.
func resolve(promptsRoot, id string, stack []string, seen map[string]bool) (string, error) {
	if seen[id] {
		// Re-entering an id already on the active path closes a cycle. Build the
		// chain from where it first appears so the message shows exactly the loop.
		chain := cycleChain(stack, id)
		return "", &IncludeCycleError{Chain: chain}
	}

	p, err := Load(promptsRoot, id)
	if err != nil {
		return "", err
	}

	seen[id] = true
	stack = append(stack, id)
	// Pop on the way out so sibling branches do not see this id as "on the path".
	defer func() {
		delete(seen, id)
	}()

	lines := strings.Split(p.Body, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		incID, ok := parseIncludeDirective(line)
		if !ok {
			out = append(out, line)
			continue
		}
		// A directive must name a real, kebab-case prompt; otherwise it can never
		// resolve, which is a not-found reference (R6) attributed to this prompt.
		if !IsKebabCase(incID) {
			return "", &IncludeNotFoundError{MissingID: incID, ReferencedBy: id}
		}
		included, err := resolve(promptsRoot, incID, stack, seen)
		if err != nil {
			// Translate an included prompt's absence into the richer, attributed
			// not-found error so the message names who referenced the missing id.
			var nf *IncludeNotFoundError
			if errors.Is(err, ErrPromptNotFound) && !errors.As(err, &nf) {
				return "", &IncludeNotFoundError{MissingID: incID, ReferencedBy: id}
			}
			return "", err
		}
		out = append(out, included)
	}

	// Trim trailing newlines from the composed body so an inclusion of this prompt
	// never injects blank padding around itself; the caller controls surrounding
	// whitespace. This keeps the output deterministic and tidy (R4).
	return strings.TrimRight(strings.Join(out, "\n"), "\n"), nil
}

// cycleChain builds the cycle path to report: the active stack from the first
// occurrence of id onward, with id appended again to close the loop visibly. If
// id is somehow not on the stack (defensive), the chain is just stack + id.
func cycleChain(stack []string, id string) []string {
	start := 0
	for i, s := range stack {
		if s == id {
			start = i
			break
		}
	}
	chain := make([]string, 0, len(stack)-start+1)
	chain = append(chain, stack[start:]...)
	chain = append(chain, id)
	return chain
}

// parseIncludeDirective reports whether line (after trimming) is an inclusion
// directive `{{include: <id>}}` and, if so, returns the trimmed <id>. A line that
// is not exactly a directive returns ("", false) and is treated as literal body
// text, so prose that merely mentions the token is never expanded. The id portion
// is trimmed but not otherwise validated here (the caller checks kebab-case).
func parseIncludeDirective(line string) (string, bool) {
	trimmed := strings.TrimSpace(line)
	const open = "{{include:"
	const close = "}}"
	if !strings.HasPrefix(trimmed, open) || !strings.HasSuffix(trimmed, close) {
		return "", false
	}
	inner := trimmed[len(open) : len(trimmed)-len(close)]
	id := strings.TrimSpace(inner)
	if id == "" {
		return "", false
	}
	return id, true
}
