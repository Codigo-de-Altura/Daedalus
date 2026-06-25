# Ticket 10-02 — Sitio de documentación publicado

> **Epic:** epic-10-distribution-and-docs · **Tipo:** tooling+docs · **Implementador:** Obi-Wan (tooling/CI) + C-3PO (navegación/contenido) · **Validador:** Yoda
> **Origen:** release-readiness (epic-10) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

La guía de usuario (10-01) vive como markdown en `docs/`. Para que sea consumible por terceros sin clonar el repo, debe publicarse como **sitio navegable** con índice, navegación y búsqueda. El contenido ya existe y es markdown; falta el **andamiaje de publicación** y su **automatización**.

## Feature / Qué se construye

La publicación del manual `docs/` como **sitio estático hospedado**, con despliegue automatizado:

- **Generador de sitio**: MkDocs con el tema **Material** (encaja con el markdown existente; aporta navegación jerárquica y búsqueda). Configuración declarativa (`mkdocs.yml`) con la **navegación** que refleja el recorrido de adopción de la guía (10-01): instalación → primeros pasos → conceptos → flujo → referencia → ejemplos → troubleshooting, más la sección de contributing.
- **Build reproducible del sitio**: `mkdocs build` produce el sitio a partir de `docs/` sin warnings de enlaces rotos (modo estricto).
- **Despliegue automatizado**: un workflow de **GitHub Actions** construye y publica el sitio a **GitHub Pages** automáticamente al actualizar la documentación en `main` (push que toque `docs/` o `mkdocs.yml`).
- **Navegación coherente**: la estructura del sitio coincide con el índice del manual; no hay páginas inalcanzables ni entradas de nav que apunten a archivos inexistentes.

El contenido de la documentación no se reescribe aquí (es de 10-01); este ticket aporta la **configuración de publicación**, la **navegación** y la **automatización del deploy**.

## Requerimientos

- R1. Existe configuración de **MkDocs (tema Material)** (`mkdocs.yml`) que toma el contenido de `docs/` y declara una navegación que refleja el recorrido de adopción de la guía.
- R2. `mkdocs build` (modo **estricto**: enlaces rotos = error) construye el sitio sin fallos a partir del `docs/` actual.
- R3. Existe un **workflow de GitHub Actions** que construye y **despliega a GitHub Pages** automáticamente cuando se actualiza la documentación en `main`.
- R4. La **navegación del sitio coincide** con la estructura/índice del manual: todas las páginas alcanzables, sin entradas de nav rotas, separación uso vs. contributing preservada.
- R5. El setup es **reproducible** (versiones de MkDocs/tema/plugins fijadas) y **no acopla** el sitio a un entorno local concreto.
- R6. El workflow **no interfiere** con los otros checks de CI ni con el pipeline de release (10-03); usa permisos mínimos de Pages.
- R7. Todo lo escrito (config, nav, comentarios) en **inglés** (CLAUDE.md §1).

## Criterios de aceptación

- [ ] CA1. `mkdocs build --strict` (o equivalente) construye el sitio sin errores ni enlaces rotos desde el `docs/` actual.
- [ ] CA2. `mkdocs.yml` declara una navegación que refleja el recorrido de adopción y todas sus entradas resuelven a páginas existentes.
- [ ] CA3. Existe un workflow de GitHub Actions que despliega a GitHub Pages al actualizar la documentación en `main`.
- [ ] CA4. La estructura del sitio coincide con el índice del manual; no hay páginas huérfanas ni entradas de nav rotas.
- [ ] CA5. Las versiones de las herramientas del sitio están fijadas; el build es reproducible.
- [ ] CA6. El workflow usa permisos mínimos y no rompe los demás workflows.
- [ ] CA7. Config/nav en inglés.

## Fuera de alcance

- El **contenido** de la documentación (ticket 10-01).
- Dominio propio/branding avanzado del sitio (se acepta la URL de GitHub Pages por defecto; un dominio custom es evolución futura).
- Versionado del sitio por release (p. ej. `mike`) — futuro; aquí se publica la última.

## Referencias

- CLAUDE.md — §1 (inglés), §6 (`docs/` como manual).
- Manual: `docs/README.md` y capítulos (la nav debe reflejar su índice).
- GitHub Pages + Actions (deploy de sitio estático).
- epic-10-distribution-and-docs/epic.md — alcance (sitio hospedado, deploy automatizado).
