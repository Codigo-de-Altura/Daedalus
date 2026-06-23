# Validación automática — Ticket 09-02

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El proyecto compila y la suite de tests es ejecutable (`go test ./...`).
- Existe(n) input(s) canónico(s) de referencia y sus golden files de compilación versionados en el repositorio.
- Se dispone de un entorno limpio (sin red requerida) para correr los tests.

## Checks

1. **Suite unitaria pasa** — Comando: `go test ./...` · Esperado: la suite ejecuta y pasa; cubre modelo de dominio, validación de esquemas y (de)serialización determinista.
2. **Golden files presentes y comparados** — Comando: inspeccionar el repositorio y correr los tests de compilación · Esperado: existen golden files para al menos un input canónico de referencia y un test compara la salida real de `build`/`sync` contra el golden.
3. **Determinismo de compilación (RNF-5)** — Comando: ejecutar el test que compila dos veces el mismo input y compara las salidas · Esperado: las salidas son idénticas (mismo input → mismo output).
4. **La red de seguridad falla ante cambio no intencional** — Comando: alterar deliberadamente y de forma temporal un golden file (o la salida) y correr el test de golden · Esperado: el test **falla**, demostrando que detecta desviaciones; luego se revierte la alteración.
5. **Regeneración controlada de golden** — Comando: usar el mecanismo de update (p. ej. flag) para regenerar los golden ante un cambio intencional y volver a correr la suite · Esperado: los golden se regeneran de forma revisable y la suite vuelve a pasar.
6. **Reproducibilidad / sin dependencias de entorno** — Comando: correr la suite sin red y observar uso de reloj/rutas · Esperado: los tests pasan sin red y no dependen de timestamps ni rutas absolutas.
7. **Golden/fixtures legibles y ordenados** — Comando: inspeccionar los golden files y fixtures · Esperado: están en texto legible, ordenados y versionados (RNF-6).
8. **Idioma inglés** — Comando: revisar código de test, nombres y fixtures · Esperado: están en inglés.

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
