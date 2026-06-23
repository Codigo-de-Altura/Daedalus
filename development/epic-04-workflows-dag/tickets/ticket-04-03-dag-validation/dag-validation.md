# Ticket 04-03 — Validación del DAG de workflows

> **Epic:** epic-04-workflows-dag · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-4.3 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Un workflow de Daedalus es un **DAG declarativo en YAML** (ticket 04-01) cuyo modelo ya carga y serializa las fases `{ id, agent, inputs, outputs, gate, depends_on }`. Pero un YAML sintácticamente válido puede describir un grafo **semánticamente inválido**: con ciclos (deja de ser un DAG), con artefactos que una fase consume pero ninguna fase anterior produce, o con referencias a agentes que no existen en el catálogo/definiciones del workspace.

Este ticket cubre la **validación semántica del DAG**: un validador que, dado un workflow cargado, detecta esos problemas y devuelve **errores accionables** (qué falla, dónde y por qué), de modo que el usuario pueda corregir su workflow. Es coherente con la filosofía del proyecto de validaciones/linters de definiciones (RF-9.3) y con la mitigación de "sobre-ingeniería del DAG: mantenerlo simple y validado" (PRD §13).

No incluye la ejecución del workflow (fuera de scope de Fase 1 — PRD §4.2, D5) ni la visualización (ticket 04-02), aunque la vista puede consumir el resultado de esta validación.

## Feature / Qué se construye

Un validador de workflows DAG que opera sobre el modelo canónico (ticket 04-01) y detecta, como mínimo, tres clases de problema, reportando cada hallazgo de forma accionable:

1. **Ciclos** — el grafo de dependencias entre fases contiene al menos un ciclo, por lo que no es un DAG. El error identifica las fases involucradas en el ciclo.
2. **Artefactos faltantes** — una fase declara un `input` (artefacto) que no es el `brief` inicial ni es producido como `output` por ninguna fase predecesora. El error identifica la fase, el artefacto y por qué falta.
3. **Agentes inexistentes** — una fase referencia en `agent` un agente que no existe entre las definiciones/catálogo de agentes del workspace. El error identifica la fase y el agente referenciado.

El validador produce un resultado que distingue **válido** de **inválido** y, en caso de inválido, una lista de hallazgos con suficiente información para corregir (fase, campo, valor observado, motivo). El conjunto de problemas detectables puede ampliarse, pero estos tres son obligatorios. El reporte es legible y accionable (RF-9.3, RNF-8: operaciones seguras/preview).

## Requerimientos

- R1. Existe un validador que recibe un workflow ya cargado (modelo del ticket 04-01) y produce un resultado **válido / inválido** con una lista de hallazgos.
- R2. **Detección de ciclos:** si el grafo de dependencias contiene un ciclo, el validador lo reporta como inválido e identifica las fases que forman el ciclo.
- R3. **Artefactos faltantes:** si una fase consume un `input` que no es el artefacto inicial (`brief`) ni es `output` de ninguna fase predecesora, el validador lo reporta, identificando fase y artefacto.
- R4. **Agentes inexistentes:** si una fase referencia un `agent` ausente del catálogo/definiciones de agentes del workspace, el validador lo reporta, identificando fase y agente.
- R5. Cada hallazgo es **accionable**: indica al menos la fase afectada, el tipo de problema, el valor observado y el motivo, en lenguaje claro.
- R6. Un workflow correcto (DAG sin ciclos, todos los inputs producidos por predecesores o iniciales, todos los agentes existentes) se reporta como **válido**, sin hallazgos.
- R7. La validación es **determinista**: el mismo workflow produce siempre el mismo resultado y el mismo orden de hallazgos (apto para diffs y tests golden — RNF-5).
- R8. La validación no panic ante entradas degeneradas (workflow vacío, fase sin dependencias, listas vacías); las maneja como casos definidos.
- R9. El validador es agnóstico del backend: valida el modelo canónico, sin lógica específica de Claude Code.

## Criterios de aceptación

- [ ] CA1. Un workflow con un ciclo en sus dependencias se reporta como inválido, con un hallazgo que identifica las fases del ciclo.
- [ ] CA2. Un workflow donde una fase consume un artefacto no producido por ninguna fase predecesora (ni inicial) se reporta como inválido, identificando fase y artefacto faltante.
- [ ] CA3. Un workflow que referencia un agente inexistente se reporta como inválido, identificando fase y agente.
- [ ] CA4. Un workflow correcto se reporta como **válido**, sin hallazgos.
- [ ] CA5. Cada hallazgo incluye fase afectada, tipo de problema, valor observado y motivo, de forma accionable.
- [ ] CA6. La validación es determinista: misma entrada → mismo resultado y mismo orden de hallazgos.
- [ ] CA7. Entradas degeneradas (workflow vacío, fases sin dependencias) se manejan sin panic.

## Fuera de alcance

- Modelo, parseo y edición del workflow YAML (ticket 04-01); este ticket asume un workflow ya cargado.
- Visualización del DAG en la TUI (ticket 04-02), aunque la vista pueda consumir este resultado.
- **Ejecución** del workflow / orquestación de agentes (fuera de scope de Fase 1 — PRD §4.2, D5).
- Validación de features avanzadas del DAG (paralelismo, condicionales) — backlog.
- Validación del esquema del propio agente (epic 02) o del manifiesto (epic 01/08), más allá de comprobar la existencia del agente referenciado.

## Referencias

- PRD.md — RF-4.3, RF-9.3 (validaciones/linters de definiciones), §8.3 (esquema de fase y pipeline), §13 (mitigación: DAG simple y validado), RNF-5 (determinismo), RNF-8 (operaciones seguras).
- init.md — §3 (glosario: workflow, gate, artefacto, agente), §6 (pipeline SDD por defecto), §8 (catálogo de agentes built-in: referencia de agentes existentes).
- epic-04-workflows-dag/epic.md — objetivo, alcance y criterios del epic.
- ticket-04-01-dag-yaml-model — modelo canónico del workflow que este validador consume.
