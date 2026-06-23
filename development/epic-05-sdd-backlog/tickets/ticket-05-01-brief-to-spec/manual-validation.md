# Validación manual — Ticket 05-01

> Para alguien **sin background de testing**. Seguí los pasos y compará lo que ves con lo que se espera. Si algo no coincide → avisá al **orquestador** (Workflow B).

## Preparación

1. Tené un repo de prueba con un workspace `.daedalus/` ya inicializado.
2. Tené a mano el binario `daedalus` (o el comando que te indique el orquestador para correrlo).
3. Abrí una terminal en la raíz de ese repo.

## Casos

### Caso 1 — Capturar un brief

- **Hacé:** Capturá un brief de prueba (por ejemplo, una descripción corta de una idea de producto) con la operación de brief de Daedalus.
- **Esperá ver:** Un archivo markdown de brief guardado dentro del workspace, con el texto que ingresaste.

### Caso 2 — El brief queda conectado al agente analyst

- **Hacé:** Revisá la definición/relación del brief que acabás de crear.
- **Esperá ver:** El brief aparece asociado al agente *analyst* (el paso "brief → spec" del flujo por defecto).

### Caso 3 — Lugar de la spec

- **Hacé:** Mirá dónde quedaría/queda la spec.
- **Esperá ver:** Una ruta dentro de `.daedalus/specs/` con un nombre en kebab-case (palabras en minúscula separadas por guiones).

### Caso 4 — Daedalus NO corre el agente

- **Hacé:** Completá la operación de principio a fin.
- **Esperá ver:** Daedalus **no** lanza ningún agente ni proceso de IA. Solo deja la definición lista. La generación real del contenido la harías vos corriendo el agente en tu backend.

### Caso 5 — La spec es tuya para editar

- **Hacé:** Editá a mano una spec materializada y volvé a correr la operación de brief.
- **Esperá ver:** Tus cambios manuales siguen ahí; Daedalus no los pisa sin avisarte (te ofrece preview/confirmación).

## Si algo no coincide

Anotá qué esperabas y qué viste, y reportalo al **orquestador** (Workflow B) para que delegue el fix.
