# Daedalus — PRD (Product Requirements Document) · Fase 1

> **Estado:** Borrador v0.1 · **Fecha:** 2026-06-23 · **Autor:** Andy + Daedalus session
> **Tipo de documento:** *Plano* (Spec-Driven Development). Describe **qué** se construye, **por qué** y **bajo qué requerimientos/arquitectura** — **no** es una guía de implementación paso a paso.

---

## 1. Resumen ejecutivo

**Daedalus** es una **TUI/CLI** (terminal) en **Go + Charm** que **automatiza el setup y la gestión** de toda la "estructura de IA" necesaria para desarrollar software con agentes siguiendo una metodología **SDD (Spec-Driven Development)** y prácticas ágiles (epics, tickets, líneas de tiempo).

Hoy, desarrollar con IA es rápido; el dolor está en el **andamiaje** (scaffolding): definir prompts globales, agentes, sub-agentes, workflows, estructura de documentación y backlog. Daedalus elimina ese dolor: lo apuntás a un repositorio y administra —de forma **agnóstica del backend de agentes**— agentes, prompts, **workflows declarativos (DAG en YAML)** y el backlog SDD, **compilándolos** al formato nativo de la herramienta de agentes que uses (p. ej. Claude Code).

Daedalus busca competir con productos tipo **Amazon AIDLC**, pero **bien implementado, lightweight y sin boilerplate**: el usuario edita definiciones limpias, no cinco formatos nativos.

---

## 2. Problema y contexto

- Los avances en IA permiten construir aplicaciones muy rápido, pero el **setup de herramientas, workflows y agentes** es lento, repetitivo y propenso a inconsistencias.
- Iniciativas tipo **AIDLC** intentan estandarizar el ciclo de desarrollo con IA, pero tienden a ser **pesadas y con mucho boilerplate**.
- Quien trabaja con estrategias ágiles + IA termina **rehaciendo manualmente** el mismo scaffolding en cada proyecto: prompts globales, definición de agentes, sub-agentes especializados (documentación, implementación, validación), workflows, estructura de epics/tickets.
- No existe una herramienta **lightweight, agnóstica y centrada en SDD** que sea la **fuente de verdad** de esa estructura y la compile a la herramienta de agentes elegida.

---

## 3. Visión del producto

> Apuntás Daedalus a un repo, describís tu idea en un **brief**, y obtenés —listo para usar— un ecosistema de desarrollo con IA: spec/PRD, arquitectura, epics, tickets, agentes y workflows, todo versionado en git y compilado al backend que prefieras. Controlás **toda** la estructura del proceso desde una TUI bella y eficiente.

Principios rectores:

1. **Lightweight, sin boilerplate.** Binario único, arranque instantáneo, definiciones mínimas y limpias.
2. **Spec-Driven Development.** La documentación son **planos** (features, requerimientos, arquitectura), no recetas de implementación.
3. **Agnóstico del backend.** Una definición canónica que se compila a múltiples backends de agentes.
4. **Fuente de verdad única, versionable en git.** Pensado para **equipos**.
5. **Control total de la estructura.** El usuario tiene control absoluto sobre prompts, agentes, workflows y backlog (la *ejecución* de agentes vive fuera de Daedalus en Fase 1).

---

## 4. Objetivos y no-objetivos (scope de Fase 1)

### 4.1 Objetivos
- Inicializar (`init`) la estructura de IA de un proyecto en un workspace `.daedalus/` dentro de cualquier repo.
- Gestionar **agentes** a partir de un **catálogo built-in** (importar, clonar, editar).
- Gestionar **prompts** globales y compartidos.
- Definir y editar **workflows** como **DAG declarativo en YAML**.
- Gestionar el **backlog SDD**: spec/PRD, arquitectura, epics, tickets — con la spec **generada desde un brief** por un agente *analista*.
- **Compilar** (`build`/`sync`) la definición canónica al formato nativo del backend (primer adaptador: **Claude Code** → `.claude/`).
- TUI **estética, eficiente y con buena UX**.
- Soporte a **equipos**: estado versionado en git, convenciones compartidas.
- Ecosistema de soporte: **logging** y **testing**.

### 4.2 No-objetivos (Fase 1)
- **Ejecutar/orquestar agentes en vivo** (lanzar subprocesos, streamear logs de runs, intervenir mid-run). → Visión Fase 2+.
- **Registry/marketplace remoto** de agentes. → Post-MVP.
- **Interfaz gráfica (GUI)**. → Fase 2 (exploratoria).
- Adaptadores para backends distintos a Claude Code. → Interfaz lista, implementación posterior.

---

## 5. Usuarios

| Persona | Necesidad principal |
|---|---|
| **Dev/Tech lead que adopta IA** (usuario primario) | Arrancar proyectos con IA sin rehacer scaffolding; controlar prompts/agentes/workflows. |
| **Equipo de desarrollo** | Compartir y versionar la misma estructura IA, con convenciones consistentes. |
| **Daedalus mismo (dogfooding)** | El propio desarrollo de Daedalus usa su metodología SDD y su `.daedalus/`. |

---

## 6. Decisiones de producto (esta sesión)

| # | Decisión | Valor elegido |
|---|---|---|
| D1 | Nombre del proyecto | **Daedalus** (arquitecto/artesano mítico; planos, laberinto, alas) |
| D2 | Stack TUI | **Go + Charm** (Bubble Tea, Lipgloss, Bubbles, Glamour, Huh) |
| D3 | Runtime de agentes | **Agnóstico multi-backend**; Claude Code = primer adaptador |
| D4 | Topología | **CLI global** → workspace `.daedalus/` por repo |
| D5 | Ejecución de agentes | **Fuera de scope MVP**: Daedalus gestiona config; el usuario corre los agentes |
| D6 | Importación de agentes | **Catálogo built-in** (remoto = post-MVP) |
| D7 | Público objetivo | **Equipos** (estado en git) |
| D8 | Compilación | **Fuente canónica agnóstica + compilación por adaptador** al formato nativo |
| D9 | Representación de workflow | **DAG declarativo en YAML** |
| D10 | Origen de la spec | **Agente genera spec desde un brief**, el usuario refina |
| D11 | Dogfooding | **Sí**: Daedalus se desarrolla con su propia metodología/`.daedalus/` |
| D12 | Alcance de esta sesión | **PRD + `init.md`** (estructura de carpetas: pasada posterior) |

---

## 7. Conceptos clave (glosario operativo)

- **Backend / Runtime de agentes:** la herramienta que efectivamente ejecuta los agentes (p. ej. Claude Code). Daedalus **no** es un backend.
- **Definición canónica:** representación **agnóstica** de agentes/prompts/workflows que vive en `.daedalus/` y es la fuente de verdad.
- **Adaptador:** módulo que sabe **compilar** la definición canónica al formato nativo de un backend específico.
- **Compilación (`build`/`sync`):** proceso que genera los artefactos nativos del backend (p. ej. `.claude/agents/*.md`, `.claude/commands/*.md`) desde la definición canónica.
- **Agente:** unidad de trabajo con rol y prompt (p. ej. *analyst*, *architect*, *planner*, *validator*, *documenter*).
- **Workflow:** **DAG declarativo (YAML)** de fases; cada fase referencia un agente, sus artefactos de entrada/salida y un *gate* de validación.
- **Artefacto:** documento producido/consumido en el pipeline (brief, spec/PRD, arquitectura, epic, ticket, doc).
- **Backlog SDD:** conjunto de specs, epics y tickets que describen el trabajo.

---

## 8. Modelo de dominio y arquitectura conceptual

### 8.1 Fuente de verdad agnóstica → compilación por adaptador

```
                 ┌─────────────────────────────┐
   brief ──────► │  .daedalus/ (canónico)      │
                 │  agents/ prompts/ workflows/│
                 │  specs/ epics/ tickets/     │
                 └──────────────┬──────────────┘
                                │  daedalus build (adaptador)
                                ▼
                 ┌─────────────────────────────┐
                 │  Formato nativo del backend │
                 │  ej. .claude/agents/*.md    │
                 │      .claude/commands/*.md  │
                 └──────────────┬──────────────┘
                                │  el usuario ejecuta (fuera de Daedalus)
                                ▼
                        Backend (Claude Code, …)
```

- **Una** fuente de verdad versionada → **portabilidad** entre backends (recompilar, no reescribir).
- El usuario edita **definiciones limpias**, no formatos nativos.

### 8.2 Estructura propuesta de `.daedalus/` (workspace por repo)

> *Propuesta para iterar; se materializa en una pasada posterior.*

```
.daedalus/
  daedalus.yaml        # manifiesto: nombre, backend(s), versión, convenciones
  init.md              # lineamiento maestro del proyecto
  agents/              # definiciones agnósticas (yaml + prompt md)
  prompts/             # prompts globales/compartidos
  workflows/           # DAGs declarativos (yaml), incl. sdd-default.yaml
  specs/               # spec/PRD generadas/refinadas
  architecture/        # documentos de arquitectura
  epics/               # epics
  tickets/             # tickets
  docs/                # documentación derivada
  .state/              # estado de progreso (git-tracked)
```

### 8.3 Pipeline SDD por defecto (`workflows/sdd-default.yaml`)

```
brief
  └─► spec/PRD        (agent: analyst)
        └─► arquitectura   (agent: architect)
              └─► epics     (agent: planner)
                    └─► tickets   (agent: planner)
                          └─► ⟨implementación externa: dev/agente⟩
                                └─► validación  (agent: validator)
                                      └─► docs    (agent: documenter)
```

Cada paso del DAG define: `{ agent, inputs[], outputs[], gate }`.

---

## 9. Requerimientos funcionales

> Agrupados por área (candidatas a **epics**). Cada ítem es una *feature description*, no una guía de implementación.

### EPIC-1 · Inicialización y scaffolding
- **RF-1.1** `daedalus init` crea/gestiona el workspace `.daedalus/` en el repo objetivo (nuevo o existente).
- **RF-1.2** Detecta si ya existe `.daedalus/` y ofrece *upgrade*/merge no destructivo.
- **RF-1.3** Genera el `daedalus.yaml` (manifiesto) y el `init.md` base del proyecto.
- **RF-1.4** Permite elegir el/los backend(s) objetivo (MVP: Claude Code).

### EPIC-2 · Gestión de agentes
- **RF-2.1** Catálogo **built-in** de agentes (mínimo: *analyst, architect, planner, validator, documenter*).
- **RF-2.2** Importar/clonar un agente del catálogo al workspace y **editarlo** (rol, prompt, parámetros).
- **RF-2.3** Importar agentes desde **archivos locales** (incluye estructuras `.claude/agents` existentes).
- **RF-2.4** Validación de la definición de agente (esquema canónico).

### EPIC-3 · Gestión de prompts
- **RF-3.1** Editar **prompts globales** y prompts compartidos reutilizables.
- **RF-3.2** Mecanismo de composición/inclusión de prompts (DRY: convenciones, glosario, etc.).
- **RF-3.3** Previsualización del prompt renderizado.

### EPIC-4 · Workflows (DAG)
- **RF-4.1** Crear/editar workflows como **DAG declarativo en YAML**.
- **RF-4.2** Visualizar el DAG en la TUI (nodos = fases/agentes, aristas = dependencias).
- **RF-4.3** Validar el DAG (ciclos, artefactos faltantes, referencias a agentes inexistentes).
- **RF-4.4** Incluir el **`sdd-default.yaml`** como workflow de fábrica.

### EPIC-5 · Backlog SDD (spec, arquitectura, epics, tickets)
- **RF-5.1** Capturar un **brief** y, vía agente *analyst*, generar una **spec/PRD** (el usuario refina).
- **RF-5.2** Gestionar documentos de **arquitectura**.
- **RF-5.3** Crear/gestionar **epics** y **tickets** con metadatos (estado, prioridad, dependencias, links a artefactos).
- **RF-5.4** Trazabilidad spec → epic → ticket.

### EPIC-6 · Compilación a backend (adaptador)
- **RF-6.1** `daedalus build`/`sync` compila la definición canónica al formato nativo del backend.
- **RF-6.2** Adaptador **Claude Code**: genera `.claude/agents/*.md` (frontmatter), `.claude/commands/*.md`, settings relevantes.
- **RF-6.3** Build **idempotente** y no destructivo de cambios manuales fuera del área gestionada.
- **RF-6.4** Reporte de diff/preview antes de escribir.

### EPIC-7 · TUI / Experiencia de usuario
- **RF-7.1** Navegación por áreas (init, agentes, prompts, workflows, backlog, build).
- **RF-7.2** Estética cuidada (Lipgloss), render de markdown en terminal (Glamour), formularios (Huh).
- **RF-7.3** Atajos de teclado consistentes y ayuda contextual.
- **RF-7.4** Operación fluida y de bajo consumo.

### EPIC-8 · Estado, persistencia y colaboración
- **RF-8.1** Todo el estado relevante vive en archivos **versionables en git**.
- **RF-8.2** Formatos amigables a diff/merge (yaml/markdown legibles).
- **RF-8.3** Convenciones de equipo (naming, estructura) explicitadas y validables.

### EPIC-9 · Ecosistema (logging y testing)
- **RF-9.1** **Logging** estructurado de las operaciones de Daedalus (init, build, validaciones).
- **RF-9.2** Estrategia de **testing** del propio Daedalus (unidad + golden files de compilación).
- **RF-9.3** Validaciones/linters de definiciones (agentes, workflows, manifiesto).

---

## 10. Requerimientos no funcionales

- **RNF-1 · Lightweight:** binario único, sin dependencias de runtime; arranque < 200 ms objetivo.
- **RNF-2 · Performance/UX:** interacción fluida, sin bloqueos perceptibles en operaciones comunes.
- **RNF-3 · Portabilidad:** Windows, macOS, Linux.
- **RNF-4 · Estética:** salida visualmente pulida y consistente (tema/colores, markdown renderizado).
- **RNF-5 · Determinismo:** la compilación es reproducible (mismo input → mismo output; *golden files*).
- **RNF-6 · Git-friendly:** artefactos en formatos de texto, ordenados y estables para minimizar ruido en diffs.
- **RNF-7 · Extensibilidad:** la interfaz de **adaptador** permite añadir backends sin tocar el núcleo.
- **RNF-8 · Seguridad/safety:** operaciones de escritura no destructivas por defecto; preview/confirm.

---

## 11. Arquitectura técnica (alto nivel)

> Visión arquitectónica, **no** guía de implementación.

- **Lenguaje/UI:** Go + **Charm** (Bubble Tea = arquitectura Elm; Lipgloss = estilos; Bubbles = componentes; Glamour = render markdown; Huh = formularios).
- **Núcleo (core):** modelo de dominio canónico (agents, prompts, workflows, backlog) + validación de esquemas.
- **Adaptadores:** interfaz `Compiler` con implementación inicial **Claude Code**.
- **Persistencia:** filesystem sobre `.daedalus/`; formatos YAML/Markdown.
- **CLI + TUI:** comandos (`init`, `build`/`sync`, validaciones) y navegación interactiva sobre el mismo core.
- **Observabilidad:** logging estructurado; (telemetría/runs en vivo → fuera de scope MVP).

---

## 12. Métricas de éxito

- **Tiempo a "proyecto listo"**: de un repo vacío + brief a estructura IA compilada en **minutos**, no horas.
- **Reducción de boilerplate**: número de archivos/decisiones manuales eliminadas vs. setup manual.
- **Portabilidad demostrada**: misma definición canónica compilando a ≥1 backend sin reescritura.
- **Dogfooding**: Daedalus desarrollado end-to-end con su propia metodología.

---

## 13. Riesgos y mitigaciones

| Riesgo | Mitigación |
|---|---|
| El esquema canónico no captura particularidades de un backend | Adaptadores con *extensiones*/escape hatches; empezar con un solo backend bien soportado. |
| Sobre-ingeniería del DAG en MVP | Mantener el DAG simple y validado; features avanzadas (paralelismo, condicionales) → backlog. |
| Compilación destructiva de cambios manuales | Build idempotente, áreas gestionadas marcadas, preview/diff antes de escribir. |
| Fricción de UX en terminal | Aprovechar Charm; pruebas de usabilidad tempranas; ayuda contextual. |
| Scope creep hacia ejecución de agentes | Línea clara: Fase 1 = config; ejecución/observabilidad = Fase 2+. |

---

## 14. Fuera de scope / Fase 2+

- Ejecución y **orquestación de agentes en vivo** (subprocesos, logs en streaming, intervención mid-run).
- **GUI** (exploratoria, idea aún no consolidada).
- **Registry/marketplace remoto** de agentes y workflows.
- Adaptadores adicionales (Codex, Gemini CLI, etc.) — la interfaz queda preparada.

---

## 15. Decisiones abiertas (para refinar)

- Detalle fino del **esquema canónico** de agente y de workflow (campos exactos, gates).
- Mapeo exacto **canónico → Claude Code** (frontmatter, comandos, settings).
- Convenciones de **naming** y de **estado** del backlog (epics/tickets).
- Estrategia de **migración/upgrade** de un `.daedalus/` existente.

---

## 16. Próximos pasos

1. Validar/ajustar este PRD.
2. Consolidar `init.md` (lineamiento maestro) — *entregado en esta sesión*.
3. **Fase 2 de planificación:** definir prompts, agentes (catálogo built-in), epics y tickets para construir Daedalus (dogfooding).
4. Materializar la **estructura de carpetas** del repo y del `.daedalus/`.
