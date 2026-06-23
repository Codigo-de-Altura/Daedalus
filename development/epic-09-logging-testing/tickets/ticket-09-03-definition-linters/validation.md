# Validación automática — Ticket 09-03

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible, y la suite de tests es ejecutable (`go test ./...`).
- Existen fixtures de definiciones **válidas** e **inválidas** para cada familia: agentes, workflows (DAG) y manifiesto (`daedalus.yaml`), incluyendo casos de ciclo, artefacto faltante, agente inexistente, ids duplicados y manifiesto malformado.

## Checks

1. **Agente inválido detectado** — Comando: correr el linter de definiciones sobre un agente con campo requerido faltante o tipo incorrecto · Esperado: se reporta el hallazgo con archivo, campo y qué se esperaba vs. qué se encontró.
2. **Ciclo en el DAG detectado** — Comando: correr el linter sobre un workflow con un ciclo · Esperado: se reporta el ciclo de forma accionable (sin panic).
3. **Artefacto faltante / agente inexistente** — Comando: correr el linter sobre un workflow con un input no producido por ninguna fase y otro con referencia a un agente inexistente · Esperado: ambos casos se detectan y reportan con ubicación.
4. **Ids de fase duplicados / dependencias malformadas** — Comando: correr el linter sobre un workflow con ids de fase duplicados o `depends_on` malformado · Esperado: se detecta y reporta.
5. **Manifiesto inválido detectado** — Comando: correr el linter sobre un `daedalus.yaml` con campo requerido faltante/malformado o backend desconocido · Esperado: se detecta y reporta de forma accionable.
6. **Sin falsos positivos en definiciones válidas** — Comando: correr los tres linters sobre fixtures válidos de cada familia · Esperado: pasan sin hallazgos.
7. **Sin panic ante entradas malformadas** — Comando: correr los linters sobre entradas severamente malformadas · Esperado: errores controlados, sin panic.
8. **Determinismo, orden estable y agnosticismo de backend** — Comando: correr un linter dos veces sobre el mismo input inválido y comparar los hallazgos; revisar la capa de validación · Esperado: mismo conjunto de hallazgos en orden estable, mensajes en inglés y sin referencias a un backend concreto.
9. **Tests unitarios** — Comando: `go test ./...` · Esperado: pasan, incluyendo los tests de los linters de definiciones.

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
