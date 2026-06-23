# Ticket 05-03 — Crear y gestionar epics y tickets con metadatos

> **Epic:** epic-05-sdd-backlog · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-5.3 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

En el pipeline SDD (init.md §6), el agente *planner* deriva **epics** y **tickets** desde la spec + arquitectura. Epics y tickets son el corazón del backlog SDD: cada uno es un artefacto markdown con **metadatos** (estado, prioridad, dependencias, links a artefactos de origen) que el equipo gestiona y versiona en git.

Este ticket cubre el **CRUD y la gestión de metadatos** de epics y tickets dentro del workspace, con la estructura y convenciones de `CLAUDE.md §6` (epics `epic-NN-<slug>`, tickets `ticket-NN-MM-<slug>`).

> Fase 1: Daedalus **gestiona la definición** de epics y tickets; **no ejecuta** el agente *planner* ni la implementación (que ocurre fuera de Daedalus).

## Feature / Qué se construye

Una capacidad para **crear y gestionar epics y tickets** con metadatos consistentes:

1. Crear/editar/gestionar **epics** (`epic-NN-<slug>/`) y **tickets** (`ticket-NN-MM-<slug>/`) como artefactos markdown.
2. Asociarles **metadatos** estables: **estado**, **prioridad**, **dependencias** (entre tickets/epics) y **links a artefactos** de origen (spec, arquitectura).
3. Mantenerlos en formato **diff-friendly** con metadatos ordenados y reproducibles.

Los epics describen objetivo/alcance/criterios; los tickets describen el feature (qué/requerimientos/criterios). Ambos son **planos**.

## Requerimientos

- R1. El usuario puede **crear y gestionar epics** con id/carpeta `epic-NN-<slug>` (kebab-case) según CLAUDE.md §6.
- R2. El usuario puede **crear y gestionar tickets** con id/carpeta `ticket-NN-MM-<slug>` (NN = epic, MM = secuencia) según CLAUDE.md §6.
- R3. Epics y tickets soportan **metadatos** estables: **estado**, **prioridad**, **dependencias** y **links a artefactos** de origen (spec/arquitectura).
- R4. Las **dependencias** entre tickets/epics se representan de forma explícita y consistente.
- R5. Los **links a artefactos** de origen quedan registrados (todo ticket referencia su epic; todo epic referencia su spec/arquitectura — init.md §7, CLAUDE.md §7).
- R6. Los metadatos son **consistentes** (conjunto de valores estable para estado/prioridad) y **diff-friendly** (ordenados, reproducibles — RNF-6).
- R7. **Fase 1 — sin ejecución de agente:** Daedalus **no** lanza ni corre el agente *planner* ni la implementación; solo gestiona la definición.
- R8. La operación es **no destructiva** respecto de epics/tickets editados por el usuario (preview/confirmación — RNF-8).

## Criterios de aceptación

- [ ] CA1. El usuario puede crear y gestionar un epic con carpeta `epic-NN-<slug>` (kebab-case).
- [ ] CA2. El usuario puede crear y gestionar un ticket con carpeta `ticket-NN-MM-<slug>` bajo su epic.
- [ ] CA3. Epics y tickets tienen metadatos de **estado**, **prioridad**, **dependencias** y **links a artefactos**.
- [ ] CA4. Las dependencias entre tickets/epics se expresan de forma explícita y consistente.
- [ ] CA5. Todo ticket referencia su epic y todo epic referencia su spec/arquitectura de origen.
- [ ] CA6. Los metadatos son consistentes y diff-friendly (ordenados, reproducibles).
- [ ] CA7. Daedalus **no ejecuta** el agente *planner* ni la implementación (Fase 1).
- [ ] CA8. Las operaciones son no destructivas respecto de ediciones del usuario.

## Fuera de alcance

- Captura de brief y generación de spec/PRD (ticket 05-01).
- Gestión de documentos de **arquitectura** (ticket 05-02).
- **Trazabilidad** navegable y verificable end-to-end spec → epic → ticket (ticket 05-04). Aquí solo se registran los links; la navegación/verificación es del 05-04.
- **Ejecución** del agente *planner* y de la implementación (Fase 2+ / fuera de Daedalus).
- Compilación al backend (epic 06).

## Referencias

- PRD.md — RF-5.3, §8.3 (pipeline SDD), RNF-6/RNF-8.
- init.md — §5 (contrato de artefactos: epic, ticket), §6 (pipeline), §7 (convenciones: kebab-case, trazabilidad).
- CLAUDE.md — §6 (estructura `development/`, naming epics/tickets, contrato de documentos del ticket), §7 (trazabilidad).
- epic-05-sdd-backlog/epic.md — objetivo y criterios del epic.
