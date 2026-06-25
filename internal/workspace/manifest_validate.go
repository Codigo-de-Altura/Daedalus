package workspace

import (
	"fmt"
	"sort"
	"strings"
)

// The canonical manifest schema (RF-1.3 / RF-9.3).
//
// This is the single, legible source of truth for what makes a parsed Manifest
// *valid*, complementing parseManifest (manifest_read.go), which only enforces
// that the on-disk shape can be read at all. parseManifest already rejects a file
// that cannot be parsed into name/version/backends; this validator deepens that
// into the semantic rules a healthy manifest must satisfy, mirroring the actionable
// single-pass style of catalog.ValidateAgent and workflows.Validate. The rules are
// derived from the manifest the scaffolder writes (newManifest/renderManifest), so
// validation can never contradict what `daedalus init` emits.
//
//	Field         Severity  Rule
//	-----         --------  ----
//	name          error     non-empty (trimmed)
//	version       error     equals SchemaVersion (the schema this binary speaks)
//	backends      error     at least one; each non-empty, supported, and unique
//	conventions   error     each key non-empty and unique; each value non-empty
//	conventions   warning   keys match the canonical set (naming/markdown/yaml):
//	                        an unknown or a missing canonical key is advisory, not
//	                        a hard failure, so a team that extends conventions is
//	                        not blocked while still being told the manifest drifted
//
// It is backend-agnostic (R7): "supported backend" is answered by IsSupportedBackend
// against SupportedBackends (data), never by hardcoding a backend name. The function
// is pure — no I/O, no backend calls — and deterministic: findings are collected in
// a fixed field order and stable-sorted, so the same manifest always yields the same
// error text (R8).

// canonicalConventionKeys is the ordered set of convention keys a freshly
// scaffolded manifest carries (see newManifest). It is the reference the validator
// uses to advise on convention drift; it is intentionally a coherence check
// (warning), not a hard schema rule, so the manifest format can grow additively.
var canonicalConventionKeys = []string{"naming", "markdown", "yaml"}

// ManifestSeverity classifies a manifest finding. The set is closed so a caller
// can branch on it; it mirrors the error/warning split used across the validators.
type ManifestSeverity string

const (
	// ManifestError is a hard violation that makes the manifest invalid.
	ManifestError ManifestSeverity = "error"
	// ManifestWarning is an advisory finding that does not, by itself, make the
	// manifest invalid (e.g. a convention key that drifted from the canonical set).
	ManifestWarning ManifestSeverity = "warning"
)

// ManifestFinding is a single actionable manifest validation finding (R5): which
// field failed, at what severity, what was observed, and what was expected. The
// parts let a user fix the problem without guessing.
type ManifestFinding struct {
	// Field is the canonical field the finding is about ("name", "version",
	// "backends", or "conventions[<key>]").
	Field string
	// Severity is whether this is a hard violation or an advisory.
	Severity ManifestSeverity
	// Observed describes what the manifest actually contained.
	Observed string
	// Expected describes what the schema requires instead.
	Expected string
}

// Error renders one finding as a single actionable line, severity-prefixed and
// self-contained with both the observed and expected halves.
func (f ManifestFinding) Error() string {
	return fmt.Sprintf("[%s] %s: observed %s; expected %s", f.Severity, f.Field, f.Observed, f.Expected)
}

// ManifestValidationError aggregates every finding for a manifest: the validator
// reports all detectable problems in a single pass rather than stopping at the
// first, so a user fixes them in one cycle. It implements error so it flows through
// error-returning gates unchanged. A non-nil *ManifestValidationError always carries
// at least one ERROR-severity finding (advisories alone never produce one).
type ManifestValidationError struct {
	// Findings are the findings in stable, deterministic order (R8).
	Findings []ManifestFinding
}

// Error renders all findings, one per line, so the message is self-describing in
// the deterministic order ValidateManifest produced.
func (e *ManifestValidationError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "manifest is invalid (%d issue%s):", len(e.Findings), manifestPluralS(len(e.Findings)))
	for _, f := range e.Findings {
		b.WriteString("\n  - ")
		b.WriteString(f.Error())
	}
	return b.String()
}

// HasErrors reports whether any finding is an ERROR-severity violation. The
// validator only returns a non-nil error when this holds, but a caller that
// inspects the findings directly can use it too.
func (e *ManifestValidationError) HasErrors() bool {
	for _, f := range e.Findings {
		if f.Severity == ManifestError {
			return true
		}
	}
	return false
}

// manifestPluralS is a tiny local pluralizer for the aggregate message.
func manifestPluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// ValidateManifest checks a parsed Manifest against the canonical schema and
// returns nil when it has no findings, or a *ManifestValidationError listing every
// finding (R4/R5). A manifest with only advisory (warning) findings still returns a
// non-nil error so the caller can surface the advice, but HasErrors reports false —
// callers that gate on hard validity check that.
//
// Findings are collected in fixed field order (name, version, backends, then
// conventions in declaration order) and stable-sorted, so the result is fully
// deterministic (R8). The function performs no I/O and references no concrete
// backend (R7).
func ValidateManifest(m Manifest) *ManifestValidationError {
	var findings []ManifestFinding

	// name: required, non-empty.
	if strings.TrimSpace(m.Name) == "" {
		findings = append(findings, ManifestFinding{
			Field:    "name",
			Severity: ManifestError,
			Observed: "empty",
			Expected: "a non-empty project name",
		})
	}

	// version: required, must equal the schema version this binary speaks.
	switch {
	case strings.TrimSpace(m.Version) == "":
		findings = append(findings, ManifestFinding{
			Field:    "version",
			Severity: ManifestError,
			Observed: "empty",
			Expected: fmt.Sprintf("the workspace schema version %q", SchemaVersion),
		})
	case m.Version != SchemaVersion:
		findings = append(findings, ManifestFinding{
			Field:    "version",
			Severity: ManifestError,
			Observed: fmt.Sprintf("%q", m.Version),
			Expected: fmt.Sprintf("the workspace schema version %q", SchemaVersion),
		})
	}

	findings = append(findings, validateManifestBackends(m.Backends)...)
	findings = append(findings, validateManifestConventions(m.Conventions)...)

	if len(findings) == 0 {
		return nil
	}
	sortManifestFindings(findings)
	return &ManifestValidationError{Findings: findings}
}

// validateManifestBackends checks the backends list: at least one, each non-empty,
// each supported, and no duplicates. "Supported" is decided by IsSupportedBackend
// against the data-driven SupportedBackends set, so the rule never hardcodes a
// backend name (R7).
func validateManifestBackends(backends []string) []ManifestFinding {
	var findings []ManifestFinding
	if len(backends) == 0 {
		return []ManifestFinding{{
			Field:    "backends",
			Severity: ManifestError,
			Observed: "empty",
			Expected: fmt.Sprintf("at least one supported backend (%s)", strings.Join(SupportedBackends, ", ")),
		}}
	}

	seen := make(map[string]int, len(backends))
	for i, raw := range backends {
		name := strings.TrimSpace(raw)
		field := fmt.Sprintf("backends[%d]", i)
		if name == "" {
			findings = append(findings, ManifestFinding{
				Field:    field,
				Severity: ManifestError,
				Observed: "empty",
				Expected: "a non-empty backend name",
			})
			continue
		}
		if first, dup := seen[name]; dup {
			findings = append(findings, ManifestFinding{
				Field:    fmt.Sprintf("backends[%s]", name),
				Severity: ManifestError,
				Observed: fmt.Sprintf("duplicate backend (first declared at index %d)", first),
				Expected: "each backend listed at most once",
			})
		} else {
			seen[name] = i
		}
		if !IsSupportedBackend(name) {
			findings = append(findings, ManifestFinding{
				Field:    fmt.Sprintf("backends[%s]", name),
				Severity: ManifestError,
				Observed: fmt.Sprintf("unsupported backend %q", name),
				Expected: fmt.Sprintf("one of the supported backends: %s", strings.Join(SupportedBackends, ", ")),
			})
		}
	}
	return findings
}

// validateManifestConventions checks the conventions block: each key non-empty and
// unique with a non-empty value (errors), and advises (warnings) when a key is not
// in the canonical set or a canonical key is missing — coherence guidance that does
// not block a team extending its conventions.
func validateManifestConventions(conventions []convention) []ManifestFinding {
	var findings []ManifestFinding

	seen := make(map[string]int, len(conventions))
	known := make(map[string]struct{}, len(canonicalConventionKeys))
	for _, k := range canonicalConventionKeys {
		known[k] = struct{}{}
	}

	for i, c := range conventions {
		key := strings.TrimSpace(c.Key)
		if key == "" {
			findings = append(findings, ManifestFinding{
				Field:    fmt.Sprintf("conventions[%d]", i),
				Severity: ManifestError,
				Observed: "empty key",
				Expected: "a non-empty convention key",
			})
			continue
		}
		field := fmt.Sprintf("conventions[%s]", key)
		if first, dup := seen[key]; dup {
			findings = append(findings, ManifestFinding{
				Field:    field,
				Severity: ManifestError,
				Observed: fmt.Sprintf("duplicate key (first declared at index %d)", first),
				Expected: "each convention key to be unique",
			})
		} else {
			seen[key] = i
		}
		if strings.TrimSpace(c.Value) == "" {
			findings = append(findings, ManifestFinding{
				Field:    field,
				Severity: ManifestError,
				Observed: "empty value",
				Expected: "a non-empty convention value",
			})
		}
		if _, ok := known[key]; !ok {
			findings = append(findings, ManifestFinding{
				Field:    field,
				Severity: ManifestWarning,
				Observed: fmt.Sprintf("unknown convention key %q", key),
				Expected: fmt.Sprintf("a canonical convention key (%s)", strings.Join(canonicalConventionKeys, ", ")),
			})
		}
	}

	// Advise when a canonical convention key is absent, so a manifest that drifted
	// from the scaffolded set is surfaced (a warning, never a hard failure).
	for _, k := range canonicalConventionKeys {
		if _, ok := seen[k]; !ok {
			findings = append(findings, ManifestFinding{
				Field:    fmt.Sprintf("conventions[%s]", k),
				Severity: ManifestWarning,
				Observed: "missing",
				Expected: fmt.Sprintf("the canonical convention %q to be present", k),
			})
		}
	}
	return findings
}

// sortManifestFindings imposes a fully deterministic order on the findings (R8),
// independent of collection order: by a fixed field-group rank (name, version,
// backends, conventions), then by the full field string, then by severity, then by
// observed text as a final tie-break.
func sortManifestFindings(findings []ManifestFinding) {
	sort.SliceStable(findings, func(i, j int) bool {
		ri, rj := manifestFieldRank(findings[i].Field), manifestFieldRank(findings[j].Field)
		if ri != rj {
			return ri < rj
		}
		if findings[i].Field != findings[j].Field {
			return findings[i].Field < findings[j].Field
		}
		if findings[i].Severity != findings[j].Severity {
			return findings[i].Severity == ManifestError // errors before warnings
		}
		return findings[i].Observed < findings[j].Observed
	})
}

// manifestFieldRank maps a field to its schema-order rank so findings read in the
// order a reader expects (name, version, backends, conventions) rather than
// alphabetically.
func manifestFieldRank(field string) int {
	switch {
	case field == "name":
		return 0
	case field == "version":
		return 1
	case strings.HasPrefix(field, "backends"):
		return 2
	case strings.HasPrefix(field, "conventions"):
		return 3
	default:
		return 4
	}
}
