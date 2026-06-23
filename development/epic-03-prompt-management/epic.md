# Epic 03 — Prompt Management

> **Origen:** EPIC-3 del PRD (RF-3.1 … RF-3.3). **Estilo:** SDD (plano, no guía de implementación).

## Objetivo

Gestionar prompts globales y compartidos reutilizables, con un mecanismo de composición/inclusión (DRY: convenciones, glosario, estilo) y previsualización del prompt renderizado.

## Alcance

**Incluye:** edición de prompts globales y compartidos, composición/inclusión de fragmentos, preview del prompt final renderizado.

**No incluye:** la compilación de prompts a comandos nativos del backend (epic 06).

## Tickets

| Ticket | Tipo | Foco | Origen |
|---|---|---|---|
| `ticket-03-01-global-shared-prompts` | backend | Editar prompts globales y prompts compartidos reutilizables. | RF-3.1 |
| `ticket-03-02-prompt-composition` | backend | Mecanismo de composición/inclusión de prompts (DRY). | RF-3.2 |
| `ticket-03-03-prompt-preview` | frontend | Previsualización del prompt renderizado en la TUI. | RF-3.3 |

## Criterios de aceptación del epic

- Se crean/editan prompts globales y compartidos persistidos en `.daedalus/prompts/`.
- La composición resuelve inclusiones de forma determinista, sin duplicación.
- La preview muestra el prompt final renderizado tal como se compilaría.
- Trazabilidad a RF-3.x.
