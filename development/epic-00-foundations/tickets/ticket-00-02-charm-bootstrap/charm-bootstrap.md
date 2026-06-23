# Ticket 00-02 — Dependencias Charm y bootstrap de Bubble Tea

> **Epic:** epic-00-foundations · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** PRD D2 (stack TUI: Go + Charm) · PRD §11 (Bubble Tea = arquitectura Elm) · epic.md (ticket-00-02) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

El stack de Daedalus es Go + Charm (decisión D2): Bubble Tea para la arquitectura tipo Elm, y Lipgloss, Bubbles, Glamour y Huh como complementos de presentación. Sobre el layout creado en ticket-00-01, este ticket incorpora las dependencias del ecosistema Charm y un programa Bubble Tea mínimo que arranca y cierra limpio. El objetivo es probar que el toolkit está correctamente integrado y que la TUI completa (epic-07) podrá construirse encima, sin todavía implementar pantallas de producto.

## Feature / Qué se construye

La integración del ecosistema Charm en el módulo Go y un bootstrap mínimo de Bubble Tea: un programa que inicializa el modelo, renderiza una vista mínima y responde a una tecla de salida (p. ej. `q`/`Ctrl+C`) cerrando limpiamente sin dejar la terminal en un estado roto. Las dependencias Charm relevantes quedan declaradas y disponibles para los epics siguientes.

## Requerimientos

- R1 — Las dependencias del ecosistema Charm requeridas por el stack quedan declaradas en `go.mod`: como mínimo Bubble Tea (`bubbletea`) y Lipgloss; Bubbles, Glamour y Huh disponibles para uso posterior (declaradas o documentadas como dependencias del proyecto).
- R2 — Existe un modelo Bubble Tea mínimo (con `Init`, `Update`, `View`) en el código del binario, idiomático respecto a la arquitectura Elm.
- R3 — El programa arranca, renderiza una vista mínima identificable de Daedalus y responde a una tecla/secuencia de salida cerrando con `tea.Quit`.
- R4 — El cierre es limpio: restaura el estado de la terminal y retorna código de salida 0; no entra en pánico.
- R5 — El programa soporta ejecución no interactiva para validación (p. ej. detección de ausencia de TTY o un modo/flag que permita arrancar y cerrar sin entrada humana) de modo que un agente pueda verificar el arranque/cierre.
- R6 — Las dependencias quedan fijadas y verificables (`go.sum` consistente; `go mod verify` pasa).
- R7 — El bootstrap no implementa navegación ni pantallas de producto; es el esqueleto mínimo sobre el que epic-07 construirá la TUI.

## Criterios de aceptación

- [ ] CA1 — `go.mod` declara Bubble Tea y Lipgloss (y el resto del set Charm previsto) como dependencias.
- [ ] CA2 — `go build ./...` compila con las dependencias Charm integradas.
- [ ] CA3 — Existe un modelo Bubble Tea con los métodos `Init`, `Update` y `View`.
- [ ] CA4 — El programa arranca y cierra limpio en modo no interactivo, retornando código de salida 0 sin pánico.
- [ ] CA5 — `go mod verify` y `go mod tidy` (sin cambios pendientes) confirman consistencia de dependencias.
- [ ] CA6 — `go vet ./...` no reporta problemas.

## Fuera de alcance

- Pantallas, navegación, atajos y estética de la TUI de producto (epic-07).
- Render de markdown con Glamour y formularios con Huh aplicados a features (epics posteriores).
- Comandos de producto (init, build) y cualquier lógica de dominio.
- Logging estructurado (ticket-00-04).

## Referencias

- PRD D2 — Stack TUI: Go + Charm (Bubble Tea, Lipgloss, Bubbles, Glamour, Huh)
- PRD §11 — Arquitectura técnica (Bubble Tea = arquitectura Elm)
- CLAUDE.md §2 (frontend = capa TUI con Charm)
- epic-00-foundations/epic.md (ticket-00-02)
