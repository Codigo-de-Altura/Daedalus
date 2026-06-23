# Epic 05 — SDD Backlog (spec, architecture, epics, tickets)

> **Origen:** EPIC-5 del PRD (RF-5.1 … RF-5.4). **Estilo:** SDD (plano, no guía de implementación).

## Objetivo

Gestionar el backlog SDD dentro del workspace: capturar un brief y generar —vía agente *analyst*— una spec/PRD que el usuario refina; gestionar documentos de arquitectura; crear/gestionar epics y tickets con metadatos (estado, prioridad, dependencias, links a artefactos); y mantener la trazabilidad spec → epic → ticket.

## Alcance

**Incluye:** captura de brief y generación de spec, gestión de arquitectura, CRUD de epics/tickets con metadatos, trazabilidad.

**No incluye:** la **ejecución** del agente analyst (Daedalus gestiona la definición; el usuario corre el agente en el backend) ni la compilación (epic 06).

## Tickets

| Ticket | Tipo | Foco | Origen |
|---|---|---|---|
| `ticket-05-01-brief-to-spec` | backend | Capturar un brief y, vía agente analyst, generar una spec/PRD (el usuario refina). | RF-5.1 |
| `ticket-05-02-architecture-docs` | backend | Gestionar documentos de arquitectura. | RF-5.2 |
| `ticket-05-03-epics-tickets-management` | backend | Crear/gestionar epics y tickets con metadatos (estado, prioridad, dependencias, links). | RF-5.3 |
| `ticket-05-04-traceability` | backend | Trazabilidad spec → epic → ticket. | RF-5.4 |

## Criterios de aceptación del epic

- A partir de un brief se materializa una spec en `.daedalus/specs/`, editable por el usuario.
- Epics y tickets se crean con metadatos consistentes y links a sus artefactos de origen.
- La trazabilidad spec → epic → ticket es navegable y verificable.
- Los formatos son diff-friendly (markdown + metadatos estables).
- Trazabilidad a RF-5.x.
