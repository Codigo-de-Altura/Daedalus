package workspace

import (
	"strings"
	"testing"
)

// validManifest is the manifest a fresh `daedalus init` writes: the reference a
// linter must accept without findings (CA6). Built through newManifest so the test
// can never drift from what the scaffolder emits.
func validManifest() Manifest {
	return newManifest("my-project", []string{DefaultBackend})
}

// TestValidateManifestAcceptsScaffolded covers CA6: the manifest the scaffolder
// writes is valid — no findings, no false positives.
func TestValidateManifestAcceptsScaffolded(t *testing.T) {
	if err := ValidateManifest(validManifest()); err != nil {
		t.Fatalf("scaffolded manifest reported invalid: %v", err)
	}
}

// TestValidateManifestMissingName covers CA5: an empty name is a hard error naming
// the field with an expectation.
func TestValidateManifestMissingName(t *testing.T) {
	m := validManifest()
	m.Name = ""

	err := ValidateManifest(m)
	if err == nil || !err.HasErrors() {
		t.Fatalf("empty name accepted: %v", err)
	}
	if !hasManifestField(err, "name") {
		t.Errorf("no finding for field name; got: %v", err)
	}
}

// TestValidateManifestWrongVersion covers CA5: a version other than the schema
// version is a hard error.
func TestValidateManifestWrongVersion(t *testing.T) {
	m := validManifest()
	m.Version = "999"

	err := ValidateManifest(m)
	if err == nil || !err.HasErrors() {
		t.Fatalf("wrong version accepted: %v", err)
	}
	if !hasManifestField(err, "version") {
		t.Errorf("no finding for field version; got: %v", err)
	}
}

// TestValidateManifestUnknownBackend covers CA5: an unsupported backend is a hard
// error, and the message lists the supported set without hardcoding it in the rule.
func TestValidateManifestUnknownBackend(t *testing.T) {
	m := validManifest()
	m.Backends = []string{"made-up-backend"}

	err := ValidateManifest(m)
	if err == nil || !err.HasErrors() {
		t.Fatalf("unknown backend accepted: %v", err)
	}
	if !strings.Contains(err.Error(), "made-up-backend") {
		t.Errorf("finding does not name the offending backend; got: %v", err)
	}
}

// TestValidateManifestNoBackends covers CA5: an empty backends list is a hard error
// (at least one backend is required).
func TestValidateManifestNoBackends(t *testing.T) {
	m := validManifest()
	m.Backends = nil

	err := ValidateManifest(m)
	if err == nil || !err.HasErrors() {
		t.Fatalf("empty backends accepted: %v", err)
	}
	if !hasManifestField(err, "backends") {
		t.Errorf("no finding for field backends; got: %v", err)
	}
}

// TestValidateManifestDuplicateBackend covers CA5: a backend listed twice is a hard
// error.
func TestValidateManifestDuplicateBackend(t *testing.T) {
	m := validManifest()
	m.Backends = []string{DefaultBackend, DefaultBackend}

	err := ValidateManifest(m)
	if err == nil || !err.HasErrors() {
		t.Fatalf("duplicate backend accepted: %v", err)
	}
}

// TestValidateManifestUnknownConventionIsWarning covers the coherence rule: an
// extra convention key is advisory (warning), not a hard failure — a team may
// extend conventions. A non-nil error with only warnings reports HasErrors()==false.
func TestValidateManifestUnknownConventionIsWarning(t *testing.T) {
	m := validManifest()
	m.Conventions = append(m.Conventions, convention{Key: "extra", Value: "something"})

	err := ValidateManifest(m)
	if err == nil {
		t.Fatal("expected an advisory finding for the unknown convention key, got none")
	}
	if err.HasErrors() {
		t.Errorf("an unknown convention key must be advisory, not a hard error: %v", err)
	}
}

// TestValidateManifestDeterministic covers CA8: validating the same invalid
// manifest twice yields identical, stably-ordered findings.
func TestValidateManifestDeterministic(t *testing.T) {
	m := validManifest()
	m.Name = ""
	m.Version = "x"
	m.Backends = []string{"bad", "bad"}

	first := ValidateManifest(m)
	second := ValidateManifest(m)
	if first == nil || second == nil {
		t.Fatal("expected findings on an invalid manifest")
	}
	if first.Error() != second.Error() {
		t.Errorf("non-deterministic findings:\n1: %s\n2: %s", first.Error(), second.Error())
	}
}

// hasManifestField reports whether any finding's field starts with the given prefix
// (so "backends" matches "backends[0]" too).
func hasManifestField(err *ManifestValidationError, prefix string) bool {
	for _, f := range err.Findings {
		if strings.HasPrefix(f.Field, prefix) {
			return true
		}
	}
	return false
}
