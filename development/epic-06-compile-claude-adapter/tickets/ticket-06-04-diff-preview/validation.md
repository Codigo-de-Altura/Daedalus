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

**Estado:** _APPROVED_ / _REJECTED_ — _(a completar por Leia tras ejecutar los checks.)_

### Hallazgos

| # | Severidad | Check | Observado | Esperado |
|---|---|---|---|---|
| | | | | |

> Severidad: `blocker` / `major` / `minor`. Un hallazgo por fila. Si `REJECTED`, el orquestador traslada estos hallazgos a `observations.md`.
