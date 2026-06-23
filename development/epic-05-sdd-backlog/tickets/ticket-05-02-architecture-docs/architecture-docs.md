# Ticket 05-02 — Gestionar documentos de arquitectura

> **Epic:** epic-05-sdd-backlog · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-5.2 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

En el pipeline SDD (init.md §6), después de la spec/PRD viene la **arquitectura**, producida por el agente *architect* y consumida por el *planner* y por el dev/agente externo. Los documentos de arquitectura son **planos** (visión arquitectónica, no guía de implementación) y viven en el workspace canónico como artefactos versionables.

Este ticket cubre la **gestión** de esos documentos dentro del workspace: su ubicación canónica, su forma diff-friendly y su lugar en la cadena de trazabilidad spec → arquitectura.

> Fase 1: Daedalus **gestiona la definición** del artefacto de arquitectura; **no ejecuta** el agente *architect*. La generación del contenido la produce el usuario corriendo el agente en su backend.

## Feature / Qué se construye

Una capacidad para **gestionar documentos de arquitectura** en `.daedalus/architecture/`:

1. Crear/gestionar artefactos de arquitectura markdown con ubicación canónica determinista.
2. Vincularlos a su **spec de origen** (la cadena `spec → arquitectura` del `sdd-default.yaml`).
3. Mantenerlos como artefactos **editables** y diff-friendly para que el usuario los refine.

Los documentos de arquitectura son **planos**: describen estructura, componentes y decisiones a alto nivel, no recetas de implementación.

## Requerimientos

- R1. Existe una **ubicación canónica determinista** para los documentos de arquitectura: `.daedalus/architecture/<slug>.md`, con `<slug>` en kebab-case.
- R2. El usuario puede **crear y gestionar** documentos de arquitectura como artefactos markdown dentro del workspace.
- R3. Cada documento de arquitectura puede **vincularse a su spec de origen** (paso `spec → arquitectura` del `sdd-default.yaml`), preservando la trazabilidad.
- R4. Los documentos son **editables** y **no se sobrescriben de forma destructiva** (RNF-8): preview/confirmación ante cambios.
- R5. **Fase 1 — sin ejecución de agente:** Daedalus **no** lanza ni corre el agente *architect*; solo gestiona la definición y el destino del artefacto.
- R6. Los artefactos son **markdown diff-friendly** con metadatos estables (RNF-6).
- R7. La operación es **no destructiva** respecto de documentos de arquitectura editados por el usuario.

## Criterios de aceptación

- [ ] CA1. Existe/queda preparada la ubicación canónica `.daedalus/architecture/<slug>.md` (kebab-case).
- [ ] CA2. El usuario puede crear y gestionar documentos de arquitectura como markdown en el workspace.
- [ ] CA3. Un documento de arquitectura puede vincularse a su spec de origen (rastro `spec → arquitectura`).
- [ ] CA4. Los documentos son editables y no se sobrescriben de forma destructiva (preview/confirmación).
- [ ] CA5. Daedalus **no ejecuta** el agente *architect* (Fase 1).
- [ ] CA6. Los documentos son markdown con metadatos estables, aptos para diffs git limpios.

## Fuera de alcance

- Captura de brief y generación de spec/PRD (ticket 05-01).
- CRUD de **epics y tickets** con metadatos (ticket 05-03).
- **Trazabilidad** navegable spec → epic → ticket end-to-end (ticket 05-04).
- **Ejecución** del agente *architect* (Fase 2+; decisión D5).
- Compilación al backend (epic 06).

## Referencias

- PRD.md — RF-5.2, §8.3 (pipeline SDD), §11 (arquitectura técnica como plano), RNF-6/RNF-8.
- init.md — §5 (contrato de artefactos: arquitectura), §6 (pipeline SDD), §7 (convenciones).
- CLAUDE.md — §6 (estructura `development/`), §7 (filosofía SDD).
- epic-05-sdd-backlog/epic.md — objetivo y criterios del epic.
