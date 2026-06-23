# Ticket 04-04 — Workflow de fábrica `sdd-default.yaml`

> **Epic:** epic-04-workflows-dag · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-4.4 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

El pipeline SDD por defecto de Daedalus es **brief → spec/PRD → arquitectura → epics → tickets → ⟨implementación externa⟩ → validación → docs**, donde cada paso es ejecutado por un agente del catálogo built-in (analyst, architect, planner, validator, documenter) y cruza un gate de validación (PRD §8.3, init.md §6). Para que un usuario obtenga ese pipeline "listo para usar" sin tener que escribirlo a mano, Daedalus debe **incluir de fábrica** un workflow `sdd-default.yaml`.

Este ticket cubre la **provisión del workflow de fábrica `sdd-default.yaml`**: un archivo canónico, conforme al modelo del ticket 04-01, que materializa ese pipeline y queda disponible en `.daedalus/workflows/` del workspace. Reutiliza el modelo de DAG (04-01) y debe pasar la validación del DAG (04-03). No incluye su ejecución (fuera de scope de Fase 1 — PRD §4.2, D5).

## Feature / Qué se construye

Un workflow de fábrica `sdd-default.yaml`, en el formato canónico del DAG YAML (ticket 04-01), que reproduce el pipeline SDD por defecto. Conceptualmente, sus fases son:

```
brief
  └─► spec          (agent: analyst)     inputs: [brief]          outputs: [spec]
        └─► architecture (agent: architect) inputs: [spec]        outputs: [architecture]
              └─► epics    (agent: planner)  inputs: [architecture] outputs: [epics]
                    └─► tickets (agent: planner) inputs: [epics]    outputs: [tickets]
                          └─► ⟨implementación externa: dev/agente⟩
                                └─► validation (agent: validator) inputs: [tickets, implementation] outputs: [validation]
                                      └─► docs   (agent: documenter) inputs: [validation] outputs: [docs]
```

- Cada fase respeta el esquema `{ id, agent, inputs, outputs, gate, depends_on }` del ticket 04-01.
- Los agentes referenciados son los del catálogo built-in (init.md §8): analyst, architect, planner, validator, documenter.
- El paso de **implementación** es **externo** a Daedalus (lo ejecuta un dev/agente en el backend); el workflow lo refleja como el punto donde el artefacto de implementación entra al pipeline para ser validado, sin que Daedalus lo ejecute.
- Cada arista cruza un **gate** de validación del artefacto de entrada (init.md §6).

El alcance es: el contenido del `sdd-default.yaml` y su disponibilidad como workflow de fábrica en el workspace. El YAML debe cargar en el modelo (04-01) y pasar la validación del DAG (04-03).

## Requerimientos

- R1. Daedalus provee de fábrica un workflow `sdd-default.yaml` disponible en el área de workflows del workspace (`.daedalus/workflows/`).
- R2. El `sdd-default.yaml` está escrito en el formato canónico del DAG YAML (ticket 04-01) y **carga sin error** en el modelo de dominio.
- R3. El workflow reproduce el pipeline SDD por defecto: las fases **spec, architecture, epics, tickets, validation, docs** (más el artefacto inicial `brief` y el punto de implementación externa), en ese orden de dependencias.
- R4. Cada fase asigna el agente correcto del catálogo built-in: spec→analyst, architecture→architect, epics→planner, tickets→planner, validation→validator, docs→documenter.
- R5. Cada fase declara sus `inputs`, `outputs` y un `gate`, y sus `depends_on` reflejan el orden del pipeline.
- R6. El paso de **implementación externa** se refleja sin que el workflow implique ejecutar agentes (Fase 1: Daedalus configura, no ejecuta — PRD §4.2, D5).
- R7. El `sdd-default.yaml` **pasa** la validación del DAG (ticket 04-03): sin ciclos, sin artefactos faltantes, sin agentes inexistentes.
- R8. El archivo es determinista y git-friendly: claves estables y ordenadas, formato estable (RNF-5, RNF-6).

## Criterios de aceptación

- [ ] CA1. Existe un `sdd-default.yaml` provisto de fábrica en el área de workflows del workspace.
- [ ] CA2. El `sdd-default.yaml` carga sin error en el modelo de dominio del ticket 04-01.
- [ ] CA3. El workflow contiene las fases spec, architecture, epics, tickets, validation y docs, encadenadas por dependencias en el orden del pipeline SDD.
- [ ] CA4. Cada fase referencia el agente correcto: analyst, architect, planner, planner, validator, documenter respectivamente.
- [ ] CA5. Cada fase declara `inputs`, `outputs` y `gate`, y `depends_on` consistente con el pipeline.
- [ ] CA6. El `sdd-default.yaml` pasa la validación del DAG (ticket 04-03) sin hallazgos.
- [ ] CA7. El archivo es determinista (claves ordenadas, formato estable) y apto para diffs git limpios.

## Fuera de alcance

- Modelo, parseo y edición del workflow YAML (ticket 04-01).
- Implementación del validador del DAG (ticket 04-03); aquí solo se requiere que el `sdd-default.yaml` **pase** esa validación.
- Visualización del DAG en la TUI (ticket 04-02).
- **Ejecución** del pipeline / orquestación de agentes (fuera de scope de Fase 1 — PRD §4.2, D5).
- Definición del contenido/prompts de cada agente del catálogo (epic 02) y la compilación al backend (epic 06).
- Variantes o personalización del pipeline por proyecto más allá del default de fábrica.

## Referencias

- PRD.md — RF-4.4, §8.3 (pipeline SDD por defecto y `workflows/sdd-default.yaml`), §7 (glosario), RNF-5/RNF-6 (determinismo, git-friendly), §4.2/D5 (ejecución fuera de scope).
- init.md — §6 (pipeline SDD por defecto), §5 (contrato de artefactos: quién produce/consume cada uno), §8 (catálogo de agentes built-in), §4.2 (área `workflows/`).
- epic-04-workflows-dag/epic.md — objetivo, alcance y criterios del epic.
- ticket-04-01-dag-yaml-model — formato canónico del workflow.
- ticket-04-03-dag-validation — validación que el `sdd-default.yaml` debe pasar.
