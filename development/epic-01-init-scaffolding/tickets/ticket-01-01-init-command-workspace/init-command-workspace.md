# Ticket 01-01 — Comando `daedalus init` crea el workspace `.daedalus/`

> **Epic:** epic-01-init-scaffolding · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-1.1 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Daedalus apunta a cualquier repositorio (nuevo o existente) y materializa allí un workspace canónico `.daedalus/`, que es la **fuente de verdad agnóstica** de toda la estructura de IA del proyecto (agentes, prompts, workflows y backlog SDD). Antes de poder gestionar agentes, prompts o backlog, el repo objetivo necesita ese workspace creado de forma consistente y determinista.

Este ticket cubre la **creación inicial** del workspace: el comando `daedalus init` ejecutándose en un repo donde `.daedalus/` **no** existe todavía. La detección de un workspace preexistente y su upgrade/merge no destructivo viven en el ticket 01-02; la generación del contenido del manifiesto e `init.md` base viven en el 01-03; la selección de backend en el 01-04. Aquí se establece el **esqueleto de carpetas** y el comando que lo dispara.

## Feature / Qué se construye

El comando `daedalus init` que, ejecutado en la raíz de un repositorio objetivo, crea el árbol de directorios canónico `.daedalus/` según la estructura definida en `init.md` §4.2 y PRD §8.2:

```
.daedalus/
  daedalus.yaml        # manifiesto (contenido → ticket 01-03)
  init.md              # lineamiento maestro del proyecto target (contenido → ticket 01-03)
  agents/
  prompts/
  workflows/
  specs/
  architecture/
  epics/
  tickets/
  docs/
  .state/
```

El comando crea las carpetas y deja los archivos raíz (`daedalus.yaml`, `init.md`) en su lugar como artefactos del workspace (su contenido determinista se aborda en 01-03). La operación es **no destructiva** y debe ser **determinista** (mismo repo objetivo → misma estructura).

## Requerimientos

- R1. Existe un comando `daedalus init` invocable desde la CLI que opera sobre el directorio de trabajo actual como repo objetivo.
- R2. Cuando `.daedalus/` no existe en el repo objetivo, `init` crea el árbol completo de directorios de `init.md` §4.2: `agents/`, `prompts/`, `workflows/`, `specs/`, `architecture/`, `epics/`, `tickets/`, `docs/`, `.state/`.
- R3. `init` deja en su lugar los artefactos raíz `daedalus.yaml` e `init.md` dentro de `.daedalus/` (el contenido determinista es responsabilidad del ticket 01-03; aquí basta su existencia como parte del scaffolding).
- R4. La operación es **no destructiva**: nunca borra ni sobrescribe archivos del repo objetivo fuera del área `.daedalus/` que se está creando.
- R5. La creación es **determinista**: nombres, jerarquía y orden de la estructura son estables y reproducibles (apto para diffs git limpios).
- R6. El comando reporta al usuario el resultado de la operación (workspace creado y ruta), de forma clara y legible en terminal.
- R7. Funciona tanto en un repo vacío como en un repo existente que aún no tenga `.daedalus/` (la coexistencia con `.daedalus/` previo es del ticket 01-02).
- R8. Comportamiento portable entre Windows, macOS y Linux (separadores de ruta y permisos de directorio coherentes).

## Criterios de aceptación

- [ ] CA1. Ejecutar `daedalus init` en un directorio sin `.daedalus/` crea el directorio `.daedalus/` con **todas** las subcarpetas de `init.md` §4.2.
- [ ] CA2. Tras `init`, existen los artefactos raíz `.daedalus/daedalus.yaml` y `.daedalus/init.md`.
- [ ] CA3. Ningún archivo del repo objetivo fuera de `.daedalus/` es modificado o eliminado por la operación.
- [ ] CA4. Dos ejecuciones de `init` sobre dos copias idénticas de un repo vacío producen estructuras idénticas (determinismo).
- [ ] CA5. El comando emite un mensaje de resultado indicando que el workspace fue creado y dónde.
- [ ] CA6. La estructura generada coincide exactamente (nombres y jerarquía) con `init.md` §4.2 / PRD §8.2.

## Fuera de alcance

- Detección de un `.daedalus/` preexistente y upgrade/merge no destructivo (ticket 01-02).
- Generación del **contenido** determinista de `daedalus.yaml` e `init.md` (ticket 01-03).
- Selección y registro de backend(s) en el manifiesto (ticket 01-04).
- Contenido de agentes, prompts, workflows o backlog (epics 02–05) y compilación al backend (epic 06).

## Referencias

- PRD.md — RF-1.1, §8.2 (estructura `.daedalus/`), RNF-5 (determinismo), RNF-8 (no destructivo).
- init.md — §4.2 (workspace `.daedalus/`), §7 (convenciones: kebab-case, git-friendly, idempotencia).
- epic-01-init-scaffolding/epic.md — objetivo y criterios del epic.
