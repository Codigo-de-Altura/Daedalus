# Operación fluida y de bajo consumo

> **Epic:** epic-07-tui-ux · **Tipo:** frontend · **Implementador:** Padmé · **Validador:** Leia · **Origen:** RF-7.4 · **Estilo:** SDD

## Contexto

Daedalus busca ser **lightweight** y de **interacción fluida, sin bloqueos perceptibles** en operaciones comunes (RNF-2; arranque objetivo < 200 ms, RNF-1). La capa TUI (Bubble Tea, arquitectura Elm) debe mantener el bucle de actualización/render responsivo: el input del usuario nunca debe "congelarse" mientras ocurre trabajo de carga, render o consumo del core.

Este ticket cubre la **fluidez percibida** de la TUI: que la navegación, el render (incluido markdown) y los formularios respondan sin trabar, y que el consumo de recursos se mantenga bajo en operaciones comunes.

## Feature / Qué se construye

Garantías de **fluidez** en la capa de presentación:

- El input del usuario se procesa sin bloqueos perceptibles incluso mientras hay operaciones en curso (carga de datos del core, render de documentos grandes).
- El trabajo potencialmente lento no congela el bucle de la TUI (se ejecuta de forma asíncrona respecto al render, con estados de loading visibles).
- El consumo de recursos (CPU en reposo, memoria) se mantiene bajo durante operaciones comunes; sin spinners/redraws que quemen CPU innecesariamente.

## Requerimientos

1. La interacción (navegar, abrir áreas, abrir/scrollear documentos, usar formularios) es **fluida**, sin bloqueos perceptibles para el usuario en operaciones comunes.
2. Las operaciones potencialmente lentas (carga desde el core, render de markdown extenso) **no bloquean** el bucle de la TUI; el usuario puede seguir interactuando o, como mínimo, ve un estado de loading y conserva la capacidad de cancelar/volver.
3. En reposo (sin input), la TUI **no consume CPU de forma apreciable** (sin redibujado/polling innecesario).
4. El arranque de la TUI es rápido, consistente con el objetivo de producto (RNF-1).
5. El uso de memoria se mantiene acotado en operaciones comunes (sin crecimiento descontrolado al navegar repetidamente entre áreas).
6. No hay degradación perceptible al repetir navegación o re-render (sin fugas que enlentezcan con el uso).

## Criterios de aceptación

- [ ] Navegar, abrir áreas y usar formularios se siente fluido, sin congelamientos perceptibles.
- [ ] Las operaciones lentas no bloquean el input; se muestra loading y se puede volver/cancelar.
- [ ] En reposo, la TUI no consume CPU de forma apreciable.
- [ ] El arranque es rápido (consistente con RNF-1).
- [ ] El uso de memoria permanece acotado tras navegación repetida.
- [ ] No hay degradación de respuesta con el uso prolongado.
- [ ] Trazabilidad a RF-7.4 (y RNF-2).

## Fuera de alcance

- Navegación, tema/markdown/formularios y atajos como features funcionales (tickets 07-01, 07-02, 07-03); aquí solo se evalúa su **fluidez** y consumo.
- Optimización de la lógica de dominio/compilación del core (epics de backend).
- Benchmarks de rendimiento del backend / golden files (epic de ecosistema).

## Referencias

- `PRD.md` — RF-7.4, RNF-1 (lightweight, arranque < 200 ms), RNF-2 (performance/UX), §13 (fricción de UX).
- `epic.md` — epic-07-tui-ux (criterio: interacción fluida sin bloqueos perceptibles).
- `CLAUDE.md` — §2 (frontend, Bubble Tea = Elm).
