package tui

import "strings"

// This file owns the line-level diff the build preview renders. The compile core
// hands the presentation layer the raw Current and Target content of every
// artifact (compile.PlannedArtifact); turning that pair into a readable list of
// added/removed/context lines is a PRESENTATION concern, so it lives here and not
// in the core. The algorithm is a classic longest-common-subsequence (LCS) line
// diff: deterministic (the same inputs always yield the same hunks), dependency
// free, and good enough for the small text artifacts Daedalus compiles.

// diffOp classifies one line of a rendered diff.
type diffOp int

const (
	// diffContext: the line is identical in Current and Target (shown for context).
	diffContext diffOp = iota
	// diffAdd: the line exists only in Target (a "+" line — new content).
	diffAdd
	// diffRemove: the line exists only in Current (a "-" line — removed content).
	diffRemove
)

// diffLine is one rendered diff row: an operation and the line's text (without a
// trailing newline). The View layer maps the op to a color and a +/-/ prefix so
// the styling stays in one place (the theme) rather than baked into the data.
type diffLine struct {
	op   diffOp
	text string
}

// lineDiff computes a deterministic line-level diff from current → target. The
// result reads top to bottom like a unified diff without hunks: removed lines
// (only in current) carry diffRemove, added lines (only in target) carry diffAdd,
// and shared lines carry diffContext. It is pure and allocation-bounded so the
// View can call it directly while building the diff body.
func lineDiff(current, target string) []diffLine {
	a := splitLines(current)
	b := splitLines(target)

	// lcs[i][j] = length of the longest common subsequence of a[i:] and b[j:].
	// Building it from the end lets us walk forward to emit a stable diff.
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	var out []diffLine
	i, j := 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			out = append(out, diffLine{op: diffContext, text: a[i]})
			i++
			j++
		case lcs[i+1][j] >= lcs[i][j+1]:
			out = append(out, diffLine{op: diffRemove, text: a[i]})
			i++
		default:
			out = append(out, diffLine{op: diffAdd, text: b[j]})
			j++
		}
	}
	for ; i < len(a); i++ {
		out = append(out, diffLine{op: diffRemove, text: a[i]})
	}
	for ; j < len(b); j++ {
		out = append(out, diffLine{op: diffAdd, text: b[j]})
	}
	return out
}

// splitLines splits text into lines for diffing, dropping a single trailing empty
// element so a file that ends in "\n" does not produce a phantom blank last line
// (which would otherwise show as a spurious diff). An empty input yields no lines,
// so a created-from-nothing artifact diffs as all-added.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	return lines
}
