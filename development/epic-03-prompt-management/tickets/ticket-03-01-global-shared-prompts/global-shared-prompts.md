# Global & Shared Prompts — Edición de prompts reutilizables

> **Epic:** epic-03-prompt-management · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-3.1 · **Estilo:** SDD

## Contexto

Daedalus gestiona, de forma agnóstica del backend, la estructura de IA de un proyecto. Dentro de esa estructura, los **prompts** son fragmentos de texto reutilizables que alimentan a los agentes y a las convenciones del proyecto (glosario, estilo, lineamientos). Hoy ese contenido se rehace manualmente en cada proyecto, lo que genera inconsistencias.

Este ticket cubre la **edición y persistencia** de dos clases de prompts dentro del workspace canónico `.daedalus/prompts/`:

- **Prompts globales:** lineamientos que aplican a todo el proyecto (p. ej. estilo, idioma, convenciones SDD).
- **Prompts compartidos reutilizables:** fragmentos referenciables por agentes u otros prompts (p. ej. glosario, definición de roles, política de commits).

No cubre la composición/inclusión entre prompts (ticket-03-02) ni la previsualización en la TUI (ticket-03-03), ni la compilación al backend (epic-06).

## Feature / Qué se construye

Capacidad del **core (backend)** para **crear, leer, editar, listar y persistir** prompts globales y compartidos en `.daedalus/prompts/`, con una identidad estable (id/slug) y metadatos mínimos, en formato Markdown legible y git-friendly.

El feature expone operaciones de dominio sobre prompts (no la UI, que se trata en frontend) y garantiza persistencia determinista y no destructiva.

## Requerimientos

- **R1.** El sistema persiste prompts en `.daedalus/prompts/` como archivos Markdown (`.md`), uno por prompt, con nombre `kebab-case` derivado del id/slug del prompt.
- **R2.** Cada prompt tiene metadatos mínimos: `id` (slug único, `kebab-case`), `kind` (`global` | `shared`), `title` y `description` opcional. Los metadatos viven en frontmatter del archivo o en una estructura equivalente legible.
- **R3.** El core soporta operaciones de dominio: **crear**, **leer**, **listar** (con filtro por `kind`), **editar** (título, descripción, cuerpo) y **eliminar** un prompt.
- **R4.** El `id` de un prompt es **único** dentro del workspace; intentar crear un prompt con un `id` existente es un error explícito (no sobrescribe).
- **R5.** La persistencia es **determinista** (mismo input → mismo archivo, orden y formato estables) y **no destructiva**: editar un prompt no altera otros archivos del workspace ni contenido fuera del área gestionada.
- **R6.** La distinción `global` vs `shared` es explícita y consultable; los prompts globales y compartidos coexisten en `.daedalus/prompts/` sin colisión de nombres.
- **R7.** El cuerpo del prompt es texto Markdown arbitrario; el core lo persiste sin reinterpretarlo ni resolver inclusiones (eso es ticket-03-02).
- **R8.** Errores recuperables (id duplicado, prompt inexistente, slug inválido) se reportan de forma explícita y sin dejar el workspace en estado inconsistente.

## Criterios de aceptación

- [ ] Se puede **crear** un prompt `global` y queda persistido como archivo `.md` en `.daedalus/prompts/`.
- [ ] Se puede **crear** un prompt `shared` y queda persistido en `.daedalus/prompts/`.
- [ ] **Listar** prompts devuelve todos los prompts con su `kind`, y permite **filtrar** por `global` o `shared`.
- [ ] **Editar** título/descripción/cuerpo de un prompt persiste el cambio sin alterar otros archivos.
- [ ] **Eliminar** un prompt remueve solo su archivo.
- [ ] Crear un prompt con un `id` ya existente **falla** con error explícito y no sobrescribe.
- [ ] El nombre de archivo es `kebab-case` y deriva del `id`.
- [ ] Crear/editar el mismo prompt dos veces con el mismo input produce **archivos idénticos** (determinismo).
- [ ] Un `id`/slug inválido (caracteres no permitidos, vacío) es **rechazado** con error explícito.

## Fuera de alcance

- Composición/inclusión de prompts y resolución de fragmentos (ticket-03-02).
- Previsualización renderizada en la TUI (ticket-03-03).
- UI/TUI de edición (frontend).
- Compilación de prompts al backend de agentes (epic-06).

## Referencias

- `PRD.md` — RF-3.1, sección 8.2 (estructura de `.daedalus/`), sección 9 EPIC-3.
- `epic.md` — epic-03-prompt-management.
- `CLAUDE.md` — §6 (estructura `development/`), §7 (filosofía SDD: determinismo, no destrucción).
