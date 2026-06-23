# Validación automática — Ticket 05-04

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible (o invocable vía el runner del proyecto).
- Existe un workspace `.daedalus/` con al menos una spec, un documento de arquitectura, un epic y un ticket vinculados entre sí.

## Checks

1. **Navegación descendente** — Comando: partir de una spec y recorrer hacia sus epics y tickets · Esperado: se alcanzan los epics y tickets asociados a esa spec.
2. **Navegación ascendente** — Comando: partir de un ticket y remontar a su epic y a su spec/arquitectura · Esperado: se alcanza el epic y la spec/arquitectura de origen.
3. **Verificación de links existentes** — Comando: ejecutar la verificación de trazabilidad sobre un workspace consistente · Esperado: confirma que todo ticket referencia un epic existente y todo epic una spec/arquitectura existente; sin inconsistencias.
4. **Detección de inconsistencias** — Comando: sembrar un link roto / un ticket huérfano / una referencia inexistente y ejecutar la verificación · Esperado: cada inconsistencia es detectada e informada (link roto, referencia inexistente, ticket/epic huérfano).
5. **Reuso de links (sin duplicar verdad)** — Comando: inspeccionar de dónde toma la trazabilidad sus relaciones · Esperado: se apoya en los links ya registrados en epics/tickets/arquitectura, sin duplicar la fuente de verdad.
6. **Sin ejecución de agente / determinismo** — Comando: ejecutar la verificación dos veces sobre el mismo workspace y observar procesos externos · Esperado: Daedalus **no** lanza agentes; el resultado es idéntico en ambas corridas (determinista).

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
