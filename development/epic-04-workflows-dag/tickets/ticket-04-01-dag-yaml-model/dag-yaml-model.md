# Ticket 04-01 — Modelo y edición del DAG de workflows en YAML

> **Epic:** epic-04-workflows-dag · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-4.1 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Un **workflow** en Daedalus es un **DAG declarativo en YAML** (decisión D9 del PRD) que vive en el área canónica `.daedalus/workflows/` y describe el pipeline SDD del proyecto: una secuencia de fases donde cada fase referencia un **agente**, los **artefactos** que consume, los que produce y un **gate** de validación que el artefacto de entrada debe cruzar para avanzar (init.md §3, §6).

Este ticket cubre el **modelo de dominio canónico del workflow** y su **edición** como archivo YAML: el esquema de cada fase `{ agent, inputs, outputs, gate }`, la representación de las dependencias entre fases (las aristas del DAG) y las operaciones de carga, parseo y serialización determinista. Es la base sobre la que se apoyan la visualización (ticket 04-02), la validación estructural (ticket 04-03) y el workflow de fábrica `sdd-default.yaml` (ticket 04-04).

La **ejecución** del workflow está **fuera de scope de Fase 1**: aquí solo se modela y edita la definición; nadie corre los agentes (PRD §4.2, decisión D5).

## Feature / Qué se construye

Un modelo de dominio canónico para workflows DAG y su (de)serialización a YAML legible y determinista. Cada workflow es un documento YAML en `.daedalus/workflows/<name>.yaml` con una lista ordenada de **fases**; cada fase tiene la forma:

```
phases:
  - id: spec
    agent: analyst
    inputs:  [brief]
    outputs: [spec]
    gate: spec-gate
    depends_on: [brief]
```

- `id` — identificador único de la fase dentro del workflow (kebab-case).
- `agent` — referencia al agente que produce el/los `outputs` de la fase (analyst, architect, planner, validator, documenter…).
- `inputs` — artefactos que la fase consume.
- `outputs` — artefactos que la fase produce.
- `gate` — criterio de validación que el artefacto de entrada/salida debe cumplir para avanzar de fase.
- `depends_on` — fases/artefactos predecesores; define las **aristas** del DAG.

El alcance incluye: el modelo en memoria del workflow y sus fases, la carga/parseo desde YAML, la serialización determinista (claves estables y ordenadas, diffs limpios) y las operaciones básicas de edición de la definición (crear un workflow, añadir/editar/eliminar fases). La verificación semántica del grafo (ciclos, artefactos faltantes, agentes inexistentes) es responsabilidad del ticket 04-03; aquí basta con que el modelo represente fielmente el DAG y se (de)serialice sin pérdida.

## Requerimientos

- R1. Existe un modelo de dominio canónico que representa un **workflow** como un DAG: un conjunto ordenado de **fases**, cada una con `{ id, agent, inputs[], outputs[], gate, depends_on[] }`.
- R2. Un workflow se define y edita como archivo YAML en el área canónica `.daedalus/workflows/<name>.yaml`.
- R3. El modelo carga/parsea un workflow desde su YAML y lo serializa de vuelta sin pérdida de información (round-trip).
- R4. La serialización a YAML es **determinista**: claves estables y ordenadas, salida reproducible (mismo modelo → mismo YAML), apta para diffs git limpios (RNF-5, RNF-6).
- R5. Cada fase expresa sus **dependencias** (`depends_on`), que constituyen las aristas del DAG; la lista de fases más sus dependencias describe el grafo completo.
- R6. El modelo soporta las operaciones de edición de la definición: crear un workflow nuevo, y añadir, editar y eliminar fases.
- R7. El parseo de un YAML malformado o que no cumple el esquema de fase falla con un error claro y accionable (no panic), indicando qué campo/fase es inválido.
- R8. El modelo es agnóstico del backend: no contiene nada específico de Claude Code ni de ningún runtime de agentes.

## Criterios de aceptación

- [ ] CA1. Un workflow YAML con fases `{ id, agent, inputs, outputs, gate, depends_on }` se carga correctamente en el modelo de dominio.
- [ ] CA2. Serializar un workflow cargado produce un YAML equivalente (round-trip sin pérdida).
- [ ] CA3. La serialización es determinista: serializar dos veces el mismo modelo produce bytes idénticos, con claves ordenadas de forma estable.
- [ ] CA4. El modelo expresa las dependencias entre fases (`depends_on`) de modo que el conjunto de aristas del DAG es recuperable.
- [ ] CA5. Se pueden crear un workflow y añadir/editar/eliminar fases sobre el modelo, y el resultado se serializa a un YAML válido.
- [ ] CA6. Un YAML malformado o con una fase que incumple el esquema produce un error claro y accionable (sin panic), señalando el campo/fase problemático.
- [ ] CA7. El modelo no contiene referencias a un backend concreto.

## Fuera de alcance

- **Ejecución** del workflow / orquestación de agentes (fuera de scope de Fase 1 — PRD §4.2, D5).
- Validación semántica del grafo: ciclos, artefactos faltantes, agentes inexistentes (ticket 04-03).
- Visualización del DAG en la TUI (ticket 04-02).
- Provisión del workflow de fábrica `sdd-default.yaml` (ticket 04-04).
- Features avanzadas del DAG: paralelismo, condicionales (backlog — PRD §13, §14).
- Compilación del workflow al formato nativo del backend (epic 06).

## Referencias

- PRD.md — RF-4.1, §8.3 (pipeline SDD por defecto y esquema de fase `{agent, inputs, outputs, gate}`), §7 (glosario: workflow, gate, artefacto), D9 (DAG declarativo en YAML), RNF-5/RNF-6 (determinismo, git-friendly).
- init.md — §3 (glosario: workflow = `{agent, inputs, outputs, gate}`, gate), §6 (pipeline SDD por defecto), §7 (convenciones: kebab-case, YAML ordenado y determinista).
- epic-04-workflows-dag/epic.md — objetivo, alcance y criterios del epic.
