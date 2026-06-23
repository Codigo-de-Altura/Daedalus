# Epic 00 — Foundations

> **Origen:** nuevo (no deriva de un RF concreto; es el terreno base sobre el que se construyen EPIC-1…EPIC-9). Terreno de **Obi-Wan**.
> **Estilo:** SDD — este documento es un plano (objetivo, alcance, tickets, criterios), no una guía de implementación.

## Objetivo

Establecer el andamiaje base del repositorio Go de Daedalus: módulo y layout del proyecto, dependencias del ecosistema Charm, entorno de desarrollo local sin fricción (Docker / Docker Compose / Makefile / scripts), baseline de logging/telemetría estructurada e integración continua. Al cerrar este epic, cualquier desarrollador clona el repo y, con un comando, compila, testea y corre Daedalus.

## Alcance

**Incluye:** estructura del proyecto Go (`cmd/`, `internal/`), gestión de dependencias, bootstrap mínimo de Bubble Tea, scaffolding de dev local, baseline de logging, pipeline de CI.

**No incluye:** features de producto (init, agentes, prompts, workflows, compilación, TUI completa) — viven en los epics siguientes. La estrategia fina de testing y logging de producto se profundiza en `epic-09`.

## Tickets

| Ticket | Tipo | Foco | Origen |
|---|---|---|---|
| `ticket-00-01-go-module-and-layout` | backend | Inicializar el módulo Go y el layout del repo (`cmd/daedalus/`, `internal/`), README base. | init.md §4.1 |
| `ticket-00-02-charm-bootstrap` | backend | Incorporar dependencias Charm y un bootstrap mínimo de Bubble Tea que arranca y cierra limpio. | PRD D2 |
| `ticket-00-03-local-dev-scaffolding` | backend | Docker, Docker Compose, Makefile y scripts para onboarding seamless. | HANDOFF §5 (Obi-Wan) |
| `ticket-00-04-logging-baseline` | backend | Baseline de logging estructurado en puntos de decisión, sin datos sensibles. | RF-9.1 (baseline) |
| `ticket-00-05-ci-pipeline` | backend | Pipeline de CI: build, test y lint en cada cambio. | RNF-2/RNF-5 |

## Criterios de aceptación del epic

- El repositorio compila (`go build`) y corre un binario mínimo de Daedalus.
- Un desarrollador nuevo levanta el entorno local con un solo comando documentado.
- Existe logging estructurado base utilizable por el resto de los epics.
- CI valida build + test + lint de forma reproducible.
- Trazabilidad: cada ticket referencia su origen en PRD/init/HANDOFF.
