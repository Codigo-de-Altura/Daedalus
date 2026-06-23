# Validación — Prompt Composition

> La corre **Yoda** (validador backend). Solo reporta; no implementa ni arregla.

## Precondiciones

- Core de Daedalus compilable (`go build ./...` sin errores).
- Feature de prompts base (ticket-03-01) disponible para crear prompts de prueba.
- Workspace `.daedalus/prompts/` con fragmentos de prueba (uno que incluye a otro, un caso recursivo y un caso con ciclo).
- Suite de tests del core ejecutable.

## Checks

### Check 1 — Compila el core
- **Comando:** `go build ./...`
- **Esperado:** compila sin errores.

### Check 2 — Inclusión simple
- **Comando:** resolver un prompt A que incluye un fragmento B.
- **Esperado:** el texto compuesto contiene el contenido de B expandido en la posición de la referencia.

### Check 3 — Inclusión recursiva
- **Comando:** resolver un prompt A que incluye B, donde B incluye C.
- **Esperado:** el texto final contiene el contenido de C resuelto a través de B.

### Check 4 — Determinismo
- **Comando:** resolver el mismo prompt dos veces y comparar la salida.
- **Esperado:** textos byte-idénticos.

### Check 5 — Detección de ciclo
- **Comando:** resolver un prompt con un ciclo de inclusión (A→B→A).
- **Esperado:** error explícito de ciclo; el proceso termina sin colgarse ni desbordar la pila.

### Check 6 — Referencia inexistente
- **Comando:** resolver un prompt que referencia un `id` que no existe.
- **Esperado:** error explícito que nombra el id faltante.

### Check 7 — DRY: un solo archivo fuente
- **Comando:** crear dos prompts que referencian el mismo fragmento y revisar `.daedalus/prompts/`.
- **Esperado:** el fragmento compartido existe como un único archivo; no hay copias duplicadas en disco.

### Check 8 — No mutación de fuentes
- **Comando:** comparar los archivos fuente de los prompts antes/después de resolver.
- **Esperado:** los archivos fuente quedan byte-idénticos (la composición no reescribe el origen).

### Check 9 — Suite de tests
- **Comando:** `go test ./...`
- **Esperado:** todos los tests del área de composición pasan.

## Mapeo a criterios

| Check | Criterio de aceptación |
|---|---|
| 2 | Inclusión simple expande el fragmento |
| 3 | Inclusión recursiva |
| 4 | Determinismo |
| 5 | Detección de ciclos sin colgarse |
| 6 | Referencia inexistente falla con error explícito |
| 7 | DRY: fragmento en un solo archivo |
| 8 | No mutación de archivos fuente |
| 1, 9 | Build y tests verdes |

## Verdict

**Estado:** _APPROVED / REJECTED_ (a completar por Yoda al ejecutar).

**Hallazgos** (uno por ítem):

| # | Severidad | Observado | Esperado |
|---|---|---|---|
| | | | |
