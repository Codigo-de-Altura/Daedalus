# Ticket 05-01 — Capturar un brief y generar la spec/PRD (vía agente analyst)

> **Epic:** epic-05-sdd-backlog · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-5.1 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

El pipeline SDD por defecto (init.md §6) arranca en un **brief** humano que se transforma en una **spec/PRD** mediante el agente *analyst*; el humano luego refina ese artefacto. La spec/PRD es el primer artefacto canónico del backlog y la entrada de las fases posteriores (arquitectura, epics, tickets).

Este ticket cubre la **gestión de la definición** dentro del workspace: capturar el brief, asociarlo a la definición del agente *analyst*, y materializar la spec/PRD resultante en `.daedalus/specs/` como artefacto editable.

> **IMPORTANTE (Fase 1):** Daedalus **gestiona la definición** del brief, del agente y del artefacto spec; **NO ejecuta el agente analyst**. La ejecución del agente ocurre **fuera** de Daedalus, en el backend que use el usuario (decisión D5 del PRD). Aquí se construye la captura del brief, el cableado al agente y el lugar/forma donde aterriza la spec generada para que el usuario la refine.

## Feature / Qué se construye

Una capacidad para:

1. **Capturar un brief** del usuario como artefacto markdown dentro del workspace (entrada del pipeline).
2. **Asociar** ese brief a la definición del agente *analyst* (inputs/outputs del paso `brief → spec/PRD` del `sdd-default.yaml`).
3. **Materializar** la spec/PRD resultante en `.daedalus/specs/<slug>.md` como documento **editable por el usuario** (el humano refina).

La spec/PRD es un **plano** (qué/por qué/requerimientos), no una guía de implementación. El formato es markdown diff-friendly con metadatos estables.

## Requerimientos

- R1. El usuario puede **capturar un brief** y persistirlo como artefacto markdown dentro del workspace (área de entrada del pipeline SDD).
- R2. El brief se **vincula** a la definición del agente *analyst* según el paso `brief → spec/PRD` del workflow `sdd-default.yaml` (inputs/outputs/gate).
- R3. Existe una **ubicación canónica determinista** para la spec/PRD resultante: `.daedalus/specs/<slug>.md`, con `<slug>` en kebab-case.
- R4. La spec/PRD materializada es un **artefacto editable**: el usuario la refina manualmente sin que Daedalus la sobrescriba de forma destructiva.
- R5. **Fase 1 — sin ejecución de agente:** Daedalus **no** lanza ni corre el agente *analyst*; solo gestiona la definición (brief, vínculo al agente, destino de la spec). La generación efectiva del contenido la produce el usuario corriendo el agente en su backend.
- R6. Los artefactos producidos (brief, spec/PRD) son **markdown diff-friendly** con metadatos estables (RNF-6).
- R7. La operación es **no destructiva**: no borra ni pisa specs o briefs editados por el usuario sin preview/confirmación (RNF-8).
- R8. La trazabilidad se preserva: la spec referencia su brief de origen, dejando el rastro `brief → spec`.

## Criterios de aceptación

- [ ] CA1. El usuario puede capturar un brief y queda persistido como artefacto markdown en el workspace.
- [ ] CA2. El brief queda asociado a la definición del agente *analyst* (paso `brief → spec/PRD` del `sdd-default.yaml`).
- [ ] CA3. Existe/queda preparada la ubicación canónica `.daedalus/specs/<slug>.md` (kebab-case) para la spec resultante.
- [ ] CA4. La spec materializada es editable por el usuario y no es sobrescrita de forma destructiva por Daedalus.
- [ ] CA5. Daedalus **no ejecuta** el agente *analyst* (Fase 1): no se lanza ningún proceso de agente; solo se gestiona la definición.
- [ ] CA6. Brief y spec son markdown con metadatos estables, aptos para diffs git limpios.
- [ ] CA7. La spec referencia su brief de origen (rastro de trazabilidad `brief → spec`).

## Fuera de alcance

- **Ejecución/orquestación** del agente *analyst* (Fase 2+; decisión D5). Daedalus solo gestiona la definición.
- Gestión de documentos de **arquitectura** (ticket 05-02).
- CRUD de **epics y tickets** con metadatos (ticket 05-03).
- **Trazabilidad** navegable spec → epic → ticket end-to-end (ticket 05-04).
- Compilación de la definición al backend (epic 06).

## Referencias

- PRD.md — RF-5.1, §8.3 (pipeline SDD), §5 (decisión D5: ejecución fuera de scope), §5 contrato de artefactos (brief, spec/PRD), RNF-6/RNF-8.
- init.md — §5 (contrato de artefactos: brief, spec/PRD), §6 (pipeline SDD por defecto), §7 (convenciones).
- CLAUDE.md — §6 (estructura `development/`), §7 (filosofía SDD).
- epic-05-sdd-backlog/epic.md — objetivo y criterios del epic.
