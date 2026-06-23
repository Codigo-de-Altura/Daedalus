# Validación manual — Estética, render de markdown y formularios

> Para alguien **sin experiencia en testing**. Seguí los pasos tal cual. Si algo no coincide con lo esperado, no intentes arreglarlo: avisá al **orquestador** (Workflow B).

## Preparación

1. Abrí una terminal en el repositorio del proyecto.
2. Arrancá la TUI de Daedalus (el orquestador te dará el comando si no lo sabés).
3. Asegurate de poder llegar a una vista que muestre un documento (markdown) y a una vista con un formulario.

## Casos

### Caso 1 — Todo se ve coherente
- **Hacé:** recorré algunas áreas de la TUI.
- **Esperá ver:** un estilo consistente (colores, títulos, resaltado de lo seleccionado) que se siente como una sola aplicación pulida, no pantallas con apariencias distintas.

### Caso 2 — Leer un documento bien formateado
- **Hacé:** abrí un documento (por ejemplo una spec o un prompt) dentro de la TUI.
- **Esperá ver:** el texto formateado y legible: títulos destacados, listas con viñetas, y si hay tablas o bloques de código, que se vean ordenados (no como texto plano con símbolos crudos).

### Caso 3 — Completar un formulario correcto
- **Hacé:** abrí un formulario, completá los campos con datos válidos y enviá.
- **Esperá ver:** el formulario se acepta y el flujo continúa.

### Caso 4 — Formulario con error
- **Hacé:** abrí un formulario e intentá enviarlo dejando vacío un campo obligatorio (o poniendo algo claramente inválido).
- **Esperá ver:** un mensaje de error claro que te dice qué corregir; no te deja avanzar hasta arreglarlo.

### Caso 5 — Cancelar sin quedar atrapado
- **Hacé:** abrí un formulario y cancelalo a mitad de camino.
- **Esperá ver:** volvés a la pantalla anterior sin problemas; no quedás trabado.

## Si algo no coincide

Si cualquier caso no se comporta como dice "Esperá ver", reportalo al **orquestador** (Workflow B) describiendo qué hiciste y qué viste.
