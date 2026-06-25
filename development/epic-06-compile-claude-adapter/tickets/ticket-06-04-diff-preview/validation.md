# Validación — Reporte de diff / preview

> La corre **Leia** (validadora frontend). Solo reporta; no implementa ni arregla.

---

## Precondiciones

- Binario `daedalus` compilado y disponible, con la TUI operativa.
- Repo de prueba con `.daedalus/` válido (`daedalus.yaml` → Claude Code).
- Tres escenarios disponibles: (a) workspace **sin** `.claude/` previo (todo nuevo); (b) `.claude/` ya generado e idéntico (sin cambios); (c) definición canónica modificada respecto a un `.claude/` previo (cambios).
- Terminal capaz de mostrar la TUI (colores, render markdown).

## Checks numerados

### Check 1 — Se muestra el preview antes de escribir
- **Comando:** Iniciar el build en modo que dispara preview.
- **Esperado:** Aparece un reporte de diff/preview **antes** de cualquier escritura; nada se ha escrito todavía.

### Check 2 — Clasificación de artefactos (todo nuevo)
- **Comando:** Correr el preview en el escenario (a) (sin `.claude/` previo).
- **Esperado:** Todos los artefactos aparecen como **nuevos**; conteo correcto.

### Check 3 — Clasificación (sin cambios)
- **Comando:** Correr el preview en el escenario (b) (`.claude/` idéntico).
- **Esperado:** El reporte indica claramente **sin cambios** para los artefactos; comunica que no hay nada que escribir.

### Check 4 — Clasificación y detalle (modificados)
- **Comando:** Correr el preview en el escenario (c) (definición canónica cambiada).
- **Esperado:** Los artefactos afectados aparecen como **modificados**, con **detalle del cambio** (diff de contenido) legible; los no afectados, como sin cambios.

### Check 5 — Confirmación explícita escribe
- **Comando:** En el preview con cambios, **confirmar**.
- **Esperado:** Recién entonces se escriben los artefactos; el resultado coincide con lo previsualizado.

### Check 6 — Cancelar no escribe nada
- **Comando:** En el preview con cambios, **cancelar**.
- **Esperado:** **No se escribe nada**; el disco queda como antes.

### Check 7 — Modo solo-preview
- **Comando:** Invocar el modo solo-preview.
- **Esperado:** Muestra el diff y **sale sin escribir**.

### Check 8 — Legibilidad y controles
- **Comando:** Recorrer el reporte con los controles de navegación.
- **Esperado:** Presentación estética y consistente (colores, resúmenes, conteos); atajos claros para confirmar / cancelar / navegar.

## Mapeo a criterios

| Check | Criterio de aceptación |
|---|---|
| 1 | El usuario ve un reporte de diff/preview antes de que se escriba algo. |
| 2, 3, 4 | Cada artefacto se clasifica como nuevo / modificado / sin cambios. |
| 4 | Para modificados, se muestra el detalle del cambio de forma legible. |
| 5, 6 | Confirmación explícita; cancelar no escribe nada. |
| 7 | Existe modo solo-preview que no escribe. |
| 3 | Cuando no hay cambios, el reporte lo indica claramente. |
| 8 | Presentación legible y consistente, con controles claros. |

## Verdict

**Estado:** **APPROVED** — validated by Leia on 2026-06-25.

All 8 checks pass across three independent evidence paths: (1) the key-driven test
suite (`go test ./internal/tui/...` — 18 build-preview tests green, exercising the
exact keys y/enter/n/esc/up/down and every state), (2) real `View()` frames captured
from a disposable harness driving a fabricated plan (created + modified + unchanged +
orphan) through the ready/modified-diff/unchanged/written/cancelled/empty/error/
read-only states, and (3) the CLI wiring run headless against a real `.daedalus/`
workspace (2 agents + 1 prompt, claude-code backend). Code health is clean:
`go build ./...`, `go test ./... -count=1`, `go vet ./...` all pass; `gofmt -l internal cmd`
is empty. The 06-01 contract change (non-TTY `build` no longer writes without `--yes`;
`TestRunBuildCompilesClaudeArtifacts` updated to `--yes`) is coherent and covered by
dedicated dry-run / preview / write tests.

Evidence per check:
- Check 1 — `.claude/` absent before confirmation (TestConfirmWritesToDisk); `--preview`
  leaves no `.claude/` (CLI, Test-Path=False).
- Check 2 — all-new plan classifies every artifact `[new]`; CLI preview shows "4 new".
- Check 3 — empty state renders "No changes — every artifact is already up to date";
  second `--yes` run is idempotent ("0 created, 0 updated, 4 unchanged").
- Check 4 — modified artifact shows a readable line diff with `-`/`+` and context, both
  in the TUI frame and the CLI preview (`- description: …spec/PRD.` / `+ …CHANGED…`).
- Check 5 — confirm (y/enter) writes; written summary matches the preview (4 created).
- Check 6 — cancel (n/esc) ends as Cancelled and leaves disk untouched; non-TTY dry-run
  writes nothing.
- Check 7 — read-only `--preview` has a nil buildFn (structurally cannot write), gate
  says "nothing will be written"; CLI `--preview` exits 0 with no `.claude/`.
- Check 8 — consistent chrome/title, per-backend counts, status badges, contextual help
  footer reflecting the active state, scroll hint, no dead-ends. Error states are
  actionable; absent workspace → exit 4 + "run 'daedalus init'"; invalid definition →
  exit 3 + finding-by-finding message, nothing written.

The 3 `minor` cosmetics found in the first pass were polished by Padmé (presentation
only, no behavior change) and re-verified by Leia on 2026-06-25 — all three are now
**RESOLVED**. The full suite stayed green (`go test ./internal/tui/... -count=1` and
`go test ./... -count=1`), `gofmt -l internal cmd` empty. Re-check evidence (real frame,
ready state with a modified artifact selected + two orphans):

```
╭────────────────────────────────────────────────────────╮
│ claude-code: 1 new, 1 modified, 1 unchanged, 2 orphans │   ← border flush (fix #1)
╰────────────────────────────────────────────────────────╯

  [new]        .claude/agents/analyst.md
> [modified]   .claude/agents/coder.md                        ← cursor only on navigable rows
  [unchanged]  .claude/agents/idle.md
                                                              ← paths aligned in one column (fix #2)
Orphans — left untouched · not selectable                     ← separated, subtle, no cursor (fix #3)
[orphan]     .claude/agents/removed-old.md
[orphan]     .claude/commands/gone.md
```

### Hallazgos

| # | Severidad | Check | Observado | Esperado | Estado |
|---|---|---|---|---|---|
| 1 | minor | 8 | Summary box right border sat one cell short on ANSI-colored count lines. | Border flush regardless of color codes. | **RESOLVED** — `summaryBoxView` normalizes visible width via `lipgloss.Width` before bordering; frame shows the `│` flush on the colored count line. |
| 2 | minor | 8 | `[orphan]` badge not column-aligned with the fixed-width status badges, shifting its path one column left. | All badges padded to one fixed width so paths align. | **RESOLVED** — `padBadge` pads every label to `badgeWidth=11`; all paths now start in the same column. |
| 3 | minor | 8 | Orphans in the list were non-navigable with no detail, reading as a possible cursor dead-spot. | Make the orphans' non-interactive nature self-evident. | **RESOLVED** — separate subtle section headed "Orphans — left untouched · not selectable"; no cursor on those rows. |

> Severidad: `blocker` / `major` / `minor`. Un hallazgo por fila. Si `REJECTED`, el orquestador traslada estos hallazgos a `observations.md`.
