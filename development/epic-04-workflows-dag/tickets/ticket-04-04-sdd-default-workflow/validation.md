# Validación automática — Ticket 04-04

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado (o el workflow es ejercitable vía la suite de tests del proyecto).
- Está disponible el modelo de carga de workflows (ticket 04-01) y el validador del DAG (ticket 04-03).
- El catálogo/definiciones de agentes built-in está disponible (analyst, architect, planner, validator, documenter).

## Checks

1. **Existencia de fábrica** — Comando: verificar que `sdd-default.yaml` está provisto en el área de workflows del workspace (`.daedalus/workflows/`) · Esperado: el archivo existe.
2. **Carga sin error** — Comando: cargar `sdd-default.yaml` con el modelo del ticket 04-01 · Esperado: carga correctamente, sin error.
3. **Fases del pipeline presentes** — Comando: inspeccionar las fases del workflow cargado · Esperado: contiene spec, architecture, epics, tickets, validation y docs, encadenadas por dependencias en el orden del pipeline SDD.
4. **Agentes correctos** — Comando: revisar el `agent` de cada fase · Esperado: spec→analyst, architecture→architect, epics→planner, tickets→planner, validation→validator, docs→documenter.
5. **Inputs/outputs/gate/depends_on** — Comando: revisar cada fase · Esperado: cada fase declara `inputs`, `outputs` y `gate`, y sus `depends_on` son consistentes con el pipeline.
6. **Pasa la validación del DAG** — Comando: correr el validador del DAG (ticket 04-03) sobre `sdd-default.yaml` · Esperado: resultado **válido**, sin hallazgos (sin ciclos, sin artefactos faltantes, sin agentes inexistentes).
7. **Determinista / git-friendly** — Comando: revisar el archivo · Esperado: claves estables y ordenadas, formato estable, apto para diffs limpios.

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

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
