package backlog

import "sort"

// Epic is the in-memory canonical model of an epic (R1). An epic describes an
// objective/scope/criteria and is the parent of a set of tickets. Its metadata is the
// stable set the ticket requires (R3/CA3): status, priority, dependencies and links to
// its originating artifacts (spec/architecture).
type Epic struct {
	// ID is the canonical epic id `epic-NN-<slug>` (R1/CA1). It is the folder name and
	// the unique key within the backlog.
	ID string
	// Title is the short, human-facing name of the epic. Never empty.
	Title string
	// Status and Priority are closed, validated metadata (R3/R6).
	Status   Status
	Priority Priority
	// SpecRef and ArchitectureRef are OPTIONAL links to the epic's originating spec and
	// architecture artifacts (R5/CA5). They are workspace-relative slugs (e.g.
	// "payments", "payments-arch"); empty when unlinked. At least conceptually an epic
	// derives from a spec/architecture, but R3 phrases links as metadata, so neither is
	// mandatory here — recording them is what matters; end-to-end verification is 05-04.
	SpecRef         string
	ArchitectureRef string
	// DependsOn is the explicit, consistent list of artifact ids this epic depends on
	// (R4/CA4): other epic ids (or ticket ids). It is stored as a list of ids, mirroring
	// the workflows package's DependsOn. Order is preserved as authored; duplicates are
	// removed at the edit/create boundary.
	DependsOn []string
	// Body is the epic's Markdown content (objective/scope/criteria). Persisted verbatim
	// (R6); seeded with a placeholder on create (R7).
	Body string
}

// Ticket is the in-memory canonical model of a ticket (R2). A ticket describes a
// feature (what/requirements/criteria) and always belongs to an epic.
type Ticket struct {
	// ID is the canonical ticket id `ticket-NN-MM-<slug>` (R2/CA2). It is the folder
	// name and the unique key within its epic.
	ID string
	// EpicID is the id of the parent epic (R5/CA5): every ticket references its epic.
	// Never empty; it is also the on-disk parent folder, so the link is both metadata
	// and structure.
	EpicID string
	// Title is the short, human-facing name of the ticket. Never empty.
	Title string
	// Status and Priority are closed, validated metadata (R3/R6).
	Status   Status
	Priority Priority
	// SpecRef and ArchitectureRef are OPTIONAL links to originating artifacts (R5/CA5),
	// as for an epic. A ticket commonly inherits these from its epic, but may record its
	// own; recording is what matters here (verification is 05-04).
	SpecRef         string
	ArchitectureRef string
	// DependsOn is the explicit list of artifact ids this ticket depends on (R4/CA4):
	// other ticket ids (or epic ids). Stored as a list, order preserved, deduplicated at
	// the boundary.
	DependsOn []string
	// Body is the ticket's Markdown content (the feature spec). Persisted verbatim (R6);
	// seeded with a placeholder on create (R7).
	Body string
}

// EpicEntry is a single epic listing row: the minimum a caller needs to present epics
// for selection without loading their bodies.
type EpicEntry struct {
	ID       string
	Title    string
	Status   Status
	Priority Priority
}

// TicketEntry is a single ticket listing row, including its parent epic so a flat
// listing across the backlog is unambiguous.
type TicketEntry struct {
	ID       string
	EpicID   string
	Title    string
	Status   Status
	Priority Priority
}

// dedupePreserveOrder returns ids with duplicates removed, keeping the first
// occurrence's position. Dependency lists must be consistent (R4/CA4): a repeated id
// is meaningless and would produce noisy diffs, so we collapse duplicates while
// preserving the authored order (we never silently re-sort the user's list, because
// the order can carry intent).
func dedupePreserveOrder(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]bool, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// sortEpicEntries orders epic entries by id so a listing is deterministic regardless
// of filesystem order (R6).
func sortEpicEntries(entries []EpicEntry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
}

// sortTicketEntries orders ticket entries by id so a listing is deterministic (R6).
func sortTicketEntries(entries []TicketEntry) {
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
}
