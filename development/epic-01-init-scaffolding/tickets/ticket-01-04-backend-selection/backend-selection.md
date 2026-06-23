# Ticket 01-04 — Selección de backend(s) objetivo (MVP: Claude Code)

> **Epic:** epic-01-init-scaffolding · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-1.4 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Daedalus es **agnóstico del backend de agentes** (D3): la definición canónica vive en `.daedalus/` y se **compila** al formato nativo de la herramienta elegida mediante un **adaptador**. Para que la compilación (epic 06) sepa a qué formato apuntar, el workspace debe registrar **qué backend(s)** son el objetivo. En el MVP el único backend soportado es **Claude Code**, pero la interfaz debe quedar preparada para más (RNF-7, PRD §14).

Este ticket cubre la **elección y registro** del/los backend(s) durante `daedalus init`, persistiéndolos en el campo `backends` del manifiesto que define el ticket 01-03.

## Feature / Qué se construye

El paso de `daedalus init` que determina el/los backend(s) objetivo y los registra en `.daedalus/daedalus.yaml`:

- Durante `init`, el usuario puede elegir el/los backend(s) objetivo de un conjunto de opciones soportadas.
- En el MVP el conjunto soportado es **{ Claude Code }**, que también es el **valor por defecto**.
- La selección queda **persistida** en el campo `backends` del manifiesto (escrito por la generación del ticket 01-03).
- Elegir un backend **no soportado** se rechaza con un mensaje claro (la interfaz de adaptadores está lista pero la implementación de otros backends es post-MVP).
- El diseño deja la puerta abierta a múltiples backends sin romper el formato del manifiesto (RNF-7).

## Requerimientos

- R1. `daedalus init` permite **seleccionar** el/los backend(s) objetivo entre las opciones soportadas.
- R2. El conjunto de backends soportados en el MVP es exactamente **{ Claude Code }**.
- R3. El **valor por defecto** (sin interacción explícita / modo no interactivo) es **Claude Code**.
- R4. La selección se **persiste** en el campo `backends` del manifiesto `.daedalus/daedalus.yaml`.
- R5. Solicitar un backend **no soportado** se **rechaza** con un mensaje de error claro; no se escribe un valor inválido en el manifiesto.
- R6. El formato del campo `backends` admite, a nivel de estructura, **uno o más** backends (preparado para multi-backend, aunque el MVP solo soporte uno).
- R7. La selección es **determinista** dado el input (la misma elección produce el mismo registro en el manifiesto).
- R8. Coherente con el ticket 01-03 (no duplica el manifiesto: escribe/actualiza el campo `backends` del mismo artefacto) y con 01-02 (no destructivo).

## Criterios de aceptación

- [ ] CA1. Tras `init`, el manifiesto registra el backend seleccionado en el campo `backends`.
- [ ] CA2. Sin elección explícita (default/no interactivo), el backend registrado es **Claude Code**.
- [ ] CA3. Es posible elegir **Claude Code** explícitamente y queda registrado.
- [ ] CA4. Solicitar un backend no soportado produce un error claro y no escribe un valor inválido en el manifiesto.
- [ ] CA5. El campo `backends` tiene una forma que admite uno o más backends (estructura lista para multi-backend).
- [ ] CA6. La misma elección produce el mismo valor registrado entre corridas (determinismo).

## Fuera de alcance

- La **compilación** efectiva al formato nativo del backend / implementación del adaptador Claude Code (epic 06).
- Soporte real de backends distintos a Claude Code (post-MVP; solo se deja la interfaz/forma de datos lista).
- Definición completa del esquema del manifiesto más allá del campo `backends` (ticket 01-03).
- Creación del scaffolding (01-01) y detección/upgrade (01-02).

## Referencias

- PRD.md — RF-1.4, D3 (multi-backend; Claude Code primer adaptador), RNF-7 (extensibilidad de adaptadores), §14 (adaptadores adicionales fuera de scope).
- init.md — §4.2 (`daedalus.yaml` registra backend(s)), §3 (glosario: backend, adaptador, compilación).
- epic-01-init-scaffolding/epic.md — criterio: "la selección de backend queda registrada en el manifiesto".
- Ticket 01-03 — manifiesto que persiste el campo `backends`.
