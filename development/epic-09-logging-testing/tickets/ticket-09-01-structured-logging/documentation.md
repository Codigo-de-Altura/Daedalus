# Documentation

The end-user documentation for this feature lives in the Daedalus manual:

→ [Configuration](../../../../docs/guide/configuration.md) — section
**"Reading operation logs to troubleshoot"**, under `## Logging`.

This ticket deepens the existing logging baseline, so the documentation extends
the chapter that already covers logging (levels, `DAEDALUS_LOG_LEVEL`, capturing,
and privacy) rather than adding a new chapter. The new subsection explains the
structured decision-point events emitted by `init`, `build`/`sync`, and
`daedalus validate`, and how a reader uses them to troubleshoot — for example,
finding the `definition rejected` event (with its `reason` and workspace-relative
`definition` path) when a build aborts, or the `convention violated` events that
name the offending file and convention.

This file is intentionally a pointer. Product documentation is maintained as a
single manual under `docs/` (see CLAUDE.md §6).
