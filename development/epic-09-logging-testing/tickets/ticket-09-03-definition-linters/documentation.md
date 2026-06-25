# Documentation

The end-user documentation for this feature lives in the Daedalus manual:

→ [Validating conventions](../../../../docs/guide/validating-conventions.md) —
section **"Validating definitions"**.

This ticket extends the existing `daedalus validate` command with a second axis,
so the documentation extends the chapter that already covers that command rather
than adding a new one. The new section describes the three linted definition
families — **Agents** (schema validity), **Workflows (DAG)** (cycles, missing
artifacts, unknown agents, duplicate phase ids, malformed dependencies), and the
**Manifest** (well-formed required fields, supported backends, coherent
conventions) — with a short actionable example, and clarifies that the exit code
now spans both axes (any error in either fails the check) while a fresh
`daedalus init` workspace lints clean.

(The chapter title and the manual-index link were kept; the index description was
updated to mention the new definitions axis.)

This file is intentionally a pointer. Product documentation is maintained as a
single manual under `docs/` (see CLAUDE.md §6).
