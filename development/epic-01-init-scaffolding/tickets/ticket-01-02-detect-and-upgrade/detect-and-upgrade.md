# Ticket 01-02 — Detección de `.daedalus/` existente y upgrade/merge no destructivo

> **Epic:** epic-01-init-scaffolding · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-1.2 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

El workspace `.daedalus/` vive versionado en git y es editado por el equipo a lo largo del tiempo. Re-ejecutar `daedalus init` sobre un repo que **ya** tiene un workspace no debe destruir trabajo manual (agentes, prompts, workflows, backlog que el equipo escribió o ajustó). Por seguridad (RNF-8) y por la convención de idempotencia (init.md §7), la herramienta debe **detectar** un `.daedalus/` preexistente y, en lugar de recrearlo ciegamente, ofrecer un **upgrade/merge no destructivo** con **preview** antes de escribir.

Este ticket cubre la rama del comando `init` que se activa cuando `.daedalus/` ya existe. La creación inicial (cuando no existe) es del ticket 01-01.

## Feature / Qué se construye

La lógica de **detección + upgrade no destructivo** dentro de `daedalus init`:

- Al ejecutar `init`, la herramienta detecta si ya existe un `.daedalus/` en el repo objetivo.
- Si existe, **no** sobrescribe ciegamente: calcula qué le faltaría a la estructura para estar completa según `init.md` §4.2 (carpetas o artefactos raíz ausentes) y presenta un **preview/diff** de los cambios propuestos.
- El merge es **no destructivo**: los archivos y carpetas existentes con contenido manual **no se sobrescriben ni se borran**; solo se **agregan** los faltantes para completar/actualizar la estructura.
- El usuario obtiene visibilidad de qué cambiaría antes de que se escriba nada (preview), coherente con el principio "preview/confirm" del PRD (RNF-8) e idempotencia (init.md §7).

## Requerimientos

- R1. `daedalus init` detecta de forma fiable si `.daedalus/` ya existe en el repo objetivo.
- R2. Cuando existe, la operación entra en modo **upgrade/merge** en lugar de creación desde cero.
- R3. El modo upgrade calcula la diferencia entre la estructura actual y la estructura canónica esperada (init.md §4.2): carpetas y artefactos raíz **faltantes**.
- R4. El modo upgrade es **no destructivo**: archivos y carpetas existentes (incluido contenido editado manualmente) **no** se sobrescriben ni se eliminan.
- R5. Antes de escribir cambios, se presenta un **preview** legible de lo que se agregaría (qué carpetas/artefactos faltantes se crearían). Nada fuera de ese conjunto se toca.
- R6. Si la estructura existente ya está completa, el upgrade no aplica cambios (idempotencia: re-ejecutar no produce diffs).
- R7. El reporte al usuario distingue claramente "creado desde cero" (01-01) de "upgrade sobre workspace existente" (este ticket).
- R8. Comportamiento determinista y portable (Windows/macOS/Linux).

## Criterios de aceptación

- [ ] CA1. Ejecutar `init` sobre un repo con `.daedalus/` preexistente NO recrea ni sobrescribe los archivos existentes.
- [ ] CA2. Un archivo editado manualmente dentro de `.daedalus/` (p. ej. un cambio en `init.md`) conserva su contenido tras re-ejecutar `init`.
- [ ] CA3. Si falta una subcarpeta o artefacto raíz de la estructura canónica, el upgrade la agrega sin tocar lo demás.
- [ ] CA4. Antes de escribir, la herramienta muestra un preview de los cambios propuestos (faltantes a crear).
- [ ] CA5. Re-ejecutar `init` sobre un workspace ya completo no produce cambios (idempotente; preview vacío o sin acciones).
- [ ] CA6. El mensaje de resultado indica que se trató de un upgrade sobre un workspace existente.

## Fuera de alcance

- Creación inicial del workspace cuando no existe (ticket 01-01).
- Generación del contenido determinista de `daedalus.yaml` e `init.md` (ticket 01-03).
- Selección/registro de backend (ticket 01-04).
- Migración de versiones de **esquema** del manifiesto entre releases de Daedalus (queda como decisión abierta del PRD §15; aquí el upgrade es estructural/no destructivo, no migración semántica de esquema).
- Merge de tres vías con resolución de conflictos de contenido dentro de archivos existentes.

## Referencias

- PRD.md — RF-1.2, RNF-8 (no destructivo, preview/confirm), §15 (decisión abierta: estrategia de upgrade/migración).
- init.md — §4.2 (estructura canónica), §7 (idempotencia: no destruir cambios manuales, ofrecer preview/diff).
- epic-01-init-scaffolding/epic.md — criterio: "re-ejecutar init no destruye cambios manuales y ofrece preview/merge".
- Ticket 01-01 — creación inicial del workspace.
