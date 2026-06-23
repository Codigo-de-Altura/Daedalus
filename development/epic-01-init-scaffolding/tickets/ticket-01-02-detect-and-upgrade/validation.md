# Validación automática — Ticket 01-02

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible.
- Disponibilidad de directorios temporales para preparar repos objetivo en distintos estados (workspace completo, workspace con carpeta faltante, archivo editado manualmente).
- El ticket 01-01 (creación inicial) está implementado, ya que este ticket opera sobre un `.daedalus/` previamente creado.

## Checks

1. **Detección de workspace existente** — Comando: crear `.daedalus/` con `init`, luego ejecutar `daedalus init` de nuevo y capturar salida · Esperado: la herramienta reporta que es un upgrade sobre un workspace existente (no una creación desde cero).
2. **No sobrescribe contenido manual** — Comando: editar `.daedalus/init.md` con una marca conocida (p. ej. `MANUAL-EDIT-123`), ejecutar `init`, releer el archivo · Esperado: la marca `MANUAL-EDIT-123` sigue presente intacta.
3. **No elimina archivos existentes** — Comando: agregar un archivo manual dentro de `.daedalus/agents/` (p. ej. `mi-agente.yaml`), ejecutar `init`, verificar su existencia · Esperado: el archivo sigue presente.
4. **Completa faltantes** — Comando: borrar una subcarpeta canónica (p. ej. `.daedalus/docs/`), ejecutar `init`, verificar · Esperado: la subcarpeta faltante se recrea, y el resto del workspace queda intacto.
5. **Preview antes de escribir** — Comando: con una carpeta faltante, ejecutar `init` (en modo preview o capturando su salida previa a la escritura) · Esperado: se muestra un preview que lista los faltantes a crear antes de aplicarlos.
6. **Idempotencia** — Comando: sobre un workspace completo, ejecutar `init` dos veces y comparar el árbol de `.daedalus/` antes/después · Esperado: sin cambios; preview vacío o sin acciones en la segunda corrida.
7. **Distinción del mensaje** — Comando: comparar la salida de `init` en repo nuevo vs. repo con workspace existente · Esperado: los mensajes distinguen "creado" de "upgrade".

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 2, 3 |
| CA2 | 2 |
| CA3 | 4 |
| CA4 | 5 |
| CA5 | 6 |
| CA6 | 1, 7 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
