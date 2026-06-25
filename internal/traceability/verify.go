package traceability

import (
	"fmt"
	"strings"
)

// Report is the result of verifying the traceability chain (R2/R3/R6): the ordered list
// of findings plus convenience predicates. A consistent workspace yields no error-level
// findings (it may still carry warning-level traceability gaps). It implements error so
// an inconsistent report can flow through error-returning gates, but callers typically
// inspect Findings, HasErrors and the counts directly.
type Report struct {
	// Findings are every traceability problem detected, in deterministic order (R6/CA6):
	// errors before warnings, then by kind, subject and observed value.
	Findings []Finding
}

// HasErrors reports whether any finding is a hard inconsistency (SeverityError). This is
// what a `verify` command keys its non-zero exit code on: soft gaps (warnings) do NOT
// make verification fail, honoring 05-03's optional-origin decision.
func (r *Report) HasErrors() bool {
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Counts returns the number of error- and warning-level findings, for a summary line.
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

// Consistent reports whether the chain has no hard inconsistencies (no error findings).
// It is the inverse of HasErrors, named for the message a caller prints.
func (r *Report) Consistent() bool {
	return !r.HasErrors()
}

// Error renders all findings, one per line, in the deterministic order Verify produced,
// so the message is byte-stable for a given workspace (R6/CA6). It is safe to call on a
// clean report (it says the chain is consistent).
func (r *Report) Error() string {
	var b strings.Builder
	errs, warns := r.Counts()
	if len(r.Findings) == 0 {
		b.WriteString("traceability chain is consistent")
		return b.String()
	}
	fmt.Fprintf(&b, "traceability chain has %d error%s and %d warning%s:",
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

// Verify checks the whole chain over the Graph and returns an ordered Report (R2/R3/R6).
// It is pure (no I/O — the Graph was already read by Build) and deterministic: it walks
// artifacts in sorted id order, collects findings, and stable-sorts them. The checks
// implement the severity split documented in finding.go:
//
//   - Architecture `spec:` that does not resolve  -> broken-link (error).
//   - Epic `spec:`/`architecture:` that does not resolve -> broken-link (error).
//   - Ticket `epic:` that does not resolve -> orphan-ticket (error).
//   - Ticket `spec:`/`architecture:` that does not resolve -> broken-link (error).
//   - Epic/ticket with NO origin link at all -> missing-origin (warning).
//
// A present-but-dangling reference is always an error (the link is broken); an ABSENT
// link is a warning (a gap, legal per 05-03). The two are never conflated.
func (g *Graph) Verify() *Report {
	report := &Report{}

	report.Findings = append(report.Findings, g.verifyArchitectures()...)
	report.Findings = append(report.Findings, g.verifyEpics()...)
	report.Findings = append(report.Findings, g.verifyTickets()...)

	sortFindings(report.Findings)
	return report
}

// verifyArchitectures checks each architecture document's recorded origin spec resolves.
// An architecture doc with no spec recorded is a missing-origin gap (warning); a recorded
// spec that does not exist is a broken-link error. Walks in sorted slug order.
func (g *Graph) verifyArchitectures() []Finding {
	var findings []Finding
	for _, slug := range g.archSlugs {
		a := g.Architectures[slug]
		if a.SpecRef == "" {
			findings = append(findings, Finding{
				Subject:  a.Slug,
				Severity: SeverityWarning,
				Kind:     KindMissingOrigin,
				Observed: "",
				Reason:   "architecture document records no origin spec; link it with a spec to complete the trace (optional, but recommended)",
			})
			continue
		}
		if _, ok := g.Specs[a.SpecRef]; !ok {
			findings = append(findings, Finding{
				Subject:  a.Slug,
				Severity: SeverityError,
				Kind:     KindBrokenLink,
				Observed: a.SpecRef,
				Reason:   "architecture document references a spec that does not exist; create the spec or correct the reference",
			})
		}
	}
	return findings
}

// verifyEpics checks each epic's origin links. An epic with NEITHER a spec NOR an
// architecture recorded is a missing-origin gap (warning). Any recorded spec/architecture
// that does not resolve is a broken-link error. Walks in sorted id order.
func (g *Graph) verifyEpics() []Finding {
	var findings []Finding
	for _, id := range g.epicIDs {
		e := g.Epics[id]

		if e.SpecRef == "" && e.ArchRef == "" {
			findings = append(findings, Finding{
				Subject:  e.ID,
				Severity: SeverityWarning,
				Kind:     KindMissingOrigin,
				Observed: "",
				Reason:   "epic records no origin spec or architecture; link one to complete the trace (optional per the backlog model, but recommended)",
			})
		}
		if e.SpecRef != "" {
			if _, ok := g.Specs[e.SpecRef]; !ok {
				findings = append(findings, Finding{
					Subject:  e.ID,
					Severity: SeverityError,
					Kind:     KindBrokenLink,
					Observed: e.SpecRef,
					Reason:   "epic references a spec that does not exist; create the spec or correct the reference",
				})
			}
		}
		if e.ArchRef != "" {
			if _, ok := g.Architectures[e.ArchRef]; !ok {
				findings = append(findings, Finding{
					Subject:  e.ID,
					Severity: SeverityError,
					Kind:     KindBrokenLink,
					Observed: e.ArchRef,
					Reason:   "epic references an architecture document that does not exist; create the document or correct the reference",
				})
			}
		}
	}
	return findings
}

// verifyTickets checks each ticket's parent epic exists (orphan-ticket error if not) and
// that any recorded origin links resolve (broken-link error if not). A ticket that
// records no origin of its own is NOT flagged as a gap here: it legitimately inherits its
// origin from its epic (the ascending walk resolves that), so flagging it would be a
// false positive — the epic-level missing-origin warning already covers the chain's gap.
// Walks in sorted id order.
func (g *Graph) verifyTickets() []Finding {
	var findings []Finding
	for _, id := range g.ticketIDs {
		t := g.Tickets[id]

		if _, ok := g.Epics[t.EpicID]; !ok {
			findings = append(findings, Finding{
				Subject:  t.ID,
				Severity: SeverityError,
				Kind:     KindOrphanTicket,
				Observed: t.EpicID,
				Reason:   "ticket references a parent epic that does not exist; the epic was removed or the reference is wrong — restore the epic or correct the reference",
			})
		}
		if t.SpecRef != "" {
			if _, ok := g.Specs[t.SpecRef]; !ok {
				findings = append(findings, Finding{
					Subject:  t.ID,
					Severity: SeverityError,
					Kind:     KindBrokenLink,
					Observed: t.SpecRef,
					Reason:   "ticket references a spec that does not exist; create the spec or correct the reference",
				})
			}
		}
		if t.ArchRef != "" {
			if _, ok := g.Architectures[t.ArchRef]; !ok {
				findings = append(findings, Finding{
					Subject:  t.ID,
					Severity: SeverityError,
					Kind:     KindBrokenLink,
					Observed: t.ArchRef,
					Reason:   "ticket references an architecture document that does not exist; create the document or correct the reference",
				})
			}
		}
	}
	return findings
}
