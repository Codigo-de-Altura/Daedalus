# Validación manual — Ticket 05-03

> Para alguien **sin background de testing**. Seguí los pasos y compará lo que ves con lo que se espera. Si algo no coincide → avisá al **orquestador** (Workflow B).

## Preparación

1. Tené un repo de prueba con un workspace `.daedalus/` ya inicializado.
2. Tené a mano el binario `daedalus` (o el comando que te indique el orquestador para correrlo).
3. Que exista al menos una spec (y/o documento de arquitectura) para usar como origen.

## Casos

### Caso 1 — Crear un epic

- **Hacé:** Creá un epic nuevo con la operación de Daedalus.
- **Esperá ver:** Una carpeta con nombre tipo `epic-NN-<slug>` (NN = número, slug en minúsculas con guiones).

### Caso 2 — Crear un ticket dentro del epic

- **Hacé:** Creá un ticket bajo ese epic.
- **Esperá ver:** Una carpeta tipo `ticket-NN-MM-<slug>` dentro del epic (NN = número del epic, MM = secuencia del ticket).

### Caso 3 — Metadatos del ticket

- **Hacé:** Mirá los metadatos del ticket que creaste.
- **Esperá ver:** Campos de **estado**, **prioridad**, **dependencias** y **links a artefactos** (spec/arquitectura).

### Caso 4 — Dependencias

- **Hacé:** Declará que un ticket depende de otro.
- **Esperá ver:** La dependencia queda anotada de forma clara y consistente en los metadatos.

### Caso 5 — Trazabilidad de origen

- **Hacé:** Revisá el ticket y el epic.
- **Esperá ver:** El ticket apunta a su epic, y el epic apunta a su spec/arquitectura de origen.

### Caso 6 — Daedalus NO corre el agente

- **Hacé:** Completá las operaciones de creación/gestión.
- **Esperá ver:** Daedalus **no** lanza el agente *planner* ni ninguna implementación. Solo gestiona las definiciones.

### Caso 7 — Tus ediciones se respetan

- **Hacé:** Editá a mano un epic/ticket y volvé a correr la operación de gestión.
- **Esperá ver:** Tus cambios siguen ahí; Daedalus no los pisa sin avisarte (te ofrece preview/confirmación).

## Si algo no coincide

Anotá qué esperabas y qué viste, y reportalo al **orquestador** (Workflow B) para que delegue el fix.
