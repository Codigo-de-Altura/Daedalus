# Ticket 05-04 — Trazabilidad spec → epic → ticket

> **Epic:** epic-05-sdd-backlog · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-5.4 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

La **trazabilidad** es un principio rector del SDD (init.md §7, CLAUDE.md §7): todo ticket referencia su epic, y todo epic referencia su spec/arquitectura de origen. Los tickets 05-01..05-03 producen y vinculan los artefactos (brief → spec, spec → arquitectura, spec/arquitectura → epic → ticket); este ticket hace que esa cadena sea **navegable y verificable** de punta a punta.

> Fase 1: Daedalus gestiona la definición y la trazabilidad de los artefactos; **no ejecuta** agentes.

## Feature / Qué se construye

Una capacidad de **trazabilidad spec → epic → ticket** que sea:

1. **Navegable:** desde una spec se puede llegar a sus epics y a sus tickets, y desde un ticket remontar a su epic y a su spec/arquitectura de origen.
2. **Verificable:** los links son consistentes y detectables como rotos/faltantes (referencias a artefactos inexistentes, tickets sin epic, epics sin spec).
3. Construida sobre los **links** que ya registran los tickets 05-02 y 05-03; aquí se consolida la cadena y su verificación.

## Requerimientos

- R1. La cadena **spec → epic → ticket** es **navegable** en ambos sentidos (descendente: spec→epics→tickets; ascendente: ticket→epic→spec/arquitectura).
- R2. La trazabilidad es **verificable**: se puede comprobar que todo ticket referencia un epic existente y que todo epic referencia una spec/arquitectura existente (init.md §7, CLAUDE.md §7).
- R3. Se **detectan inconsistencias**: links rotos, referencias a artefactos inexistentes, tickets huérfanos (sin epic) y epics huérfanos (sin spec de origen).
- R4. La trazabilidad se apoya en los **links** ya registrados en epics/tickets (ticket 05-03) y en arquitectura (ticket 05-02), sin duplicar la fuente de verdad.
- R5. **Fase 1 — sin ejecución de agente:** la verificación de trazabilidad **no** ejecuta agentes; opera sobre los artefactos del workspace.
- R6. El resultado de la verificación es **determinista** y diff-friendly/reportable (mismo workspace → mismo resultado — RNF-5/RNF-6).

## Criterios de aceptación

- [ ] CA1. Desde una spec se puede navegar a sus epics y tickets asociados.
- [ ] CA2. Desde un ticket se puede remontar a su epic y a su spec/arquitectura de origen.
- [ ] CA3. La verificación confirma que todo ticket referencia un epic existente y todo epic una spec/arquitectura existente.
- [ ] CA4. Se detectan e informan inconsistencias: links rotos, referencias inexistentes, tickets/epics huérfanos.
- [ ] CA5. La trazabilidad reutiliza los links existentes (05-02/05-03) sin duplicar la fuente de verdad.
- [ ] CA6. La verificación **no ejecuta** agentes (Fase 1) y es determinista sobre el mismo workspace.

## Fuera de alcance

- Captura de brief y generación de spec/PRD (ticket 05-01).
- Gestión de documentos de **arquitectura** (ticket 05-02).
- CRUD de **epics y tickets** y registro de sus metadatos/links (ticket 05-03). Aquí se consolida y verifica la cadena, no se crean los artefactos.
- **Ejecución** de agentes (Fase 2+; decisión D5).
- Compilación al backend (epic 06).

## Referencias

- PRD.md — RF-5.4, §8.3 (pipeline SDD), RNF-5 (determinismo), RNF-6 (git-friendly).
- init.md — §5 (contrato de artefactos), §6 (pipeline), §7 (convenciones: trazabilidad).
- CLAUDE.md — §6 (estructura `development/`), §7 (filosofía SDD: trazabilidad).
- epic-05-sdd-backlog/epic.md — objetivo y criterios del epic (trazabilidad navegable y verificable).
