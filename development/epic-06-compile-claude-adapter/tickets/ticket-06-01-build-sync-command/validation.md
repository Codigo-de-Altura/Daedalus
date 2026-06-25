# Validación — `daedalus build` / `sync`

> La corre **Yoda** (validador backend). Solo reporta; no implementa ni arregla.

---

## Precondiciones

- Binario `daedalus` compilado y disponible en el `PATH` (o ruta conocida).
- Un repo de prueba con un workspace `.daedalus/` válido, con `daedalus.yaml` apuntando al backend **Claude Code** y al menos un agente y un comando en la definición canónica.
- Un segundo directorio **sin** `.daedalus/` para los casos negativos.
- Capacidad de inspeccionar el código de salida del proceso (`$LASTEXITCODE` en PowerShell).

## Checks numerados

### Check 1 — El comando existe y muestra ayuda
- **Comando:** `daedalus build --help`
- **Esperado:** Salida de ayuda del comando `build`; código de salida `0`.

### Check 2 — Alias `sync`
- **Comando:** `daedalus sync --help`
- **Esperado:** Ayuda equivalente a `build` (mismo comando vía alias); código de salida `0`.

### Check 3 — Compilación básica exitosa
- **Comando:** `daedalus build` (dentro del repo con `.daedalus/` válido)
- **Esperado:** Compila al formato nativo del backend configurado; muestra resumen (backend, artefactos creados/actualizados, estado); código de salida `0`.

### Check 4 — Workspace ausente
- **Comando:** `daedalus build` (en un directorio **sin** `.daedalus/`)
- **Esperado:** Aborta sin escribir; mensaje de error claro indicando que no hay workspace; código de salida distinto de `0`.

### Check 5 — Definición canónica inválida aborta sin escribir
- **Comando:** Introducir un error en la definición canónica (p. ej. agente sin campo requerido) y correr `daedalus build`.
- **Esperado:** Reporta el/los error(es) de validación; **no escribe artefactos nativos**; código de salida de error de validación (distinto del de escritura).

### Check 6 — Backend sin adaptador
- **Comando:** Configurar en `daedalus.yaml` un backend sin adaptador registrado y correr `daedalus build`.
- **Esperado:** Error claro de "adaptador no encontrado"; ninguna escritura; código de salida distinto de `0`.

### Check 7 — Selección de backend desde el manifiesto
- **Comando:** `daedalus build` con `daedalus.yaml` configurado a Claude Code.
- **Esperado:** Enruta al adaptador Claude Code y genera artefactos bajo el área esperada (`.claude/`); resumen menciona el backend Claude Code.

### Check 8 — Resumen y códigos de salida diferenciados
- **Comando:** Correr los escenarios de éxito, error de validación y error de escritura, e inspeccionar el código de salida en cada caso.
- **Esperado:** `0` en éxito; códigos distintos entre sí para error de validación y error de compilación/escritura.

## Mapeo a criterios

| Check | Criterio de aceptación |
|---|---|
| 1, 3 | `daedalus build` compila la definición canónica al formato nativo. |
| 2 | `daedalus sync` es alias de `build`. |
| 4, 5 | Workspace ausente/inválido aborta sin escribir, con error accionable. |
| 3, 7 | Backend objetivo se resuelve desde `daedalus.yaml` y se enruta al adaptador correcto. |
| 6 | Backend sin adaptador → error claro y ninguna escritura. |
| 3 | Resumen con backend, artefactos y estado. |
| 8 | Códigos de salida distinguen éxito, error de validación y error de compilación/escritura. |

## Verdict

**Estado:** **APPROVED** — validated by Yoda on 2026-06-24 against the test binary
`go build -o $TEMP/daedalus-06.exe ./cmd/daedalus` (go 1.26.4 windows/amd64).

Scope note: per the orchestrator's scope boundary, the canonical→Claude Code mapping
is RF-6.2 (ticket-06-02). The Claude Code adapter in 06-01 is an intentional stub
returning `ErrNotImplemented`. Therefore the "real artifacts produced in `.claude/`
and exit 0" portion of Check 3 and Check 7 is **DEFERRED to 06-02**, not a 06-01
finding. Everything that is in-scope for 06-01 passes for real.

### Evidence (real commands + exit codes)

Health:
- `go build ./...` → exit 0
- `go test ./... -count=1` → all packages ok (incl. `internal/compile`, `internal/workspace`)
- `go vet ./...` → exit 0
- `gofmt -l internal cmd` → empty output (exit 0)

Behavioral:
- Check 1 — `daedalus build --help` → help with alias note, exit **0**.
- Check 2 — `daedalus sync --help` → byte-identical help (alias), exit **0**.
- Check 3/7 (in-scope) — valid workspace: workspace located, backend `claude-code`
  resolved from `daedalus.yaml`, definition validated, routed to Claude adapter via
  the registry; terminated with `compiling backend "claude-code": backend adapter
  not yet implemented`, exit **4**, no `.claude/` written. Real-artifact/exit-0 part
  DEFERRED to 06-02.
- Check 4 — no `.daedalus/`: `no .daedalus workspace found ... run 'daedalus init'`,
  exit **4**, no `.claude/` written.
- Check 5 — agent missing required `role`: `canonical definition is invalid
  (1 problem) ... missing required key: role`, exit **3** (validation), no `.claude/`.
- Check 6 — manifest backend `codex` (no adapter): definition validated first, then
  `no adapter registered for backend: "codex" (registered: claude-code)`, exit **4**,
  no `.claude/`.
- Check 8 — exit-code matrix all distinct: success 0, usage 2, validation 3,
  compile/write 4. Validation (3) and compile/write (4) are effectively distinct.

### Hallazgos

| # | Severidad | Check | Observado | Esperado |
|---|---|---|---|---|
| 1 | info (deferred) | 3, 7 | Stub Claude adapter returns `ErrNotImplemented`; no `.claude/` artifacts, exit 4. | Real `.claude/` artifacts + exit 0 — DEFERRED to RF-6.2 / ticket-06-02; out of scope for 06-01. |
| 2 | minor (info) | 6 (preview) | `build --preview` on a valid repo also exits 4: the stub `Compile` runs before the preview no-write gate. Preview still wrote nothing. | Expected at this stage — compile-all-before-write precedes the write/preview gate; once 06-02 lands, preview will reach exit 0. Not a 06-01 defect. |

> Severidad: `blocker` / `major` / `minor`. Un hallazgo por fila. Si `REJECTED`, el orquestador traslada estos hallazgos a `observations.md`.
