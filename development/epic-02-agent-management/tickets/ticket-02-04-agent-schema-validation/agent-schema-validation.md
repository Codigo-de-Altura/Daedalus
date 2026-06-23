# Ticket 02-04 — Validación de agente contra el esquema canónico

> **Epic:** epic-02-agent-management · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-2.4 · **Estilo:** SDD

## Contexto

Toda definición de agente —venga del catálogo built-in (ticket-02-01), de un clon editado (ticket-02-02) o de un import local (ticket-02-03)— debe ser **válida y consistente** antes de poder usarse o compilarse al backend (epic-06). Para garantizarlo, Daedalus necesita un **esquema canónico de agente** y una validación que lo aplique, devolviendo **errores accionables** (RF-2.4, EPIC-9 RF-9.3). Esta validación es la pieza transversal que los demás tickets de este epic consumen como *gate* de calidad.

## Feature / Qué se construye

El **esquema canónico de agente** (los campos que conforman una definición válida: identificador, rol, prompt, parámetros y sus reglas) y un **validador** que verifica una definición contra ese esquema. Cuando la definición es inválida, el validador devuelve **errores accionables**: qué campo falla, qué se observó y qué se esperaba, de modo que el usuario pueda corregir sin adivinar. La validación es la base reutilizada por los tickets 02-01, 02-02 y 02-03.

## Requerimientos

- R1 — Existe un **esquema canónico de agente** que define los campos obligatorios y opcionales de una definición de agente (al menos: identificador, rol, prompt, parámetros) y sus reglas básicas (tipos, no-vacío, `kebab-case` en el identificador).
- R2 — Existe un **validador** que, dada una definición de agente, determina si es **válida** o **inválida** contra el esquema canónico.
- R3 — Ante una definición inválida, el validador produce **errores accionables**: identifican el campo afectado, lo **observado** y lo **esperado** (RF-9.3).
- R4 — El validador reporta **todos** los problemas detectables en una pasada (no solo el primero), para minimizar ciclos de corrección.
- R5 — La validación es **determinista** (RNF-5): la misma definición produce siempre el mismo veredicto y el mismo conjunto de errores, en orden estable.
- R6 — El validador es **reutilizable** por los demás tickets del epic (catálogo, clon/edición, import) como gate de calidad.
- R7 — La validación **no ejecuta** agentes ni invoca el backend; opera sobre la definición canónica (Fase 1, D5).

## Criterios de aceptación

- [ ] CA1 — El esquema canónico de agente declara los campos obligatorios (al menos identificador, rol, prompt) y sus reglas.
- [ ] CA2 — Una definición válida pasa la validación sin errores.
- [ ] CA3 — Una definición a la que le falta un campo obligatorio falla y el error indica campo, observado y esperado.
- [ ] CA4 — Una definición con un identificador que no cumple `kebab-case` falla con un error accionable.
- [ ] CA5 — Una definición con múltiples problemas reporta todos los hallazgos en una sola pasada.
- [ ] CA6 — La misma definición produce el mismo veredicto y los mismos errores en ejecuciones repetidas (determinismo).
- [ ] CA7 — Trazabilidad a RF-2.4.

## Fuera de alcance

- Catálogo built-in y materialización (ticket-02-01); lo consume.
- Clonado y edición (ticket-02-02); lo consume.
- Import desde archivos locales (ticket-02-03); lo consume.
- Validación del DAG de workflows (epic-04) y del manifiesto (epic-08/09); este ticket cubre solo el esquema de agente.
- Compilación al formato nativo del backend (epic-06).

## Referencias

- PRD §9 — RF-2.4 (validación de la definición de agente), RF-9.3 (validaciones/linters)
- PRD §15 (decisión abierta: detalle fino del esquema canónico de agente)
- init.md §5 (agente como YAML + prompt MD), §7 (convenciones)
- CLAUDE.md §6 (estructura de tickets)
- epic-02-agent-management/epic.md (ticket-02-04)
