# Validación automática — Ticket 01-01

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible en el PATH (o invocable vía el runner del proyecto).
- Existe un directorio temporal vacío que actúa como repo objetivo limpio, sin `.daedalus/` previo.

## Checks

1. **Comando existe** — Comando: `daedalus init --help` (o equivalente de ayuda) · Esperado: el comando `init` está registrado y retorna ayuda sin error.
2. **Crea el directorio raíz** — Comando: ejecutar `daedalus init` en el repo objetivo vacío y verificar la existencia de `.daedalus/` · Esperado: el directorio `.daedalus/` existe.
3. **Crea todas las subcarpetas** — Comando: verificar la existencia de `.daedalus/agents/`, `prompts/`, `workflows/`, `specs/`, `architecture/`, `epics/`, `tickets/`, `docs/`, `.state/` · Esperado: las 9 subcarpetas existen.
4. **Artefactos raíz presentes** — Comando: verificar la existencia de `.daedalus/daedalus.yaml` y `.daedalus/init.md` · Esperado: ambos archivos existen.
5. **No destructivo** — Comando: sembrar el repo objetivo con archivos arbitrarios (p. ej. `README.md`, `src/foo.txt`) antes de `init`, ejecutar `init`, y comparar esos archivos antes/después · Esperado: ningún archivo fuera de `.daedalus/` fue modificado ni eliminado.
6. **Determinismo** — Comando: ejecutar `init` en dos copias idénticas de un repo vacío y comparar el listado recursivo de `.daedalus/` (rutas ordenadas) · Esperado: ambos listados son idénticos.
7. **Mensaje de resultado** — Comando: capturar la salida estándar de `daedalus init` · Esperado: incluye una confirmación de workspace creado y su ruta.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 2, 3 |
| CA2 | 4 |
| CA3 | 5 |
| CA4 | 6 |
| CA5 | 7 |
| CA6 | 3, 4 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
