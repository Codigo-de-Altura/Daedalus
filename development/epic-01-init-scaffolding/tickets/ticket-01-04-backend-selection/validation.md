# Validación automática — Ticket 01-04

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible.
- Directorio temporal vacío como repo objetivo.
- El ticket 01-03 (manifiesto con campo `backends`) está implementado.

## Checks

1. **Default registra Claude Code** — Comando: ejecutar `daedalus init` sin elección explícita (modo default/no interactivo) y leer el campo `backends` del manifiesto · Esperado: contiene Claude Code.
2. **Selección explícita de Claude Code** — Comando: ejecutar `init` eligiendo explícitamente Claude Code (flag/opción) y leer `backends` · Esperado: registrado Claude Code.
3. **Backend no soportado rechazado** — Comando: ejecutar `init` solicitando un backend inexistente (p. ej. `--backend foo`) · Esperado: error claro; el manifiesto no contiene un valor inválido (o no se escribe).
4. **Forma multi-backend** — Comando: inspeccionar la estructura del campo `backends` en el manifiesto · Esperado: forma que admite uno o más backends (p. ej. lista).
5. **Persistencia en el manifiesto** — Comando: verificar que el valor de backend vive en `.daedalus/daedalus.yaml` (no en otro archivo) · Esperado: registrado en el manifiesto del ticket 01-03.
6. **Determinismo** — Comando: ejecutar `init` con la misma elección en dos repos limpios y comparar el campo `backends` · Esperado: valor idéntico en ambos.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 2, 5 |
| CA2 | 1 |
| CA3 | 2 |
| CA4 | 3 |
| CA5 | 4 |
| CA6 | 6 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
