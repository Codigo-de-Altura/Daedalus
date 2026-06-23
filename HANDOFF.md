# Daedalus — HANDOFF / Spec de ejecución

> **Qué es este documento:** la **compresión** de la sesión de descubrimiento + todos los requerimientos para construir el andamiaje del proyecto. Está pensado para que un **chat nuevo y limpio** ejecute sin releer la conversación original.
>
> **Cómo usarlo:** abrí un chat nuevo en `F:\Codigo de Altura\Daedalus`, leé en orden [`PRD.md`](./PRD.md), [`init.md`](./init.md) y este `HANDOFF.md`, y ejecutá el **Checklist de ejecución** (sección 9).
>
> **Estado:** v0.1 · **Fecha:** 2026-06-23 · **Idioma de trabajo:** ver sección 7.

---

## 1. Contexto del proyecto (resumen)

**Daedalus** es una **TUI/CLI en Go + Charm** que automatiza el setup y la gestión —de forma **agnóstica del backend de agentes**— de la estructura de IA de un proyecto (agentes, prompts, **workflows DAG en YAML**, backlog SDD) y la **compila** al formato nativo de la herramienta elegida (primer adaptador: **Claude Code → `.claude/`**). Lightweight, sin boilerplate, para **equipos**, versionado en git. **No ejecuta** agentes en MVP. Detalle completo en `PRD.md` e `init.md`.

**Metodología:** **SDD (Spec-Driven Development)**. Todo documento es un **plano** (features, requerimientos, arquitectura), **no** una guía de implementación.

**Arquitectura (confirmada esta sesión):** un **solo proyecto Go**. *Frontend* = capa **TUI/presentación** (Bubble Tea, Lipgloss, Bubbles, Glamour, Huh). *Backend* = **core** (dominio, adaptadores, compilación, persistencia, logging/telemetría). CQRS/Docker/scripts se aplican **donde tenga sentido**.

---

## 2. Objetivo de la próxima fase (lo que ejecuta el chat nuevo)

1. Crear **`CLAUDE.md`** (no existe todavía) — sección 6.
2. Crear los **5 agentes** en `.claude/agents/` — sección 5.
3. Crear la estructura **`development/epics/...`** con **todos los epics y sus tickets** — secciones 3, 4 y 8.

Todo en estilo **SDD**.

---

## 3. Estructura de carpetas `development/`

```
development/
  epics/
    epic-00-<slug>/
      epic.md                      # descripción SDD del epic (objetivo, alcance, tickets, criterios)
      tickets/
        ticket-00-01-<slug>/
          <slug>.md                # SPEC del ticket: el feature a implementar (qué/requerimientos/criterios de aceptación)
          validation.md            # validación AUTOMÁTICA: cómo validar el feature (la corre el agente validador)
          documentation.md         # guía de uso para el USUARIO FINAL (la escribe el technical writer)
          manual-validation.md     # OPCIONAL: casos de prueba manuales para alguien sin background de testing
          observations.md          # se CREA solo si la validación automática falla (feedback para reimplementar)
```

**Convenciones de nombres:** `kebab-case`. Epics: `epic-NN-<slug>`. Tickets: `ticket-NN-MM-<slug>` (NN = epic, MM = secuencia dentro del epic). Carpeta del ticket = id del ticket.

### Contrato de cada documento del ticket

| Documento | Quién lo produce | Propósito | Notas |
|---|---|---|---|
| `<slug>.md` | Humano (o planner) | **Spec** del feature a implementar. SDD: descripción, requerimientos, criterios de aceptación. | El slug describe el ticket. |
| `validation.md` | Humano (o planner) | Cómo **validar automáticamente** que el feature quedó bien. La corre el **agente validador** correspondiente. | Debe ser ejecutable/verificable por un agente. |
| `documentation.md` | **C-3PO** (technical writer) | **Guía de uso para el usuario final** (quien clona/descarga el producto), no para devs del proyecto. Se va escribiendo a la par del desarrollo y alimenta una **guía de uso** creciente. | Foco en el usuario, no en internals. |
| `manual-validation.md` | Humano (o planner) | **Casos de prueba manuales** para alguien **sin** background de testing: qué probar, **cómo correrlo** y **qué esperar**. | **No para todos los tickets.** Específico, **no redundante**. |
| `observations.md` | **Validador** (Yoda/Leia) | Observaciones cuando la validación automática **falla**; feedback para que el implementador corrija. | Se crea/actualiza en cada fallo; se reimplementa hasta pasar. |

---

## 4. Workflows de desarrollo

> Estos workflows se documentan en `CLAUDE.md` y guían al **orquestador** (el chat principal de Claude Code).

### Workflow A — "**implementa el Epic X**"

El orquestador procesa los tickets del epic **en orden**. Por cada ticket:

1. **Implementación.** El implementador correspondiente toma `<slug>.md` y lo implementa:
   - ticket de **backend/core** → **Obi-Wan**
   - ticket de **frontend/TUI** → **Padmé**
2. **Validación automática.** El **validador** correspondiente corre `validation.md`:
   - **backend** → **Yoda** · **frontend** → **Leia**
   - **Si PASA** → **C-3PO** escribe/actualiza `documentation.md` → si el ticket tiene `manual-validation.md`, **el workflow SE DETIENE** y queda en manos del usuario hacer la validación manual.
   - **Si FALLA** → se crea/actualiza `observations.md` y se **reasigna** la implementación al implementador correspondiente. **Loop hasta que pase.**
3. El orquestador **continúa ticket por ticket** hasta toparse con una **validación manual**, donde se detiene.
4. **Al finalizar una implementación**, el orquestador **presenta el plan/resumen al usuario**.
5. **Antes de commitear**, **espera confirmación explícita** del usuario.

### Workflow B — "**fix / observaciones**" (más simple)

Cuando una **validación manual falla** o se encuentra un error: el usuario le describe los problemas al **orquestador**; este **delega** el fix a los agentes correspondientes. Al finalizar, si la validación pasa, el usuario puede pedirle que **commitee** (mismas reglas de commit).

### Reglas de commit (críticas)

- **Esperar confirmación** del usuario antes de cualquier `git commit`.
- **Sin `Co-Authored-By` ni ninguna atribución al agente/IA.** El commit es **responsabilidad absoluta del dueño de la cuenta de git** del usuario. *(Esta instrucción del usuario anula el default de agregar trailer de co-autoría.)*
- Mensajes de commit en **inglés**.

---

## 5. Agentes `.claude/agents/`

**Formato de header** (verificado contra `QuienPuede/.claude/agents/aragorn.md`, para que Claude Code los reconozca):

```markdown
---
name: <lowercase>
description: <rol en una línea>
tools: Read, Edit, Write, Glob, Grep, Bash, Agent
model: sonnet
color: <color>
---

# <Nombre> — <Rol>

<persona detallada: identidad, expertise, cómo trabaja, qué rechaza hacer…>
```

**Regla de idioma para TODOS los agentes** (incluir en cada persona): conversan con el usuario **siempre en español**; todo lo que escriben en **código, comentarios, documentación, mensajes de commit, logs y nombres de archivo es siempre en inglés**.

### Roster (Star Wars Ep. 1–6; la personalidad define el rol)

| Archivo | Nombre | Rol | Personalidad / base | Tools sugeridas |
|---|---|---|---|---|
| `obiwan.md` | **Obi-Wan Kenobi** | **Backend / core (Go)** | Maestro disciplinado y experimentado. | Read, Edit, Write, Glob, Grep, Bash, Agent |
| `padme.md` | **Padmé Amidala** | **Frontend / TUI (Charm)** | Elegancia, UX, representa al usuario. | Read, Edit, Write, Glob, Grep, Bash, Agent |
| `c3po.md` | **C-3PO** | **Technical writer** | "Fluent in over six million forms of communication"; preciso y claro. | Read, Edit, Write, Glob, Grep |
| `yoda.md` | **Yoda** | **Validador backend** | Gran maestro que juzga con rigor ("do or do not"). | Read, Glob, Grep, Bash, Write |
| `leia.md` | **Leia Organa** | **Validadora frontend** | Exigente, abogada del usuario, ojo afilado. | Read, Glob, Grep, Bash, Write |

### Notas de persona

- **Obi-Wan (backend):** arquitecto Go senior. **CQRS si es necesario** (no por dogma). Experto en **patrones de diseño** (los aplica desde la experiencia, no por moda), **clean code**, **SOLID**, buenas prácticas. **Fanático de logging y telemetría**: busca **visibilidad absoluta** logueando en **puntos de decisión críticos** (no en cada entrada/salida de método), **sin ensuciar** los logs ni loguear datos sensibles. **Fanático del scaffolding de desarrollo local**: Docker, Docker Compose, scripts, Makefiles → **onboarding seamless** para cualquier dev. Analiza el *blast radius* antes de cambiar; prefiere cambios aditivos; interfaces sobre implementaciones. *(Mismo espíritu que el `aragorn.md` de QuienPuede, pero el stack acá es **Go**, no .NET.)*
- **Padmé (frontend/TUI):** dominio de **Bubble Tea / Lipgloss / Bubbles / Glamour / Huh**. Estética pulida y consistente, ergonomía de teclado, ayuda contextual, render de markdown hermoso. Aboga por la experiencia del usuario.
- **C-3PO (technical writer):** escribe la **guía de uso para el usuario final** (quien clona/descarga el producto). Claro, estructurado, no técnico-internista. Cada `documentation.md` alimenta una guía de uso creciente del producto.
- **Yoda / Leia (validadores):** corren la `validation.md` del ticket, **no** implementan. Si falla, escriben `observations.md` con feedback accionable. Rigurosos y escépticos.

> Modelo por defecto: `model: sonnet` (igual que la convención existente), salvo que el usuario indique otra cosa. Asignar `color` distinto a cada uno.

---

## 6. `CLAUDE.md` (a crear)

Debe contener, en estilo SDD:

1. **Idioma:** toda comunicación con el usuario en **español**; código/docs/comentarios/commits/logs/nombres en **inglés**.
2. **Rol del orquestador** y delegación a los 5 agentes (cuándo usar cada uno).
3. **Workflow A** ("implementa el Epic X") y **Workflow B** (fix/observaciones) — sección 4.
4. **Reglas de commit:** confirmación previa del usuario; **sin co-authored ni atribución a IA**; responsabilidad del dueño de la cuenta; mensajes en inglés.
5. **Estructura `development/`** y **contrato de documentos del ticket** — sección 3.
6. **Filosofía SDD:** los docs son planos, no guías de implementación.
7. Referencia a `PRD.md`, `init.md` y a este `HANDOFF.md`.

---

## 7. Reglas de idioma (global)

- **Conversación con el usuario:** español.
- **Todo lo escrito** (código, comentarios, documentación, mensajes de commit, logs, nombres de archivo): **inglés**.

> Nota: los documentos de planificación de este repo (PRD, init, handoff, epics/tickets) están en **español** por decisión del proyecto; el **código y la documentación de producto** van en **inglés**.

---

## 8. Epics a crear (derivados del PRD)

> El chat nuevo crea cada `epic.md` y deriva sus **tickets** desde los requerimientos funcionales (RF) del `PRD.md`. Granularidad sugerida: 1 ticket por feature coherente. Marcar tickets como **backend** o **frontend** para enrutar al implementador/validador.

| Epic | Origen PRD | Foco |
|---|---|---|
| `epic-00-foundations` | (nuevo) | Scaffolding del repo Go, módulos Charm, **dev local (Docker/Makefile/scripts)**, baseline de **logging/telemetría**, CI. *(Terreno de Obi-Wan.)* |
| `epic-01-init-scaffolding` | EPIC-1 (RF-1.x) | `daedalus init`, workspace `.daedalus/`, manifiesto, detección/upgrade. |
| `epic-02-agent-management` | EPIC-2 (RF-2.x) | Catálogo built-in, importar/clonar/editar agentes, validación de esquema. |
| `epic-03-prompt-management` | EPIC-3 (RF-3.x) | Prompts globales/compartidos, composición, preview. |
| `epic-04-workflows-dag` | EPIC-4 (RF-4.x) | DAG YAML: crear/editar/visualizar/validar; `sdd-default.yaml`. |
| `epic-05-sdd-backlog` | EPIC-5 (RF-5.x) | Brief→spec (analyst), arquitectura, epics, tickets, trazabilidad. |
| `epic-06-compile-claude-adapter` | EPIC-6 (RF-6.x) | `daedalus build`/`sync`, adaptador Claude Code, idempotencia, diff/preview. |
| `epic-07-tui-ux` | EPIC-7 (RF-7.x) | Navegación, estética, atajos, render markdown, fluidez. |
| `epic-08-state-collab` | EPIC-8 (RF-8.x) | Estado en git, formatos diff-friendly, convenciones de equipo. |
| `epic-09-logging-testing` | EPIC-9 (RF-9.x) | Logging estructurado, testing (golden files de compilación), linters de definiciones. |

> Recordar: cada ticket = carpeta con `<slug>.md`, `validation.md`, `documentation.md`, y **opcionalmente** `manual-validation.md`. `observations.md` se crea recién al fallar una validación.

---

## 9. Checklist de ejecución (para el chat nuevo)

- [ ] Leer `PRD.md`, `init.md` y este `HANDOFF.md`.
- [ ] Crear **`CLAUDE.md`** (sección 6).
- [ ] Crear los **5 agentes** en `.claude/agents/` (`obiwan.md`, `padme.md`, `c3po.md`, `yoda.md`, `leia.md`) con el header verificado y las personas de la sección 5.
- [ ] Crear `development/epics/` con los **10 epics** (sección 8): cada uno con su `epic.md`.
- [ ] Para cada epic, crear sus **tickets** (carpeta + documentos del contrato, sección 3), derivados de los RF del PRD.
- [ ] Confirmar reglas de commit y de idioma en `CLAUDE.md`.
- [ ] Presentar el plan al usuario antes de avanzar con cargas grandes; esperar confirmación antes de commitear.

---

## 10. Decisiones tomadas (trazabilidad)

Ver `PRD.md` §6 (D1–D12) +:
- **D13:** Arquitectura = un solo proyecto Go; frontend = TUI (Charm), backend = core Go.
- **D14:** Agentes = Obi-Wan (backend), Padmé (frontend), C-3PO (technical writer), Yoda (validador backend), Leia (validadora frontend).
- **D15:** Commits sin co-authored / sin atribución a IA; confirmación previa del usuario.
- **D16:** Esta sesión entrega solo este HANDOFF (compresión); la implementación (CLAUDE.md, agentes, epics) va en chat nuevo.

---

## 11. Ítems abiertos

- Granularidad final de tickets por epic (la define el chat nuevo al crear el backlog).
- Dónde se ensambla la **guía de uso** del usuario final (¿`docs/usage/`?) a partir de los `documentation.md`.
- Esquema canónico fino de agente/workflow y mapeo exacto canónico → Claude Code (heredado del PRD §15).
- `model`/`color` definitivos de cada agente.
