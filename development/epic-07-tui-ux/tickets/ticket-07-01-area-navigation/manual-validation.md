# Validación manual — Navegación por áreas

> Para alguien **sin experiencia en testing**. Seguí los pasos tal cual. Si algo no coincide con lo esperado, no intentes arreglarlo: avisá al **orquestador** (Workflow B).

## Preparación

1. Abrí una terminal en el repositorio del proyecto.
2. Arrancá la TUI de Daedalus (el orquestador te dirá el comando exacto si no lo sabés).
3. Esperá a ver la **pantalla inicial** (la raíz).

## Casos

### Caso 1 — Ver todas las áreas
- **Hacé:** mirá la pantalla inicial.
- **Esperá ver:** una lista con seis áreas: init, agentes, prompts, workflows, backlog y build.

### Caso 2 — Entrar y volver de cada área
- **Hacé:** entrá a "init", luego volvé a la pantalla inicial. Repetí lo mismo con agentes, prompts, workflows, backlog y build.
- **Esperá ver:** cada área se abre al entrar, y siempre podés volver a la pantalla inicial con la misma tecla.

### Caso 3 — Nunca quedar atrapado
- **Hacé:** entrá a un área, metete en cualquier sub-pantalla que ofrezca, y andá usando "volver" varias veces.
- **Esperá ver:** cada vez que pedís volver, retrocedés un paso, hasta llegar de nuevo a la pantalla inicial. En ningún momento te quedás sin forma de salir.

### Caso 4 — Saber dónde estás
- **Hacé:** entrá a un área cualquiera.
- **Esperá ver:** algún indicador (resaltado, título o "miga de pan") que te diga en qué área estás.

### Caso 5 — Áreas sin datos o con problemas
- **Hacé:** entrá a un área que todavía no tenga contenido.
- **Esperá ver:** un mensaje claro de "vacío" (o de carga, o de error si algo falla), y de todas formas podés volver con la tecla de retroceso.

## Si algo no coincide

Si cualquier caso no se comporta como dice "Esperá ver", reportalo al **orquestador** (Workflow B) describiendo qué hiciste y qué viste.
