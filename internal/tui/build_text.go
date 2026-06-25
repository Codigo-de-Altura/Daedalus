package tui

import (
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Codigo-de-Altura/Daedalus/internal/compile"
)

// build_text.go is the NON-TTY rendering of the build plan/diff. The same plan the
// interactive preview shows is rendered as plain, deterministic text for the paths
// that have no terminal: `daedalus build --preview` piped/in CI, and `daedalus
// build` with neither a TTY nor --yes (which must NOT write). Keeping it next to
// the TUI lets both share the counting/labeling helpers so the two renderings can
// never disagree on classification.
//
// It writes nothing to the filesystem — it only formats a *compile.PlanResult — so
// every textual path is, by construction, a safe dry run.

// RenderPlanText writes a deterministic textual report of a build plan to w: a
// per-backend summary (created/updated/unchanged/orphans), then each changed
// artifact with a compact +/- content diff (created files shown as all-added,
// updated as a line diff). Unchanged artifacts are summarized in the counts and
// not diffed. It is the shared body of both non-TTY build paths; the caller adds
// the surrounding "nothing written" notice appropriate to its flag combination.
//
// It is pure and deterministic (plan order is the core's deterministic emission
// order), so it is straightforward to assert in tests and stable as a golden.
func RenderPlanText(w io.Writer, res *compile.PlanResult) {
	if res == nil {
		return
	}
	fmt.Fprintf(w, "Preview of compiling %s (no files written):\n", filepath.ToSlash(res.Root))

	if !planHasChanges(res) {
		fmt.Fprintln(w, "  No changes — every artifact is already up to date. Nothing to write.")
		return
	}

	for _, bp := range res.Backends {
		c, u, n := countStatuses(bp)
		fmt.Fprintf(w, "  %s: %d new, %d modified, %d unchanged (of %d artifact%s)\n",
			bp.Backend, c, u, n, len(bp.Artifacts), plural2(len(bp.Artifacts)))

		for _, a := range bp.Artifacts {
			switch a.Status {
			case compile.StatusCreated:
				fmt.Fprintf(w, "    [new]      %s\n", a.RelPath)
				writeDiffLinesText(w, createdDiff(a.Target))
			case compile.StatusUpdated:
				fmt.Fprintf(w, "    [modified] %s\n", a.RelPath)
				writeDiffLinesText(w, lineDiff(a.Current, a.Target))
			default:
				fmt.Fprintf(w, "    [unchanged] %s\n", a.RelPath)
			}
		}
		// Orphans: detected, reported, never deleted (safe default, Check-8).
		for _, o := range bp.Orphans {
			fmt.Fprintf(w, "    [orphan]   %s (no longer produced; left untouched)\n", o)
		}
	}
}

// writeDiffLinesText prints a diff as plain +/- prefixed lines, indented under its
// artifact. Context lines use two leading spaces so additions/removals stand out
// even without color (REQ-3, but in a no-color sink).
func writeDiffLinesText(w io.Writer, lines []diffLine) {
	for _, dl := range lines {
		switch dl.op {
		case diffAdd:
			fmt.Fprintf(w, "      + %s\n", dl.text)
		case diffRemove:
			fmt.Fprintf(w, "      - %s\n", dl.text)
		default:
			fmt.Fprintf(w, "        %s\n", dl.text)
		}
	}
}

// createdDiff turns a new file's content into an all-added diff so created files
// render with the same +/- vocabulary as updated ones.
func createdDiff(target string) []diffLine {
	lines := splitLines(target)
	out := make([]diffLine, 0, len(lines))
	for _, l := range lines {
		out = append(out, diffLine{op: diffAdd, text: l})
	}
	return out
}

// summaryText renders an Outcome's per-backend counts as plain text, used by the
// TUI's post-write screen so the confirmed result is reported the same way the CLI
// reports it.
func summaryText(out *compile.Outcome) string {
	var b strings.Builder
	for _, bo := range out.Backends {
		fmt.Fprintf(&b, "%s: %d new, %d modified, %d unchanged (of %d artifact%s)\n",
			bo.Backend, len(bo.Created), len(bo.Updated), len(bo.Unchanged),
			bo.Planned, plural2(bo.Planned))
		for _, o := range bo.Orphans {
			fmt.Fprintf(&b, "  [orphan] %s (left untouched)\n", o)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// countStatuses partitions a backend plan's artifacts into created/updated/
// unchanged counts. Shared by the TUI summary and the textual report so they can
// never drift.
func countStatuses(bp compile.BackendPlan) (created, updated, unchanged int) {
	for _, a := range bp.Artifacts {
		switch a.Status {
		case compile.StatusCreated:
			created++
		case compile.StatusUpdated:
			updated++
		default:
			unchanged++
		}
	}
	return created, updated, unchanged
}

// isWorkspaceMissing reports whether err is the "no .daedalus workspace" failure,
// so the preview can render the same actionable "run daedalus init" guidance the
// CLI does.
func isWorkspaceMissing(err error) bool {
	return errors.Is(err, compile.ErrWorkspaceNotFound)
}

// plural2 returns the plural "s" suffix for n, kept local so the tui package does
// not reach into the command package's helper.
func plural2(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
