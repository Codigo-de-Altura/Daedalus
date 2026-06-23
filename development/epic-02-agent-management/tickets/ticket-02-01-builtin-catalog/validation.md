# Validación — Ticket 02-01 — Catálogo built-in de agentes

> La corre **Yoda**. Verifica que el feature de `builtin-catalog.md` cumple sus criterios de aceptación. El validador **solo reporta**, nunca arregla.

## Precondiciones

- El repositorio compila (`go build ./...` sin errores).
- Existe un workspace `.daedalus/` de prueba (o se puede inicializar uno) en un directorio temporal limpio.
- El catálogo built-in está embebido en el binario (no requiere red ni archivos externos).

## Checks

### Check 1 — El catálogo lista los 5 agentes canónicos
- **Comando:** ejecutar la operación de listado del catálogo built-in.
- **Esperado:** la salida incluye, como mínimo, `analyst`, `architect`, `planner`, `validator`, `documenter`, cada uno con su rol/descripción.

### Check 2 — Cada agente tiene rol y prompt no vacíos y valida
- **Comando:** para cada agente del catálogo, inspeccionar su definición canónica y correr la validación de esquema (ticket-02-04).
- **Esperado:** rol y prompt no vacíos; cada definición pasa la validación sin errores.

### Check 3 — Materializar un agente crea su definición canónica
- **Comando:** materializar `analyst` en un workspace limpio y listar `.daedalus/agents/`.
- **Esperado:** aparece la definición canónica de `analyst` (YAML + prompt MD) bajo `.daedalus/agents/`.

### Check 4 — Materialización no destructiva
- **Comando:** materializar `analyst` por segunda vez sobre un workspace que ya lo tiene.
- **Esperado:** no se sobreescribe silenciosamente; se informa el conflicto u ofrece preview/confirmación.

### Check 5 — Identificadores en kebab-case
- **Comando:** inspeccionar los identificadores de los agentes del catálogo y de los materializados.
- **Esperado:** todos en `kebab-case`.

### Check 6 — Determinismo
- **Comando:** materializar el mismo agente dos veces en workspaces limpios y comparar el contenido generado.
- **Esperado:** contenido idéntico byte a byte (orden de claves estable).

## Mapeo a criterios

| Check | Criterio(s) |
|---|---|
| Check 1 | CA1 |
| Check 2 | CA2 |
| Check 3 | CA3 |
| Check 4 | CA4 |
| Check 5 | CA5 |
| Check 6 | CA6 |
| Todos | CA7 (trazabilidad RF-2.1) |

## Verdict

**APPROVED / REJECTED** — _(a completar por Yoda al ejecutar)._

Hallazgos (si REJECTED), uno por ítem:
- **Severidad:** _(bloqueante / mayor / menor)_
- **Observado:** _(qué se vio)_
- **Esperado:** _(qué debía verse, con el criterio asociado)_
