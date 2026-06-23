# Ticket 00-04 — Baseline de logging estructurado

> **Epic:** epic-00-foundations · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-9.1 (logging estructurado, baseline) · PRD §11 (observabilidad) · epic.md (ticket-00-04) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

El epic-09 profundiza la estrategia fina de logging/telemetría de producto, pero el resto de los epics necesita desde ya un baseline de logging estructurado reutilizable en los puntos de decisión del core (init, build, validaciones). Este ticket establece ese baseline mínimo: un logger estructurado, configurable por nivel, sin datos sensibles, expuesto al core para que los epics siguientes lo usen de forma consistente. No es la observabilidad completa; es el piso común.

## Feature / Qué se construye

Un componente de logging estructurado en `internal/` que el core de Daedalus usa para registrar eventos en puntos de decisión. Emite registros estructurados (clave/valor, p. ej. JSON o `slog`), soporta niveles (debug/info/warn/error), permite configurar el nivel por entorno, y nunca registra datos sensibles. Incluye un punto de inicialización claro que el binario y los futuros comandos puedan reutilizar.

## Requerimientos

- R1 — Existe un componente de logging estructurado en `internal/` (p. ej. sobre `log/slog`), que produce registros con campos clave/valor en un formato consistente y parseable.
- R2 — Soporta niveles estándar (al menos `debug`, `info`, `warn`, `error`) y permite configurar el nivel mínimo (p. ej. vía variable de entorno o config), con un default razonable.
- R3 — Provee una función de inicialización/constructor que devuelve un logger listo para inyectar en el core, evitando estado global oculto donde sea razonable.
- R4 — El bootstrap del binario (ticket-00-02) inicializa el logger y emite al menos un evento estructurado en un punto de decisión base (p. ej. arranque/cierre del programa).
- R5 — El logging no registra datos sensibles; existe una convención explícita (documentada en código/comentarios) de qué no se loguea (secretos, rutas con credenciales, contenido sensible de usuario).
- R6 — El logger es reutilizable por los epics siguientes (init, build, validaciones): su API es estable y no acoplada a una feature concreta.
- R7 — Existe cobertura de tests mínima que verifica el formato estructurado, el filtrado por nivel y la ausencia de campos prohibidos en un evento de ejemplo.
- R8 — La salida de logging es separable de la salida de la TUI (p. ej. va a `stderr` o a un sink configurable) para no corromper el render de Bubble Tea.

## Criterios de aceptación

- [ ] CA1 — Existe un paquete/componente de logging estructurado bajo `internal/`.
- [ ] CA2 — El logger emite registros estructurados (clave/valor) y respeta el nivel configurado (un evento `debug` se suprime con nivel `info`, etc.).
- [ ] CA3 — El nivel de log es configurable (p. ej. variable de entorno) con un default definido.
- [ ] CA4 — El binario inicializa el logger y emite al menos un evento estructurado en arranque/cierre.
- [ ] CA5 — Los tests del paquete de logging pasan (`go test ./internal/...` para logging) y verifican formato, nivel y ausencia de datos sensibles.
- [ ] CA6 — `go build ./...` y `go vet ./...` pasan sin errores.
- [ ] CA7 — La salida de log no se mezcla con el render de la TUI (va a stderr o a un sink configurable).

## Fuera de alcance

- Telemetría, métricas, trazas distribuidas y observabilidad avanzada (Fase 2+).
- Estrategia fina de logging de producto por feature (epic-09).
- Logging específico de comandos de producto (init, build) — esos epics lo usan, no lo definen aquí.
- Pipeline de CI (ticket-00-05).

## Referencias

- PRD RF-9.1 — Logging estructurado de operaciones de Daedalus (baseline)
- PRD §11 — Observabilidad (logging estructurado; runs en vivo fuera de scope)
- epic-00-foundations/epic.md (ticket-00-04; "sin datos sensibles")
- CLAUDE.md §2 (logging/telemetría en el backend/core), §1 (idioma del código)
