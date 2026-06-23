# Formatos diff-friendly

> **Epic:** epic-08-state-collab · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-8.2 · **Estilo:** SDD

---

## Contexto

Daedalus es una **fuente de verdad versionada en git para equipos** (PRD §3, D7). Cuando varias personas editan la misma estructura de IA, los **diffs y merges** deben ser limpios: un cambio pequeño en una definición canónica debe producir un diff pequeño y legible, no una reescritura completa del archivo por reordenamiento de claves, indentación inconsistente o campos volátiles. El PRD lo exige en RNF-5 (determinismo: mismo input → mismo output, golden files) y RNF-6 (git-friendly: artefactos de texto, ordenados y estables para minimizar ruido en diffs).

Este ticket define el requerimiento de que **toda serialización** que produce Daedalus use **formatos amigables a diff/merge** con **output determinista**.

## Feature / Qué se construye

La definición —como **plano**— de la **política de serialización diff-friendly y determinista** de Daedalus. Cubre todo artefacto de texto que Daedalus escribe como estado relevante (definiciones canónicas en YAML, documentos del backlog en Markdown, estado de progreso).

Esto incluye:

- **Legibilidad:** YAML y Markdown legibles por humanos, con estructura jerárquica clara.
- **Determinismo:** mismo input → mismo output byte a byte; sin campos volátiles (timestamps, ordenamientos aleatorios) embebidos en el estado canónico.
- **Claves ordenadas:** las claves de YAML se serializan en un **orden estable** (canónico/ordenado), de modo que reescribir un archivo sin cambios semánticos no genere diff.
- **Estabilidad de formato:** indentación, comillas, anchos y estilo de bloque consistentes y fijos, para que el ruido en diffs provenga solo de cambios reales.

## Requerimientos

1. **YAML determinista con claves ordenadas.** La serialización de YAML usa un orden de claves estable y reproducible (canónico o lexicográfico definido), no el orden de iteración de un mapa no determinista.
2. **Idempotencia de escritura.** Reescribir un artefacto cuyo contenido semántico no cambió produce **cero diff** (sin reordenamientos ni reformateo espurios).
3. **Sin volátiles en el estado canónico.** No se embeben timestamps, rutas absolutas de la máquina, ni identificadores aleatorios dentro del estado relevante versionado.
4. **Markdown estable y legible.** Los documentos en Markdown mantienen una estructura/estilo consistente (encabezados jerárquicos, tablas para metadatos), sin reformateo automático que cambie líneas no tocadas.
5. **Estilo de formato fijo.** Indentación, comillas y estilo de bloque de YAML son consistentes y configurados de forma fija (mismo estilo en todo el workspace).
6. **Coherencia con golden files.** El output es reproducible y apto para verificación por **golden files** (RNF-5).
7. **Trazabilidad.** El plano referencia RF-8.2, RNF-5 y RNF-6.

## Criterios de aceptación

- [ ] La serialización de YAML emite claves en un **orden estable** y reproducible.
- [ ] Reescribir un artefacto sin cambios semánticos produce **cero diff** (idempotente).
- [ ] El mismo input produce el **mismo output byte a byte** en ejecuciones repetidas (determinismo).
- [ ] No hay campos volátiles (timestamps, rutas de máquina, ids aleatorios) en el estado canónico versionado.
- [ ] Markdown se mantiene legible y estable (sin reformateo que altere líneas no editadas).
- [ ] El estilo de formato YAML (indentación, comillas, bloques) es consistente y fijo.
- [ ] El output es verificable por golden files.
- [ ] Trazabilidad explícita a RF-8.2 (y RNF-5 / RNF-6).

## Fuera de alcance

- **Qué** estado es relevante y **dónde** vive (inventario y ubicación → ticket-08-01).
- **Convenciones de naming/estructura** del equipo y su validación (→ ticket-08-03).
- Algoritmos de **merge automático** o resolución de conflictos (fuera de scope; git maneja el merge).
- Backend remoto / sincronización en la nube (epic.md — fuera de scope).
- Formato nativo de los artefactos compilados del backend `.claude/…` (EPIC-6).

## Referencias

- `PRD.md` — RF-8.2; RNF-5 (determinismo, golden files); RNF-6 (git-friendly: texto ordenado y estable).
- `init.md` — §7 (convenciones: "YAML: claves estables y ordenadas, output determinista, diffs limpios"; "Markdown: encabezados jerárquicos, tablas para metadatos").
- `development/epics/epic-08-state-collab/epic.md` — criterio "serialización determinista (claves ordenadas, diffs limpios)".
