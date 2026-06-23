# Validación — Navegación por áreas

> La corre **Leia** (validadora frontend, abogada del usuario). Solo reporta hallazgos; nunca implementa ni arregla.

## Precondiciones

- El binario de Daedalus compila y la TUI arranca.
- Existe un workspace `.daedalus/` (o se puede arrancar la TUI en estado inicial sin él) que permita alcanzar las seis áreas.
- Se valida desde la pantalla raíz de la TUI.

## Checks

| # | Comando / Acción | Esperado |
|---|---|---|
| 1 | Arrancar la TUI. | Aparece la pantalla raíz listando las seis áreas: init, agentes, prompts, workflows, backlog, build. |
| 2 | Entrar a **init** desde la raíz. | La TUI muestra el área init; el área activa es identificable. |
| 3 | Volver a la raíz desde init con el atajo de retorno. | Se regresa a la pantalla raíz sin error. |
| 4 | Repetir entrada/salida para **agentes, prompts, workflows, backlog, build**. | Cada área se alcanza y se abandona con los mismos atajos; ninguna queda inaccesible. |
| 5 | Dentro de un área con sub-pantallas, navegar hacia adentro y luego usar "volver" repetidamente. | Cada paso de retorno funciona hasta llegar a la raíz; no hay dead ends. |
| 6 | **Consistencia de atajos:** verificar que entrar/volver/salir usan las mismas teclas en las seis áreas. | El comportamiento de los atajos es idéntico en todas las áreas. |
| 7 | **Estado loading:** entrar a un área cuya carga de datos esté en curso. | Se muestra un estado de carga; el atajo de volver sigue disponible. |
| 8 | **Estado empty:** entrar a un área sin datos (p. ej. sin agentes/prompts). | Se muestra un estado vacío claro; se puede volver. |
| 9 | **Estado error:** forzar/observar un área que falle al cargar. | Se muestra un estado de error legible; se puede volver sin que la TUI se bloquee. |
| 10 | **Ayuda contextual:** verificar que el área activa expone cómo navegar (al menos indicador del área y de la acción de volver). | El usuario puede inferir cómo entrar/volver desde la propia UI. |

## Mapeo a criterios

| Criterio de aceptación | Checks |
|---|---|
| Pantalla raíz lista las seis áreas | 1 |
| Se entra a cada área desde la raíz | 2, 4 |
| Se vuelve con atajos consistentes | 3, 4, 6 |
| No hay dead ends | 5 |
| Área activa identificable | 2, 10 |
| Estados loading/empty/error no bloquean navegación | 7, 8, 9 |
| Trazabilidad a RF-7.1 | 1–10 |

## Verdict

**APPROVED / REJECTED**

Hallazgos (uno por ítem):

- **[severidad: blocker/major/minor]** Observado: `<qué se vio>`. Esperado: `<qué debía pasar>`.
