# Validación — Formatos diff-friendly

> La corre **Yoda** (validador backend). Solo reporta; no implementa ni corrige.

---

## Precondiciones

- Existe un repo con un workspace `.daedalus/` inicializado y estado relevante presente.
- El repo es un repositorio git válido (`git status` funciona).
- Está disponible el plano `diff-friendly-formats.md` (origen RF-8.2).

## Checks

### Check 1 — YAML con claves ordenadas
- **Comando:** Inspeccionar uno o más artefactos YAML del workspace (`.daedalus/**/*.yaml`) y verificar el orden de claves contra el orden canónico/ordenado definido en el plano.
- **Esperado:** Las claves aparecen en un orden estable y reproducible (canónico o lexicográfico definido), no en orden arbitrario.

### Check 2 — Determinismo de serialización (mismo input → mismo output)
- **Comando:** Ejecutar dos veces la operación de serialización/escritura sobre el mismo input, capturando el output; comparar byte a byte (p. ej. hash de los archivos generados).
- **Esperado:** Output idéntico byte a byte entre ejecuciones; mismos hashes.

### Check 3 — Idempotencia (cero diff al reescribir)
- **Comando:** Reescribir un artefacto sin cambios semánticos y luego `git diff --stat` / `git diff` sobre la ruta.
- **Esperado:** Cero diff (sin reordenamiento de claves ni reformateo espurio).

### Check 4 — Sin campos volátiles en el estado canónico
- **Comando:** Buscar timestamps, rutas absolutas de máquina e identificadores aleatorios dentro de los artefactos de estado canónico versionado.
- **Esperado:** No hay valores volátiles embebidos en el estado relevante versionado.

### Check 5 — Estilo de formato YAML consistente
- **Comando:** Revisar indentación, comillas y estilo de bloque en varios archivos YAML del workspace.
- **Esperado:** Estilo consistente y fijo en todo el workspace, acorde al plano.

### Check 6 — Markdown legible y estable
- **Comando:** Reescribir un documento Markdown del backlog sin cambios semánticos y `git diff` sobre la ruta; revisar encabezados jerárquicos y tablas de metadatos.
- **Esperado:** Estructura legible y estable; sin reformateo que altere líneas no editadas; cero diff al reescribir sin cambios.

### Check 7 — Verificable por golden files
- **Comando:** Comparar el output contra golden files de referencia (RNF-5).
- **Esperado:** El output coincide exactamente con los golden files.

### Check 8 — No hay estado fuera de archivos git-trackeados
- **Comando:** `git status --porcelain --ignored .daedalus/` para confirmar que la serialización no produce estado relevante por fuera del área git-trackeada.
- **Esperado:** Todo lo serializado como estado relevante queda dentro de `.daedalus/` git-trackeado; nada relevante aparece como ignorado o fuera del repo.

## Mapeo a criterios

| Check | Criterio de aceptación (spec) | RF / RNF |
|---|---|---|
| 1 | YAML emite claves en orden estable | RF-8.2 |
| 2 | Mismo input → mismo output byte a byte | RF-8.2 / RNF-5 |
| 3 | Reescritura sin cambios = cero diff (idempotente) | RF-8.2 / RNF-6 |
| 4 | Sin volátiles en el estado canónico | RF-8.2 |
| 5 | Estilo de formato YAML fijo y consistente | RF-8.2 / RNF-6 |
| 6 | Markdown legible y estable | RF-8.2 / RNF-6 |
| 7 | Output verificable por golden files | RNF-5 |
| 8 | Sin estado fuera de archivos git-trackeados | RF-8.1 |

## Verdict

**APPROVED / REJECTED**

Hallazgos (uno por ítem):

| # | Severidad | Observado | Esperado |
|---|---|---|---|
| | | | |
