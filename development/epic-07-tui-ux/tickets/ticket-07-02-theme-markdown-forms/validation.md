# Validación — Estética, render de markdown y formularios

> La corre **Leia** (validadora frontend, abogada del usuario). Solo reporta hallazgos; nunca implementa ni arregla.

## Precondiciones

- El binario compila y la TUI arranca.
- Hay al menos un documento markdown alcanzable (p. ej. una spec, un prompt o documentación) para probar el render.
- Hay al menos un flujo con formulario alcanzable (p. ej. captura de un campo).

## Checks

| # | Comando / Acción | Esperado |
|---|---|---|
| 1 | Recorrer varias áreas y observar estilos (encabezados, selección activa, bordes). | El lenguaje visual es coherente entre áreas; no hay estilos disonantes ni colores ad-hoc. |
| 2 | Abrir un documento markdown en la TUI. | Se renderiza legible: encabezados, listas, énfasis se ven con formato (no como texto plano crudo). |
| 3 | Abrir un markdown con **tabla** y **bloque de código**. | La tabla y el código se renderizan correctamente y acordes al tema. |
| 4 | Abrir un formulario y enviarlo con datos válidos. | El formulario acepta el envío y continúa el flujo. |
| 5 | **Validación de entrada:** enviar un formulario con un campo inválido/vacío requerido. | Se muestra un mensaje de error claro; el envío no procede hasta corregir. |
| 6 | Cancelar un formulario a mitad de carga. | Se cancela y se regresa sin romper la navegación (sin dead end). |
| 7 | **Estado loading:** observar una vista mientras carga. | El indicador de carga sigue el tema. |
| 8 | **Estado empty:** observar una vista sin datos. | El estado vacío es claro y sigue el tema. |
| 9 | **Estado error:** observar una vista en error. | El estado de error es legible y sigue el tema. |
| 10 | **Consistencia de atajos / ayuda contextual:** dentro del formulario, verificar que los atajos de envío/cancelación están indicados. | El usuario puede inferir cómo enviar/cancelar desde la propia UI. |

## Mapeo a criterios

| Criterio de aceptación | Checks |
|---|---|
| Tema central aplicado consistentemente | 1, 7, 8, 9 |
| Markdown se renderiza correctamente | 2, 3 |
| Formularios reutilizables capturan entrada | 4 |
| Formularios validan y muestran errores | 5 |
| Enviar/cancelar sin romper navegación | 6, 10 |
| Estados loading/empty/error con lenguaje visual del tema | 7, 8, 9 |
| Trazabilidad a RF-7.2 / RNF-4 | 1–10 |

## Verdict

**APPROVED / REJECTED**

Hallazgos (uno por ítem):

- **[severidad: blocker/major/minor]** Observado: `<qué se vio>`. Esperado: `<qué debía pasar>`.
