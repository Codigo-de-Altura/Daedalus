# Validación manual — Global & Shared Prompts

> Para alguien **sin experiencia en testing**. Seguí los pasos tal cual. Si algo no coincide con lo que dice "Esperá ver", anotalo y avisá.

## Preparación

1. Pedí a quien te ayuda el binario/comando de Daedalus ya compilado, o tener el proyecto listo para correr.
2. Abrí una terminal en una carpeta de prueba vacía (no uses un proyecto real).
3. Inicializá el workspace de Daedalus en esa carpeta (paso de `init`, si aplica). Debería aparecer una carpeta `.daedalus/`.

## Casos

### Caso 1 — Crear un prompt global

- **Hacé:** creá un prompt **global** con título "Estilo del proyecto" y un texto cualquiera en el cuerpo.
- **Esperá ver:** un mensaje de éxito, y dentro de `.daedalus/prompts/` un archivo nuevo `.md` con nombre en minúsculas y guiones (ej. `estilo-del-proyecto.md`).

### Caso 2 — Crear un prompt compartido

- **Hacé:** creá un prompt **compartido (shared)** con título "Glosario" y algo de texto.
- **Esperá ver:** otro archivo `.md` dentro de `.daedalus/prompts/`, sin que el del Caso 1 desaparezca o cambie.

### Caso 3 — Listar los prompts

- **Hacé:** pedí la lista de prompts.
- **Esperá ver:** los dos prompts creados, cada uno indicando si es **global** o **compartido**.

### Caso 4 — Editar un prompt

- **Hacé:** editá el cuerpo del prompt "Glosario" agregando una línea.
- **Esperá ver:** el cambio guardado en su archivo. El otro prompt ("Estilo del proyecto") debe quedar **igual que antes**.

### Caso 5 — Intentar duplicar un id

- **Hacé:** intentá crear otro prompt con el mismo nombre/identificador que "Glosario".
- **Esperá ver:** un **mensaje de error** claro avisando que ya existe; el "Glosario" original **no** debe cambiar.

### Caso 6 — Eliminar un prompt

- **Hacé:** eliminá el prompt "Estilo del proyecto".
- **Esperá ver:** desaparece **solo** ese archivo de `.daedalus/prompts/`; "Glosario" sigue ahí.

## Si algo no coincide

Si algún resultado no coincide con lo esperado, no intentes arreglarlo: avisá al **orquestador** describiendo qué hiciste y qué viste (Workflow B).
