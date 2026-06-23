# Reporte de diff / preview antes de escribir

> **Epic:** epic-06-compile-claude-adapter · **Tipo:** frontend · **Implementador:** Padmé · **Validador:** Leia · **Origen:** RF-6.4 · **Estilo:** SDD

---

## Contexto

La compilación escribe artefactos al backend (RF-6.1, RF-6.2) de forma idempotente y no destructiva (RF-6.3). Antes de **escribir**, el usuario debe poder **ver qué cambiaría** y **confirmar**. Este ticket define el **reporte de diff/preview** en la TUI: muestra los cambios que produciría el build (altas, modificaciones, sin-cambios) y pide confirmación explícita del usuario antes de tocar el disco.

Es un ticket de **frontend/TUI** (Charm: Bubble Tea, Lipgloss, Glamour, Huh). Consume el resultado de compilación del núcleo (los artefactos que *se generarían*) y lo presenta de forma legible; no implementa el mapeo ni la escritura en sí.

Referencias: PRD RF-6.4, RNF-8 (operaciones no destructivas por defecto; preview/confirm), §13 (mitigación: preview/diff antes de escribir), RF-7.x (TUI/UX).

## Feature / Qué se construye

Un **modo preview** del build que:

- Calcula el diff entre el estado actual del área gestionada en disco y lo que el build **generaría**.
- **Presenta el diff** en la TUI de forma legible: por archivo, clasificando cada cambio como **nuevo / modificado / sin cambios** (y, si aplica, eliminado dentro del área gestionada).
- Para archivos modificados, muestra el detalle del cambio (diff de contenido) renderizado de forma clara.
- Pide **confirmación explícita** del usuario antes de escribir. Si el usuario **no confirma**, **no se escribe nada**.
- Ofrece un modo de solo-preview (mostrar el diff y salir sin escribir).

> El cálculo del diff se apoya en el determinismo del adaptador (RF-6.2) y en la noción de "área gestionada" (RF-6.3). Este ticket aporta la **presentación y el gate de confirmación**.

## Requerimientos

- **REQ-1.** Antes de escribir, el flujo de build muestra un **reporte de diff/preview** de los cambios que produciría.
- **REQ-2.** El reporte clasifica cada artefacto: **nuevo**, **modificado**, **sin cambios** (y eliminado dentro del área gestionada si corresponde).
- **REQ-3.** Para artefactos **modificados**, el reporte muestra el **detalle del cambio** (diff de contenido) de forma legible (Glamour/Lipgloss).
- **REQ-4.** El reporte pide **confirmación explícita**; sin confirmación, **no se escribe nada** (RNF-8).
- **REQ-5.** Existe un modo **solo-preview** (mostrar diff y salir sin escribir).
- **REQ-6.** La presentación es **estética y legible** en terminal (RNF-4, RF-7.2): resúmenes claros, conteos, colores consistentes.
- **REQ-7.** El preview refleja la realidad: si no hay cambios, lo comunica claramente ("sin cambios"); si hay, los lista con exactitud.
- **REQ-8.** Atajos/controles consistentes para confirmar, cancelar y navegar el diff (RF-7.3).

## Criterios de aceptación

- [ ] Al compilar, el usuario ve un **reporte de diff/preview** antes de que se escriba algo.
- [ ] Cada artefacto aparece clasificado como nuevo / modificado / sin cambios.
- [ ] Para artefactos modificados, se muestra el detalle del cambio de forma legible.
- [ ] El usuario debe **confirmar explícitamente**; si cancela, no se escribe nada.
- [ ] Existe un modo solo-preview que muestra el diff y no escribe.
- [ ] Cuando no hay cambios, el reporte lo indica claramente.
- [ ] La presentación es legible y estéticamente consistente, con controles claros.

## Fuera de alcance

- El **cálculo del output** y el mapeo canónico → Claude Code → RF-6.2 (ticket-06-02).
- La **escritura** efectiva y su idempotencia/no-destrucción → RF-6.3 (ticket-06-03).
- La orquestación del **comando** `build`/`sync` → RF-6.1 (ticket-06-01).
- Ejecución de agentes → Fase 2+.

## Referencias

- `PRD.md` — RF-6.4, RNF-8 (preview/confirm), RNF-4 (estética), §13 (preview/diff como mitigación), RF-7.2/RF-7.3 (TUI/UX/atajos).
- `init.md` — §7 (idempotencia: siempre ofrece preview/diff).
- `CLAUDE.md` — §3 (Padmé = frontend/TUI; Leia = validadora frontend), §7 (idempotencia y no destrucción: preview/diff).
- `epic.md` — Epic 06, criterio "el usuario ve un diff/preview y confirma antes de que se escriba".
