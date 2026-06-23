# Atajos de teclado y ayuda contextual

> **Epic:** epic-07-tui-ux · **Tipo:** frontend · **Implementador:** Padmé · **Validador:** Leia · **Origen:** RF-7.3 · **Estilo:** SDD

## Contexto

Una TUI eficiente se opera con el teclado. Para que Daedalus sea predecible y aprendible, los **atajos de teclado** deben ser **consistentes** en toda la aplicación (la misma tecla hace lo mismo en todas partes) y debe existir **ayuda contextual** accesible: en cada área el usuario puede ver qué atajos están disponibles aquí y ahora.

Este ticket entrega el **sistema de atajos compartido** y el mecanismo de **ayuda contextual** que las áreas (ticket-07-01) y los componentes (ticket-07-02) consumen.

## Feature / Qué se construye

1. **Sistema de keybindings consistente:** un registro central de atajos con acciones de navegación y comunes (mover selección, entrar, volver, raíz, salir, ayuda, enviar/cancelar en formularios). Las áreas declaran qué atajos exponen, pero las teclas para acciones equivalentes son idénticas en toda la TUI.
2. **Ayuda contextual:** un mecanismo (p. ej. barra de ayuda y/o vista de ayuda expandida) que muestra los atajos **disponibles en el contexto actual**. Accesible desde cualquier área con un atajo dedicado y consistente.

## Requerimientos

1. Existe un registro/definición **central** de atajos; las acciones comunes (mover, entrar, volver, raíz, salir, ayuda) tienen teclas únicas y consistentes en toda la TUI.
2. Una misma acción usa la **misma tecla** en todas las áreas y componentes; no hay colisiones ni significados divergentes.
3. Existe **ayuda contextual** que lista los atajos disponibles en el contexto actual (área/pantalla/formulario).
4. La ayuda es **accesible desde cualquier área** mediante un atajo dedicado y consistente.
5. La ayuda contextual refleja el contexto: muestra solo (o destaca) lo que aplica en la pantalla actual.
6. Existe una vista breve siempre visible (p. ej. barra de ayuda) y/o una vista de ayuda expandible.
7. La definición de atajos es la **fuente de verdad** que también alimenta lo que la ayuda muestra (consistencia entre lo que se anuncia y lo que funciona).

## Criterios de aceptación

- [ ] Existe un registro central de atajos con acciones comunes y de navegación.
- [ ] La misma acción usa la misma tecla en todas las áreas/componentes (consistencia verificable).
- [ ] Existe ayuda contextual que lista los atajos del contexto actual.
- [ ] La ayuda es accesible desde cualquier área con un atajo consistente.
- [ ] Lo que la ayuda anuncia coincide con lo que efectivamente funciona.
- [ ] Trazabilidad a RF-7.3.

## Fuera de alcance

- La estructura de navegación entre áreas (ticket-07-01); aquí se provee el sistema de atajos que aquella consume.
- El tema visual, render markdown y formularios (ticket-07-02); aquí solo se cubre la **indicación** de atajos de envío/cancelación.
- Configurabilidad/remapeo de atajos por el usuario (no requerido en Fase 1).
- Performance/fluidez (ticket-07-04).

## Referencias

- `PRD.md` — RF-7.3, RNF-4, riesgo "Fricción de UX en terminal → ayuda contextual" (§13).
- `epic.md` — epic-07-tui-ux.
- `CLAUDE.md` — §2 (frontend: UX, atajos), §6.
