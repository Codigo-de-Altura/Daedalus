# Validación automática — Ticket 05-01

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible (o invocable vía el runner del proyecto).
- Existe un workspace `.daedalus/` inicializado en un repo objetivo de prueba.
- El catálogo de agentes incluye una definición del agente *analyst* y existe el workflow `sdd-default.yaml`.

## Checks

1. **Captura de brief** — Comando: capturar un brief de prueba mediante la operación de Daedalus · Esperado: queda persistido un artefacto markdown de brief dentro del workspace.
2. **Vínculo al agente analyst** — Comando: inspeccionar la definición/registro del brief y su relación con el paso `brief → spec/PRD` del `sdd-default.yaml` · Esperado: el brief queda asociado a la definición del agente *analyst* (inputs/outputs).
3. **Ubicación canónica de la spec** — Comando: verificar la ruta destino de la spec · Esperado: `.daedalus/specs/<slug>.md` con `<slug>` en kebab-case.
4. **Spec editable / no destructiva** — Comando: editar manualmente una spec materializada y repetir la operación de gestión del brief · Esperado: la edición manual del usuario no es sobrescrita ni borrada (se ofrece preview/confirmación).
5. **Sin ejecución de agente (Fase 1)** — Comando: ejecutar la operación completa y observar procesos/llamadas externas · Esperado: Daedalus **no** lanza ni ejecuta el agente *analyst*; ningún subproceso de agente es invocado.
6. **Formato diff-friendly** — Comando: inspeccionar brief y spec · Esperado: markdown con metadatos estables, ordenados y reproducibles (aptos para diffs limpios).
7. **Trazabilidad brief → spec** — Comando: inspeccionar la spec materializada · Esperado: referencia explícita a su brief de origen.

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
