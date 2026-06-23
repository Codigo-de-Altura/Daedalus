# Validación — Convenciones de equipo

> La corre **Yoda** (validador backend). Solo reporta; no implementa ni corrige.

---

## Precondiciones

- Existe un repo con un workspace `.daedalus/` inicializado y backlog SDD presente.
- El repo es un repositorio git válido (`git status` funciona).
- Está disponible el plano `team-conventions.md` (origen RF-8.3) con las convenciones explícitas.

## Checks

### Check 1 — Convenciones explícitas presentes
- **Comando:** Localizar la fuente referenciable de convenciones (naming, estructura, formato) y confirmar alineación con `init.md` §7 y CLAUDE.md §6.
- **Esperado:** Las convenciones están enunciadas de forma explícita y única; no son implícitas.

### Check 2 — Validación de naming (kebab-case + patrones de id)
- **Comando:** Correr la validación de convenciones sobre el workspace y revisar el chequeo de `kebab-case` y patrones `epic-NN-<slug>` / `ticket-NN-MM-<slug>` para epics, tickets, agentes y workflows. Probar también con un nombre deliberadamente inválido.
- **Esperado:** Nombres conformes pasan; un nombre inválido es detectado y reportado con su ubicación.

### Check 3 — Validación de estructura
- **Comando:** Correr la validación contra un workspace correcto y contra uno con un directorio/documento faltante o fuera de lugar.
- **Esperado:** El layout correcto pasa; faltantes o elementos fuera de lugar se detectan y reportan.

### Check 4 — Validación de formato
- **Comando:** Ejecutar el chequeo de coherencia de formato (YAML con claves ordenadas, Markdown estructurado) sobre artefactos del workspace.
- **Esperado:** Artefactos conformes pasan; desviaciones de formato se reportan.

### Check 5 — Validación de trazabilidad
- **Comando:** Verificar que la validación comprueba que cada ticket referencia su epic y cada epic sus RF de origen; probar con un ticket sin referencia a epic.
- **Esperado:** Trazabilidad presente pasa; ausencia de referencia se detecta y reporta.

### Check 6 — Reporte accionable
- **Comando:** Inspeccionar la salida de la validación ante violaciones inyectadas.
- **Esperado:** Cada violación reporta ubicación (archivo/elemento) y la convención incumplida, de forma accionable.

### Check 7 — Determinismo de la validación
- **Comando:** Correr la validación dos veces sobre el mismo workspace sin cambios y comparar la salida.
- **Esperado:** Mismo workspace → misma salida (orden y contenido estables); serialización del reporte determinista.

### Check 8 — No hay estado fuera de archivos git-trackeados
- **Comando:** `git status --porcelain --ignored .daedalus/` tras correr la validación, para confirmar que no genera estado relevante fuera del área git-trackeada.
- **Esperado:** La validación no persiste estado relevante fuera de `.daedalus/` git-trackeado.

## Mapeo a criterios

| Check | Criterio de aceptación (spec) | RF |
|---|---|---|
| 1 | Convenciones explicitadas en fuente referenciable | RF-8.3 |
| 2 | Validación de kebab-case y patrones de id | RF-8.3 |
| 3 | Validación de estructura del workspace/backlog | RF-8.3 |
| 4 | Validación de coherencia de formato | RF-8.3 / RF-8.2 |
| 5 | Validación de trazabilidad ticket→epic, epic→RF | RF-8.3 |
| 6 | Reporte accionable de violaciones | RF-8.3 |
| 7 | Validación determinista | RF-8.3 / RNF-5 |
| 8 | Sin estado fuera de archivos git-trackeados | RF-8.1 |

## Verdict

**APPROVED / REJECTED**

Hallazgos (uno por ítem):

| # | Severidad | Observado | Esperado |
|---|---|---|---|
| | | | |
