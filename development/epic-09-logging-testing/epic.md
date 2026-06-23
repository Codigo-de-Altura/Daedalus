# Epic 09 — Ecosystem (Logging & Testing)

> **Origen:** EPIC-9 del PRD (RF-9.1 … RF-9.3). **Estilo:** SDD (plano, no guía de implementación).

## Objetivo

Consolidar el ecosistema de soporte del propio Daedalus: logging estructurado de sus operaciones (init, build, validaciones), una estrategia de testing (unidad + golden files de compilación) y validaciones/linters de las definiciones (agentes, workflows, manifiesto).

## Alcance

**Incluye:** logging estructurado de operaciones, testing unitario + golden files de compilación, linters de definiciones.

**No incluye:** telemetría/observabilidad de runs en vivo de agentes (fuera de scope de Fase 1). El baseline mínimo de logging se establece en `epic-00`; aquí se profundiza.

## Tickets

| Ticket | Tipo | Foco | Origen |
|---|---|---|---|
| `ticket-09-01-structured-logging` | backend | Logging estructurado de las operaciones de Daedalus (init, build, validaciones). | RF-9.1 |
| `ticket-09-02-testing-and-golden-files` | backend | Testing unitario + golden files de compilación. | RF-9.2 |
| `ticket-09-03-definition-linters` | backend | Validaciones/linters de definiciones (agentes, workflows, manifiesto). | RF-9.3 |

## Criterios de aceptación del epic

- Las operaciones clave loguean en puntos de decisión, sin datos sensibles.
- Existe suite de tests unitarios y golden files que fijan la salida de compilación.
- Los linters detectan definiciones inválidas (agentes, workflows, manifiesto) con mensajes accionables.
- Trazabilidad a RF-9.x.
