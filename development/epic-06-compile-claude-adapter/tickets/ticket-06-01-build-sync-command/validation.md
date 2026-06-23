# Validación — `daedalus build` / `sync`

> La corre **Yoda** (validador backend). Solo reporta; no implementa ni arregla.

---

## Precondiciones

- Binario `daedalus` compilado y disponible en el `PATH` (o ruta conocida).
- Un repo de prueba con un workspace `.daedalus/` válido, con `daedalus.yaml` apuntando al backend **Claude Code** y al menos un agente y un comando en la definición canónica.
- Un segundo directorio **sin** `.daedalus/` para los casos negativos.
- Capacidad de inspeccionar el código de salida del proceso (`$LASTEXITCODE` en PowerShell).

## Checks numerados

### Check 1 — El comando existe y muestra ayuda
- **Comando:** `daedalus build --help`
- **Esperado:** Salida de ayuda del comando `build`; código de salida `0`.

### Check 2 — Alias `sync`
- **Comando:** `daedalus sync --help`
- **Esperado:** Ayuda equivalente a `build` (mismo comando vía alias); código de salida `0`.

### Check 3 — Compilación básica exitosa
- **Comando:** `daedalus build` (dentro del repo con `.daedalus/` válido)
- **Esperado:** Compila al formato nativo del backend configurado; muestra resumen (backend, artefactos creados/actualizados, estado); código de salida `0`.

### Check 4 — Workspace ausente
- **Comando:** `daedalus build` (en un directorio **sin** `.daedalus/`)
- **Esperado:** Aborta sin escribir; mensaje de error claro indicando que no hay workspace; código de salida distinto de `0`.

### Check 5 — Definición canónica inválida aborta sin escribir
- **Comando:** Introducir un error en la definición canónica (p. ej. agente sin campo requerido) y correr `daedalus build`.
- **Esperado:** Reporta el/los error(es) de validación; **no escribe artefactos nativos**; código de salida de error de validación (distinto del de escritura).

### Check 6 — Backend sin adaptador
- **Comando:** Configurar en `daedalus.yaml` un backend sin adaptador registrado y correr `daedalus build`.
- **Esperado:** Error claro de "adaptador no encontrado"; ninguna escritura; código de salida distinto de `0`.

### Check 7 — Selección de backend desde el manifiesto
- **Comando:** `daedalus build` con `daedalus.yaml` configurado a Claude Code.
- **Esperado:** Enruta al adaptador Claude Code y genera artefactos bajo el área esperada (`.claude/`); resumen menciona el backend Claude Code.

### Check 8 — Resumen y códigos de salida diferenciados
- **Comando:** Correr los escenarios de éxito, error de validación y error de escritura, e inspeccionar el código de salida en cada caso.
- **Esperado:** `0` en éxito; códigos distintos entre sí para error de validación y error de compilación/escritura.

## Mapeo a criterios

| Check | Criterio de aceptación |
|---|---|
| 1, 3 | `daedalus build` compila la definición canónica al formato nativo. |
| 2 | `daedalus sync` es alias de `build`. |
| 4, 5 | Workspace ausente/inválido aborta sin escribir, con error accionable. |
| 3, 7 | Backend objetivo se resuelve desde `daedalus.yaml` y se enruta al adaptador correcto. |
| 6 | Backend sin adaptador → error claro y ninguna escritura. |
| 3 | Resumen con backend, artefactos y estado. |
| 8 | Códigos de salida distinguen éxito, error de validación y error de compilación/escritura. |

## Verdict

**Estado:** _APPROVED_ / _REJECTED_ — _(a completar por Yoda tras ejecutar los checks.)_

### Hallazgos

| # | Severidad | Check | Observado | Esperado |
|---|---|---|---|---|
| | | | | |

> Severidad: `blocker` / `major` / `minor`. Un hallazgo por fila. Si `REJECTED`, el orquestador traslada estos hallazgos a `observations.md`.
