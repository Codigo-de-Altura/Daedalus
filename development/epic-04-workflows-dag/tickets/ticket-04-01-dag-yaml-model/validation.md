# Validación automática — Ticket 04-01

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible (o el modelo es ejercitable vía la suite de tests del proyecto).
- Existe un archivo YAML de workflow de ejemplo, válido, con fases que cubren `{ id, agent, inputs, outputs, gate, depends_on }` (puede usarse uno mínimo de 2-3 fases con dependencias).
- Existe un archivo YAML de workflow malformado/ inválido para los checks negativos.

## Checks

1. **Carga de workflow válido** — Comando: cargar el YAML de workflow de ejemplo en el modelo de dominio · Esperado: la carga tiene éxito y el modelo contiene todas las fases con sus campos `{ id, agent, inputs, outputs, gate, depends_on }`.
2. **Round-trip sin pérdida** — Comando: cargar el workflow y volver a serializarlo a YAML; comparar el contenido semántico (fases, agentes, inputs, outputs, gates, dependencias) con el original · Esperado: no hay pérdida de información entre el original y el reserializado.
3. **Determinismo de la serialización** — Comando: serializar el mismo modelo dos veces y comparar los bytes resultantes · Esperado: ambas salidas son idénticas, con claves ordenadas de forma estable.
4. **Aristas del DAG recuperables** — Comando: inspeccionar las dependencias (`depends_on`) de las fases cargadas · Esperado: el conjunto de aristas del DAG coincide con las dependencias declaradas en el YAML.
5. **Edición del modelo** — Comando: crear un workflow nuevo y añadir, editar y eliminar fases; luego serializar · Esperado: el YAML resultante es válido y refleja los cambios aplicados.
6. **Error en YAML malformado** — Comando: intentar cargar el YAML malformado/ inválido · Esperado: retorna un error claro y accionable (sin panic) que señala el campo/fase problemático.
7. **Agnóstico de backend** — Comando: revisar el modelo y su (de)serialización · Esperado: no hay referencias a Claude Code ni a ningún backend/runtime concreto.

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
