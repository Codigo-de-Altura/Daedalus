# Validación — Prompt Preview (TUI)

> La corre **Leia** (validadora frontend). Solo reporta; no implementa ni arregla.

## Precondiciones

- TUI de Daedalus compilable y ejecutable (`go build ./...` y binario que arranca).
- Workspace `.daedalus/prompts/` con prompts de prueba: uno con inclusiones válidas (texto compuesto largo) y uno con error de composición (ciclo o referencia inexistente).
- Core de composición (ticket-03-02) disponible para alimentar la preview.

## Checks

### Check 1 — Compila y arranca la TUI
- **Comando:** `go build ./...` y luego ejecutar el binario de la TUI.
- **Esperado:** compila sin errores y la TUI arranca sin pánico.

### Check 2 — Render de la vista de preview
- **Comando:** navegar a un prompt y abrir su preview.
- **Esperado:** la vista de preview se muestra con el contenido renderizado; no hay glitches de layout.

### Check 3 — Contenido compuesto y renderizado con Glamour
- **Comando:** abrir la preview de un prompt con inclusiones y Markdown (encabezados, listas, código).
- **Esperado:** se ve el texto **compuesto** (inclusiones resueltas), con el Markdown formateado por Glamour; no aparecen las directivas de inclusión crudas.

### Check 4 — Scroll en contenido largo
- **Comando:** abrir un prompt cuyo contenido excede la altura del viewport y usar las teclas de desplazamiento.
- **Esperado:** el contenido se desplaza correctamente arriba/abajo sin recortes ni saltos erráticos.

### Check 5 — Atajos y ayuda contextual
- **Comando:** usar los atajos para abrir/cerrar la preview y desplazarse; abrir la ayuda.
- **Esperado:** los atajos funcionan y aparecen documentados en la ayuda contextual.

### Check 6 — Solo lectura
- **Comando:** estando en la preview, intentar escribir/editar.
- **Esperado:** la preview no modifica el prompt; no hay entrada de edición.

### Check 7 — Error de composición
- **Comando:** abrir la preview de un prompt con ciclo o referencia inexistente.
- **Esperado:** se muestra un mensaje de error legible; la TUI no crashea ni muestra contenido corrupto.

### Check 8 — Estilo consistente
- **Comando:** comparar visualmente la preview con otras vistas de la TUI.
- **Esperado:** tema, colores y márgenes (Lipgloss) consistentes con el resto de la app.

## Mapeo a criterios

| Check | Criterio de aceptación |
|---|---|
| 2 | Abrir preview de un prompt seleccionado |
| 3 | Texto compuesto renderizado con Glamour |
| 4 | Scroll en contenido largo |
| 5 | Atajos funcionan y figuran en la ayuda |
| 6 | Preview solo lectura |
| 7 | Error de composición legible, sin crash |
| 8 | Estilo consistente con la TUI |
| 1 | Build y arranque de la TUI |

## Verdict

**Estado:** _APPROVED / REJECTED_ (a completar por Leia al ejecutar).

**Hallazgos** (uno por ítem):

| # | Severidad | Observado | Esperado |
|---|---|---|---|
| | | | |
