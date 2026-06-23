# Estética, render de markdown y formularios

> **Epic:** epic-07-tui-ux · **Tipo:** frontend · **Implementador:** Padmé · **Validador:** Leia · **Origen:** RF-7.2 · **Estilo:** SDD

## Contexto

Daedalus apunta a una TUI **estética, eficiente y con buena UX** (PRD §4.1, RNF-4). Para lograrlo, la capa de presentación (Charm) necesita: un **tema visual coherente** (Lipgloss), capacidad de **renderizar markdown** en terminal (Glamour) —dado que gran parte del contenido del producto son documentos markdown (specs, prompts, docs)— y **formularios** que capturen y **validen** entrada del usuario (Huh).

Este ticket define los componentes visuales y de entrada compartidos que las áreas (ticket-07-01) consumen.

## Feature / Qué se construye

Tres pilares de presentación compartidos:

1. **Tema visual (Lipgloss):** una paleta y estilos consistentes (colores, bordes, énfasis, estados) aplicados de forma uniforme a través de la TUI, de modo que toda la salida luzca pulida y coherente.
2. **Render de markdown (Glamour):** un componente que toma markdown (specs, prompts, documentación) y lo muestra renderizado y legible en terminal, consistente con el tema.
3. **Formularios (Huh):** componentes de formulario reutilizables para capturar entrada (campos de texto, selección, confirmación) con **validación** de entrada y mensajes de error claros.

## Requerimientos

1. Existe un **tema** central (paleta + estilos) reutilizable; los componentes y áreas toman sus estilos de ahí (no colores hardcodeados dispersos).
2. El estilo es **coherente** entre áreas: encabezados, listas, selección activa, estados (loading/empty/error) comparten lenguaje visual.
3. Existe un **componente de render de markdown** (Glamour) que muestra documentos markdown legibles y acordes al tema.
4. El render de markdown maneja contenido típico del producto: encabezados, listas, tablas, bloques de código, énfasis.
5. Existen **formularios** (Huh) reutilizables para capturar entrada del usuario.
6. Los formularios **validan** la entrada y muestran mensajes de error comprensibles cuando la entrada es inválida.
7. Los formularios soportan envío y cancelación, integrándose con la navegación (no rompen el retorno; sin dead ends).
8. La estética es consistente en los tres estados transversales: **loading**, **empty** y **error**.

## Criterios de aceptación

- [ ] Existe un tema central reutilizable y aplicado de forma consistente en la TUI.
- [ ] El markdown se renderiza correctamente (encabezados, listas, tablas, código, énfasis) y acorde al tema.
- [ ] Existen formularios reutilizables que capturan entrada.
- [ ] Los formularios validan la entrada y muestran errores claros ante input inválido.
- [ ] Los formularios permiten enviar y cancelar sin romper la navegación.
- [ ] Los estados loading/empty/error comparten el lenguaje visual del tema.
- [ ] Trazabilidad a RF-7.2 (y consistencia con RNF-4).

## Fuera de alcance

- La estructura de navegación entre áreas (ticket-07-01).
- El sistema global de atajos y la ayuda contextual (ticket-07-03).
- Performance/fluidez del render (ticket-07-04).
- La lógica de dominio que produce el markdown o consume los datos de formularios (epics de backend).

## Referencias

- `PRD.md` — RF-7.2, RNF-4, D2 (stack Charm: Lipgloss, Glamour, Huh).
- `epic.md` — epic-07-tui-ux.
- `CLAUDE.md` — §2 (frontend = Lipgloss/Glamour/Huh), §6.
