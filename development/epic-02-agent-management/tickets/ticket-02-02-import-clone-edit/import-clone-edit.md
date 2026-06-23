# Ticket 02-02 — Importar / clonar y editar un agente

> **Epic:** epic-02-agent-management · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-2.2 · **Estilo:** SDD

## Contexto

Con el catálogo built-in disponible (ticket-02-01), el usuario necesita partir de un agente canónico y adaptarlo a su proyecto sin perder el original. El principio rector es el **control total de la estructura** (PRD §3): el usuario edita definiciones limpias (rol, prompt, parámetros), no formatos nativos. Clonar y editar es la operación central de personalización de agentes y debe garantizar que el agente del catálogo built-in permanece intacto.

## Feature / Qué se construye

La capacidad de **clonar** un agente del catálogo built-in al workspace `.daedalus/agents/` bajo un nuevo identificador, y luego **editar** su definición canónica: **rol, prompt y parámetros**. El clon es una copia independiente; las ediciones sobre el clon **no afectan** la definición original del catálogo built-in. Tras editar, la definición resultante sigue siendo una definición canónica válida.

## Requerimientos

- R1 — Se puede clonar cualquier agente del catálogo built-in al workspace bajo un identificador destino (`kebab-case`), produciendo una copia canónica independiente en `.daedalus/agents/`.
- R2 — El clon es **independiente del original**: editar el clon no muta el agente del catálogo built-in ni ningún otro agente del workspace.
- R3 — Se puede editar la definición del agente clonado: al menos **rol**, **prompt** y **parámetros**.
- R4 — El clonado es **no destructivo** (RNF-8): si el identificador destino ya existe en el workspace, no se sobreescribe silenciosamente; se informa/ofrece preview o confirmación.
- R5 — Tras editar, la definición resultante se valida contra el esquema canónico (ticket-02-04); una edición que rompe el esquema produce un error accionable y no deja la definición en estado inválido sin avisar.
- R6 — La salida es **determinista** y diff-friendly (RNF-5, RNF-6): claves estables y ordenadas.
- R7 — Las operaciones de clonado/edición operan sobre la **definición canónica** (YAML + prompt MD), no sobre formatos nativos del backend.

## Criterios de aceptación

- [ ] CA1 — Clonar un agente del catálogo crea una nueva definición canónica en `.daedalus/agents/` bajo el identificador destino.
- [ ] CA2 — Tras clonar y editar el clon (rol/prompt/parámetros), la definición del catálogo built-in original permanece sin cambios.
- [ ] CA3 — Se puede modificar rol, prompt y parámetros del agente clonado y los cambios persisten en su definición canónica.
- [ ] CA4 — Clonar sobre un identificador ya existente no sobreescribe silenciosamente (operación no destructiva).
- [ ] CA5 — Una edición que viola el esquema canónico produce un error accionable.
- [ ] CA6 — El identificador destino sigue `kebab-case`.
- [ ] CA7 — Trazabilidad a RF-2.2.

## Fuera de alcance

- Definición del catálogo built-in y su materialización (ticket-02-01); este ticket lo consume como origen.
- Importación desde archivos locales / `.claude/agents` (ticket-02-03).
- Definición del esquema canónico y sus errores accionables (ticket-02-04); este ticket lo consume.
- Compilación al formato nativo del backend (epic-06).
- Edición visual avanzada / formularios en TUI (epic-07).

## Referencias

- PRD §9 — RF-2.2 (importar/clonar y editar un agente)
- PRD §3 — Principio: control total de la estructura; definiciones limpias
- init.md §5 (agente como YAML + prompt MD), §7 (convenciones, kebab-case)
- CLAUDE.md §6 (estructura de tickets)
- epic-02-agent-management/epic.md (ticket-02-02)
