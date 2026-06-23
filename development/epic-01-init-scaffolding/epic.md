# Epic 01 — Init & Scaffolding

> **Origen:** EPIC-1 del PRD (RF-1.1 … RF-1.4). **Estilo:** SDD (plano, no guía de implementación).

## Objetivo

Implementar `daedalus init`: la creación y gestión del workspace canónico `.daedalus/` dentro de cualquier repo (nuevo o existente), incluyendo el manifiesto `daedalus.yaml`, el `init.md` base del proyecto target, la elección de backend(s) objetivo y la detección de un `.daedalus/` preexistente con upgrade/merge no destructivo.

## Alcance

**Incluye:** comando `init`, materialización de la estructura `.daedalus/` (ver `init.md` §4.2), generación del manifiesto e `init.md` base, selección de backend (MVP: Claude Code), detección + upgrade no destructivo.

**No incluye:** el contenido de agentes/prompts/workflows/backlog (epics 02–05) ni la compilación al backend (epic 06).

## Tickets

| Ticket | Tipo | Foco | Origen |
|---|---|---|---|
| `ticket-01-01-init-command-workspace` | backend | `daedalus init` crea/gestiona el workspace `.daedalus/` en el repo objetivo. | RF-1.1 |
| `ticket-01-02-detect-and-upgrade` | backend | Detecta `.daedalus/` existente y ofrece upgrade/merge no destructivo. | RF-1.2 |
| `ticket-01-03-manifest-and-init-md` | backend | Genera `daedalus.yaml` (manifiesto) y el `init.md` base del proyecto. | RF-1.3 |
| `ticket-01-04-backend-selection` | backend | Permite elegir el/los backend(s) objetivo (MVP: Claude Code). | RF-1.4 |

## Criterios de aceptación del epic

- `daedalus init` produce un `.daedalus/` válido y completo en un repo vacío y en uno existente.
- Re-ejecutar `init` sobre un workspace existente no destruye cambios manuales y ofrece preview/merge.
- El manifiesto y el `init.md` base se generan de forma determinista.
- La selección de backend queda registrada en el manifiesto.
- Trazabilidad a RF-1.x.
