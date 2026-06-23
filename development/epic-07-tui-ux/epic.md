# Epic 07 — TUI / User Experience

> **Origen:** EPIC-7 del PRD (RF-7.1 … RF-7.4). **Estilo:** SDD (plano, no guía de implementación). Terreno de **Padmé**.

## Objetivo

Entregar la experiencia de usuario de la TUI: navegación por áreas (init, agentes, prompts, workflows, backlog, build), estética cuidada (Lipgloss), render de markdown en terminal (Glamour), formularios (Huh), atajos de teclado consistentes con ayuda contextual, y operación fluida de bajo consumo.

## Alcance

**Incluye:** shell de navegación entre áreas, tema visual y componentes compartidos, render de markdown, formularios, sistema de atajos + ayuda contextual, fluidez/performance percibida.

**No incluye:** la lógica de dominio/compilación detrás de cada área (vive en los epics de backend; la TUI la consume vía interfaces del core).

## Tickets

| Ticket | Tipo | Foco | Origen |
|---|---|---|---|
| `ticket-07-01-area-navigation` | frontend | Navegación por áreas (init, agentes, prompts, workflows, backlog, build). | RF-7.1 |
| `ticket-07-02-theme-markdown-forms` | frontend | Estética (Lipgloss), render markdown (Glamour), formularios (Huh). | RF-7.2 |
| `ticket-07-03-keybindings-and-help` | frontend | Atajos de teclado consistentes y ayuda contextual. | RF-7.3 |
| `ticket-07-04-performance-fluidity` | frontend | Operación fluida y de bajo consumo. | RF-7.4 |

## Criterios de aceptación del epic

- Se navega entre todas las áreas con atajos consistentes y sin dead ends.
- El tema visual es coherente; el markdown se renderiza correctamente; los formularios validan entrada.
- Existe ayuda contextual accesible en cada área.
- La interacción es fluida, sin bloqueos perceptibles (RNF-2).
- Trazabilidad a RF-7.x.
