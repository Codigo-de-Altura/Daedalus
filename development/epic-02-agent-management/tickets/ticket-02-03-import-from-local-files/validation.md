# Validación — Ticket 02-03 — Importar agentes desde archivos locales

> La corre **Yoda**. Verifica que el feature de `import-from-local-files.md` cumple sus criterios de aceptación. El validador **solo reporta**, nunca arregla.

## Precondiciones

- El repositorio compila (`go build ./...` sin errores).
- Existe un workspace `.daedalus/` de prueba en un directorio temporal limpio.
- Se dispone de archivos de origen de prueba: (a) una definición de agente canónica local válida; (b) una estructura `.claude/agents/` de ejemplo con al menos un agente (frontmatter Claude Code); (c) un archivo de agente inválido contra el esquema.

## Checks

### Check 1 — Import desde archivo canónico local
- **Comando:** importar el archivo de agente canónico válido y listar `.daedalus/agents/`.
- **Esperado:** aparece su definición canónica bajo `.daedalus/agents/`.

### Check 2 — Import desde `.claude/agents/`
- **Comando:** importar la estructura `.claude/agents/` de ejemplo.
- **Esperado:** los agentes Claude Code (frontmatter) se convierten a definiciones canónicas válidas en `.daedalus/agents/`.

### Check 3 — Origen inválido produce error accionable
- **Comando:** importar el archivo de agente inválido contra el esquema.
- **Esperado:** error accionable (campo, observado, esperado); el agente no se importa en estado inválido sin aviso.

### Check 4 — Import no destructivo
- **Comando:** importar un origen cuyo identificador ya existe en el workspace.
- **Esperado:** no se sobreescribe silenciosamente; se informa el conflicto u ofrece preview/confirmación.

### Check 5 — Identificadores en kebab-case
- **Comando:** inspeccionar los identificadores de los agentes importados.
- **Esperado:** todos en `kebab-case` (normalizados o reportados de forma accionable si el origen no cumplía).

### Check 6 — Determinismo
- **Comando:** importar el mismo origen dos veces en workspaces limpios y comparar el resultado.
- **Esperado:** misma definición canónica en ambos casos.

## Mapeo a criterios

| Check | Criterio(s) |
|---|---|
| Check 1 | CA1 |
| Check 2 | CA2 |
| Check 3 | CA3 |
| Check 4 | CA4 |
| Check 5 | CA5 |
| Check 6 | CA6 |
| Todos | CA7 (trazabilidad RF-2.3) |

## Verdict

**APPROVED / REJECTED** — _(a completar por Yoda al ejecutar)._

Hallazgos (si REJECTED), uno por ítem:
- **Severidad:** _(bloqueante / mayor / menor)_
- **Observado:** _(qué se vio)_
- **Esperado:** _(qué debía verse, con el criterio asociado)_
