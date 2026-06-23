# Validación — Estado versionado en git

> La corre **Yoda** (validador backend). Solo reporta; no implementa ni corrige.

---

## Precondiciones

- Existe un repo con un workspace `.daedalus/` inicializado.
- El repo es un repositorio git válido (`git status` funciona).
- Está disponible el inventario de estado relevante definido en `git-versioned-state.md` (origen RF-8.1).

## Checks

### Check 1 — Inventario de estado relevante presente
- **Comando:** Revisar `git-versioned-state.md` y localizar el inventario de categorías de estado relevante con su ruta en `.daedalus/`.
- **Esperado:** Cada categoría (definiciones canónicas, backlog SDD, estado de progreso) tiene una ruta explícita dentro de `.daedalus/`.

### Check 2 — Todo el estado relevante está git-trackeado
- **Comando:** `git ls-files .daedalus/` y comparar con el inventario; luego `git status --porcelain --ignored .daedalus/` para detectar piezas de estado relevante marcadas como ignoradas (`!!`).
- **Esperado:** Toda ruta de estado relevante del inventario aparece en `git ls-files`; ninguna pieza de estado relevante aparece como ignorada.

### Check 3 — No hay estado fuera de archivos git-trackeados
- **Comando:** Verificar que Daedalus no escribe estado relevante fuera del repo: inspeccionar que no haya persistencia en `$HOME`, temporales del SO, bases de datos binarias locales ni dependencia exclusiva de variables de entorno. Buscar en la spec/implementación referencias a rutas absolutas de usuario/temporales para el estado.
- **Esperado:** No existe estado relevante persistido fuera de `.daedalus/` (es decir, fuera del área git-trackeada).

### Check 4 — `.daedalus/.state/` es git-tracked
- **Comando:** `git ls-files .daedalus/.state/` y `git check-ignore -v .daedalus/.state` (debe no devolver match de ignore).
- **Esperado:** `.daedalus/.state/` contiene archivos rastreados y **no** está cubierto por ninguna regla de `.gitignore` generada/sugerida por Daedalus.

### Check 5 — Formato de texto
- **Comando:** Verificar que los archivos de estado relevante son texto (YAML/Markdown), no binarios: `git ls-files .daedalus/ | <inspección de extensión/contenido>`.
- **Esperado:** Todo el estado relevante está en archivos de texto YAML/Markdown.

### Check 6 — Determinismo de serialización del estado
- **Comando:** Disparar dos veces la operación que escribe estado relevante sin cambios de input y comparar: `git diff --stat .daedalus/` tras la segunda escritura.
- **Esperado:** Mismo input → mismo output byte a byte; la segunda escritura no produce diff (sin ruido por reordenamiento de claves ni timestamps volátiles).

### Check 7 — Separación estado canónico vs. derivado
- **Comando:** Revisar el plano y confirmar que los artefactos derivados/efímeros (p. ej. salida de `build` en `.claude/`, caches) están explícitamente fuera del estado canónico versionado.
- **Esperado:** El plano distingue inequívocamente estado canónico de derivados; no se confunde lo derivado con estado relevante versionado.

## Mapeo a criterios

| Check | Criterio de aceptación (spec) | RF |
|---|---|---|
| 1 | Inventario explícito de estado relevante y rutas | RF-8.1 |
| 2 | Ninguna categoría de estado fuera de git-tracked | RF-8.1 |
| 3 | No hay estado fuera de archivos git-trackeados | RF-8.1 |
| 4 | `.daedalus/.state/` git-tracked y no ignorado | RF-8.1 |
| 5 | Estado en formatos de texto (YAML/Markdown) | RF-8.1 / RF-8.2 |
| 6 | Serialización determinista del estado | RF-8.1 / RNF-6 |
| 7 | Separación canónico vs. derivado | RF-8.1 |

## Verdict

**APPROVED / REJECTED**

Hallazgos (uno por ítem):

| # | Severidad | Observado | Esperado |
|---|---|---|---|
| | | | |
