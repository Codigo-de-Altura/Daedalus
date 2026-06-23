# Validación — Atajos de teclado y ayuda contextual

> La corre **Leia** (validadora frontend, abogada del usuario). Solo reporta hallazgos; nunca implementa ni arregla.

## Precondiciones

- El binario compila y la TUI arranca.
- Se pueden alcanzar las distintas áreas y al menos un formulario (depende de ticket-07-01 y ticket-07-02).

## Checks

| # | Comando / Acción | Esperado |
|---|---|---|
| 1 | En la raíz, abrir la ayuda con el atajo dedicado. | Se muestra la ayuda con los atajos disponibles en ese contexto. |
| 2 | Entrar a un área y abrir la ayuda con el mismo atajo. | La ayuda se abre con la misma tecla y muestra los atajos del contexto actual. |
| 3 | **Consistencia de atajos:** comparar la tecla de "volver" (y de "salir", "raíz", "mover", "entrar") en las seis áreas. | Cada acción usa la misma tecla en todas las áreas; no hay divergencias ni colisiones. |
| 4 | **Ayuda contextual:** abrir ayuda en un formulario. | La ayuda lista los atajos del formulario (enviar/cancelar/mover entre campos), no solo los de navegación general. |
| 5 | Verificar que un atajo anunciado por la ayuda efectivamente ejecuta su acción. | La acción ocurre exactamente como la ayuda la describe (anunciado == real). |
| 6 | Buscar acciones equivalentes con teclas distintas según el área. | No existen; una acción equivalente nunca cambia de tecla entre contextos. |
| 7 | Verificar barra de ayuda breve siempre visible y/o vista expandida. | Hay una indicación de atajos accesible sin memorizar; la expandida da el listado completo del contexto. |
| 8 | **Estados loading/empty/error:** abrir ayuda en cada uno. | El atajo de ayuda y el de volver siguen disponibles y anunciados. |

## Mapeo a criterios

| Criterio de aceptación | Checks |
|---|---|
| Registro central con acciones comunes/navegación | 1, 3 |
| Misma acción = misma tecla en todas partes | 3, 6 |
| Ayuda contextual lista atajos del contexto | 1, 2, 4, 7 |
| Ayuda accesible desde cualquier área con atajo consistente | 1, 2, 8 |
| Anunciado coincide con lo que funciona | 5 |
| Trazabilidad a RF-7.3 | 1–8 |

## Verdict

**APPROVED / REJECTED**

Hallazgos (uno por ítem):

- **[severidad: blocker/major/minor]** Observado: `<qué se vio>`. Esperado: `<qué debía pasar>`.
