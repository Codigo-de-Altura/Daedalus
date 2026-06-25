package backlog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ErrEpicExists is the sentinel returned by Create/Apply when an epic with the
// requested id already has a file. Exposed so callers map a duplicate id to an explicit
// "already exists, not overwritten" error via errors.Is, never an overwrite (R8/CA8).
var ErrEpicExists = errors.New("epic already exists")

// ErrTicketExists is the sentinel returned when a ticket with the requested id already
// has a file under its epic.
var ErrTicketExists = errors.New("ticket already exists")

// ErrParentEpicMissing is returned by ticket creation when the parent epic does not
// exist. A ticket is nested under its epic on disk (R2/CA2), so it cannot be created
// without a parent — this is a structural prerequisite, not a soft link check, so it is
// enforced in the store (unlike the optional spec/architecture links, which are checked
// at the CLI layer).
var ErrParentEpicMissing = errors.New("parent epic does not exist")

// --- Epics ---

// EpicCreatePlan is the result of planning an epic Create: the resolved epic, the file
// it will land in and the fully rendered bytes — computed without touching the
// filesystem (R6/R8). Content is captured at plan time so `--preview` and Apply describe
// identical bytes, and validation happens before any I/O.
type EpicCreatePlan struct {
	Epic    Epic
	Path    string
	Content string
}

// PlanCreateEpic validates an epic and computes the plan to persist it under epicsRoot,
// without writing anything (R6/R8). Defaults are applied first (status/priority), then
// the epic is validated; an invalid epic is rejected here — before any I/O — with the
// rich *ValidationError. Dependencies are deduplicated for consistency (R4). A body-less
// epic is seeded with a deterministic placeholder stating Daedalus did not run the
// planner (R7/CA7). It does NOT check for an existing file (Apply-time, non-destructive
// concern) nor verify referenced artifacts' existence (CLI-layer / 05-04 concern).
func PlanCreateEpic(epicsRoot string, e Epic) (*EpicCreatePlan, error) {
	e = applyEpicDefaults(e)
	if trimmedEmpty(e.Body) {
		e.Body = epicPlaceholderBody(e)
	}
	if err := e.Validate(); err != nil {
		return nil, err
	}
	return &EpicCreatePlan{
		Epic:    e,
		Path:    epicPath(epicsRoot, e.ID),
		Content: RenderEpic(e),
	}, nil
}

// Apply writes the planned epic non-destructively: it creates the epic's folder if
// needed and the file only if it does not already exist (O_CREATE|O_EXCL). A duplicate
// id is reported as ErrEpicExists rather than clobbered (R8/CA8). Re-creating into a
// clean workspace yields byte-identical bytes (R6).
func (p *EpicCreatePlan) Apply() error {
	if err := os.MkdirAll(filepath.Dir(p.Path), 0o755); err != nil {
		return err
	}
	created, err := ensureFile(p.Path, p.Content)
	if err != nil {
		return err
	}
	if !created {
		return fmt.Errorf("%w: %q", ErrEpicExists, p.Epic.ID)
	}
	return nil
}

// CreateEpic validates and persists a new epic in one call, non-destructively (R8).
func CreateEpic(epicsRoot string, e Epic) error {
	plan, err := PlanCreateEpic(epicsRoot, e)
	if err != nil {
		return err
	}
	return plan.Apply()
}

// applyEpicDefaults fills unset status/priority with the canonical defaults and
// deduplicates the dependency list, so a caller that omits them gets a consistent,
// valid artifact rather than empty enum values.
func applyEpicDefaults(e Epic) Epic {
	if e.Status == "" {
		e.Status = DefaultStatus
	}
	if e.Priority == "" {
		e.Priority = DefaultPriority
	}
	e.DependsOn = dedupePreserveOrder(e.DependsOn)
	return e
}

// epicPlaceholderBody is the deterministic placeholder seeded into a body-less epic
// (R7/CA7). It states Daedalus did not run the planner and, when linked, names the
// origin artifacts. Pure function of the epic so the seed is byte-stable (R6).
func epicPlaceholderBody(e Epic) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n\n", e.Title)
	sb.WriteString("> Epic placeholder. Daedalus manages this artifact's definition but does not run\n")
	fmt.Fprintf(&sb, "> the %q agent (workflow %q, phase %q). Generate the epic on your backend, then\n",
		PlannerAgent, DefaultWorkflowName, PhaseEpics)
	sb.WriteString("> replace this placeholder with the objective, scope and acceptance criteria.\n")
	writeOriginLines(&sb, e.SpecRef, e.ArchitectureRef)
	return sb.String()
}

// ListEpics returns the epics under epicsRoot as entries sorted by id (R6). A
// non-existent epics directory is treated as "no epics" (empty list), not an error.
// Only subdirectories whose name is a valid epic id and that contain an epic.md are
// considered; a folder that fails to parse is skipped rather than aborting the listing.
func ListEpics(epicsRoot string) ([]EpicEntry, error) {
	dirEntries, err := os.ReadDir(epicsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []EpicEntry{}, nil
		}
		return nil, err
	}

	out := make([]EpicEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if !de.IsDir() || !IsEpicID(de.Name()) {
			continue
		}
		e, err := LoadEpic(epicsRoot, de.Name())
		if err != nil {
			continue
		}
		out = append(out, EpicEntry{ID: e.ID, Title: e.Title, Status: e.Status, Priority: e.Priority})
	}
	sortEpicEntries(out)
	return out, nil
}

// EpicEditSpec describes the changes to apply to a persisted epic. Every field is
// optional; the presence of a change is signaled by the *Set* booleans rather than zero
// values, because an explicit empty value (e.g. clear the spec link) and "leave it
// untouched" must be distinguishable. The id is NOT editable — it is the folder
// identity (renaming is create+remove, not an in-place mutation).
type EpicEditSpec struct {
	SetTitle        bool
	Title           string
	SetStatus       bool
	Status          Status
	SetPriority     bool
	Priority        Priority
	SetSpec         bool
	Spec            string
	SetArchitecture bool
	Architecture    string
	SetDependsOn    bool
	DependsOn       []string
	SetBody         bool
	Body            string
}

// IsEmpty reports whether the edit would change nothing.
func (s EpicEditSpec) IsEmpty() bool {
	return !s.SetTitle && !s.SetStatus && !s.SetPriority && !s.SetSpec &&
		!s.SetArchitecture && !s.SetDependsOn && !s.SetBody
}

// EditEpic applies spec to the persisted epic and persists the result atomically
// (load -> mutate -> validate -> atomic write), returning the edited Epic. An invalid
// result is rejected before any write and the existing file is left intact (R8/CA8).
// Only this epic's file is touched.
func EditEpic(epicsRoot, epicID string, spec EpicEditSpec) (Epic, error) {
	e, err := LoadEpic(epicsRoot, epicID)
	if err != nil {
		return Epic{}, err
	}

	if spec.SetTitle {
		e.Title = spec.Title
	}
	if spec.SetStatus {
		e.Status = spec.Status
	}
	if spec.SetPriority {
		e.Priority = spec.Priority
	}
	if spec.SetSpec {
		e.SpecRef = spec.Spec
	}
	if spec.SetArchitecture {
		e.ArchitectureRef = spec.Architecture
	}
	if spec.SetDependsOn {
		e.DependsOn = dedupePreserveOrder(spec.DependsOn)
	}
	if spec.SetBody {
		e.Body = spec.Body
	}

	if err := e.Validate(); err != nil {
		return Epic{}, err
	}
	if err := writeAtomic(epicPath(epicsRoot, epicID), RenderEpic(e)); err != nil {
		return Epic{}, err
	}
	return e, nil
}

// RemoveEpic deletes the epic's entire folder (epic.md plus its nested tickets) under
// epicsRoot. Removing an epic necessarily removes its tickets, because they live inside
// it on disk; this is reported to the caller so the cascade is never a surprise (the CLI
// can warn). An absent epic is ErrEpicNotFound. The id is validated first so a malformed
// id can never be turned into an unexpected path.
func RemoveEpic(epicsRoot, epicID string) error {
	if !IsEpicID(epicID) {
		return fmt.Errorf("epic id %q is not a valid epic-NN-<slug> id", epicID)
	}
	dir := epicDir(epicsRoot, epicID)
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %q", ErrEpicNotFound, epicID)
		}
		return err
	}
	return os.RemoveAll(dir)
}

// --- Tickets ---

// TicketCreatePlan is the result of planning a ticket Create.
type TicketCreatePlan struct {
	Ticket  Ticket
	Path    string
	Content string
}

// PlanCreateTicket validates a ticket and computes the plan to persist it nested under
// its parent epic, without writing anything (R6/R8). The parent epic MUST already exist
// (ErrParentEpicMissing otherwise): a ticket is structurally a child of its epic on disk
// (R2/CA2). Defaults are applied, dependencies deduplicated, and the parent epic id is
// recorded as the mandatory R5/CA5 link. A body-less ticket is seeded with a placeholder
// (R7/CA7).
func PlanCreateTicket(epicsRoot string, t Ticket) (*TicketCreatePlan, error) {
	if !IsEpicID(t.EpicID) {
		// Surface as a validation error so the CLI reports it actionably.
		if err := t.Validate(); err != nil {
			return nil, err
		}
	}
	// The parent epic must exist on disk: a ticket cannot be nested under a missing epic.
	if _, err := os.Stat(epicPath(epicsRoot, t.EpicID)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %q", ErrParentEpicMissing, t.EpicID)
		}
		return nil, err
	}

	t = applyTicketDefaults(t)
	if trimmedEmpty(t.Body) {
		t.Body = ticketPlaceholderBody(t)
	}
	if err := t.Validate(); err != nil {
		return nil, err
	}
	return &TicketCreatePlan{
		Ticket:  t,
		Path:    ticketPath(epicsRoot, t.EpicID, t.ID),
		Content: RenderTicket(t),
	}, nil
}

// Apply writes the planned ticket non-destructively (O_CREATE|O_EXCL). A duplicate id is
// reported as ErrTicketExists rather than clobbered (R8/CA8).
func (p *TicketCreatePlan) Apply() error {
	if err := os.MkdirAll(filepath.Dir(p.Path), 0o755); err != nil {
		return err
	}
	created, err := ensureFile(p.Path, p.Content)
	if err != nil {
		return err
	}
	if !created {
		return fmt.Errorf("%w: %q", ErrTicketExists, p.Ticket.ID)
	}
	return nil
}

// CreateTicket validates and persists a new ticket in one call, non-destructively (R8).
func CreateTicket(epicsRoot string, t Ticket) error {
	plan, err := PlanCreateTicket(epicsRoot, t)
	if err != nil {
		return err
	}
	return plan.Apply()
}

// applyTicketDefaults fills unset status/priority and deduplicates dependencies.
func applyTicketDefaults(t Ticket) Ticket {
	if t.Status == "" {
		t.Status = DefaultStatus
	}
	if t.Priority == "" {
		t.Priority = DefaultPriority
	}
	t.DependsOn = dedupePreserveOrder(t.DependsOn)
	return t
}

// ticketPlaceholderBody is the deterministic placeholder seeded into a body-less ticket
// (R7/CA7). It names the parent epic (the R5 link) and any origin artifacts.
func ticketPlaceholderBody(t Ticket) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "# %s\n\n", t.Title)
	sb.WriteString("> Ticket placeholder. Daedalus manages this artifact's definition but does not run\n")
	fmt.Fprintf(&sb, "> the %q agent (workflow %q, phase %q). Generate the ticket on your backend, then\n",
		PlannerAgent, DefaultWorkflowName, PhaseTickets)
	sb.WriteString("> replace this placeholder with the feature, requirements and acceptance criteria.\n\n")
	fmt.Fprintf(&sb, "Parent epic: %s\n", t.EpicID)
	writeOriginLines(&sb, t.SpecRef, t.ArchitectureRef)
	return sb.String()
}

// writeOriginLines appends "Source spec/architecture" lines for any recorded origin
// link, reinforcing the R5 trace in the human-readable body (not only the frontmatter).
func writeOriginLines(sb *strings.Builder, specRef, archRef string) {
	if !trimmedEmpty(specRef) {
		fmt.Fprintf(sb, "Source spec: %s\n", specRef)
	}
	if !trimmedEmpty(archRef) {
		fmt.Fprintf(sb, "Source architecture: %s\n", archRef)
	}
}

// ListTickets returns the tickets of one epic as entries sorted by id (R6). A
// non-existent epic or tickets directory is treated as "no tickets" (empty list), not an
// error, so listing an epic with no tickets is well-defined. A folder that fails to parse
// is skipped.
func ListTickets(epicsRoot, epicID string) ([]TicketEntry, error) {
	if !IsEpicID(epicID) {
		return nil, fmt.Errorf("epic id %q is not a valid epic-NN-<slug> id", epicID)
	}
	ticketsRoot := filepath.Join(epicDir(epicsRoot, epicID), TicketsSubdir)
	dirEntries, err := os.ReadDir(ticketsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []TicketEntry{}, nil
		}
		return nil, err
	}

	out := make([]TicketEntry, 0, len(dirEntries))
	for _, de := range dirEntries {
		if !de.IsDir() || !IsTicketID(de.Name()) {
			continue
		}
		t, err := LoadTicket(epicsRoot, epicID, de.Name())
		if err != nil {
			continue
		}
		out = append(out, TicketEntry{ID: t.ID, EpicID: t.EpicID, Title: t.Title, Status: t.Status, Priority: t.Priority})
	}
	sortTicketEntries(out)
	return out, nil
}

// TicketEditSpec describes the changes to apply to a persisted ticket. As for epics,
// *Set* booleans distinguish an explicit change from a no-op. The id and the parent epic
// id are NOT editable: both are structural identity (the nested folder), and moving a
// ticket to another epic is create+remove, not an in-place mutation.
type TicketEditSpec struct {
	SetTitle        bool
	Title           string
	SetStatus       bool
	Status          Status
	SetPriority     bool
	Priority        Priority
	SetSpec         bool
	Spec            string
	SetArchitecture bool
	Architecture    string
	SetDependsOn    bool
	DependsOn       []string
	SetBody         bool
	Body            string
}

// IsEmpty reports whether the edit would change nothing.
func (s TicketEditSpec) IsEmpty() bool {
	return !s.SetTitle && !s.SetStatus && !s.SetPriority && !s.SetSpec &&
		!s.SetArchitecture && !s.SetDependsOn && !s.SetBody
}

// EditTicket applies spec to the persisted ticket and persists the result atomically.
// An invalid result is rejected before any write and the existing file is left intact
// (R8/CA8).
func EditTicket(epicsRoot, epicID, ticketID string, spec TicketEditSpec) (Ticket, error) {
	t, err := LoadTicket(epicsRoot, epicID, ticketID)
	if err != nil {
		return Ticket{}, err
	}

	if spec.SetTitle {
		t.Title = spec.Title
	}
	if spec.SetStatus {
		t.Status = spec.Status
	}
	if spec.SetPriority {
		t.Priority = spec.Priority
	}
	if spec.SetSpec {
		t.SpecRef = spec.Spec
	}
	if spec.SetArchitecture {
		t.ArchitectureRef = spec.Architecture
	}
	if spec.SetDependsOn {
		t.DependsOn = dedupePreserveOrder(spec.DependsOn)
	}
	if spec.SetBody {
		t.Body = spec.Body
	}

	if err := t.Validate(); err != nil {
		return Ticket{}, err
	}
	if err := writeAtomic(ticketPath(epicsRoot, epicID, ticketID), RenderTicket(t)); err != nil {
		return Ticket{}, err
	}
	return t, nil
}

// RemoveTicket deletes the ticket's folder under its epic and nothing else (the epic and
// sibling tickets are untouched). An absent ticket is ErrTicketNotFound.
func RemoveTicket(epicsRoot, epicID, ticketID string) error {
	if !IsEpicID(epicID) {
		return fmt.Errorf("epic id %q is not a valid epic-NN-<slug> id", epicID)
	}
	if !IsTicketID(ticketID) {
		return fmt.Errorf("ticket id %q is not a valid ticket-NN-MM-<slug> id", ticketID)
	}
	dir := ticketDir(epicsRoot, epicID, ticketID)
	if _, err := os.Stat(dir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %q", ErrTicketNotFound, ticketID)
		}
		return err
	}
	return os.RemoveAll(dir)
}

// --- shared write helpers ---

// ensureFile creates a file at path with the given content only if it does not exist.
// O_EXCL makes the create-or-skip decision atomic and non-destructive: an existing file
// is never truncated, so an artifact the user refined survives a re-create attempt
// untouched (R8/CA8). Mirrors the sibling helper.
func ensureFile(path, content string) (bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, fs.ErrExist) {
			return false, nil
		}
		return false, err
	}
	_, writeErr := f.WriteString(content)
	closeErr := f.Close()
	if writeErr != nil {
		return false, writeErr
	}
	if closeErr != nil {
		return false, closeErr
	}
	return true, nil
}

// writeAtomic writes content to path atomically via a temp file + rename in the same
// directory, so a reader sees old or new content but never a partial write, and a crash
// mid-write leaves the original intact. Unlike ensureFile it replaces an existing file,
// which is what an edit must do. Mirrors the sibling helper.
func writeAtomic(path, content string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
