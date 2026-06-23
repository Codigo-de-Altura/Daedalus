# Validación automática — Ticket 09-01

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible, y la suite de tests del proyecto es ejecutable (`go test ./...`).
- Existe un workspace `.daedalus/` de prueba sobre el que ejercitar `init`, `build`/`sync` y las validaciones (uno nuevo y uno preexistente para forzar la rama de upgrade/merge).
- Se puede capturar la salida de logs de una corrida (a fichero o stream) para inspeccionarla.

## Checks

1. **Logs de `init` en puntos de decisión** — Comando: ejecutar `init` sobre un repo sin `.daedalus/` y sobre uno con `.daedalus/` preexistente, capturando logs · Esperado: los logs muestran inicio, detección del `.daedalus/` preexistente, decisión crear vs. upgrade/merge, backend(s) elegido(s), artefactos generados y resultado.
2. **Logs de `build`/`sync` en puntos de decisión** — Comando: ejecutar `build`/`sync` y capturar logs · Esperado: los logs muestran inicio, adaptador/backend seleccionado, decisión de escritura idempotente vs. no-op, artefactos compilados y resultado.
3. **Logs de validaciones** — Comando: correr una validación de definiciones sobre definiciones válidas e inválidas, capturando logs · Esperado: los logs indican la definición evaluada y su resultado (válida/inválida) con el motivo del rechazo.
4. **Estructura y niveles** — Comando: inspeccionar los eventos de log capturados · Esperado: son estructurados (clave/valor) con campos consistentes entre operaciones; los niveles info/warn/error se usan de forma coherente.
5. **Sin datos sensibles** — Comando: revisar los logs capturados buscando secretos, tokens, credenciales o contenido íntegro de prompts/briefs · Esperado: no aparecen datos sensibles; las rutas se registran relativas al workspace cuando aplica.
6. **Determinismo intacto (golden / RNF-5)** — Comando: ejecutar `build` dos veces sobre el mismo input —con y sin logging activo— y comparar los artefactos generados · Esperado: los artefactos son idénticos; el logging no altera la salida.
7. **Idioma inglés** — Comando: revisar mensajes y claves de los logs · Esperado: están en inglés.
8. **Tests unitarios** — Comando: `go test ./...` · Esperado: pasan, incluyendo los que cubren la instrumentación de logging de las operaciones.
9. **Agnóstico de backend** — Comando: revisar la instrumentación de logging · Esperado: no hay referencias a Claude Code ni a ningún backend/runtime concreto.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1 |
| CA2 | 2 |
| CA3 | 3 |
| CA4 | 4 |
| CA5 | 5 |
| CA6 | 6 |
| CA7 | 7 |
| CA8 | 9 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
