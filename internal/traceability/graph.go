package traceability

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Codigo-de-Altura/Daedalus/internal/architecture"
	"github.com/Codigo-de-Altura/Daedalus/internal/backlog"
	"github.com/Codigo-de-Altura/Daedalus/internal/specs"
)

// Graph is the consolidated, navigable view of the SDD chain assembled from the
// workspace's artifacts (R1/CA1/CA2). It holds the resolved sets of specs, architecture
// documents, epics and tickets, plus the cross-references between them — all READ from
// the existing artifacts (R4/CA5), never duplicated or re-authored.
//
// Navigation is provided in both directions:
//
//   - Descending: SpecChain (spec -> its epics -> their tickets).
//   - Ascending:  TicketChain (ticket -> its epic -> origin spec/architecture).
//
// The Graph is a pure data structure built by Build; it performs no I/O after
// construction and is safe to query repeatedly.
type Graph struct {
	// Specs are the spec slugs that have a materialized spec (`<slug>.md`), keyed by
	// slug. A brief without a spec is not a node in the traceability chain (the chain is
	// spec -> epic -> ticket); only materialized specs participate.
	Specs map[string]SpecNode
	// Architectures are the architecture documents keyed by slug.
	Architectures map[string]ArchNode
	// Epics are the epics keyed by id.
	Epics map[string]EpicNode
	// Tickets are the tickets keyed by id.
	Tickets map[string]TicketNode

	// specSlugs/archSlugs/epicIDs/ticketIDs are the sorted key lists, kept so every
	// traversal is deterministic regardless of map iteration order (R6/CA6).
	specSlugs []string
	archSlugs []string
	epicIDs   []string
	ticketIDs []string
}

// SpecNode is a spec in the chain. It carries only the identity needed for navigation;
// the full model stays in internal/specs (the source of truth, R4/CA5).
type SpecNode struct {
	Slug  string
	Title string
}

// ArchNode is an architecture document in the chain, with the spec slug it links back to
// (normalized from its frontmatter `spec:` reference), or "" if unlinked.
type ArchNode struct {
	Slug    string
	Title   string
	SpecRef string // normalized spec slug (no .md), "" if none recorded
}

// EpicNode is an epic in the chain, with the normalized origin links it recorded.
type EpicNode struct {
	ID      string
	Title   string
	SpecRef string // normalized spec slug, "" if none
	ArchRef string // normalized architecture slug, "" if none
}

// TicketNode is a ticket in the chain, with its parent epic id and any normalized origin
// links it recorded.
type TicketNode struct {
	ID      string
	EpicID  string
	Title   string
	SpecRef string // normalized spec slug, "" if none
	ArchRef string // normalized architecture slug, "" if none
}

// normalizeRef turns a recorded origin reference into a bare slug for matching. The CLI
// persists spec/architecture links as filenames (`<slug>.md`), but a hand-edited
// frontmatter might carry a bare slug; we accept both by stripping a trailing ".md" and
// surrounding whitespace, so the resolver matches the artifact regardless of which form
// the reference took. An empty reference stays empty (an absent link, not a broken one).
func normalizeRef(ref string) string {
	ref = strings.TrimSpace(ref)
	return strings.TrimSuffix(ref, ".md")
}

// Build reads every spec, architecture document, epic and ticket from the workspace and
// assembles the navigable Graph (R1/R4/CA5). It is read-only: it consumes the List/Load
// functions of the sibling packages and never writes. The three roots are the canonical
// workspace directories; the CLI supplies them so this package stays free of the
// workspace-location convention.
//
// A malformed artifact that fails to load is skipped at the listing level (the sibling
// List functions already skip unparseable files), so Build never fails on a single
// corrupt file; the verification pass reports the resulting gaps. Build itself returns an
// error only on a genuine I/O failure reading a directory.
func Build(specsRoot, archRoot, epicsRoot string) (*Graph, error) {
	g := &Graph{
		Specs:         map[string]SpecNode{},
		Architectures: map[string]ArchNode{},
		Epics:         map[string]EpicNode{},
		Tickets:       map[string]TicketNode{},
	}

	// Specs: only those with a materialized spec participate in the chain.
	specEntries, err := specs.List(specsRoot)
	if err != nil {
		return nil, err
	}
	for _, e := range specEntries {
		if !e.HasSpec {
			continue
		}
		g.Specs[e.Slug] = SpecNode{Slug: e.Slug, Title: e.Title}
	}

	// Architecture documents.
	archEntries, err := architecture.List(archRoot)
	if err != nil {
		return nil, err
	}
	for _, e := range archEntries {
		g.Architectures[e.Slug] = ArchNode{Slug: e.Slug, Title: e.Title, SpecRef: normalizeRef(e.SpecRef)}
	}

	// Epics, and the tickets nested under each.
	epicEntries, err := backlog.ListEpics(epicsRoot)
	if err != nil {
		return nil, err
	}
	for _, ee := range epicEntries {
		epic, err := backlog.LoadEpic(epicsRoot, ee.ID)
		if err != nil {
			continue
		}
		g.Epics[epic.ID] = EpicNode{
			ID:      epic.ID,
			Title:   epic.Title,
			SpecRef: normalizeRef(epic.SpecRef),
			ArchRef: normalizeRef(epic.ArchitectureRef),
		}

		ticketEntries, err := backlog.ListTickets(epicsRoot, epic.ID)
		if err != nil {
			continue
		}
		for _, te := range ticketEntries {
			g.addTicket(epicsRoot, epic.ID, te.ID)
		}
	}

	// Orphan sweep: pick up tickets whose parent epic did NOT load (e.g. its epic.md was
	// deleted), so the verification can flag them as orphans. backlog.ListEpics only
	// yields epics with a valid epic.md, so a ticket under an epic-shaped folder with no
	// epic.md is invisible to the walk above. Here we scan the epics tree directly — the
	// aggregator is allowed to be filesystem-aware — and add any ticket folder not already
	// captured. Reading the layout (epic folder / tickets / ticket folder) reuses the
	// backlog layout constants so it stays in sync with 05-03; the ticket's metadata still
	// comes from backlog.LoadTicket (the single source of truth, R4/CA5).
	g.sweepOrphanTickets(epicsRoot)

	g.indexKeys()
	return g, nil
}

// addTicket loads a ticket via the backlog package (the source of truth) and records it
// as a node. A ticket that fails to load is skipped (a malformed file is not a chain
// node); its absence surfaces as a gap during verification rather than crashing Build.
func (g *Graph) addTicket(epicsRoot, epicID, ticketID string) {
	if _, already := g.Tickets[ticketID]; already {
		return
	}
	ticket, err := backlog.LoadTicket(epicsRoot, epicID, ticketID)
	if err != nil {
		return
	}
	g.Tickets[ticket.ID] = TicketNode{
		ID:      ticket.ID,
		EpicID:  ticket.EpicID,
		Title:   ticket.Title,
		SpecRef: normalizeRef(ticket.SpecRef),
		ArchRef: normalizeRef(ticket.ArchitectureRef),
	}
}

// sweepOrphanTickets scans every `<epics>/<epic-dir>/tickets/<ticket-dir>` folder and
// adds any well-formed ticket that the epic walk did not already capture — i.e. tickets
// whose parent epic has no loadable epic.md. This is what makes orphan-ticket detection
// possible (R3/CA4): an orphan is precisely a ticket on disk whose epic is gone. All I/O
// errors are treated as "nothing to sweep" (a best-effort additive pass), since the main
// walk already produced the primary graph.
func (g *Graph) sweepOrphanTickets(epicsRoot string) {
	epicDirs, err := os.ReadDir(epicsRoot)
	if err != nil {
		return
	}
	for _, ed := range epicDirs {
		if !ed.IsDir() || !backlog.IsEpicID(ed.Name()) {
			continue
		}
		ticketsDir := filepath.Join(epicsRoot, ed.Name(), backlog.TicketsSubdir)
		ticketDirs, err := os.ReadDir(ticketsDir)
		if err != nil {
			continue
		}
		for _, td := range ticketDirs {
			if !td.IsDir() || !backlog.IsTicketID(td.Name()) {
				continue
			}
			g.addTicket(epicsRoot, ed.Name(), td.Name())
		}
	}
}

// indexKeys computes the sorted key lists once so every traversal is deterministic
// (R6/CA6) without re-sorting on each query.
func (g *Graph) indexKeys() {
	g.specSlugs = sortedKeys(g.Specs)
	g.archSlugs = sortedKeysArch(g.Architectures)
	g.epicIDs = sortedKeysEpic(g.Epics)
	g.ticketIDs = sortedKeysTicket(g.Tickets)
}

// --- navigation ---

// SpecChain is the descending view from one spec: the spec, its epics, and each epic's
// tickets (R1/CA1). Epics and tickets are in deterministic id order.
type SpecChain struct {
	Spec  SpecNode
	Epics []EpicWithTickets
}

// EpicWithTickets pairs an epic with its tickets for the descending view.
type EpicWithTickets struct {
	Epic    EpicNode
	Tickets []TicketNode
}

// TicketChain is the ascending view from one ticket: the ticket, its epic (if it
// exists), and the origin spec/architecture (R1/CA2). EpicFound is false when the
// ticket's parent epic does not exist (an orphan), so a caller can render the break.
type TicketChain struct {
	Ticket    TicketNode
	Epic      EpicNode
	EpicFound bool
	// OriginSpec/OriginArch are the resolved origin nodes, taken from the ticket's own
	// links if present, otherwise inherited from its epic's links (a ticket commonly
	// inherits its origin from its epic). Found flags report whether each resolved.
	OriginSpec      SpecNode
	OriginSpecFound bool
	OriginArch      ArchNode
	OriginArchFound bool
}

// DescendFromSpec returns the descending chain from the spec slug (R1/CA1): its epics
// (those whose recorded origin spec resolves to this slug) and their tickets. It returns
// ok=false if the spec is not a node in the graph. The walk is deterministic: epics in
// sorted id order, tickets in sorted id order.
func (g *Graph) DescendFromSpec(slug string) (SpecChain, bool) {
	spec, ok := g.Specs[slug]
	if !ok {
		return SpecChain{}, false
	}

	chain := SpecChain{Spec: spec}
	for _, epicID := range g.epicIDs {
		epic := g.Epics[epicID]
		if epic.SpecRef != slug {
			continue
		}
		chain.Epics = append(chain.Epics, EpicWithTickets{
			Epic:    epic,
			Tickets: g.ticketsOfEpic(epicID),
		})
	}
	return chain, true
}

// AscendFromTicket returns the ascending chain from the ticket id (R1/CA2): its parent
// epic and its origin spec/architecture, resolving the ticket's own origin links first
// and falling back to the epic's links when the ticket records none. It returns ok=false
// if the ticket is not a node in the graph.
func (g *Graph) AscendFromTicket(id string) (TicketChain, bool) {
	ticket, ok := g.Tickets[id]
	if !ok {
		return TicketChain{}, false
	}

	chain := TicketChain{Ticket: ticket}
	epic, epicFound := g.Epics[ticket.EpicID]
	chain.Epic = epic
	chain.EpicFound = epicFound

	// Resolve the origin spec: prefer the ticket's own link, else inherit the epic's.
	specRef := ticket.SpecRef
	if specRef == "" && epicFound {
		specRef = epic.SpecRef
	}
	if specRef != "" {
		if s, found := g.Specs[specRef]; found {
			chain.OriginSpec = s
			chain.OriginSpecFound = true
		}
	}

	// Resolve the origin architecture the same way.
	archRef := ticket.ArchRef
	if archRef == "" && epicFound {
		archRef = epic.ArchRef
	}
	if archRef != "" {
		if a, found := g.Architectures[archRef]; found {
			chain.OriginArch = a
			chain.OriginArchFound = true
		}
	}

	return chain, true
}

// TicketsOfEpic returns the tickets whose parent epic is epicID, in sorted id order. It
// is the descending half of navigation from an epic (R1/CA1), exported so the CLI can
// show an epic's tickets directly without first locating its spec.
func (g *Graph) TicketsOfEpic(epicID string) []TicketNode {
	return g.ticketsOfEpic(epicID)
}

// ticketsOfEpic returns the tickets whose parent epic is epicID, in sorted id order.
func (g *Graph) ticketsOfEpic(epicID string) []TicketNode {
	var out []TicketNode
	for _, tID := range g.ticketIDs {
		if t := g.Tickets[tID]; t.EpicID == epicID {
			out = append(out, t)
		}
	}
	return out
}

// --- sorted-key helpers (one per node type; Go has no generic map-keys-sorted in
// stdlib for these concrete maps, and keeping them explicit stays clear and dependency-
// free). ---

func sortedKeys(m map[string]SpecNode) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysArch(m map[string]ArchNode) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysEpic(m map[string]EpicNode) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedKeysTicket(m map[string]TicketNode) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
