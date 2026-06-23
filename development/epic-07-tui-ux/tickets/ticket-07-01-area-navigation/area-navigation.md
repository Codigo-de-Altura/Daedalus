# Navegación por áreas

> **Epic:** epic-07-tui-ux · **Tipo:** frontend · **Implementador:** Padmé · **Validador:** Leia · **Origen:** RF-7.1 · **Estilo:** SDD

## Contexto

Daedalus es una TUI/CLI en Go + Charm (Bubble Tea) que gestiona la estructura de IA de un proyecto. La TUI organiza el trabajo del usuario en **áreas** funcionales que mapean a los dominios del producto: inicialización, agentes, prompts, workflows, backlog y compilación (build). El usuario debe poder moverse entre estas áreas de forma predecible, sin quedar atrapado en pantallas sin salida (dead ends).

Este ticket entrega el **shell de navegación**: la estructura que enmarca y conecta las áreas. La lógica de dominio detrás de cada área vive en los epics de backend; aquí solo se construye la cáscara de navegación y su consumo de las interfaces del core.

## Feature / Qué se construye

Un **shell de navegación por áreas** dentro de la TUI que permite al usuario alcanzar y abandonar cada una de las seis áreas:

- **init** — inicialización/gestión del workspace `.daedalus/`.
- **agentes** — catálogo y gestión de agentes.
- **prompts** — prompts globales y compartidos.
- **workflows** — DAGs declarativos.
- **backlog** — spec/PRD, arquitectura, epics, tickets.
- **build** — compilación al backend.

La navegación es **consistente** (mismos atajos para entrar/salir/volver en todas las áreas) y **sin dead ends**: desde cualquier pantalla siempre existe una ruta de retorno hacia el menú/área padre y, eventualmente, a la raíz.

## Requerimientos

1. La TUI presenta un punto de entrada (menú/área raíz) que lista las seis áreas: init, agentes, prompts, workflows, backlog, build.
2. El usuario puede **entrar** a cualquier área desde la raíz mediante teclado.
3. Desde cualquier área (y desde cualquier sub-pantalla de un área) existe siempre una acción de **volver** a la pantalla anterior y/o a la raíz; no hay pantallas sin salida.
4. La navegación entre áreas es **consistente**: los atajos para entrar, volver y salir se comportan igual en todas las áreas.
5. El estado de navegación (área actual, ruta de retorno) es visible para el usuario (p. ej. indicador de área/breadcrumb o resaltado del área activa).
6. Las áreas consumen la lógica de dominio **vía las interfaces del core**; este ticket no implementa dicha lógica.
7. Estados transversales mínimos por área: **loading**, **empty** y **error** deben tener una representación que no rompa la navegación (siempre se puede volver).

## Criterios de aceptación

- [ ] Existe una pantalla raíz que lista las seis áreas (init, agentes, prompts, workflows, backlog, build).
- [ ] Se puede entrar a cada una de las seis áreas desde la raíz.
- [ ] Desde cada área se puede volver a la pantalla anterior y a la raíz con atajos consistentes.
- [ ] No existe ninguna pantalla sin ruta de salida (no hay dead ends).
- [ ] El área activa es identificable visualmente (indicador/breadcrumb/resaltado).
- [ ] Los estados loading/empty/error no impiden volver ni navegar.
- [ ] Trazabilidad a RF-7.1.

## Fuera de alcance

- La lógica de dominio de cada área (init, build, etc.): vive en los epics de backend y se consume vía interfaces del core.
- Estética detallada, render de markdown y formularios (ticket-07-02).
- Sistema de atajos global y ayuda contextual extendida (ticket-07-03); aquí solo se garantiza la **consistencia** de los atajos de navegación.
- Optimización de performance/fluidez (ticket-07-04).

## Referencias

- `PRD.md` — RF-7.1, RNF-2, RNF-4.
- `epic.md` — epic-07-tui-ux.
- `CLAUDE.md` — §6 (estructura `development/`), §2 (arquitectura frontend/backend).
