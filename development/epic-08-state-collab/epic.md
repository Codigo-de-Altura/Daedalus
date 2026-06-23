# Epic 08 — State, Persistence & Collaboration

> **Origen:** EPIC-8 del PRD (RF-8.1 … RF-8.3). **Estilo:** SDD (plano, no guía de implementación).

## Objetivo

Garantizar que todo el estado relevante de Daedalus viva en archivos versionables en git, en formatos amigables a diff/merge (YAML/Markdown legibles y estables), con convenciones de equipo (naming, estructura) explícitas y validables.

## Alcance

**Incluye:** persistencia del estado en git, formatos diff-friendly y deterministas, convenciones de equipo explicitadas y validables.

**No incluye:** un backend remoto de estado o sincronización en la nube (fuera de scope).

## Tickets

| Ticket | Tipo | Foco | Origen |
|---|---|---|---|
| `ticket-08-01-git-versioned-state` | backend | Todo el estado relevante vive en archivos versionables en git. | RF-8.1 |
| `ticket-08-02-diff-friendly-formats` | backend | Formatos amigables a diff/merge (YAML/Markdown legibles, output determinista). | RF-8.2 |
| `ticket-08-03-team-conventions` | backend | Convenciones de equipo (naming, estructura) explicitadas y validables. | RF-8.3 |

## Criterios de aceptación del epic

- Ningún estado relevante se guarda fuera de archivos git-trackeados.
- La serialización es determinista (claves ordenadas, diffs limpios).
- Las convenciones de equipo están documentadas y existe una validación que las verifica.
- Trazabilidad a RF-8.x.
