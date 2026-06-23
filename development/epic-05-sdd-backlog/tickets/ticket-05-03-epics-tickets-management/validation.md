# Validación automática — Ticket 05-03

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible (o invocable vía el runner del proyecto).
- Existe un workspace `.daedalus/` inicializado en un repo objetivo de prueba.
- Existe al menos una spec y/o documento de arquitectura para vincular como origen.

## Checks

1. **Crear epic** — Comando: crear un epic y verificar su carpeta · Esperado: `epic-NN-<slug>` en kebab-case según CLAUDE.md §6.
2. **Crear ticket** — Comando: crear un ticket bajo el epic y verificar su carpeta · Esperado: `ticket-NN-MM-<slug>` (NN=epic, MM=secuencia) en kebab-case.
3. **Metadatos presentes** — Comando: inspeccionar los metadatos de un epic y de un ticket · Esperado: incluyen **estado**, **prioridad**, **dependencias** y **links a artefactos**.
4. **Dependencias explícitas** — Comando: declarar una dependencia entre dos tickets/epics e inspeccionarla · Esperado: la dependencia queda representada de forma explícita y consistente.
5. **Links de origen / trazabilidad** — Comando: inspeccionar ticket y epic · Esperado: el ticket referencia su epic; el epic referencia su spec/arquitectura de origen.
6. **Metadatos consistentes y diff-friendly** — Comando: inspeccionar el formato de los metadatos · Esperado: valores estables para estado/prioridad, claves ordenadas y reproducibles (diffs limpios).
7. **Sin ejecución de agente (Fase 1)** — Comando: ejecutar las operaciones y observar procesos externos · Esperado: Daedalus **no** lanza el agente *planner* ni ninguna implementación.
8. **No destructivo** — Comando: editar manualmente un epic/ticket y repetir la operación de gestión · Esperado: las ediciones del usuario no son sobrescritas (preview/confirmación).

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
| CA8 | 8 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
