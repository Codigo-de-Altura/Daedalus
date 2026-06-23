# Validación — Operación fluida y de bajo consumo

> La corre **Leia** (validadora frontend, abogada del usuario). Solo reporta hallazgos; nunca implementa ni arregla.

## Precondiciones

- El binario compila y la TUI arranca.
- Las áreas, el render de markdown y los formularios son alcanzables (tickets 07-01/02/03).
- Se dispone de un documento markdown extenso para probar render/scroll y de un área cuya carga involucre trabajo del core.

## Checks

| # | Comando / Acción | Esperado |
|---|---|---|
| 1 | Navegar rápidamente entre las seis áreas con el teclado. | La respuesta es inmediata; sin congelamientos ni lag perceptible. |
| 2 | Abrir y scrollear un documento markdown extenso. | El scroll y el render responden con fluidez; no se traba el input. |
| 3 | **Sin bloqueos:** entrar a un área cuya carga sea lenta y, mientras carga, intentar volver/cancelar. | El input sigue atendido: se ve estado de loading y se puede volver/cancelar; la TUI no se congela. |
| 4 | **Consumo en reposo:** dejar la TUI quieta (sin input) y observar uso de CPU (p. ej. con el monitor del sistema). | El uso de CPU en reposo es despreciable (sin redibujado/polling constante). |
| 5 | **Arranque:** medir el tiempo desde el lanzamiento hasta la pantalla raíz interactiva. | Arranque rápido, consistente con el objetivo de producto (RNF-1, ~< 200 ms objetivo). |
| 6 | **Memoria:** navegar repetidamente entre áreas (decenas de veces) y observar memoria. | La memoria se mantiene acotada; no crece de forma descontrolada. |
| 7 | **Degradación:** tras uso prolongado, repetir el check 1. | La respuesta sigue siendo fluida; no hay enlentecimiento con el uso. |
| 8 | **Estados loading/empty/error:** verificar que aparecen sin trabar el input. | Los estados se muestran de forma asíncrona; el usuario nunca queda bloqueado. |

## Mapeo a criterios

| Criterio de aceptación | Checks |
|---|---|
| Navegación/formularios fluidos sin congelamientos | 1, 2 |
| Operaciones lentas no bloquean; loading + volver/cancelar | 3, 8 |
| Sin consumo apreciable de CPU en reposo | 4 |
| Arranque rápido (RNF-1) | 5 |
| Memoria acotada tras navegación repetida | 6 |
| Sin degradación con uso prolongado | 7 |
| Trazabilidad a RF-7.4 / RNF-2 | 1–8 |

## Verdict

**APPROVED / REJECTED**

Hallazgos (uno por ítem):

- **[severidad: blocker/major/minor]** Observado: `<qué se vio>`. Esperado: `<qué debía pasar>`.
