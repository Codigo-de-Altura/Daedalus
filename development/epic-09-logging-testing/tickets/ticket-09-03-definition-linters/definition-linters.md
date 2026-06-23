# Ticket 09-03 — Validaciones/linters de definiciones

> **Epic:** epic-09-logging-testing · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-9.3 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

El usuario edita las **definiciones canónicas** que viven en `.daedalus/`: agentes, workflows (DAG en YAML) y el manifiesto `daedalus.yaml`. Si una definición es inválida —un campo requerido faltante, un agente referenciado que no existe, un ciclo en el DAG, un manifiesto mal formado— el resultado es una compilación rota o un comportamiento inesperado, a menudo con un error oscuro y lejano a la causa.

Este ticket define las **validaciones/linters de definiciones**: un conjunto de comprobaciones que detectan definiciones inválidas **antes** de compilar y reportan el problema con **mensajes accionables** (qué archivo, qué campo/fase, qué se esperaba). Es la cara de "calidad de las definiciones" del epic de ecosistema, complementaria al logging (09-01) y a los tests (09-02).

Los linters **consolidan y profundizan** las reglas de validación de esquema ya esbozadas en epics previos (RF-2.4 validación de agente, RF-4.3 validación del DAG, RF-8.3 convenciones validables): aquí se reúnen en una capa de validación coherente, agnóstica del backend, con mensajes consistentes y accionables.

## Feature / Qué se construye

Una capa de validación de las definiciones canónicas que cubre las tres familias:

- **Agentes** — la definición cumple el esquema canónico: campos requeridos presentes, tipos correctos, rol/prompt válidos, sin campos desconocidos donde no se permiten (consolida RF-2.4).
- **Workflows (DAG)** — el grafo es válido: sin **ciclos**, sin **artefactos faltantes** (inputs que nadie produce), sin **referencias a agentes inexistentes**, ids de fase únicos, dependencias bien formadas (consolida RF-4.3).
- **Manifiesto (`daedalus.yaml`)** — el manifiesto cumple su esquema: nombre, backend(s) declarado(s), versión y convenciones bien formados; backends referenciados conocidos; convenciones de equipo coherentes (toca RF-1.3, RF-8.3).

Cada hallazgo del linter es **accionable**: identifica el **archivo/definición**, la **ubicación** (campo, fase, clave) y describe **qué se esperaba** vs. **qué se encontró**, de forma que el usuario pueda corregir sin adivinar. Las validaciones se pueden invocar de forma explícita (comando de validación) y son la base que `build`/`sync` puede aprovechar para no compilar definiciones inválidas. La capa es **agnóstica del backend**: valida el modelo canónico, no formatos nativos.

El alcance es la lógica de validación y sus mensajes; la integración fina en el flujo de TUI y los comandos pertenece a sus epics respectivos, pero las reglas y los reportes accionables se definen aquí.

## Requerimientos

- R1. Existe una capa de **linters/validaciones** que evalúa las tres familias de definiciones canónicas: **agentes**, **workflows (DAG)** y **manifiesto** (`daedalus.yaml`).
- R2. El linter de **agentes** detecta definiciones que incumplen el esquema canónico: campos requeridos faltantes, tipos inválidos, rol/prompt ausentes o malformados.
- R3. El linter de **workflows** detecta DAGs inválidos: **ciclos**, **artefactos faltantes** (inputs no producidos por ninguna fase), **referencias a agentes inexistentes**, ids de fase duplicados y dependencias malformadas.
- R4. El linter del **manifiesto** detecta manifiestos inválidos: campos requeridos faltantes/malformados (nombre, backend(s), versión, convenciones) y backends desconocidos.
- R5. Cada hallazgo es **accionable**: indica el archivo/definición, la ubicación (campo/fase/clave) y describe qué se esperaba vs. qué se encontró.
- R6. Las validaciones **no producen panic** ante entradas malformadas; reportan errores controlados.
- R7. Las validaciones son **agnósticas del backend**: operan sobre el modelo canónico, sin acoplarse a Claude Code ni a ningún runtime.
- R8. Los mensajes de los linters están en **inglés** (CLAUDE.md §1), son **deterministas** (mismo input → mismo conjunto de hallazgos, en orden estable) y los reportes son aptos para diffs/logs limpios.

## Criterios de aceptación

- [ ] CA1. Una definición de **agente** inválida (campo requerido faltante o tipo incorrecto) es detectada y reportada con archivo, campo y expectativa.
- [ ] CA2. Un **workflow** con un **ciclo** en el DAG es detectado y reportado de forma accionable.
- [ ] CA3. Un **workflow** con un **input/artefacto que ninguna fase produce** o con una **referencia a un agente inexistente** es detectado y reportado.
- [ ] CA4. Un workflow con **ids de fase duplicados** o dependencias malformadas es detectado y reportado.
- [ ] CA5. Un **manifiesto** inválido (campo requerido faltante/malformado o backend desconocido) es detectado y reportado de forma accionable.
- [ ] CA6. Toda definición **válida** de las tres familias pasa los linters sin falsos positivos.
- [ ] CA7. Ante entradas malformadas, los linters reportan errores controlados sin panic.
- [ ] CA8. Los mensajes están en inglés, son deterministas y aparecen en orden estable; las validaciones no contienen referencias a un backend concreto.

## Fuera de alcance

- La **visualización** del DAG en la TUI (ticket 04-02) y la **edición** de definiciones (sus epics respectivos: 02, 03, 04).
- El **modelo de dominio** y la (de)serialización en sí (definidos en sus epics; aquí se valida, no se modela).
- Reglas de validación **específicas de un backend** o del formato nativo compilado (epic 06; aquí se valida el modelo canónico).
- Autocorrección/fix automático de definiciones inválidas (el linter reporta; no arregla).
- La estrategia general de tests del proyecto (ticket 09-02), más allá de los tests propios de estos linters.

## Referencias

- PRD.md — RF-9.3 (validaciones/linters de definiciones: agentes, workflows, manifiesto), RF-2.4 (validación de la definición de agente), RF-4.3 (validar el DAG: ciclos, artefactos faltantes, agentes inexistentes), RF-1.3 (manifiesto `daedalus.yaml`), RF-8.3 (convenciones de equipo validables), §7 (glosario: agente, workflow, manifiesto), RNF-8 (safety).
- CLAUDE.md — §1 (mensajes/logs en inglés), §2 (backend = core: validación de esquemas), §7 (trazabilidad, determinismo).
- epic-09-logging-testing/epic.md — objetivo, alcance y criterios (los linters detectan definiciones inválidas con mensajes accionables).
