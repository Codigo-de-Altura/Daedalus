# Validación — Build idempotente y no destructivo

> La corre **Yoda** (validador backend). Solo reporta; no implementa ni arregla.

---

## Precondiciones

- Binario `daedalus` compilado y disponible.
- Repo de prueba con `.daedalus/` válido (`daedalus.yaml` → Claude Code) y artefactos ya generables por el build.
- Conjunto de **golden files** de referencia para ese workspace (o generables en la primera corrida).
- Herramienta de diff de archivos/directorios (`git diff --no-index`, `fc`, o comparación de hashes).

## Checks numerados

### Check 1 — Idempotencia: dos corridas sin cambios → diff vacío
- **Comando:** Correr `daedalus build`, snapshot del output; correr `daedalus build` otra vez; comparar contra el snapshot.
- **Esperado:** Output **idéntico** entre la primera y la segunda corrida; diff vacío; ninguna escritura espuria.

### Check 2 — Idempotencia repetida (N corridas)
- **Comando:** Correr `daedalus build` tres o más veces seguidas sin tocar la definición canónica; comparar todos los outputs.
- **Esperado:** Todos los outputs idénticos entre sí.

### Check 3 — Determinismo (golden files): mismo input → mismo output
- **Comando:** Correr `daedalus build` sobre el mismo workspace en dos directorios limpios distintos; comparar (`git diff --no-index dirA dirB`).
- **Esperado:** Outputs **idénticos byte-a-byte**; sin diferencias de orden, formato ni datos volátiles.

### Check 4 — Determinismo contra golden de referencia
- **Comando:** Comparar el output del build contra los golden files de referencia.
- **Esperado:** Coincidencia exacta. Cualquier diferencia es un hallazgo.

### Check 5 — Sin datos volátiles
- **Comando:** Inspeccionar los artefactos generados buscando timestamps, rutas absolutas, orden aleatorio de claves u otros datos dependientes del entorno.
- **Esperado:** Ninguno presente; el contenido solo depende de la definición canónica.

### Check 6 — No-destrucción de cambios manuales fuera del área gestionada
- **Comando:** Crear/editar un archivo manual **fuera del área gestionada** por el build (p. ej. un archivo del usuario que el build no produce); correr `daedalus build`; verificar ese archivo.
- **Esperado:** El cambio manual **se preserva intacto** tras la re-compilación.

### Check 7 — El área gestionada se regenera de forma acotada
- **Comando:** Modificar la definición canónica de modo que cambie un artefacto; correr `daedalus build`; revisar qué cambió.
- **Esperado:** Solo cambia el contenido del **área gestionada** afectada; nada fuera de ella se toca; el resultado es reproducible.

### Check 8 — Comportamiento por defecto no destructivo
- **Comando:** Revisar que, por defecto, el build no borra ni sobre-escribe contenido fuera del área gestionada.
- **Esperado:** Default seguro (RNF-8); ninguna destrucción inadvertida.

## Mapeo a criterios

| Check | Criterio de aceptación |
|---|---|
| 1, 2 | Dos corridas sin cambios dejan el output idéntico (idempotente; diff vacío). |
| 6 | Cambio manual fuera del área gestionada sobrevive a la re-compilación. |
| 3, 4 | Mismo input canónico → mismo output byte-a-byte (determinismo / golden files). |
| 5 | El build no escribe datos volátiles. |
| 7 | El área gestionada está delimitada y solo ella se regenera. |
| 8 | Comportamiento por defecto no destructivo. |

## Verdict

**Estado:** _APPROVED_ / _REJECTED_ — _(a completar por Yoda tras ejecutar los checks.)_

### Hallazgos

| # | Severidad | Check | Observado | Esperado |
|---|---|---|---|---|
| | | | | |

> Severidad: `blocker` / `major` / `minor`. Un hallazgo por fila. Si `REJECTED`, el orquestador traslada estos hallazgos a `observations.md`.
