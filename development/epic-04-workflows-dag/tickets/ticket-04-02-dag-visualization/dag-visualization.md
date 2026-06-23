# Ticket 04-02 — Visualización del DAG de workflows en la TUI

> **Epic:** epic-04-workflows-dag · **Tipo:** frontend · **Implementador:** Padmé · **Validador:** Leia
> **Origen:** RF-4.2 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Daedalus es una **TUI en Go + Charm** (Bubble Tea, Lipgloss, Bubbles, Glamour, Huh) cuya estética y UX son un objetivo de producto (PRD §4.1, RNF-4, decisión D2). Los workflows son **DAGs declarativos en YAML** (ticket 04-01) y el usuario necesita **entenderlos de un vistazo** sin leer el YAML crudo: qué fases hay, qué agente corre cada una y cómo dependen entre sí.

Este ticket cubre la **visualización del DAG en la TUI**: representar un workflow ya cargado como un grafo legible en terminal, donde los **nodos** son las fases/agentes y las **aristas** son las dependencias entre fases. Es una vista de **solo lectura/presentación**; consume el modelo canónico del ticket 04-01. No edita el workflow ni lo ejecuta (la ejecución está fuera de scope de Fase 1 — PRD §4.2, D5).

## Feature / Qué se construye

Una vista en la TUI que renderiza un workflow DAG cargado (modelo del ticket 04-01) como un grafo navegable y estéticamente cuidado en terminal:

- **Nodos** = fases del workflow, mostrando al menos el `id` de la fase y el `agent` asociado; idealmente también sus `inputs`/`outputs`/`gate` de forma compacta o expandible.
- **Aristas** = dependencias entre fases (`depends_on`), dibujadas de modo que se lea el orden topológico del pipeline (p. ej. brief → spec → arquitectura → … → docs).
- Estilo con **Lipgloss** consistente con el tema de la app; la vista debe ser legible y no romperse ante DAGs de tamaño moderado.

El alcance es la **presentación** del grafo dentro del flujo de navegación de la TUI (selección de un workflow del área `.daedalus/workflows/` y su render). La edición del workflow (epic/ticket 04-01) y su validación (04-03) no son parte de esta vista, aunque la vista puede señalar visualmente un workflow inválido si la información de validación está disponible.

## Requerimientos

- R1. La TUI ofrece una vista que renderiza un workflow DAG cargado (desde el modelo canónico del ticket 04-01) como un grafo en terminal.
- R2. Cada **fase** se representa como un **nodo** que muestra, como mínimo, el `id` de la fase y el `agent` que la ejecuta.
- R3. Cada **dependencia** entre fases (`depends_on`) se representa como una **arista**, de modo que se lea el orden/dirección del pipeline.
- R4. La vista es de **solo lectura**: no modifica el workflow ni dispara la ejecución de agentes.
- R5. El render usa Charm/Lipgloss y es consistente con la estética y el tema de la TUI (RNF-4); es legible para DAGs de tamaño moderado (p. ej. el `sdd-default`).
- R6. La vista se integra en la navegación de la TUI: el usuario puede seleccionar un workflow disponible y ver su DAG.
- R7. La vista degrada con gracia: ante un workflow vacío o no cargable, muestra un estado claro en lugar de romperse o entrar en panic.
- R8. La interacción es fluida y de bajo consumo (RNF-2, RNF-4); atajos de teclado consistentes con el resto de la TUI (RF-7.3).

## Criterios de aceptación

- [ ] CA1. Al seleccionar un workflow en la TUI, se muestra una vista de su DAG con nodos y aristas.
- [ ] CA2. Cada nodo muestra el `id` de la fase y el `agent` asociado.
- [ ] CA3. Las dependencias entre fases se ven como aristas que reflejan la dirección/orden del pipeline.
- [ ] CA4. El `sdd-default` (brief→spec→arquitectura→epics→tickets→validación→docs) se visualiza de forma legible y en el orden correcto.
- [ ] CA5. La vista no permite editar ni ejecutar el workflow (solo lectura).
- [ ] CA6. El estilo es consistente con el tema de la TUI y la vista no se rompe con un DAG de tamaño moderado.
- [ ] CA7. Ante un workflow vacío o no cargable, se muestra un estado claro sin panic ni glitches.

## Fuera de alcance

- Modelo, parseo y edición del workflow YAML (ticket 04-01).
- Validación semántica del DAG: ciclos, artefactos faltantes, agentes inexistentes (ticket 04-03). La vista puede reflejar el resultado de validación si está disponible, pero no lo computa.
- **Ejecución** del workflow / orquestación de agentes (fuera de scope de Fase 1 — PRD §4.2, D5).
- Layouts avanzados de grafo (paralelismo, condicionales) — backlog.

## Referencias

- PRD.md — RF-4.2, §8.3 (pipeline SDD), RF-7.1/7.2/7.3 (TUI: navegación, estética, atajos), RNF-2/RNF-4 (UX, estética), D2 (Go + Charm).
- init.md — §6 (pipeline SDD por defecto = forma del DAG a visualizar), §3 (glosario: workflow).
- epic-04-workflows-dag/epic.md — objetivo, alcance y criterios del epic.
- ticket-04-01-dag-yaml-model — modelo canónico del workflow que esta vista consume.
