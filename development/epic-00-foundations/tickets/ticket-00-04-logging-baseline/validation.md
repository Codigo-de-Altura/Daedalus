# Validación automática — Ticket 00-04

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- Tickets 00-01 y 00-02 aprobados (módulo, layout y bootstrap existentes).
- Go toolchain disponible.
- Se ejecuta desde la raíz del repositorio.

## Checks

1. **Paquete de logging presente** — Comando: `ls internal/ | grep -iE 'log'` (o búsqueda del paquete de logging bajo `internal/`) · Esperado: existe un componente/paquete de logging estructurado en `internal/`.
2. **Logging estructurado** — Comando: `grep -RE 'slog|log/slog|NewLogger|Handler' internal/` · Esperado: el componente usa logging estructurado (p. ej. `log/slog`) y expone un constructor/inicializador.
3. **Configurable por nivel** — Comando: `grep -RiE 'level|LOG_LEVEL|SetLevel|LevelInfo|LevelDebug' internal/` · Esperado: el nivel mínimo es configurable y existe un default.
4. **Tests de logging** — Comando: `go test ./internal/...` (alcance del paquete de logging) · Esperado: tests pasan; cubren formato estructurado, filtrado por nivel y ausencia de campos sensibles.
5. **Evento en arranque/cierre** — Comando: `go build -o daedalus ./cmd/daedalus && printf 'q' | ./daedalus 2> log.txt; grep -E 'level|msg|=' log.txt` · Esperado: el binario emite al menos un registro estructurado a stderr en arranque/cierre.
6. **Log separado de la TUI** — Verificación: la salida de log va a `stderr` o a un sink configurable, no a `stdout` mezclado con el render · Esperado: el render de la TUI no se corrompe con líneas de log.
7. **Build y vet** — Comando: `go build ./... && go vet ./...` · Esperado: ambos sin errores, código de salida 0.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1 |
| CA2 | 2, 4 |
| CA3 | 3 |
| CA4 | 5 |
| CA5 | 4 |
| CA6 | 7 |
| CA7 | 6 |

## Verdict

- **APPROVED** si todos los checks pasan.
- **REJECTED** si alguno falla → hallazgos con severidad, comportamiento observado y esperado (para `observations.md`).
