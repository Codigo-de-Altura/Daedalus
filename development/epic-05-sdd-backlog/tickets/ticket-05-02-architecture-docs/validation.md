# Validación automática — Ticket 05-02

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible (o invocable vía el runner del proyecto).
- Existe un workspace `.daedalus/` inicializado en un repo objetivo de prueba.
- Existe al menos una spec en `.daedalus/specs/` para vincular como origen.

## Checks

1. **Ubicación canónica** — Comando: crear un documento de arquitectura y verificar su ruta · Esperado: `.daedalus/architecture/<slug>.md` con `<slug>` en kebab-case.
2. **Crear/gestionar** — Comando: crear y luego listar/gestionar documentos de arquitectura · Esperado: el documento markdown queda persistido y es gestionable dentro del workspace.
3. **Vínculo a spec de origen** — Comando: inspeccionar un documento de arquitectura vinculado a una spec · Esperado: referencia explícita a la spec de origen (rastro `spec → arquitectura`).
4. **Editable / no destructivo** — Comando: editar manualmente un documento y repetir la operación de gestión · Esperado: la edición manual no es sobrescrita ni borrada (preview/confirmación).
5. **Sin ejecución de agente (Fase 1)** — Comando: ejecutar la operación y observar procesos externos · Esperado: Daedalus **no** lanza ni ejecuta el agente *architect*.
6. **Formato diff-friendly** — Comando: inspeccionar el documento · Esperado: markdown con metadatos estables y reproducibles (diffs limpios).

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1 |
| CA2 | 2 |
| CA3 | 3 |
| CA4 | 4 |
| CA5 | 5 |
| CA6 | 6 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
