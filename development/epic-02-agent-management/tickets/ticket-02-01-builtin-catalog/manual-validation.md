# Validación manual — Ticket 02-01 — Catálogo built-in de agentes

> Para alguien **sin experiencia en testing**. Seguí los pasos tal cual. Cada caso te dice qué hacer y qué deberías ver.

## Preparación

1. Conseguí el binario de Daedalus ya compilado (o pedíle al orquestador cómo obtenerlo).
2. Abrí una terminal en una carpeta de prueba **vacía** (no uses un repo importante).
3. Asegurate de que en esa carpeta todavía **no** exista una carpeta `.daedalus/`.

## Casos

### Caso 1 — Ver la lista de agentes built-in
- **Hacé:** ejecutá el comando de Daedalus que lista los agentes del catálogo built-in.
- **Esperá ver:** una lista que incluye al menos estos cinco nombres: `analyst`, `architect`, `planner`, `validator`, `documenter`, cada uno con una breve descripción de su rol.

### Caso 2 — Materializar un agente en el workspace
- **Hacé:** ejecutá el comando para materializar (agregar) el agente `analyst` a tu carpeta de prueba.
- **Esperá ver:** que se crea la carpeta `.daedalus/agents/` y dentro aparece la definición del agente `analyst` (un archivo de configuración y su prompt).

### Caso 3 — Intentar materializar el mismo agente otra vez
- **Hacé:** volvé a ejecutar el mismo comando del Caso 2 (materializar `analyst`) en la misma carpeta.
- **Esperá ver:** que Daedalus **no** lo pisa en silencio: te avisa que ya existe y/o te pide confirmación o te muestra una vista previa. **No** debería sobreescribir sin avisar.

## Si algo no coincide

Si algo no coincide con lo esperado → **reportar al orquestador (Workflow B)**, describiendo qué hiciste, qué esperabas ver y qué viste en su lugar.
