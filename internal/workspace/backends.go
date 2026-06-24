package workspace

import (
	"errors"
	"fmt"
	"strings"
)

// SupportedBackends is the ordered set of agent backends the MVP can target.
// It is the single source of truth for "which backends exist": validation,
// defaulting and error messages all derive from it, so adding a backend in the
// future (PRD RNF-7) is a one-line, additive change here — the seed of the
// eventual adapter registry. The first entry is the deterministic default
// (DefaultBackend); order is significant because it is the order surfaced in
// error messages and used when preserving the caller's selection.
var SupportedBackends = []string{DefaultBackend}

// ErrUnsupportedBackend is the sentinel returned (wrapped) by NormalizeBackends
// when a requested backend is not in SupportedBackends. The CLI distinguishes
// it from I/O errors via errors.Is so it can map an unsupported selection to a
// usage error (exit code 2) without touching the filesystem.
var ErrUnsupportedBackend = errors.New("unsupported backend")

// IsSupportedBackend reports whether name is a backend the MVP can target. It is
// the membership test behind NormalizeBackends and is exported so the CLI (or a
// future adapter layer) can ask the core directly rather than reimplementing the
// set — backend validity is domain knowledge, not CLI knowledge.
func IsSupportedBackend(name string) bool {
	for _, b := range SupportedBackends {
		if b == name {
			return true
		}
	}
	return false
}

// NormalizeBackends validates and canonicalizes a requested backend selection
// for persistence in the manifest. It is the single gate every selection passes
// through, so the rules live in one place:
//
//   - an empty selection (no flag / non-interactive default) resolves to the
//     deterministic default, []string{DefaultBackend} (R3/CA2);
//   - every entry must be a supported backend; the first unsupported value is
//     rejected with an error wrapping ErrUnsupportedBackend that names the bad
//     value and lists the supported set (R5/CA4);
//   - duplicates are dropped while preserving first-seen order, so the same
//     selection always yields the same canonical list (R7/CA6).
//
// It never mutates its input. On error it returns nil so a caller cannot
// accidentally persist a partially-validated slice.
func NormalizeBackends(sel []string) ([]string, error) {
	if len(sel) == 0 {
		// Default is returned as a fresh slice so callers can't alias (and later
		// mutate) the package-level default.
		return []string{DefaultBackend}, nil
	}

	seen := make(map[string]struct{}, len(sel))
	normalized := make([]string, 0, len(sel))
	for _, raw := range sel {
		name := strings.TrimSpace(raw)
		if !IsSupportedBackend(name) {
			return nil, fmt.Errorf("%w: %q (supported: %s)",
				ErrUnsupportedBackend, name, strings.Join(SupportedBackends, ", "))
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	return normalized, nil
}
