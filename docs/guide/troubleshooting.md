# Troubleshooting

[← Back to the manual index](../README.md)

Most problems you hit with Daedalus are reported with an **actionable** message
that tells you the file, the spot, the rule, and what was expected. This chapter
shows the common ones and how to read them. For where output goes, remember the
[output convention](command-reference.md): the human summary is on **stdout** and
the structured JSON logs are on **stderr**.

## An invalid definition

When a definition does not satisfy its schema, Daedalus does not just say
"invalid" — it names the **file**, the exact **spot**, the **rule**, and what was
**observed** versus what was **expected**. For example, `daedalus validate` on a
manifest whose `backends` lists an unsupported value:

```
Conventions: workspace conforms (no violations).
Definitions: 1 error and 0 warnings:
  - [error] .daedalus/daedalus.yaml: backends[nonexistent-backend]: schema: observed unsupported backend "nonexistent-backend"; expected one of the supported backends: claude-code
```

Exit `1`. Read the finding left to right — severity, file, spot, rule, then
observed-vs-expected. Here the fix is to set `backends` to a supported value
(`claude-code`) in `.daedalus/daedalus.yaml`, then re-run `daedalus validate` to
confirm the workspace conforms.

The same actionable shape appears wherever a definition is checked — for example
when [`daedalus agent edit`](managing-agents.md#editing-an-agent) rejects an edit
that would produce an empty role, listing every problem at once so you can fix
them in a single pass.

## Validation that fails

[`daedalus validate`](validating-conventions.md) checks two axes — **conventions**
and **definitions** — and a single error in either one exits `1`. It is
**read-only**: it reports violations, it never auto-fixes them. The fix is yours
(rename the file, move it into place, correct the manifest), and the next run
confirms it.

- **Errors** are genuine breaches (a bad name, a missing required directory, a
  misplaced artifact, a broken backlog link, an unsupported backend, a DAG
  cycle). They fail the check (exit `1`).
- **Warnings** flag an optional gap — for example an epic with no recorded
  origin. They are reported for visibility but do **not** fail the check; a
  workspace whose only findings are warnings still exits `0`.

Gate CI on `daedalus validate`: a non-zero exit means the workspace drifted from
the conventions or carries an invalid definition.

## A build that aborts

[`daedalus build`](compiling-to-a-backend.md) **validates first** and writes
nothing if anything is wrong, so a failed build always leaves your repository
exactly as it was:

- **No workspace** in the target directory — `build` aborts and points you at
  `daedalus init`.
- **Invalid canonical definition** — `build` reports the problems and stops
  (exit `3`); fix them and re-run.
- **No adapter for the configured backend** — `build` fails with a clear message
  (exit `4`) and writes nothing.

## Building without a terminal

`build` never writes silently. When it runs **without an interactive terminal**
— a script, CI, or a container with no TTY — there is no one to confirm at the
gate, so a plain `build` **prints the diff and writes nothing**, then tells you
how to proceed. To write from automation, pass `--yes`:

```sh
daedalus build --yes
```

`--yes` skips the interactive gate and compiles directly; it works with or
without a terminal. `--preview` always wins over `--yes` — an explicit dry run
never writes. See
[Writing from a script or CI](compiling-to-a-backend.md#writing-from-a-script-or-ci).

## The interface will not open

`daedalus` with no subcommand launches the interface, which **requires an
interactive terminal**. With piped input, in a script, in CI, or in a container
with no TTY, Daedalus does not start the interface — it prints a short notice and
exits `0`. That is expected; run it in a real terminal to use the interface, or
use the subcommands (`init`, `build`, `validate`, `trace`) from automation. See
[Command line](command-line.md#non-interactive-use).

## Reading the logs

When the summary on stdout is not enough, the structured JSON logs on **stderr**
show what Daedalus decided and why. Lower the threshold for more detail, or raise
it to silence routine logs:

```sh
# More detail
DAEDALUS_LOG_LEVEL=debug daedalus build --yes

# Only warnings and errors (silences INFO logs)
DAEDALUS_LOG_LEVEL=error daedalus validate
```

With `DAEDALUS_LOG_LEVEL=error`, the routine INFO logs are suppressed entirely,
leaving only the human-readable summary on stdout. Because logs go to stderr, you
can capture them independently:

```sh
daedalus build --yes 2> daedalus.log
```

Then look for the event you need — for example a `definition rejected` event
(logged at `WARN`, naming the `reason` and the `definition` path) when a build
aborts on an invalid definition, or a `convention violated` event when
`validate` reports a problem. See
[Configuration → Logging](configuration.md#logging) for the full details.

## A note on Windows and BOM

When you edit workspace files by hand on Windows, save them **without a
byte-order mark (BOM)**. A leading BOM breaks YAML frontmatter parsing, which
[`daedalus trace`](tracing-the-backlog.md#notes-and-limitations) and the validators
then cannot read. Most editors offer "UTF-8 without BOM" — use it for files under
`.daedalus/`.

## Still stuck?

- Re-run the command with `DAEDALUS_LOG_LEVEL=debug` and read the stderr log.
- Confirm you are pointing at the right workspace with `--path`.
- Check the relevant chapter: [Command reference](command-reference.md),
  [Validating conventions](validating-conventions.md), or
  [Compiling to a backend](compiling-to-a-backend.md).
