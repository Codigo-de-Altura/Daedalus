package linters

import (
	"fmt"
	"strings"
)

// Report is the result of linting a workspace's definitions (R1/R8): the ordered
// list of findings plus convenience predicates. A clean workspace yields no
// error-level findings (it may still carry warning-level advisories). The findings
// are in deterministic order (errors first, then by family, location, spot, rule).
type Report struct {
	// Findings are every linter finding detected, in deterministic order.
	Findings []Finding
}

// HasErrors reports whether any finding is a hard violation (SeverityError). This
// is what a validate command keys its non-zero exit code on: advisories (warnings)
// do not make linting fail.
func (r *Report) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Counts returns the number of error- and warning-level findings, for a summary
// line.
func (r *Report) Counts() (errors, warnings int) {
	for _, f := range r.Findings {
		switch f.Severity {
		case SeverityError:
			errors++
		case SeverityWarning:
			warnings++
		}
	}
	return errors, warnings
}

// Clean reports whether the workspace has no hard violations (no error findings).
// It is the inverse of HasErrors, named for the message a caller prints.
func (r *Report) Clean() bool {
	return !r.HasErrors()
}

// String renders all findings, one per line, in the deterministic order Lint
// produced, so the message is byte-stable for a given workspace (R8). It is safe to
// call on a clean report (it says the definitions are valid).
func (r *Report) String() string {
	var b strings.Builder
	errs, warns := r.Counts()
	if len(r.Findings) == 0 {
		b.WriteString("definitions are valid")
		return b.String()
	}
	fmt.Fprintf(&b, "definitions have %d error%s and %d warning%s:",
		errs, pluralS(errs), warns, pluralS(warns))
	for _, f := range r.Findings {
		b.WriteString("\n  - ")
		b.WriteString(f.Error())
	}
	return b.String()
}

// pluralS is a tiny local pluralizer for the summary line.
func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
