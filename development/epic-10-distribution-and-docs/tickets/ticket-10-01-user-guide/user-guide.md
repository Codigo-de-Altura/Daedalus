# Ticket 10-01 — Guía de usuario consumer-facing

> **Epic:** epic-10-distribution-and-docs · **Tipo:** docs · **Implementador:** C-3PO · **Validador:** usuario (validación manual)
> **Origen:** release-readiness (epic-10) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Durante Fase 1, el manual `docs/` creció **capítulo por feature** (init, agents, prompts, workflows, specs, architecture, epics/tickets, trace, compile, validate, TUI, configuration; más `contributing/`). El contenido es correcto pero está organizado para quien sigue el backlog, no para un **consumidor nuevo** que descarga la herramienta y quiere entenderla rápido. Falta una **narrativa de adopción** cohesiva y un índice que guíe del "no la conozco" al "la estoy usando productivamente".

Este ticket **reestructura y conecta** lo existente en una guía de usuario fácil de seguir, orientada al consumidor (no al desarrollador del proyecto). No reescribe desde cero: reutiliza y pule los capítulos ya escritos, cierra huecos y los ordena en un recorrido lineal con referencia.

## Feature / Qué se construye

Una **guía de usuario** consumer-facing en `docs/`, organizada como manual fácil de seguir, que cubra el ciclo completo de uso de Daedalus:

- **Instalación** — cómo obtener el binario (descarga desde GitHub Releases una vez exista — ver 10-03; y build desde fuente como alternativa), por plataforma; verificación con `daedalus --version`.
- **Primeros pasos / Quickstart** — del cero a un workspace compilado: `daedalus init`, recorrido mínimo, `daedalus build`. Un ejemplo reproducible de punta a punta.
- **Conceptos** — qué es el workspace `.daedalus/`, el modelo canónico (agentes, prompts, workflows DAG, manifiesto), y la compilación agnóstica de backend a `.claude/`. Lo justo para operar con criterio, sin teoría de más.
- **Flujo central** — el loop de trabajo: editar definiciones → `validate` → `build` (con su gate interactivo/preview/`--yes`). Cuándo se usa cada comando.
- **Referencia de comandos** — un apartado por comando (`init`, `build`/`sync`, `validate`, `trace verify`/`show`, la TUI, `--version`) con su propósito, **flags y parámetros**, exit codes y ejemplos de invocación. Debe reflejar la superficie real actual de la CLI.
- **La TUI** — cómo lanzarla y navegarla (requiere terminal interactiva).
- **Configuración** — manifiesto `daedalus.yaml`, variables de entorno (`DAEDALUS_LOG_LEVEL`), logging.
- **Ejemplos** — escenarios realistas (iniciar un proyecto, agregar un agente/prompt/workflow, validar y compilar, leer un reporte de validación).
- **Troubleshooting** — errores comunes y cómo interpretarlos (definición inválida, validación que falla, build sin TTY), apoyándose en los eventos de logging y los reportes accionables.

La guía debe ser **navegable** (índice claro, capítulos enlazados, progresión lógica) y **autosuficiente**: un usuario sin contexto del repo debe poder seguirla. Los capítulos de `contributing/` (trabajar **sobre** Daedalus) se mantienen separados de la guía de uso (operar **con** Daedalus), pero el índice los referencia.

## Requerimientos

- R1. Existe una **guía de usuario** consumer-facing en `docs/`, con un índice (`docs/README.md`) que ofrece un recorrido lineal de adopción (instalar → primeros pasos → conceptos → flujo → referencia → ejemplos → troubleshooting) además de la referencia por comando.
- R2. La guía cubre **instalación** (descarga de binario por plataforma + build desde fuente), **quickstart** end-to-end, **conceptos** mínimos, **flujo central**, **referencia de cada comando** con flags/parámetros/exit codes, **TUI**, **configuración**, **ejemplos** y **troubleshooting**.
- R3. La **referencia de comandos refleja la superficie real** de la CLI actual: comandos `init`, `build` (alias `sync`), `validate`, `trace verify`/`show`, la TUI y `--version`; flags `init {-backend,-path,-preview}`, `build {-path,-preview,-yes}`, `validate {-path}`; exit codes documentados (p. ej. validate 0/1/2). No documenta flags/comandos inexistentes.
- R4. La guía **reutiliza y consolida** el material de `docs/` existente; no duplica contenido ni deja capítulos huérfanos. Capítulos que se reescriban quedan conectados desde el índice.
- R5. Está orientada al **consumidor** (instalar/usar), no al desarrollador del proyecto; el contenido de `contributing/` permanece distinto y claramente separado.
- R6. Los **ejemplos son reproducibles** y coinciden con el comportamiento real (comandos, salidas, exit codes); no hay pasos inventados.
- R7. Toda la documentación está en **inglés** (CLAUDE.md §1) y es markdown legible, apta para renderizar como sitio (ver 10-02).
- R8. Se incluye un `manual-validation.md` que recorre la herramienta completa **siguiendo la guía paso a paso**, como gate previo al release.

## Criterios de aceptación

- [ ] CA1. Desde el índice, un usuario nuevo puede seguir un recorrido lineal de adopción que lo lleva de instalar a compilar un workspace.
- [ ] CA2. Cada comando real (`init`, `build`/`sync`, `validate`, `trace verify`/`show`, TUI, `--version`) tiene su apartado de referencia con propósito, flags/parámetros, exit codes y al menos un ejemplo.
- [ ] CA3. La referencia coincide con la superficie real de la CLI (verificable contra `daedalus <cmd> --help`); no hay flags/comandos fantasma.
- [ ] CA4. El quickstart end-to-end es reproducible: seguido al pie de la letra produce un workspace compilado.
- [ ] CA5. No quedan capítulos huérfanos ni duplicados; el índice enlaza todo el material y separa uso (guide) de contribución (contributing).
- [ ] CA6. Existe `manual-validation.md` que ejercita todos los comandos/flujos a través de la guía.
- [ ] CA7. Todo en inglés, markdown legible.

## Fuera de alcance

- La publicación del sitio (MkDocs/Pages) — ticket 10-02.
- El pipeline de release y la URL real de descarga — ticket 10-03 (la sección de instalación referencia "GitHub Releases" y se completa con la URL/tag cuando 10-03 exista).
- Cambios al comportamiento de la herramienta (salvo discrepancias que la validación manual obligue a corregir, que se enrutan a Obi-Wan/Padmé como fixes).

## Referencias

- CLAUDE.md — §1 (doc en inglés), §6 (`docs/` como manual; índice + capítulos).
- Manual existente: `docs/README.md`, `docs/getting-started/*`, `docs/guide/*`, `docs/contributing/*`.
- Superficie CLI real (verificable): `daedalus --help`, `daedalus init|build|validate|trace --help`.
- epic-10-distribution-and-docs/epic.md — objetivo, alcance y orden (guía → validación manual → sitio/release).
