# Prompt Preview — Previsualización del prompt renderizado en la TUI

> **Epic:** epic-03-prompt-management · **Tipo:** frontend · **Implementador:** Padmé · **Validador:** Leia · **Origen:** RF-3.3 · **Estilo:** SDD

## Contexto

Un prompt puede componerse a partir de fragmentos reutilizables (ticket-03-02). Antes de que ese prompt se compile al backend de agentes (epic-06), el usuario necesita **ver el resultado final** ya ensamblado, tal como quedaría. Esto le da control y confianza sobre el contenido.

Este ticket cubre la **previsualización en la TUI** (Charm) del prompt final renderizado: tomar el texto compuesto que produce el core y mostrarlo de forma legible y estética en la terminal, usando Glamour para el render de Markdown.

No cubre la lógica de composición (ticket-03-02, backend) ni la edición/persistencia de prompts (ticket-03-01, backend), ni la compilación al backend (epic-06).

## Feature / Qué se construye

Una **vista de previsualización** en la TUI que, dado un prompt seleccionado, muestra su **texto final compuesto y renderizado** (Markdown vía Glamour), dentro de la arquitectura Bubble Tea (modelo/update/view) y con estilos Lipgloss consistentes con el resto de la TUI.

La preview consume el texto compuesto provisto por el core (ticket-03-02) y se encarga solo de la presentación: render de Markdown, scroll y atajos de navegación.

## Requerimientos

- **R1.** La TUI ofrece una **vista de preview** accesible para un prompt seleccionado.
- **R2.** La preview muestra el **prompt final compuesto** (con inclusiones ya resueltas por el core), no el cuerpo crudo con las directivas de inclusión.
- **R3.** El contenido Markdown se **renderiza con Glamour** (encabezados, listas, énfasis, bloques de código) de forma legible en la terminal.
- **R4.** La vista permite **scroll** cuando el contenido excede la altura visible (viewport).
- **R5.** Hay **atajos de teclado** consistentes para abrir/cerrar la preview y desplazarse; la ayuda contextual los indica.
- **R6.** La estética es consistente con el resto de la TUI (Lipgloss: tema, colores, márgenes).
- **R7.** Si el core reporta un **error de composición** (ciclo, referencia inexistente), la preview muestra un mensaje de error claro en lugar de contenido roto.
- **R8.** La preview es de **solo lectura**: no edita ni persiste el prompt.

## Criterios de aceptación

- [ ] Desde la TUI se puede abrir la **preview** de un prompt seleccionado.
- [ ] La preview muestra el texto **compuesto** (inclusiones resueltas), renderizado como Markdown con Glamour.
- [ ] El contenido **largo** se puede recorrer con scroll dentro de la vista.
- [ ] Los **atajos** para abrir/cerrar/desplazar funcionan y figuran en la ayuda contextual.
- [ ] La preview es **solo lectura** (no altera el prompt).
- [ ] Un prompt con **error de composición** muestra un mensaje de error legible, no un crash ni contenido corrupto.
- [ ] El estilo visual es **consistente** con el resto de la TUI.

## Fuera de alcance

- Lógica de composición/resolución de inclusiones (ticket-03-02, backend).
- Edición/persistencia de prompts (ticket-03-01, backend).
- Compilación del prompt al formato nativo del backend (epic-06).

## Referencias

- `PRD.md` — RF-3.3, RF-7.2 (Glamour/Lipgloss), sección 9 EPIC-3, sección 11 (stack Charm).
- `epic.md` — epic-03-prompt-management (la preview muestra el prompt final tal como se compilaría).
- `ticket-03-02-prompt-composition` — fuente del texto compuesto.
- `CLAUDE.md` — §2 (frontend = capa TUI Charm), §3 (Padmé / Leia).
