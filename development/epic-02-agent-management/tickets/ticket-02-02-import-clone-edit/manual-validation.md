# Validación manual — Ticket 02-02 — Importar / clonar y editar un agente

> Para alguien **sin experiencia en testing**. Seguí los pasos tal cual. Cada caso te dice qué hacer y qué deberías ver.

## Preparación

1. Conseguí el binario de Daedalus ya compilado (o pedíle al orquestador cómo obtenerlo).
2. Abrí una terminal en una carpeta de prueba **vacía** (no uses un repo importante).
3. El catálogo built-in (ticket 02-01) debe estar disponible en el binario.

## Casos

### Caso 1 — Clonar un agente del catálogo
- **Hacé:** ejecutá el comando para clonar el agente `analyst` del catálogo a un nombre nuevo, por ejemplo `analyst-custom`.
- **Esperá ver:** que aparece una nueva definición de agente en `.daedalus/agents/` con el nombre `analyst-custom`.

### Caso 2 — Editar el clon
- **Hacé:** editá el agente `analyst-custom` cambiándole el rol, el prompt y algún parámetro (con el comando de edición de Daedalus).
- **Esperá ver:** que al volver a mirar `analyst-custom` los cambios quedaron guardados (rol, prompt y parámetro nuevos).

### Caso 3 — Confirmar que el original no cambió
- **Hacé:** mirá la definición del agente `analyst` del catálogo built-in (el original).
- **Esperá ver:** que el `analyst` original **sigue igual**, sin ninguno de los cambios que hiciste en el clon.

### Caso 4 — Clonar sobre un nombre que ya existe
- **Hacé:** intentá clonar otra vez usando el mismo nombre `analyst-custom`.
- **Esperá ver:** que Daedalus **no** lo pisa en silencio: te avisa que ya existe y/o te pide confirmación o te muestra una vista previa.

## Si algo no coincide

Si algo no coincide con lo esperado → **reportar al orquestador (Workflow B)**, describiendo qué hiciste, qué esperabas ver y qué viste en su lugar.
