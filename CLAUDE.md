# CLAUDE.md — Daedalus

> Documento maestro para Claude Code trabajando en **Daedalus**. Es un **plano**, no una guía de implementación paso a paso. Estilo **SDD (Spec-Driven Development)**: todo documento describe *qué* y *bajo qué requerimientos*, no *cómo* teclear cada línea.
>
> Lecturas de contexto, en orden: [`PRD.md`](./PRD.md) → [`init.md`](./init.md) → [`Daedalus/HANDOFF.md`](./HANDOFF.md) (spec de ejecución del andamiaje).

---

## 1. Idioma

- **Conversación con el usuario:** siempre en **español**.
- **Todo lo escrito** (código, comentarios, documentación de producto, mensajes de commit, nombres de rama, logs, nombres de archivo): siempre en **inglés**. Sin excepciones.
- **Documentos de planificación del repo** (PRD, init, handoff, epics, tickets): en **español** por decisión del proyecto. El código y la documentación de producto (`documentation.md` → guía de uso) van en **inglés**.

---

## 2. Qué es Daedalus

**Daedalus** es una **TUI/CLI en Go + Charm** que automatiza el setup y la gestión —de forma **agnóstica del backend de agentes**— de la estructura de IA de un proyecto (agentes, prompts, **workflows DAG en YAML**, backlog SDD) y la **compila** al formato nativo de la herramienta elegida (primer adaptador: **Claude Code → `.claude/`**). Lightweight, sin boilerplate, para **equipos**, versionado en git. **No ejecuta** agentes en la Fase 1.

**Arquitectura:** un solo proyecto Go. *Frontend* = capa TUI/presentación (Bubble Tea, Lipgloss, Bubbles, Glamour, Huh). *Backend* = core (dominio, adaptadores, compilación, persistencia, logging/telemetría). CQRS/Docker/scripts se aplican **donde tenga sentido**, no por dogma.

El detalle vive en `PRD.md` (requerimientos, decisiones D1–D16) e `init.md` (visión, glosario, contrato de artefactos).

---

## 3. Rol del orquestador

El **chat principal de Claude Code es el orquestador**. **No implementa código de producto él mismo**: coordina, verifica secuencia, escribe los `observations.md`, actualiza estados y commitea. Delega a 5 sub-agentes definidos en `.claude/agents/`:

| Agente | Cuándo usarlo |
|---|---|
| **Obi-Wan** (`obiwan`) | Implementación de **backend / core (Go)**: dominio, adaptadores, compilación, persistencia, logging, scaffolding de dev local (Docker, Makefile, scripts), CI. |
| **Padmé** (`padme`) | Implementación de **frontend / TUI (Charm)**: Bubble Tea, Lipgloss, Bubbles, Glamour, Huh; UX, atajos, render de markdown. |
| **C-3PO** (`c3po`) | **Technical writer**: mantiene el **manual de usuario** en `docs/` (organizado como manual: índice + capítulos, fácil de seguir) y deja un **puntero** en el `documentation.md` del ticket. |
| **Yoda** (`yoda`) | **Validador backend**: corre la `validation.md` de tickets de backend/core. No implementa. |
| **Leia** (`leia`) | **Validadora frontend**: corre la `validation.md` de tickets de frontend/TUI. No implementa. |

Cada ticket se marca como **backend** o **frontend** para enrutar al par implementador/validador correcto.

---

## 4. Workflows de desarrollo

### Workflow A — "implementa el Epic X"

El orquestador procesa los tickets del epic **en orden**. Por cada ticket:

1. **Implementación.** El implementador correspondiente toma el `<slug>.md` (la spec) y lo implementa:
   - ticket de **backend/core** → **Obi-Wan**
   - ticket de **frontend/TUI** → **Padmé**
   - El prompt al implementador referencia, además de la spec, el `epic.md` del ticket y —cuando haga falta contexto profundo— el `PRD.md` (por RF/D citado) y el `init.md`. Si el implementador topa con ambigüedad que la spec no resuelve, consulta esas referencias; si sigue ambigua, **se detiene y reporta** al orquestador en vez de adivinar.
2. **Validación automática.** El validador corre `validation.md`:
   - **backend** → **Yoda** · **frontend** → **Leia**. El validador **solo reporta**, nunca arregla.
   - **Si PASA** → **C-3PO** actualiza el **manual de usuario** en `docs/` (el capítulo del feature) y deja el **puntero** en `documentation.md`. Luego, si el ticket tiene `manual-validation.md`, **el workflow SE DETIENE** y queda en manos del usuario la validación manual.
   - **Si FALLA** → el orquestador escribe/actualiza `observations.md` (en inglés, un hallazgo por ítem, con severidad y comportamiento esperado) y **reasigna** la implementación al implementador correspondiente, que corrige **basándose solo en ese documento**. **Loop validación↔fix hasta que pase.**
3. El orquestador **continúa ticket por ticket** hasta toparse con una **validación manual**, donde se detiene.
4. **Al finalizar una implementación**, el orquestador **presenta el plan/resumen al usuario**.
5. **Antes de commitear**, **espera confirmación explícita** del usuario.

### Workflow B — "fix / observaciones" (más simple)

Cuando una **validación manual falla** o se encuentra un error: el usuario describe los problemas al orquestador; este **delega** el fix a los agentes correspondientes. Al finalizar, si la validación pasa, el usuario puede pedir el **commit** (mismas reglas de commit).

---

## 5. Reglas de commit (críticas)

1. **Esperar confirmación explícita** del usuario antes de cualquier `git commit`.
2. **Sin atribución a IA, nunca.** Ni `Co-Authored-By`, ni footers "Generated with Claude Code", ni emojis de robot, ni mención de la herramienta en ningún lado. El historial muestra solo la identidad del dueño de la cuenta de git. *(Esta regla anula el default de agregar trailer de co-autoría.)*
3. El commit es **responsabilidad absoluta del dueño de la cuenta de git** del usuario.
4. Mensajes de commit en **inglés** (Conventional Commits).
5. **Nunca** modificar `git config` (user.name, user.email).

---

## 6. Estructura `development/`

```
development/
  epics/
    epic-NN-<slug>/
      epic.md                      # descripción SDD del epic (objetivo, alcance, tickets, criterios)
      tickets/
        ticket-NN-MM-<slug>/
          <slug>.md                # SPEC del ticket: el feature a implementar (qué/requerimientos/criterios)
          validation.md            # validación AUTOMÁTICA: cómo validar el feature (la corre el validador)
          documentation.md         # PUNTERO al capítulo del manual de usuario en docs/ (lo mantiene C-3PO)
          manual-validation.md     # OPCIONAL: casos de prueba manuales para alguien sin background de testing
          observations.md          # se CREA solo si la validación automática falla (feedback para reimplementar)
```

**Convenciones de nombres:** `kebab-case`. Epics: `epic-NN-<slug>`. Tickets: `ticket-NN-MM-<slug>` (NN = epic, MM = secuencia dentro del epic). La carpeta del ticket es su id.

### Manual de usuario en `docs/`

La **documentación de producto** (guía de uso para el usuario final) **no** vive en `development/`: vive en `docs/` en la raíz del repo, organizada como un **manual** (índice + capítulos, fácil de seguir). `development/` contiene solo el **backlog SDD** (spec, validación, validación manual, observaciones). Cada `documentation.md` de un ticket es un **puntero** al capítulo correspondiente del manual. C-3PO mantiene el manual **a la par** de cada feature implementado.

```
docs/
  README.md                 # índice del manual
  getting-started/          # instalación, quickstart
  guide/                    # uso del producto (CLI, comandos, configuración)
  contributing/             # trabajar sobre Daedalus (build, tooling, CI)
```

### Contrato de documentos del ticket

| Documento | Quién lo produce | Propósito |
|---|---|---|
| `<slug>.md` | Humano / planner | **Spec** del feature: descripción, requerimientos, criterios de aceptación. |
| `validation.md` | Humano / planner | Cómo **validar automáticamente** que el feature quedó bien. Ejecutable/verificable por un agente (Yoda/Leia). |
| `documentation.md` | **C-3PO** | **Puntero** al capítulo correspondiente del **manual de usuario** (`docs/`). El manual —no este archivo— es la guía de uso creciente para quien clona/descarga el producto. |
| `manual-validation.md` | Humano / planner | **Casos de prueba manuales** para alguien sin background de testing: qué probar, cómo correrlo y qué esperar. **No para todos los tickets**; específico, no redundante. |
| `observations.md` | Orquestador (desde el verdict del validador) | Observaciones cuando la validación automática **falla**; feedback accionable para reimplementar. Se crea/actualiza en cada fallo. |

---

## 7. Filosofía SDD

- Todo documento es un **plano**: describe features, requerimientos y arquitectura. **No** es una receta de implementación línea a línea.
- **Trazabilidad:** todo ticket referencia su epic; todo epic referencia los RF del `PRD.md` que lo originan.
- **Idempotencia y no destrucción:** las operaciones de escritura de Daedalus ofrecen preview/diff y no destruyen cambios manuales fuera del área gestionada.
- **Determinismo:** la compilación es reproducible (golden files).

---

## 8. Referencias

- [`PRD.md`](./PRD.md) — Product Requirements Document, Fase 1 (RF-1.x … RF-9.x, decisiones D1–D12).
- [`init.md`](./init.md) — Lineamiento maestro: visión, glosario, contrato de artefactos, pipeline SDD.
- [`HANDOFF.md`](./HANDOFF.md) — Spec de ejecución del andamiaje (agentes, epics, workflows, decisiones D13–D16).
