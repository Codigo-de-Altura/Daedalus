# Ticket 02-01 — Catálogo built-in de agentes

> **Epic:** epic-02-agent-management · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-2.1 · **Estilo:** SDD

## Contexto

Daedalus gestiona la "estructura de IA" de un proyecto de forma agnóstica del backend (PRD §3, §8). Para que un usuario arranque sin partir de cero, Daedalus expone un **catálogo built-in** de agentes canónicos que cubren el pipeline SDD por defecto (init.md §6, §8). Estos agentes son las unidades con rol + prompt que luego se materializan al workspace `.daedalus/agents/` y, en epics posteriores, se compilan al formato nativo del backend (epic-06). Este ticket es la base de toda la gestión de agentes: sin un catálogo built-in no hay nada que importar, clonar ni editar.

## Feature / Qué se construye

Un **catálogo built-in** embebido en el binario de Daedalus que expone, como mínimo, los cinco agentes canónicos del pipeline SDD: **analyst, architect, planner, validator, documenter** (init.md §8). Cada entrada del catálogo es una definición de agente completa (rol, prompt, parámetros) en formato canónico agnóstico (YAML + prompt MD, init.md §5). El catálogo permite **listar** los agentes disponibles y **materializarlos** (instanciarlos) en el workspace `.daedalus/agents/` del repo, dejando una definición canónica editable como fuente de verdad.

## Requerimientos

- R1 — El catálogo built-in está embebido en el binario (no depende de red ni de archivos externos; remoto = post-MVP, D6).
- R2 — El catálogo expone **al menos** los cinco agentes canónicos: `analyst`, `architect`, `planner`, `validator`, `documenter`, cada uno con un rol declarado coherente con init.md §8.
- R3 — Cada agente del catálogo es una definición canónica completa y válida contra el esquema canónico (ticket-02-04): identificador, rol, prompt y parámetros.
- R4 — El catálogo permite **listar** los agentes disponibles con su identificador y rol/descripción.
- R5 — El catálogo permite **materializar** un agente seleccionado en `.daedalus/agents/` del workspace, produciendo su definición canónica (YAML + prompt MD) como fuente de verdad editable.
- R6 — La materialización es **no destructiva** (RNF-8): si ya existe un agente con el mismo identificador en el workspace, no se sobreescribe silenciosamente; se informa/ofrece preview o confirmación.
- R7 — Los identificadores de agente siguen `kebab-case` (init.md §7).
- R8 — La salida es **determinista** (RNF-5): materializar el mismo agente produce el mismo contenido canónico (orden de claves estable, diffs limpios).

## Criterios de aceptación

- [ ] CA1 — El catálogo built-in lista al menos los 5 agentes canónicos (`analyst`, `architect`, `planner`, `validator`, `documenter`).
- [ ] CA2 — Cada agente del catálogo tiene rol y prompt no vacíos y valida contra el esquema canónico.
- [ ] CA3 — Materializar un agente crea su definición canónica bajo `.daedalus/agents/` (YAML + prompt MD).
- [ ] CA4 — Materializar un agente ya existente no lo sobreescribe silenciosamente (operación no destructiva).
- [ ] CA5 — Los identificadores de los agentes están en `kebab-case`.
- [ ] CA6 — Materializar dos veces el mismo agente produce contenido idéntico (determinismo).
- [ ] CA7 — Trazabilidad a RF-2.1.

## Fuera de alcance

- Clonado y edición de un agente del catálogo (ticket-02-02).
- Importación desde archivos locales / `.claude/agents` (ticket-02-03).
- Definición del esquema canónico y sus errores accionables (ticket-02-04); este ticket lo consume.
- Compilación al formato nativo del backend (epic-06).
- Catálogo remoto / marketplace (post-MVP, PRD §14).
- Edición visual avanzada en TUI (epic-07).

## Referencias

- PRD §9 — RF-2.1 (catálogo built-in de agentes)
- PRD §6 — D6 (catálogo built-in; remoto post-MVP)
- init.md §8 (catálogo de agentes built-in), §6 (pipeline SDD), §5 (contrato de artefactos), §7 (convenciones)
- CLAUDE.md §6 (estructura de tickets)
- epic-02-agent-management/epic.md (ticket-02-01)
