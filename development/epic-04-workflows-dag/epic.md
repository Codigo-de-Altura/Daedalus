# Epic 04 — Workflows (DAG)

> **Origen:** EPIC-4 del PRD (RF-4.1 … RF-4.4). **Estilo:** SDD (plano, no guía de implementación).

## Objetivo

Definir, editar, visualizar y validar workflows como **DAG declarativo en YAML**: cada fase referencia un agente, sus artefactos de entrada/salida y un gate de validación. Incluye el `sdd-default.yaml` de fábrica que materializa el pipeline SDD por defecto.

## Alcance

**Incluye:** modelo y edición del DAG YAML, visualización del DAG en la TUI, validación del DAG (ciclos, artefactos faltantes, referencias a agentes inexistentes), workflow `sdd-default.yaml`.

**No incluye:** la **ejecución** de workflows/agentes (fuera de scope de Fase 1) ni features avanzadas del DAG (paralelismo/condicionales → backlog).

## Tickets

| Ticket | Tipo | Foco | Origen |
|---|---|---|---|
| `ticket-04-01-dag-yaml-model` | backend | Crear/editar workflows como DAG declarativo en YAML (`{agent, inputs, outputs, gate}`). | RF-4.1 |
| `ticket-04-02-dag-visualization` | frontend | Visualizar el DAG en la TUI (nodos = fases/agentes, aristas = dependencias). | RF-4.2 |
| `ticket-04-03-dag-validation` | backend | Validar el DAG: ciclos, artefactos faltantes, agentes inexistentes. | RF-4.3 |
| `ticket-04-04-sdd-default-workflow` | backend | Incluir `sdd-default.yaml` como workflow de fábrica. | RF-4.4 |

## Criterios de aceptación del epic

- Un workflow se define y edita en YAML con el esquema `{agent, inputs, outputs, gate}` por fase.
- La validación detecta ciclos, artefactos faltantes y referencias a agentes inexistentes, con errores accionables.
- El DAG se visualiza correctamente en la TUI.
- `sdd-default.yaml` reproduce el pipeline brief→spec→arquitectura→epics→tickets→validación→docs.
- Trazabilidad a RF-4.x.
