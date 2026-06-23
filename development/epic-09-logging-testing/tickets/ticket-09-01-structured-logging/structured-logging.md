# Ticket 09-01 — Logging estructurado de las operaciones de Daedalus

> **Epic:** epic-09-logging-testing · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-9.1 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Daedalus ejecuta operaciones de soporte —`init`, `build`/`sync`, validaciones de definiciones— que toman decisiones (detectar un `.daedalus/` existente, decidir un merge no destructivo, escribir o no un artefacto, rechazar una definición inválida). Cuando algo no sale como se espera, el usuario y el equipo necesitan **trazabilidad** de qué decidió Daedalus y por qué, sin tener que adivinar.

El **baseline mínimo de logging** ya se establece en `epic-00` (foundations): existe un logger del proyecto y un punto de entrada de logging. Este ticket **profundiza** ese baseline: define el **logging estructurado** de las operaciones clave de Daedalus, instrumentando los **puntos de decisión** de `init`, `build`/`sync` y las validaciones, con campos estructurados consistentes y **sin filtrar datos sensibles**.

Queda **fuera de scope** la telemetría/observabilidad de runs en vivo de agentes (Fase 2+ — PRD §14): aquí solo se loguean las operaciones del propio Daedalus, no la ejecución de agentes.

## Feature / Qué se construye

La instrumentación de logging estructurado de las operaciones de Daedalus sobre el logger establecido en `epic-00`. Cada operación clave emite eventos estructurados en sus **puntos de decisión**, de modo que la traza reconstruya qué hizo Daedalus y por qué:

- **`init`** — inicio de la operación, detección de un `.daedalus/` preexistente, decisión de crear vs. upgrade/merge, artefactos generados (manifiesto, `init.md`), backend(s) objetivo elegido(s), resultado final.
- **`build`/`sync`** — inicio, área canónica leída, adaptador/backend seleccionado, decisión de escritura idempotente vs. no-op, preview/diff, artefactos compilados, resultado final.
- **Validaciones** — inicio de la validación, definición evaluada (agente/workflow/manifiesto), resultado por definición (válida/inválida) y motivo de rechazo.

Cada evento de log es **estructurado** (pares clave/valor, no texto interpolado libre), con campos consistentes que permitan filtrar y correlacionar (p. ej. operación, fase/etapa, resultado, ruta del artefacto relativa al workspace). Los **niveles** de log se usan de forma consistente: info para el curso normal de la operación, warn para decisiones degradadas/no destructivas, error para fallos.

El logging es **observacional**: no altera el comportamiento de las operaciones ni su determinismo. Y es **higiénico con datos sensibles**: no registra secretos, tokens, credenciales ni contenido íntegro de prompts/briefs del usuario; las rutas se registran de forma relativa al workspace cuando aplica.

## Requerimientos

- R1. Las operaciones clave de Daedalus —`init`, `build`/`sync` y las validaciones de definiciones— emiten logs en sus **puntos de decisión** (no solo al inicio/fin), reconstruyendo qué decidió Daedalus y por qué.
- R2. Los logs son **estructurados**: pares clave/valor con campos consistentes (operación, etapa/fase, resultado y, cuando aplique, el artefacto afectado), no cadenas de texto interpoladas sin estructura.
- R3. Se reutiliza el logger/baseline establecido en `epic-00`; este ticket **profundiza** la instrumentación, no introduce un segundo subsistema de logging.
- R4. Los **niveles** de log se aplican de forma consistente: info (curso normal), warn (decisiones degradadas o no destructivas), error (fallos).
- R5. Los logs **no contienen datos sensibles**: ni secretos/tokens/credenciales, ni el contenido íntegro de prompts/briefs del usuario; las rutas se registran de forma relativa al workspace cuando corresponde.
- R6. El logging es **observacional**: no cambia el comportamiento ni el resultado de las operaciones, ni rompe el **determinismo** de la compilación (RNF-5) — la salida de artefactos no depende de que el logging esté activo.
- R7. Todo lo escrito (mensajes y claves de log) está en **inglés** (CLAUDE.md §1).
- R8. El logging es **agnóstico del backend**: no acopla la instrumentación a Claude Code ni a ningún runtime de agentes.

## Criterios de aceptación

- [ ] CA1. Al ejecutar `init`, los logs reflejan los puntos de decisión: inicio, detección de `.daedalus/` preexistente, decisión crear vs. upgrade/merge, backend(s) elegido(s), artefactos generados y resultado.
- [ ] CA2. Al ejecutar `build`/`sync`, los logs reflejan los puntos de decisión: inicio, adaptador/backend seleccionado, decisión de escritura idempotente vs. no-op, artefactos compilados y resultado.
- [ ] CA3. Al correr una validación de definiciones, los logs reflejan la definición evaluada y su resultado (válida/inválida) con el motivo de rechazo.
- [ ] CA4. Los eventos de log son estructurados (clave/valor) con campos consistentes entre operaciones, y los niveles (info/warn/error) se usan de forma coherente.
- [ ] CA5. La inspección de los logs producidos no revela secretos, tokens, credenciales ni el contenido íntegro de prompts/briefs; las rutas aparecen relativas al workspace cuando aplica.
- [ ] CA6. El resultado de las operaciones (artefactos de `build`) es idéntico con o sin la instrumentación de logging activa: el logging no afecta el determinismo (RNF-5).
- [ ] CA7. Los mensajes y claves de log están en inglés.
- [ ] CA8. La instrumentación no contiene referencias a un backend concreto.

## Fuera de alcance

- **Telemetría/observabilidad de runs en vivo** de agentes: streaming de logs de ejecución, intervención mid-run (Fase 2+ — PRD §14, §4.2).
- El **baseline mínimo de logging** (logger base, punto de entrada): ya establecido en `epic-00`.
- Métricas/tracing distribuido y exportadores externos (fuera de scope de Fase 1).
- Logging de la **capa TUI/frontend** (este ticket es backend; la TUI vive en el dominio de Padmé/Leia).

## Referencias

- PRD.md — RF-9.1 (logging estructurado de init, build, validaciones), §11 (Observabilidad: logging estructurado; runs en vivo fuera de scope), §14 (Fase 2+: ejecución/observabilidad de agentes), RNF-5 (determinismo), RNF-8 (operaciones no destructivas).
- CLAUDE.md — §1 (todo lo escrito —incluidos logs— en inglés), §2 (backend = core: logging/telemetría), §7 (determinismo).
- epic-09-logging-testing/epic.md — objetivo, alcance (el baseline mínimo se establece en epic-00; aquí se profundiza) y criterios del epic.
