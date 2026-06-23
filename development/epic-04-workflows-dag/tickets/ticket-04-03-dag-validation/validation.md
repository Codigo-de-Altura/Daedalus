# Validación automática — Ticket 04-03

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado (o el validador es ejercitable vía la suite de tests del proyecto).
- Existen workflows de prueba que cubren: (a) un workflow correcto; (b) uno con ciclo; (c) uno con artefacto faltante; (d) uno con agente inexistente; (e) casos degenerados (vacío, fases sin dependencias).
- El catálogo/definiciones de agentes del workspace está disponible para resolver la existencia de agentes.

## Checks

1. **Detección de ciclos** — Comando: validar el workflow con un ciclo en sus dependencias · Esperado: resultado inválido, con un hallazgo que identifica las fases que forman el ciclo.
2. **Artefacto faltante** — Comando: validar el workflow donde una fase consume un `input` no producido por ninguna fase predecesora ni inicial · Esperado: resultado inválido, hallazgo que identifica la fase y el artefacto faltante.
3. **Agente inexistente** — Comando: validar el workflow que referencia un `agent` ausente del catálogo · Esperado: resultado inválido, hallazgo que identifica la fase y el agente inexistente.
4. **Workflow correcto** — Comando: validar el workflow correcto · Esperado: resultado **válido**, sin hallazgos.
5. **Hallazgos accionables** — Comando: inspeccionar los hallazgos de los checks 1-3 · Esperado: cada hallazgo incluye fase afectada, tipo de problema, valor observado y motivo, en lenguaje claro.
6. **Determinismo** — Comando: validar dos veces el mismo workflow inválido y comparar resultado y orden de hallazgos · Esperado: idénticos en ambas corridas.
7. **Entradas degeneradas sin panic** — Comando: validar un workflow vacío y uno con fases sin dependencias · Esperado: se manejan como casos definidos, sin panic.
8. **Agnóstico de backend** — Comando: revisar el validador · Esperado: opera sobre el modelo canónico, sin lógica específica de Claude Code.

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
