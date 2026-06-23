# Convenciones de equipo

> **Epic:** epic-08-state-collab · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-8.3 · **Estilo:** SDD

---

## Contexto

Daedalus está pensado para **equipos** (PRD §3, D7): varias personas comparten y versionan la misma estructura de IA. Para que esa colaboración sea consistente, las **convenciones** —naming de archivos/ids, estructura de carpetas del workspace, formato de los artefactos— no pueden ser tribales ni implícitas: deben estar **explicitadas** y, sobre todo, ser **validables** de forma automática, para que cualquier desviación se detecte temprano y no contamine el repo compartido.

`init.md` §7 ya enuncia las convenciones (kebab-case, claves YAML ordenadas, trazabilidad, etc.) y CLAUDE.md §6 fija las de la estructura `development/`. Este ticket convierte ese enunciado en una **convención canónica explícita y validable** para el workspace `.daedalus/`.

Este es el ticket **de cara al equipo**: por eso incluye `documentation.md` (guía de uso para el usuario final del producto).

## Feature / Qué se construye

La definición —como **plano**— de las **convenciones de equipo de Daedalus** y de su **mecanismo de validación**. No es una guía de implementación: describe qué convenciones existen, cómo se expresan y qué debe verificar la validación.

Esto incluye:

- **Naming:** `kebab-case` para archivos e ids de epics, tickets, agentes y workflows; patrones `epic-NN-<slug>`, `ticket-NN-MM-<slug>` (NN = epic, MM = secuencia); la carpeta del ticket **es** su id.
- **Estructura:** layout esperado del workspace `.daedalus/` (`agents/`, `prompts/`, `workflows/`, `specs/`, `architecture/`, `epics/`, `tickets/`, `docs/`, `.state/`) y de `development/epics/…/tickets/…` con su contrato de documentos.
- **Formato:** YAML con claves ordenadas y Markdown estructurado (coherente con RF-8.2), encabezados jerárquicos y tablas para metadatos.
- **Trazabilidad:** todo ticket referencia su epic; todo epic referencia los RF de origen.
- **Validabilidad:** una validación automática verifica el cumplimiento de las convenciones anteriores y reporta desviaciones de forma accionable.

## Requerimientos

1. **Convenciones explícitas.** Las convenciones de naming, estructura y formato están enunciadas de forma explícita y referenciable (fuente única, alineada con `init.md` §7 y CLAUDE.md §6).
2. **Naming validable.** Existe una validación que verifica `kebab-case` y los patrones de id (`epic-NN-<slug>`, `ticket-NN-MM-<slug>`) para epics, tickets, agentes y workflows.
3. **Estructura validable.** La validación verifica que el workspace y el backlog respetan el layout/estructura esperados (directorios y documentos requeridos presentes; sin elementos fuera de lugar).
4. **Formato validable.** La validación verifica coherencia de formato (YAML con claves ordenadas, Markdown estructurado) en línea con RF-8.2.
5. **Trazabilidad validable.** La validación verifica que cada ticket referencia su epic y cada epic sus RF de origen.
6. **Reporte accionable.** Las violaciones se reportan con ubicación y descripción claras (qué archivo/elemento y qué convención incumple), de modo accionable para el equipo.
7. **Determinismo.** La validación es determinista: mismo workspace → mismo resultado.
8. **Trazabilidad del ticket.** El plano referencia RF-8.3.

## Criterios de aceptación

- [ ] Las convenciones de naming, estructura y formato están **explicitadas** en una fuente referenciable.
- [ ] Existe una **validación automática** que verifica `kebab-case` y los patrones de id de epics, tickets, agentes y workflows.
- [ ] La validación verifica la **estructura** esperada del workspace `.daedalus/` y del backlog.
- [ ] La validación verifica la **coherencia de formato** (YAML claves ordenadas, Markdown estructurado).
- [ ] La validación verifica la **trazabilidad** ticket→epic y epic→RF.
- [ ] Las violaciones se reportan de forma **accionable** (ubicación + convención incumplida).
- [ ] La validación es **determinista** (mismo workspace → mismo resultado).
- [ ] Trazabilidad explícita a RF-8.3.

## Fuera de alcance

- **Qué** estado es relevante y **dónde** vive (inventario y persistencia → ticket-08-01).
- El detalle de **serialización determinista** del output (→ ticket-08-02); aquí solo se valida la coherencia de formato como convención.
- **Auto-fix / formateo automático** de violaciones (la validación reporta, no corrige).
- Convenciones de **git** (ramas/commits) más allá de las ya fijadas en `init.md` §7 y CLAUDE.md §5.
- Backend remoto / sincronización en la nube (epic.md — fuera de scope).

## Referencias

- `PRD.md` — RF-8.3; §3 (soporte a equipos, convenciones compartidas); RNF-6.
- `init.md` — §7 (convenciones de naming, YAML, Markdown, git, trazabilidad); §4.2 (estructura del workspace).
- `CLAUDE.md` — §6 (estructura `development/`, convenciones de nombres de epics/tickets, contrato de documentos).
- `development/epics/epic-08-state-collab/epic.md` — criterio "convenciones documentadas y validación que las verifica".
