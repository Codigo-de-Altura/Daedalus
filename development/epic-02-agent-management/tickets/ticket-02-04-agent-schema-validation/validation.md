# Validación — Ticket 02-04 — Validación de agente contra el esquema canónico

> La corre **Yoda**. Verifica que el feature de `agent-schema-validation.md` cumple sus criterios de aceptación. El validador **solo reporta**, nunca arregla.

## Precondiciones

- El repositorio compila (`go build ./...` sin errores).
- Se dispone de definiciones de agente de prueba: (a) una válida; (b) una a la que le falta un campo obligatorio; (c) una con identificador que no cumple `kebab-case`; (d) una con múltiples problemas simultáneos.

## Checks

### Check 1 — El esquema declara los campos obligatorios
- **Comando:** inspeccionar la definición del esquema canónico de agente.
- **Esperado:** declara al menos identificador, rol y prompt como obligatorios, con sus reglas (tipos, no-vacío, `kebab-case` en el identificador).

### Check 2 — Definición válida pasa
- **Comando:** validar la definición válida (a).
- **Esperado:** veredicto válido, sin errores.

### Check 3 — Falta de campo obligatorio falla con error accionable
- **Comando:** validar la definición (b) sin campo obligatorio.
- **Esperado:** veredicto inválido; el error indica campo, observado y esperado.

### Check 4 — Identificador no kebab-case falla
- **Comando:** validar la definición (c) con identificador inválido.
- **Esperado:** veredicto inválido con error accionable sobre el identificador.

### Check 5 — Múltiples problemas en una sola pasada
- **Comando:** validar la definición (d) con varios problemas.
- **Esperado:** se reportan todos los hallazgos a la vez, no solo el primero.

### Check 6 — Determinismo
- **Comando:** validar la misma definición varias veces y comparar veredicto y errores.
- **Esperado:** mismo veredicto y mismo conjunto de errores, en orden estable.

## Mapeo a criterios

| Check | Criterio(s) |
|---|---|
| Check 1 | CA1 |
| Check 2 | CA2 |
| Check 3 | CA3 |
| Check 4 | CA4 |
| Check 5 | CA5 |
| Check 6 | CA6 |
| Todos | CA7 (trazabilidad RF-2.4) |

## Verdict

**APPROVED / REJECTED** — _(a completar por Yoda al ejecutar)._

Hallazgos (si REJECTED), uno por ítem:
- **Severidad:** _(bloqueante / mayor / menor)_
- **Observado:** _(qué se vio)_
- **Esperado:** _(qué debía verse, con el criterio asociado)_
