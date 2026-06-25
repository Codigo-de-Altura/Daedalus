# Validación — Build idempotente y no destructivo

> La corre **Yoda** (validador backend). Solo reporta; no implementa ni arregla.

---

## Precondiciones

- Binario `daedalus` compilado y disponible.
- Repo de prueba con `.daedalus/` válido (`daedalus.yaml` → Claude Code) y artefactos ya generables por el build.
- Conjunto de **golden files** de referencia para ese workspace (o generables en la primera corrida).
- Herramienta de diff de archivos/directorios (`git diff --no-index`, `fc`, o comparación de hashes).

## Checks numerados

### Check 1 — Idempotencia: dos corridas sin cambios → diff vacío
- **Comando:** Correr `daedalus build`, snapshot del output; correr `daedalus build` otra vez; comparar contra el snapshot.
- **Esperado:** Output **idéntico** entre la primera y la segunda corrida; diff vacío; ninguna escritura espuria.

### Check 2 — Idempotencia repetida (N corridas)
- **Comando:** Correr `daedalus build` tres o más veces seguidas sin tocar la definición canónica; comparar todos los outputs.
- **Esperado:** Todos los outputs idénticos entre sí.

### Check 3 — Determinismo (golden files): mismo input → mismo output
- **Comando:** Correr `daedalus build` sobre el mismo workspace en dos directorios limpios distintos; comparar (`git diff --no-index dirA dirB`).
- **Esperado:** Outputs **idénticos byte-a-byte**; sin diferencias de orden, formato ni datos volátiles.

### Check 4 — Determinismo contra golden de referencia
- **Comando:** Comparar el output del build contra los golden files de referencia.
- **Esperado:** Coincidencia exacta. Cualquier diferencia es un hallazgo.

### Check 5 — Sin datos volátiles
- **Comando:** Inspeccionar los artefactos generados buscando timestamps, rutas absolutas, orden aleatorio de claves u otros datos dependientes del entorno.
- **Esperado:** Ninguno presente; el contenido solo depende de la definición canónica.

### Check 6 — No-destrucción de cambios manuales fuera del área gestionada
- **Comando:** Crear/editar un archivo manual **fuera del área gestionada** por el build (p. ej. un archivo del usuario que el build no produce); correr `daedalus build`; verificar ese archivo.
- **Esperado:** El cambio manual **se preserva intacto** tras la re-compilación.

### Check 7 — El área gestionada se regenera de forma acotada
- **Comando:** Modificar la definición canónica de modo que cambie un artefacto; correr `daedalus build`; revisar qué cambió.
- **Esperado:** Solo cambia el contenido del **área gestionada** afectada; nada fuera de ella se toca; el resultado es reproducible.

### Check 8 — Comportamiento por defecto no destructivo
- **Comando:** Revisar que, por defecto, el build no borra ni sobre-escribe contenido fuera del área gestionada.
- **Esperado:** Default seguro (RNF-8); ninguna destrucción inadvertida.

## Mapeo a criterios

| Check | Criterio de aceptación |
|---|---|
| 1, 2 | Dos corridas sin cambios dejan el output idéntico (idempotente; diff vacío). |
| 6 | Cambio manual fuera del área gestionada sobrevive a la re-compilación. |
| 3, 4 | Mismo input canónico → mismo output byte-a-byte (determinismo / golden files). |
| 5 | El build no escribe datos volátiles. |
| 7 | El área gestionada está delimitada y solo ella se regenera. |
| 8 | Comportamiento por defecto no destructivo. |

## Verdict

**Estado:** **APPROVED** — validated by Yoda on 2026-06-25 against the test binary
`go build -o $TEMP/daedalus-06.exe ./cmd/daedalus` (go 1.26.4 windows/amd64) and the
package tests. Validated against the orchestrator-confirmed scope: managed area = the
exact produced path set; compare-and-skip idempotency (no mtime churn); orphans
detected + reported but NEVER deleted (safe default; auto-delete is RF-6.4).

### Evidence (real commands + exit codes)

Health:
- `go build ./...` → exit 0 · `go vet ./...` → exit 0 · `gofmt -l internal cmd` → empty
- `go test ./... -count=1` → all packages ok
- `go test ./internal/compile/... -count=1 -v` → all 24 tests PASS, including
  `TestIdempotentSecondRunNoChanges`, `TestIdempotentManyRunsIdentical`,
  `TestNonDestructionForeignFiles`, `TestBoundedRegeneration`,
  `TestOrphanDetectedNotDeleted`, `TestPlanArtifactsClassifies`,
  `TestDeterministicAcrossCleanDirs`. 06-02 goldens (`TestClaudeGolden`) still pass.

Per check (real workspace: 2 agents analyst+architect, 2 prompts commit-policy[global]
+ style-guide[shared], backend claude-code; 5 artifacts):

- Check 1/2 — idempotency: build #1 = "5 created"; builds #2 and #3 = "0 created,
  0 updated, 5 unchanged", exit 0 each. Snapshot across all 3 runs: every artifact's
  SHA-256 **and mtime** stable → compare-and-skip writes nothing on an unchanged run
  (no mtime churn, no spurious writes). PASS.
- Check 3 — determinism: same workspace built into two clean dirs A, B;
  `git diff --no-index A/.claude B/.claude` → exit 0, no output (byte-identical). PASS.
- Check 4 — golden reference: `TestClaudeGolden` passes against
  `internal/compile/testdata/golden/.claude/`. PASS.
- Check 5 — no volatile data: grep of the generated `.claude/` for absolute paths,
  ISO timestamps, AppData, temp markers, long epoch numbers → no matches. PASS.
- Check 6 — non-destruction: a manual `.claude/agents/manual-note.md` (inside the
  managed tree, not produced) AND a `MY-NOTES.md` (outside `.claude/`) both survive a
  rebuild byte-identical; the in-tree manual file is reported as an orphan ("left
  untouched"), never deleted. Produced 5 stay unchanged. PASS.
- Check 7 — bounded regeneration: editing the analyst canonical prompt → "1 updated,
  4 unchanged"; exactly `.claude/agents/analyst.md` changed (content + mtime); the
  other 4 artifacts unchanged with stable mtime; nothing outside the managed area
  touched. PASS.
- Check 8 — safe default: removing the architect canonical source made
  `.claude/agents/architect.md` an orphan; the rebuild DETECTED + REPORTED it
  (`orphans:1`, "left untouched") and PRESERVED it byte-identical — did NOT delete it.
  Default is non-destructive. PASS.

### Hallazgos

| # | Severidad | Check | Observado | Esperado |
|---|---|---|---|---|
| 1 | info | 6, 8 | Orphans (files in a managed dir no longer produced) are detected and reported as `? ... (orphan: ... left untouched)` but not deleted. | Confirmed CORRECT per scope: safe default in 06-03; auto-delete/fine-grained handling is RF-6.4. Not a defect. |

> Severidad: `blocker` / `major` / `minor`. Un hallazgo por fila. Si `REJECTED`, el orquestador traslada estos hallazgos a `observations.md`.
