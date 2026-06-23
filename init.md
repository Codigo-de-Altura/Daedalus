# Daedalus — `init.md` (Lineamiento maestro)

> **Propósito de este documento:** es el **punto de entrada** del proyecto. Cualquier persona —o agente— que vaya a trabajar en Daedalus lee este archivo primero. Define la visión, el vocabulario, las convenciones, el mapa de la estructura y el **contrato de artefactos**, de modo que la **Fase 2** (definir prompts, agentes, epics y tickets) pueda arrancar sin ambigüedad.
>
> **Estado:** v0.1 · **Fecha:** 2026-06-23 · Complementa al [`PRD.md`](./PRD.md).

---

## 1. Visión y propósito

**Daedalus** es una **TUI/CLI en Go (Charm)** que **automatiza el setup y la gestión** de la estructura de IA de un proyecto —agentes, prompts, workflows y backlog SDD— de forma **agnóstica del backend de agentes**, y la **compila** al formato nativo de la herramienta que uses (p. ej. Claude Code).

Resuelve el dolor central de desarrollar con IA hoy: **el andamiaje**. En vez de rehacer manualmente prompts globales, agentes, sub-agentes, workflows y backlog en cada proyecto, apuntás Daedalus a un repo, describís un **brief**, y obtenés un ecosistema SDD listo y versionado.

Compite con productos tipo **AIDLC**, pero **lightweight y sin boilerplate**.

---

## 2. Principios rectores

1. **SDD (Spec-Driven Development).** La documentación son **planos**: features, requerimientos, arquitectura. No recetas de implementación.
2. **Lightweight, sin boilerplate.** Binario único, arranque instantáneo, definiciones mínimas y limpias.
3. **Agnóstico del backend.** Una definición canónica que se **compila** a múltiples backends.
4. **Fuente de verdad única, en git.** Pensado para **equipos**; todo en formatos de texto diffeables.
5. **Control total de la estructura.** El usuario controla prompts, agentes, workflows y backlog. (La *ejecución* de agentes vive fuera de Daedalus en Fase 1.)
6. **Dogfooding.** Daedalus se desarrolla **usando su propia metodología** (su propio `.daedalus/`).

---

## 3. Glosario

| Término | Definición |
|---|---|
| **Backend / Runtime** | Herramienta que ejecuta agentes (p. ej. Claude Code). Daedalus **no** es un backend. |
| **Definición canónica** | Representación **agnóstica** de agentes/prompts/workflows; fuente de verdad en `.daedalus/`. |
| **Adaptador** | Módulo que **compila** lo canónico al formato nativo de un backend. |
| **Compilación (`build`/`sync`)** | Genera los artefactos nativos del backend desde lo canónico. |
| **Agente** | Unidad con rol + prompt (analyst, architect, planner, validator, documenter…). |
| **Workflow** | **DAG declarativo (YAML)** de fases; cada fase = `{agent, inputs, outputs, gate}`. |
| **Artefacto** | Documento que se produce/consume (brief, spec, arquitectura, epic, ticket, doc). |
| **Backlog SDD** | Conjunto de specs, epics y tickets. |
| **Gate** | Criterio de validación que debe cumplir un artefacto para avanzar de fase. |

---

## 4. Mapa de la estructura

### 4.1 Repositorio de Daedalus (el producto)
> Estructura objetivo; se materializa en una pasada posterior. Esta sesión entrega solo `PRD.md` e `init.md`.

```
Daedalus/
  PRD.md               # plano de producto (Fase 1)
  init.md              # este documento
  README.md            # (futuro)
  cmd/daedalus/        # (futuro) entrypoint Go/Charm
  internal/            # (futuro) core, adapters, tui
  .daedalus/           # (dogfooding) estructura IA para desarrollar Daedalus
```

### 4.2 Workspace `.daedalus/` (lo que Daedalus genera en cualquier repo)
```
.daedalus/
  daedalus.yaml        # manifiesto: nombre, backend(s), versión, convenciones
  init.md              # lineamiento maestro del proyecto target
  agents/              # definiciones agnósticas (yaml + prompt md)
  prompts/             # prompts globales/compartidos
  workflows/           # DAGs declarativos (yaml) incl. sdd-default.yaml
  specs/               # spec/PRD
  architecture/        # documentos de arquitectura
  epics/               # epics
  tickets/             # tickets
  docs/                # documentación derivada
  .state/              # estado de progreso (git-tracked)
```

---

## 5. Contrato de artefactos

> Qué es cada artefacto, **quién lo produce**, **quién lo consume** y su **formato**.

| Artefacto | Produce | Consume | Formato |
|---|---|---|---|
| **brief** | Humano (usuario) | agent: analyst | Markdown |
| **spec / PRD** | agent: analyst (refina humano) | architect, planner, equipo | Markdown |
| **arquitectura** | agent: architect | planner, dev/agente externo | Markdown |
| **epic** | agent: planner | planner (tickets), equipo | Markdown + metadatos |
| **ticket** | agent: planner | dev/agente externo, validator | Markdown + metadatos |
| **implementación** | dev / agente **externo** (fuera de Daedalus) | validator | Código en el repo |
| **validación** | agent: validator | documenter, equipo | Markdown (reporte) |
| **docs** | agent: documenter | equipo, usuarios | Markdown |
| **agente (def.)** | Humano (desde catálogo built-in) | compilador/adaptador | YAML + prompt MD |
| **workflow (def.)** | Humano | compilador, TUI (visualización) | YAML (DAG) |
| **artefactos nativos** (`.claude/…`) | `daedalus build` (adaptador) | backend (Claude Code) | Formato nativo |

---

## 6. Pipeline SDD por defecto

`workflows/sdd-default.yaml`:

```
brief
  └─► spec/PRD        (analyst)
        └─► arquitectura   (architect)
              └─► epics     (planner)
                    └─► tickets   (planner)
                          └─► ⟨implementación externa⟩
                                └─► validación  (validator)
                                      └─► docs    (documenter)
```

- La **spec nace de un brief** (RF-5.1). El humano refina.
- La **implementación** ocurre **fuera** de Daedalus (Fase 1): el dev/agente la ejecuta en el backend.
- Cada arista cruza un **gate** de validación del artefacto de entrada.

---

## 7. Convenciones

- **Naming:** `kebab-case` para archivos/ids de epics, tickets, agentes y workflows (p. ej. `epic-init-scaffolding.md`, `ticket-0012-build-claude-adapter.md`).
- **Markdown:** encabezados jerárquicos, tablas para metadatos, bloques de código para esquemas/DAGs.
- **YAML:** claves estables y ordenadas (output determinista, diffs limpios).
- **Git:** ramas por epic/ticket; commits descriptivos; el estado del backlog vive versionado.
- **Idempotencia:** `build`/`sync` no destruye cambios manuales fuera del área gestionada; siempre ofrece preview/diff.
- **Trazabilidad:** todo ticket referencia su epic; todo epic referencia la spec/arquitectura de origen.

---

## 8. Catálogo de agentes built-in (visión)

| Agente | Rol |
|---|---|
| **analyst** | Convierte el brief en spec/PRD. |
| **architect** | Define la arquitectura a partir de la spec. |
| **planner** | Deriva epics y tickets desde spec + arquitectura. |
| **validator** | Verifica artefactos/implementación contra gates y criterios. |
| **documenter** | Produce documentación derivada. |

> El detalle de prompts y parámetros de cada agente se define en **Fase 2**.

---

## 9. Cómo arrancar la Fase 2

Con el PRD y este `init.md` aprobados, la Fase 2 de **planificación** consiste en:

1. **Definir prompts** globales y compartidos (convenciones, glosario, estilo SDD).
2. **Definir los agentes** del catálogo built-in (rol, prompt, inputs/outputs, gates).
3. **Definir el `sdd-default.yaml`** (el DAG concreto con sus fases).
4. **Derivar epics y tickets** para construir Daedalus (dogfooding), a partir de las EPIC-1…EPIC-9 del PRD.
5. **Materializar** la estructura de carpetas del repo y del `.daedalus/`.

---

## 10. Referencias

- [`PRD.md`](./PRD.md) — Product Requirements Document, Fase 1.
- Charm: Bubble Tea, Lipgloss, Bubbles, Glamour, Huh.
- Inspiración/competencia: Amazon AIDLC (referencia a superar en peso y boilerplate).
