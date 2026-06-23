# Validación manual — Prompt Preview (TUI)

> Para alguien **sin experiencia en testing**. Seguí los pasos tal cual. Si algo no coincide con lo que dice "Esperá ver", anotalo y avisá.

## Preparación

1. Pedí a quien te ayuda el binario de Daedalus ya compilado y una carpeta de prueba con algunos prompts cargados (uno largo con texto reutilizado y uno "roto").
2. Abrí una terminal en esa carpeta.
3. Arrancá la TUI de Daedalus. Debería abrirse la interfaz en la terminal sin errores.

## Casos

### Caso 1 — Abrir la previsualización

- **Hacé:** navegá hasta la lista de prompts, elegí un prompt y abrí su **previsualización (preview)**.
- **Esperá ver:** una pantalla que muestra el texto del prompt ya **formateado** (títulos, listas y negritas se ven bonitos, no como símbolos sueltos).

### Caso 2 — El texto aparece completo y armado

- **Hacé:** mirá el contenido del prompt que tiene partes reutilizadas (incluye otros fragmentos).
- **Esperá ver:** el texto **completo y ensamblado**, con los fragmentos ya metidos adentro. No debe aparecer ninguna línea rara tipo "incluir tal cosa" sin resolver.

### Caso 3 — Desplazarse en un prompt largo

- **Hacé:** elegí un prompt largo y usá las teclas de flecha (o las indicadas en la ayuda) para bajar y subir.
- **Esperá ver:** el contenido se mueve suavemente; podés llegar al final y volver al principio.

### Caso 4 — Atajos y ayuda

- **Hacé:** abrí la ayuda de la pantalla y probá los atajos para cerrar la preview.
- **Esperá ver:** la ayuda lista los atajos; al usarlos, la preview se cierra y volvés a la lista.

### Caso 5 — No se puede editar desde la preview

- **Hacé:** dentro de la preview, intentá escribir o borrar texto.
- **Esperá ver:** **no pasa nada** sobre el prompt; la preview es solo para mirar.

### Caso 6 — Un prompt con error

- **Hacé:** abrí la preview del prompt "roto" (el que tiene una referencia que no existe o un bucle).
- **Esperá ver:** un **mensaje de error claro** explicando que no se pudo armar el prompt. La aplicación **no** se cierra sola ni muestra texto corrupto.

## Si algo no coincide

Si algún resultado no coincide con lo esperado, no intentes arreglarlo: avisá al **orquestador** describiendo qué hiciste y qué viste (Workflow B).
