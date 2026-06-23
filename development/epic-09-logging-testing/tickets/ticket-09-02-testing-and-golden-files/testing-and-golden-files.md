# Ticket 09-02 — Testing unitario y golden files de compilación

> **Epic:** epic-09-logging-testing · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-9.2 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Daedalus es la **fuente de verdad** de la estructura de IA de un proyecto y la **compila** al formato nativo del backend (`.claude/` como primer adaptador). Para que esa compilación sea confiable y portable, debe ser **determinista**: mismo input canónico → mismo output (RNF-5). Sin una red de seguridad de tests, cualquier cambio en el core o en un adaptador puede romper silenciosamente el contrato de salida o el comportamiento del dominio.

Este ticket define la **estrategia de testing del propio Daedalus**: una **suite de tests unitarios** del core (modelo de dominio, validación de esquemas, (de)serialización) y un conjunto de **golden files de compilación** que fijan la salida esperada de `build`/`sync`, de modo que cualquier desviación se detecte como un fallo de test. Es la materialización concreta del principio de determinismo de la metodología (CLAUDE.md §7, RNF-5).

No es un ticket de feature de producto: construye la **infraestructura de pruebas** que el resto del proyecto usa para garantizar corrección y reproducibilidad.

## Feature / Qué se construye

La estrategia y el andamiaje de testing del proyecto:

- **Tests unitarios del core** — cobertura de las piezas de dominio y sus invariantes: modelo canónico (agentes, prompts, workflows, manifiesto), validación de esquemas, (de)serialización determinista, y los caminos de decisión de `init`/`build`. Tests de la forma `go test` idiomática del proyecto.
- **Golden files de compilación** — para uno o más inputs canónicos de referencia, se fija la salida esperada del adaptador (`build`/`sync`) como archivos *golden* versionados. El test compila el input y compara, byte a byte, contra el golden; cualquier diferencia falla.
- **Determinismo verificable (RNF-5)** — los tests demuestran que compilar el mismo input produce el **mismo output**, y que la salida es **estable y ordenada** (apta para diffs git limpios — RNF-6). Esto incluye que la compilación no dependa de orden de mapas, timestamps ni rutas absolutas.
- **Actualización controlada de golden files** — existe un mecanismo explícito (p. ej. un flag de *update*) para regenerar los golden cuando un cambio de salida es **intencional**, de modo que el diff del golden quede revisable en el PR.

El alcance es la infraestructura y los casos que cubren el contrato de compilación y los invariantes del core; no busca cobertura exhaustiva al 100% sino una red de seguridad significativa sobre los caminos críticos.

## Requerimientos

- R1. Existe una **suite de tests unitarios** del core ejecutable con `go test ./...`, que cubre el modelo de dominio, la validación de esquemas y la (de)serialización determinista.
- R2. Existen **golden files de compilación**: para input(s) canónico(s) de referencia se fija la salida esperada de `build`/`sync` como archivos versionados, y un test compara la salida real contra el golden.
- R3. Los tests verifican el **determinismo** de la compilación (RNF-5): mismo input → mismo output; la salida no depende de orden de mapas, timestamps ni rutas absolutas.
- R4. Los tests verifican que la salida es **estable y ordenada** (RNF-6): la comparación contra golden es byte a byte y un cambio no intencional la hace fallar.
- R5. Existe un **mecanismo explícito** para regenerar/actualizar los golden files cuando el cambio de salida es intencional (p. ej. un flag de update), dejando el diff revisable.
- R6. Los tests son **reproducibles y deterministas** ellos mismos: no dependen de red, reloj, ni del entorno; corren en Windows, macOS y Linux (RNF-3).
- R7. Los golden files y los fixtures viven en el repositorio en formato de texto legible, ordenados para minimizar ruido en diffs (RNF-6).
- R8. Todo lo escrito (código de test, nombres, fixtures, comentarios) está en **inglés** (CLAUDE.md §1).

## Criterios de aceptación

- [ ] CA1. `go test ./...` ejecuta la suite y pasa; cubre modelo de dominio, validación de esquemas y (de)serialización.
- [ ] CA2. Existen golden files de compilación para al menos un input canónico de referencia, y un test los compara contra la salida real de `build`/`sync`.
- [ ] CA3. Un test demuestra que compilar dos veces el mismo input produce salidas idénticas (determinismo — RNF-5).
- [ ] CA4. Si la salida de compilación cambia respecto del golden de forma no intencional, el test correspondiente **falla** (la red de seguridad funciona).
- [ ] CA5. Existe un mecanismo explícito para regenerar los golden files (p. ej. flag de update) y, tras regenerarlos para un cambio intencional, los tests vuelven a pasar.
- [ ] CA6. Los tests no dependen de red, reloj ni rutas absolutas, y son reproducibles entre corridas y plataformas.
- [ ] CA7. Golden files y fixtures están en texto legible y ordenado, en el repositorio.
- [ ] CA8. El código de test, los nombres y los fixtures están en inglés.

## Fuera de alcance

- Cobertura exhaustiva al 100% de todo el código (la meta es una red de seguridad sobre caminos críticos, no un objetivo de cobertura).
- Tests de la capa **TUI/frontend** y de UX (dominio de Padmé/Leia; este ticket es backend/core).
- Tests de integración contra un backend de agentes **ejecutándose en vivo** (la ejecución de agentes está fuera de scope de Fase 1 — PRD §4.2, D5).
- Benchmarks de performance / verificación de RNF-1 (arranque < 200 ms): fuera de este ticket.
- CI/pipeline de ejecución de los tests (su scaffolding pertenece a foundations/CI; aquí se asegura que la suite existe y corre con `go test`).

## Referencias

- PRD.md — RF-9.2 (testing: unidad + golden files de compilación), RNF-5 (determinismo: golden files), RNF-6 (git-friendly: salida ordenada y estable), RNF-3 (portabilidad Windows/macOS/Linux), §8.1 (compilación reproducible → portabilidad).
- CLAUDE.md — §7 (determinismo: la compilación es reproducible, golden files), §1 (todo lo escrito en inglés), §2 (backend = core: compilación, persistencia).
- epic-09-logging-testing/epic.md — objetivo, alcance y criterios del epic (existe suite de tests unitarios y golden files que fijan la salida de compilación).
