# Validación — Ticket 02-02 — Importar / clonar y editar un agente

> La corre **Yoda**. Verifica que el feature de `import-clone-edit.md` cumple sus criterios de aceptación. El validador **solo reporta**, nunca arregla.

## Precondiciones

- El repositorio compila (`go build ./...` sin errores).
- El catálogo built-in (ticket-02-01) está disponible.
- Existe un workspace `.daedalus/` de prueba en un directorio temporal limpio.

## Checks

### Check 1 — Clonar crea una definición canónica nueva
- **Comando:** clonar `analyst` del catálogo a un identificador destino (p. ej. `analyst-custom`) y listar `.daedalus/agents/`.
- **Esperado:** aparece la definición canónica del clon bajo el identificador destino.

### Check 2 — El clon es independiente del original
- **Comando:** editar el clon (cambiar rol, prompt y parámetros) y luego inspeccionar la definición del `analyst` del catálogo built-in.
- **Esperado:** el agente del catálogo built-in permanece sin cambios; las ediciones solo afectan al clon.

### Check 3 — Edición de rol, prompt y parámetros persiste
- **Comando:** editar rol, prompt y parámetros del clon y volver a leer su definición canónica.
- **Esperado:** los tres campos reflejan los valores editados.

### Check 4 — Clonado no destructivo
- **Comando:** clonar sobre un identificador destino que ya existe en el workspace.
- **Esperado:** no se sobreescribe silenciosamente; se informa el conflicto u ofrece preview/confirmación.

### Check 5 — Edición inválida produce error accionable
- **Comando:** intentar una edición que viola el esquema canónico (p. ej. vaciar el rol).
- **Esperado:** error accionable (campo, observado, esperado); la definición no queda inválida sin aviso.

### Check 6 — Identificador en kebab-case
- **Comando:** inspeccionar el identificador del clon.
- **Esperado:** en `kebab-case`.

## Mapeo a criterios

| Check | Criterio(s) |
|---|---|
| Check 1 | CA1 |
| Check 2 | CA2 |
| Check 3 | CA3 |
| Check 4 | CA4 |
| Check 5 | CA5 |
| Check 6 | CA6 |
| Todos | CA7 (trazabilidad RF-2.2) |

## Verdict

**APPROVED / REJECTED** — _(a completar por Yoda al ejecutar)._

Hallazgos (si REJECTED), uno por ítem:
- **Severidad:** _(bloqueante / mayor / menor)_
- **Observado:** _(qué se vio)_
- **Esperado:** _(qué debía verse, con el criterio asociado)_
