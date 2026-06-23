# Prompt Composition — Composición/inclusión de prompts (DRY)

> **Epic:** epic-03-prompt-management · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-3.2 · **Estilo:** SDD

## Contexto

Los prompts globales y compartidos (ticket-03-01) son fragmentos reutilizables persistidos en `.daedalus/prompts/`. Para evitar duplicación (DRY) y mantener consistencia, un prompt debe poder **incluir/componer** otros fragmentos: convenciones, glosario, estilo, definiciones de rol, etc.

Este ticket cubre el **mecanismo de composición/inclusión** y su **resolución determinista**: dado un prompt con referencias a otros, el core produce el texto final ensamblado, de forma reproducible y sin duplicar contenido.

No cubre la edición/persistencia base de prompts (ticket-03-01), la previsualización en la TUI (ticket-03-03), ni la compilación al backend (epic-06).

## Feature / Qué se construye

Capacidad del **core (backend)** para **resolver inclusiones entre prompts**: un prompt puede referenciar otros prompts compartidos mediante una sintaxis de inclusión, y el core los expande en un único texto final renderizado, de manera **determinista** y libre de duplicación.

El feature expone una operación de **resolución/composición** que recibe el id de un prompt y devuelve su texto compuesto, junto con la detección de errores de composición (referencias inexistentes, ciclos).

## Requerimientos

- **R1.** Existe una **sintaxis de inclusión** explícita y legible dentro del cuerpo de un prompt que referencia a otro prompt por su `id`/slug (p. ej. una directiva de inclusión).
- **R2.** El core **resuelve** las inclusiones expandiéndolas en el lugar de la referencia, produciendo un único texto final compuesto.
- **R3.** La resolución es **recursiva**: un prompt incluido puede a su vez incluir otros, hasta resolver todas las referencias.
- **R4.** La resolución es **determinista**: mismo conjunto de prompts → mismo texto compuesto (orden estable, sin ambigüedad).
- **R5.** Se **detectan ciclos** de inclusión (A→B→A) y se reportan como error explícito, sin bucle infinito.
- **R6.** Una referencia a un prompt **inexistente** es un error explícito que identifica el id faltante.
- **R7.** El mecanismo es **DRY**: un mismo fragmento referenciado por varios prompts vive en un solo archivo fuente; no se duplica contenido en disco.
- **R8.** La composición **no muta** los archivos fuente: produce el texto final en memoria/salida sin reescribir los prompts originales.
- **R9.** Las convenciones de inclusión (sintaxis, resolución de `id`, manejo de espacios/encabezados) están **documentadas y son consistentes**.

## Criterios de aceptación

- [ ] Un prompt que **incluye** otro produce un texto compuesto con el fragmento referenciado expandido en su lugar.
- [ ] La inclusión es **recursiva**: un fragmento incluido que a su vez incluye otro se resuelve completamente.
- [ ] Resolver el mismo prompt dos veces produce **exactamente el mismo** texto (determinismo).
- [ ] Un **ciclo** de inclusión se detecta y se reporta como error, sin colgar el proceso.
- [ ] Una referencia a un `id` **inexistente** falla con error explícito que nombra el id.
- [ ] El fragmento compartido vive en **un solo archivo**; varios prompts pueden referenciarlo sin duplicarlo en disco.
- [ ] La resolución **no modifica** los archivos fuente de los prompts.

## Fuera de alcance

- Edición/persistencia base de prompts globales y compartidos (ticket-03-01).
- Previsualización renderizada en la TUI (ticket-03-03).
- Compilación del prompt final al backend de agentes (epic-06).

## Referencias

- `PRD.md` — RF-3.2, sección 9 EPIC-3, RNF-5 (determinismo).
- `epic.md` — epic-03-prompt-management (composición determinista, sin duplicación).
- `ticket-03-01-global-shared-prompts` — base de prompts.
- `CLAUDE.md` — §7 (filosofía SDD: determinismo, no destrucción).
