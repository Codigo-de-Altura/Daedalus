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

**Estado:** _APPROVED_ / _REJECTED_ — _(a completar por Yoda tras ejecutar los checks.)_

### Hallazgos

| # | Severidad | Check | Observado | Esperado |
|---|---|---|---|---|
| | | | | |

> Severidad: `blocker` / `major` / `minor`. Un hallazgo por fila. Si `REJECTED`, el orquestador traslada estos hallazgos a `observations.md`.
