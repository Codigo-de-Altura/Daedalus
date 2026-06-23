# Estado versionado en git

> **Epic:** epic-08-state-collab · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-8.1 · **Estilo:** SDD

---

## Contexto

Daedalus está pensado para **equipos** y su principio rector es ser una **fuente de verdad única, versionable en git** (PRD §3, D7). Todo el ecosistema SDD —agentes, prompts, workflows, backlog, estado de progreso— vive en el workspace `.daedalus/` dentro del repo objetivo. Para que la colaboración funcione, el **estado relevante** de Daedalus no puede residir en ubicaciones efímeras (cachés del sistema, directorios de usuario, temporales, bases de datos locales no commiteadas): debe vivir en archivos de texto que git pueda rastrear.

Este ticket define el requerimiento de que **todo el estado relevante** de Daedalus sea git-trackeable, sin excepciones que rompan la portabilidad entre máquinas y miembros del equipo.

## Feature / Qué se construye

La definición —como **plano**— de la **política de persistencia git-versionada** de Daedalus: qué se considera "estado relevante", dónde vive dentro de `.daedalus/`, y la garantía de que ninguna pieza de estado relevante se almacene fuera de archivos git-trackeados.

Esto incluye:

- Inventario del **estado relevante**: definiciones canónicas (agentes, prompts, workflows), backlog SDD (specs, arquitectura, epics, tickets, docs) y el **estado de progreso** (`.daedalus/.state/`).
- Ubicación canónica de cada categoría de estado dentro del workspace `.daedalus/`, según el mapa de `init.md` §4.2.
- La regla de que el directorio de estado de progreso `.daedalus/.state/` es **git-tracked** (no ignorado), tal como lo marca el PRD §8.2 e `init.md` §4.2.
- La distinción entre **estado relevante** (siempre versionado) y **artefactos derivados/efímeros** (p. ej. salidas de `build` bajo `.claude/`, caches), que quedan fuera del alcance de este estado canónico.

## Requerimientos

1. **Inventario explícito.** El plano enumera las categorías de estado relevante de Daedalus y mapea cada una a una ruta dentro de `.daedalus/`.
2. **Todo en git.** Ninguna categoría de estado relevante puede persistirse fuera de archivos rastreables por git (sin dependencias en `$HOME`, temporales del SO, bases de datos binarias locales no commiteadas, ni variables de entorno como única fuente).
3. **Estado de progreso versionado.** `.daedalus/.state/` es parte del estado relevante y debe ser git-tracked; cualquier `.gitignore` generado o sugerido por Daedalus **no** debe excluirlo.
4. **Formato de texto.** El estado relevante se persiste en formatos de texto (YAML/Markdown) coherentes con RF-8.2 (el detalle de diff-friendliness y determinismo vive en el ticket 08-02).
5. **Separación clara derivados vs. canónico.** El plano deja explícito qué es derivado/efímero (fuera del estado canónico versionado) para no confundir el alcance.
6. **Trazabilidad.** El plano referencia RF-8.1 y el mapa de estructura de `init.md` §4.2 / PRD §8.2.

## Criterios de aceptación

- [ ] Existe un inventario explícito de las categorías de **estado relevante** y su ruta dentro de `.daedalus/`.
- [ ] Ninguna categoría de estado relevante se persiste fuera de archivos git-trackeados.
- [ ] `.daedalus/.state/` está declarado como git-tracked y no es excluido por reglas de ignore generadas/sugeridas por Daedalus.
- [ ] El estado relevante se persiste en formatos de texto (YAML/Markdown).
- [ ] El plano distingue claramente estado **canónico/versionado** de artefactos **derivados/efímeros**.
- [ ] Trazabilidad explícita a RF-8.1.

## Fuera de alcance

- **Backend remoto de estado o sincronización en la nube** (epic.md — explícitamente fuera de scope).
- El detalle de **serialización determinista y formatos diff-friendly** (→ ticket-08-02).
- La validación de **convenciones de naming/estructura** del equipo (→ ticket-08-03).
- La **compilación** a artefactos nativos del backend (`.claude/…`) y su política de idempotencia (EPIC-6).

## Referencias

- `PRD.md` — RF-8.1; §3 (principio "fuente de verdad única, versionable en git"); §8.2 (estructura de `.daedalus/`, `.state/` git-tracked); RNF-6 (git-friendly).
- `init.md` — §4.2 (mapa del workspace `.daedalus/`); §7 (convenciones de git y estado).
- `development/epics/epic-08-state-collab/epic.md` — objetivo y criterios del epic.
