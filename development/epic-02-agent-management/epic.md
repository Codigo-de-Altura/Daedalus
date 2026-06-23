# Epic 02 — Agent Management

> **Origen:** EPIC-2 del PRD (RF-2.1 … RF-2.4). **Estilo:** SDD (plano, no guía de implementación).

## Objetivo

Gestionar las definiciones canónicas de agentes en el workspace: un catálogo built-in mínimo (analyst, architect, planner, validator, documenter), la importación/clonado/edición de agentes, la importación desde archivos locales (incluyendo estructuras `.claude/agents` existentes) y la validación de la definición contra el esquema canónico.

## Alcance

**Incluye:** catálogo built-in, operaciones importar/clonar/editar, import desde local, validación de esquema de agente.

**No incluye:** la compilación de los agentes al formato nativo del backend (epic 06) ni la edición visual avanzada en TUI más allá de lo necesario (epic 07 refina la UX).

## Tickets

| Ticket | Tipo | Foco | Origen |
|---|---|---|---|
| `ticket-02-01-builtin-catalog` | backend | Catálogo built-in de agentes (analyst, architect, planner, validator, documenter). | RF-2.1 |
| `ticket-02-02-import-clone-edit` | backend | Importar/clonar un agente del catálogo al workspace y editarlo (rol, prompt, parámetros). | RF-2.2 |
| `ticket-02-03-import-from-local-files` | backend | Importar agentes desde archivos locales (incluye `.claude/agents` existentes). | RF-2.3 |
| `ticket-02-04-agent-schema-validation` | backend | Validación de la definición de agente contra el esquema canónico. | RF-2.4 |

## Criterios de aceptación del epic

- El catálogo built-in expone al menos los 5 agentes canónicos y permite materializarlos en el workspace.
- Un agente puede clonarse y editarse sin afectar el original del catálogo.
- Se importan definiciones desde archivos locales, incluyendo agentes en formato Claude Code.
- Toda definición de agente se valida contra el esquema canónico, con errores accionables.
- Trazabilidad a RF-2.x.
