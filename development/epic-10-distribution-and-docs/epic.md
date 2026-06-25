# Epic 10 — Distribution & User Documentation

> **Origen:** Release-readiness post-Fase 1 (no es la Fase 2 del PRD §14, que cubre la ejecución/observabilidad de agentes). Habilita que Daedalus se **publique** y se **adopte** por consumidores externos. **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Fase 1 quedó completa: el core de Daedalus (init, edición de definiciones canónicas, validación, compilación a `.claude/`, TUI, logging, testing, linters) está implementado y verde. Lo que falta para que la herramienta sea **usable por terceros** no es más producto: es (1) una **documentación de uso orientada al consumidor** que explique instalar y operar la herramienta sin conocer su código, (2) un **sitio de documentación** publicado, y (3) un **pipeline de release** que produzca binarios descargables desde GitHub para Windows, macOS y Linux.

Hoy existe un manual markdown en `docs/` (getting-started, guide por feature, contributing) construido capítulo a capítulo durante Fase 1. Es buen material pero está orientado por-feature y mezcla audiencias; falta una narrativa cohesiva de adopción (instalar → primeros pasos → flujo central → referencia de comandos → ejemplos → troubleshooting) y una validación de que el recorrido completo funciona contra la herramienta real. Y no hay forma de que un usuario **descargue** Daedalus: no hay releases ni binarios publicados.

## Objetivo

Dejar a Daedalus **listo para publicar y adoptar**: una guía de usuario consumer-facing fácil de seguir y validada end-to-end contra la herramienta real, publicada como sitio hospedado, y un pipeline de release automatizado que entregue binarios multiplataforma descargables desde GitHub Releases.

## Alcance

**Incluye:**
- Consolidación y reestructuración del manual `docs/` en una **guía de usuario** consumer-facing coherente (instalación, quickstart, flujo central, referencia de comandos con flags/parámetros, ejemplos, troubleshooting).
- **Validación manual** end-to-end de la herramienta siguiendo la guía paso a paso (gate de release).
- **Sitio de documentación** publicado (MkDocs Material → GitHub Pages) con despliegue automatizado.
- **Pipeline de release** (GoReleaser + GitHub Actions disparado por tag) que cross-compila y publica binarios + checksums en GitHub Releases, con la versión inyectada desde el tag.

**No incluye:**
- Nuevas features de producto (el core de Fase 1 no se modifica salvo fixes que surjan de la validación manual).
- La Fase 2 del PRD (§14): ejecución/observabilidad de agentes en vivo.
- Instaladores nativos (.msi, Homebrew, .deb/.rpm) — se dejan como evolución futura sobre el pipeline de GoReleaser; este epic entrega archives + checksums.
- Internacionalización de la documentación (la doc de producto es en inglés, CLAUDE.md §1).

## Tickets

| Ticket | Tipo | Foco | Implementador / Validador |
|---|---|---|---|
| `ticket-10-01-user-guide` | docs | Guía de usuario consumer-facing cohesiva sobre `docs/`; incluye el `manual-validation.md` que recorre toda la herramienta y **bloquea** el release. | C-3PO / usuario (validación manual) |
| `ticket-10-02-docs-site` | tooling+docs | Publicar el manual como sitio (MkDocs Material) con workflow de GitHub Actions que despliega a GitHub Pages. | Obi-Wan + C-3PO / Yoda |
| `ticket-10-03-release-pipeline` | backend/CI | GoReleaser + GitHub Actions por tag `vX.Y.Z`: archives multiplataforma + checksums + versión por ldflags, publicados en GitHub Releases. | Obi-Wan / Yoda |

## Orden de ejecución

1. **10-01** se implementa primero. Tras su validación automática, su `manual-validation.md` **detiene el workflow**: el usuario recorre la herramienta siguiendo la guía y se corrigen las discrepancias guía↔herramienta antes de avanzar.
2. **10-02** y **10-03** pueden implementarse una vez la guía es estable.
3. El **primer release real** (`v0.1.0`) se publica solo después de que la validación manual de 10-01 pase.

## Criterios de aceptación del epic

- La guía permite a un usuario nuevo instalar Daedalus, inicializar un workspace, editar definiciones, validarlas y compilarlas a un backend, siguiendo solo la documentación.
- La validación manual end-to-end pasa: cada paso documentado coincide con el comportamiento real de la herramienta.
- El manual está publicado como sitio navegable y se redespliega automáticamente al actualizar `docs/`.
- Un tag de versión produce, de forma automatizada y reproducible, un GitHub Release con binarios para Windows/macOS/Linux (amd64+arm64) + checksums, con la versión correcta embebida.
- Toda la documentación de producto y los mensajes están en inglés (CLAUDE.md §1).

## Referencias

- PRD.md — RNF-3 (portabilidad Windows/macOS/Linux), RNF-6 (git-friendly), §8.1 (compilación reproducible/portabilidad); §14 (Fase 2 = runtime de agentes, explícitamente fuera de este epic).
- CLAUDE.md — §1 (doc de producto en inglés; backlog en español), §6 (estructura `docs/` como manual; `development/` como backlog SDD).
- Manual existente: `docs/` (getting-started, guide, contributing).
- Toolchain y entorno: `daedalus-go-toolchain`, binario en PATH de usuario, versión vía `--version` (hoy `0.1.0-dev`, inyectada desde `internal/buildinfo`).
