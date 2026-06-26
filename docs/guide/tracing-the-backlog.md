# Tracing the backlog

[← Back to the manual index](../README.md)

Once you have [specs](managing-specs.md),
[architecture documents](managing-architecture.md), and a backlog of
[epics and tickets](managing-epics-and-tickets.md), the **traceability** chain
ties them together: every ticket belongs to an epic, and every epic and ticket
can record the spec or architecture document it derives from. The
`daedalus trace` command lets you **verify** that this chain is consistent and
**navigate** it in both directions — from a spec down to its tickets, and from a
ticket back up to its origin.

`daedalus trace` is **read-only**. It never runs an agent and never writes
anything; it simply reads the links already recorded in your artifacts'
frontmatter. There is no separate index to keep in sync — the artifacts
themselves are the single source of truth, so if you edit a link by hand, `trace`
reflects it on the next run.

## The task, end to end

- **Check the backlog is consistent.** Run `daedalus trace verify` to confirm
  every recorded link resolves and every ticket has a parent epic. It reports any
  inconsistency, worst-first.
- **Navigate the chain.** Run `daedalus trace show <artifact-id>` to walk the
  chain from any artifact — a spec, an epic, or a ticket — in the natural
  direction for that artifact.

## What the chain looks like

The chain has three levels, linked by the frontmatter your artifacts already
carry:

```
spec  ─▶  epic  ─▶  ticket
            │           │
            └── origin links (spec / architecture) recorded in the frontmatter
```

- A **ticket** always names its parent **epic** (mandatory).
- An **epic** and a **ticket** may each record an **origin** spec and/or
  architecture document (optional — see
  [Managing epics and tickets](managing-epics-and-tickets.md)). When a ticket
  records no origin of its own, it **inherits** its epic's origin.

`trace` reasons over exactly these links; it does not invent or store new ones.

## Verifying the chain

Use `daedalus trace verify` to check the whole workspace:

```sh
daedalus trace verify
```

When every link resolves and no ticket is orphaned, the chain is consistent and
the command exits with status `0`:

```
Traceability chain is consistent (no inconsistencies).
```

### What it checks

`verify` reports three kinds of inconsistency. Two are **hard errors** that mean
the chain is genuinely broken; one is a **soft warning** for a traceability gap
that the backlog model allows.

| Kind | Severity | What it means | Affects exit code? |
|---|---|---|---|
| `broken-link` | error | An epic, ticket, or architecture document records an origin (spec or architecture) that **does not exist** — the reference is present but dangles. | **Yes** |
| `orphan-ticket` | error | A ticket's parent **epic no longer exists** (for example, the epic folder was removed). | **Yes** |
| `missing-origin` | warning | An epic (or architecture document) records **no origin link at all**. This is a traceability gap, but it is **legal** — linking a spec/architecture is optional. A ticket with no origin of its own is **not** reported: it inherits its epic's origin. | No |

### Exit codes

`verify` sets its exit code so you can gate on it from a script or CI:

| Exit code | Meaning |
|---|---|
| `0` | The chain is consistent, **or** has only warnings (a soft gap never fails the check). |
| `1` | The chain has at least one **hard error** (`broken-link` or `orphan-ticket`). |
| `2` | A usage or load error. |

### When there are inconsistencies

Findings are reported worst-first (errors before warnings) in a deterministic
order, each on its own line, naming the affected artifact, the kind of problem,
the value at fault, and how to fix it. When there are hard errors, the command
exits `1`:

```
Traceability chain has 2 errors and 1 warning:
  - [error] ticket-05-03-epics-tickets-management: orphan-ticket: observed "epic-05-sdd-backlog"; ticket references a parent epic that does not exist; the epic was removed or the reference is wrong — restore the epic or correct the reference
  - [error] epic-06-compile: broken-link: observed "compile-spec.md"; epic references a spec that does not exist; create the spec or correct the reference
  - [warning] epic-07-telemetry: missing-origin: observed ""; epic records no origin spec or architecture; link one to complete the trace (optional per the backlog model, but recommended)
```

When the only findings are warnings, the chain is still **consistent** and the
command exits `0` — the gaps are reported for visibility, not as failures:

```
Traceability chain is consistent with 1 warning (no hard errors):
  - [warning] epic-07-telemetry: missing-origin: observed ""; epic records no origin spec or architecture; link one to complete the trace (optional per the backlog model, but recommended)
```

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/` chain is verified. Defaults to the current directory. |

## Navigating the chain

Use `daedalus trace show <artifact-id>` to walk the chain from any artifact.
Daedalus infers the **direction** from the shape of the id:

- a **spec slug** descends — spec → its epics → their tickets;
- a **ticket id** (`ticket-NN-MM-<slug>`) ascends — ticket → its epic → its origin
  spec/architecture;
- an **epic id** (`epic-NN-<slug>`) shows **both** — the epic's origin and its
  tickets.

### From a spec (descending)

Pass a spec slug to see every epic linked to it and their tickets:

```sh
daedalus trace show sdd-backlog
```

```
spec sdd-backlog — SDD Backlog
  └─ epic epic-05-sdd-backlog — SDD Backlog
       └─ ticket ticket-05-03-epics-tickets-management — Epics & Tickets Management
       └─ ticket ticket-05-04-traceability — Traceability
```

If no epic links to the spec, Daedalus says so instead of showing an empty tree:

```
spec sdd-backlog — SDD Backlog
  (no epics link to this spec)
```

### From a ticket (ascending)

Pass a ticket id to climb back to its epic and its origin spec/architecture. When
the ticket records no origin of its own, the origin shown is the one it inherits
from its epic:

```sh
daedalus trace show ticket-05-04-traceability
```

```
ticket ticket-05-04-traceability — Traceability
  └─ epic epic-05-sdd-backlog — SDD Backlog
       └─ origin spec sdd-backlog — SDD Backlog
       └─ origin architecture sdd-backlog-arch — SDD Backlog Architecture
```

An **orphan** ticket — one whose parent epic no longer exists — is flagged
explicitly:

```
ticket ticket-05-04-traceability — Traceability
  └─ epic epic-05-sdd-backlog — MISSING (orphan ticket)
       └─ origin spec: (none resolved)
       └─ origin architecture: (none resolved)
```

### From an epic (both directions)

Pass an epic id to see its origin links and its tickets at once:

```sh
daedalus trace show epic-05-sdd-backlog
```

```
epic epic-05-sdd-backlog — SDD Backlog
  origin spec:         sdd-backlog.md
  origin architecture: sdd-backlog-arch.md
  tickets:
    └─ ticket ticket-05-03-epics-tickets-management — Epics & Tickets Management
    └─ ticket ticket-05-04-traceability — Traceability
```

An epic with no recorded origin shows `(none)` for that link, and an epic with no
tickets shows `tickets: (none)`.

### Options

| Option | Description |
|---|---|
| `--path <dir>` | Target repository directory whose `.daedalus/` chain is navigated. Defaults to the current directory. |

If the id is not in the chain, Daedalus tells you so and points you at the
relevant listing command — for example, for an unknown spec slug:

```
daedalus: no spec "ghost" in the traceability chain (a spec must be materialized; check 'daedalus spec list')
```

## Phase 1: read-only, no agents

`daedalus trace` does not run any agent and does not write to your workspace. It
**reads** the links already recorded in your specs, architecture documents,
epics, and tickets, and reports or navigates the chain they form. It keeps **no
index of its own**: the artifacts' frontmatter is the single source of truth, so
the result always reflects the current state of your files. Edit a link by hand
and the next `trace` run sees the change immediately.

## Notes and limitations

- `trace` is **read-only and deterministic**: the same workspace state always
  produces the same report, with findings in a stable, worst-first order. It
  never executes agents and never modifies your files.
- It holds **no separate index** — it reads the origin and parent links directly
  from each artifact's frontmatter. Fixing an inconsistency therefore means
  editing the source artifact (for example, with `daedalus epic edit` or
  `daedalus ticket edit`, or by hand).
- The **severity split is intentional.** Because recording a spec/architecture
  origin is optional in the backlog, a *missing* origin is a warning (never fails
  `verify`), while a *dangling* origin or a missing parent epic is a hard error
  (fails `verify`).
- **Fixing links by hand:** to clear an origin link, use the single-token form
  `--spec=` or `--architecture=` with `daedalus epic edit` / `daedalus ticket
  edit` (on PowerShell, `--spec ""` as two tokens is misread by the shell). When
  editing artifact files directly on Windows, save **without a BOM** — a leading
  byte-order mark breaks frontmatter parsing, which `trace` then cannot read.
- Phase 1 **verifies and navigates** the chain; it does not **run** the pipeline
  — generating the artifacts stays with your runtime (for example, Claude Code).
</content>
