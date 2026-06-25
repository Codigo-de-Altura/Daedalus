# Validación — Adaptador Claude Code (`Compiler`)

> La corre **Yoda** (validador backend). Solo reporta; no implementa ni arregla.

---

## Precondiciones

- Binario `daedalus` compilado y disponible.
- Repo de prueba con `.daedalus/` válido: al menos dos agentes y un comando en la definición canónica, `daedalus.yaml` → Claude Code.
- Un conjunto de **golden files** esperados para ese workspace (snapshot de referencia de `.claude/`), o capacidad de generarlos en la primera corrida y compararlos en las siguientes.
- Herramienta de diff de archivos disponible (p. ej. `git diff --no-index`, `fc`, o comparación de hashes).

## Checks numerados

### Check 1 — Existe la interfaz `Compiler` y el registro de adaptadores
- **Comando:** Inspección del código: ubicar la interfaz `Compiler` y el mecanismo de registro de adaptadores.
- **Esperado:** Una interfaz `Compiler` con contrato claro y un registro donde Claude Code está dado de alta. Añadir un adaptador no exige tocar el comando `build` ni el núcleo.

### Check 2 — Genera `.claude/agents/*.md` con frontmatter
- **Comando:** `daedalus build` y luego inspeccionar `.claude/agents/`.
- **Esperado:** Un archivo `.md` por agente canónico; cada uno con **frontmatter** válido (parseable) seguido del prompt. Los campos del frontmatter mapean a los campos canónicos del agente.

### Check 3 — Genera `.claude/commands/*.md`
- **Comando:** Inspeccionar `.claude/commands/` tras el build.
- **Esperado:** Archivos de comando derivados de la definición canónica.

### Check 4 — Genera settings de Claude Code
- **Comando:** Inspeccionar la ubicación de settings de Claude Code tras el build.
- **Esperado:** Settings relevantes presentes y bien formados.

### Check 5 — Nombres de archivo en `kebab-case` y estables
- **Comando:** Listar los archivos generados.
- **Esperado:** Nombres en `kebab-case` derivados del id canónico; estables entre corridas.

### Check 6 — Determinismo (golden files): mismo input → mismo output
- **Comando:** Correr `daedalus build` dos veces sobre el **mismo** workspace en directorios limpios distintos; comparar los outputs (`git diff --no-index dirA dirB` o comparar hashes de cada archivo).
- **Esperado:** Los dos outputs son **idénticos byte-a-byte**. Sin diferencias de orden de claves, formato, ni datos volátiles (timestamps, rutas absolutas, etc.).

### Check 7 — Determinismo contra golden de referencia
- **Comando:** Comparar el output de `daedalus build` contra el conjunto de **golden files** de referencia del workspace de prueba.
- **Esperado:** Coincidencia exacta con los golden files. Cualquier diferencia es un hallazgo.

### Check 8 — Extensibilidad de la interfaz (no-regresión del núcleo)
- **Comando:** Revisar que el comando `build` resuelve el adaptador vía el registro (no por un acoplamiento directo a Claude Code).
- **Esperado:** El núcleo y el comando son agnósticos; un backend nuevo se conecta registrando un `Compiler`.

## Mapeo a criterios

| Check | Criterio de aceptación |
|---|---|
| 1, 8 | Existe la interfaz `Compiler` y registro; agregar un backend no toca el núcleo. |
| 2 | `.claude/agents/*.md` con frontmatter válido (uno por agente). |
| 3 | `.claude/commands/*.md` desde la definición canónica. |
| 4 | Settings relevantes de Claude Code. |
| 5 | Nombres `kebab-case` estables respecto al id canónico. |
| 6, 7 | Determinismo verificable (golden files): mismo input → mismo output. |

## Verdict

**Estado:** **APPROVED** — validated by Yoda on 2026-06-25 against the test binary
`go build -o $TEMP/daedalus-06.exe ./cmd/daedalus` (go 1.26.4 windows/amd64) and the
package tests. Validated against the orchestrator-confirmed mapping decisions
(agents/commands/settings), not against invented expectations.

### Evidence (real commands + exit codes)

Health:
- `go build ./...` → exit 0
- `go vet ./...` → exit 0
- `gofmt -l internal cmd` → empty (exit 0)
- `go test ./... -count=1` → all packages ok
- `go test ./internal/compile/... -count=1 -v` → all 17 tests PASS, including
  `TestClaudeGolden`, `TestClaudeCompileDeterministic`, `TestClaudeSettingsIsValidJSON`,
  `TestAgentRoundTripBuildThenImport`, and the registry routing tests.

Per check (end-to-end via a real `.daedalus/` workspace: 2 agents analyst+architect,
2 prompts commit-policy[global]+style-guide[shared], backend claude-code):

- Check 1/8 — `Compiler` interface (`Backend()`/`Compile()`, pure, no I/O) +
  `Registry`/`DefaultRegistry`; `Build` routes via `reg.Lookup(backend)` with no direct
  Claude coupling. Registry tests use a `fakeCompiler`, proving a new backend is added
  by registration alone (núcleo untouched). PASS.
- Check 2 — `.claude/agents/analyst.md` and `architect.md`: one file per agent, YAML
  frontmatter PARSEABLE, fixed key order `name`/`description`/`model`, no `tools`/`color`,
  body = prompt. Mapping verified by a full CLI round-trip: importing the built
  `architect.md` back yields canonical id=architect, role intact, model=default param,
  prompt intact (`agent import` exit 0). PASS.
- Check 3 — `.claude/commands/commit-policy.md` (global) and `style-guide.md` (shared):
  commands derived from BOTH prompt kinds; `description` = title; body = resolved prompt.
  PASS.
- Check 4 — `.claude/settings.json` parsed with a real JSON parser (ConvertFrom-Json): OK.
  Top-level keys exactly `$schema` + `daedalus` (managed=true, generator=daedalus). NO
  fabricated permissions/env/hooks/model. Trailing newline present. PASS.
- Check 5 — file names kebab-case, derived from canonical id, stable across runs. PASS.
- Check 6 — determinism: built the SAME workspace into two clean dirs (A, B). All 5
  artifacts byte-identical by SHA-256; `git diff --no-index A/.claude B/.claude` → exit 0,
  no output. PASS.
- Check 7 — golden comparison: `TestClaudeGolden` passes against
  `internal/compile/testdata/golden/.claude/` (agents with/without model, role needing
  quoting, command with/without description, minimal settings). PASS.

End-to-end (06-01 Check 3/7 closure, now that the adapter is real): `daedalus build` on a
valid `.daedalus/` workspace produced a real `.claude/` (5 artifacts) and exited 0; the
summary named backend `claude-code` and listed every artifact created. VERIFIED.

### Hallazgos

| # | Severidad | Check | Observado | Esperado |
|---|---|---|---|---|
| 1 | info | 2 | Built-in catalog agents carry a `model: default` parameter, so live builds emit `model: default` (a real value, correctly NOT omitted). | Consistent with omit-empty semantics: `model` is emitted because a value exists; absent only when no model param. Confirmed correct, no action. |

> Severidad: `blocker` / `major` / `minor`. Un hallazgo por fila. Si `REJECTED`, el orquestador traslada estos hallazgos a `observations.md`.
