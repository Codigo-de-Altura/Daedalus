# Ticket 10-03 — Pipeline de release (binarios descargables)

> **Epic:** epic-10-distribution-and-docs · **Tipo:** backend/CI · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** release-readiness (epic-10) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Hoy no hay forma de que un usuario **descargue** Daedalus: se construye desde fuente (`go install`). Para adopción externa hace falta publicar **binarios precompilados** para las plataformas soportadas, descargables desde **GitHub Releases**, producidos de forma **automatizada y reproducible** a partir de un **tag de versión**.

Daedalus es un único binario Go; ya embebe su versión vía `internal/buildinfo` (hoy `0.1.0-dev`, expuesta por `--version`). Falta el andamiaje de release que cross-compile, empaquete e inyecte la versión del tag.

## Feature / Qué se construye

El **pipeline de release** de Daedalus:

- **GoReleaser** como herramienta de release: configuración (`.goreleaser.yaml`) que cross-compila el binario `cmd/daedalus` para **Windows, macOS y Linux** en **amd64 y arm64**, empaqueta cada target como **archive** (`.zip` para Windows, `.tar.gz` para Unix) e incluye **checksums** del conjunto.
- **Inyección de versión**: la versión publicada se toma del **tag git** (`vX.Y.Z`) y se inyecta por **ldflags** en `internal/buildinfo`, de modo que el binario descargado reporte la versión correcta con `--version` (no `-dev`).
- **Publicación en GitHub Releases**: el release con sus archives + checksums (+ notas de release) se publica automáticamente.
- **Automatización por tag**: un **workflow de GitHub Actions** dispara el release **al pushear un tag** `vX.Y.Z`. Reproducible, sin pasos manuales locales.
- **Verificabilidad local**: la configuración pasa `goreleaser check` y un build **snapshot/dry-run** (`--snapshot --clean`, sin publicar) produce los archives esperados en `dist/` para inspección antes de tagear.

## Requerimientos

- R1. Existe configuración de **GoReleaser** (`.goreleaser.yaml`) que cross-compila `cmd/daedalus` para **{windows, darwin, linux} × {amd64, arm64}** y empaqueta cada target como archive (zip para Windows, tar.gz para Unix).
- R2. El conjunto de release incluye un archivo de **checksums** de los artefactos.
- R3. La **versión** publicada proviene del **tag git** e inyecta por **ldflags** en `internal/buildinfo`; el binario descargado reporta esa versión con `--version`.
- R4. Existe un **workflow de GitHub Actions** que ejecuta el release con GoReleaser **al pushear un tag** `vX.Y.Z`, publicando en **GitHub Releases**.
- R5. La config pasa `goreleaser check` y un **snapshot/dry-run** local produce los archives esperados sin publicar.
- R6. El proceso es **reproducible** (versión de Go y de GoReleaser fijadas en el workflow) y no depende de un entorno local concreto; **portabilidad** de los targets (RNF-3).
- R7. El workflow usa **permisos mínimos** (publicar release) y un token estándar; no toca `git config` ni filtra secretos en logs.
- R8. Todo lo escrito (config, workflow, comentarios, notas) en **inglés** (CLAUDE.md §1).

## Criterios de aceptación

- [ ] CA1. `goreleaser check` pasa sobre `.goreleaser.yaml`.
- [ ] CA2. Un build snapshot/dry-run produce, en `dist/`, archives para los 6 targets (windows/darwin/linux × amd64/arm64) + un archivo de checksums.
- [ ] CA3. El binario producido por el pipeline reporta, con `--version`, la versión derivada del tag (no `-dev`) — verificable en snapshot con una versión simulada.
- [ ] CA4. Existe un workflow de GitHub Actions disparado por tag `vX.Y.Z` que corre GoReleaser y publica en GitHub Releases.
- [ ] CA5. Las versiones de Go y GoReleaser están fijadas en el workflow (reproducibilidad).
- [ ] CA6. El workflow usa permisos mínimos y no expone secretos; no modifica `git config`.
- [ ] CA7. Config/workflow/notas en inglés.

## Fuera de alcance

- **Instaladores nativos** (.msi, Homebrew tap, .deb/.rpm) y **script de instalación** `curl|sh`: evolución futura sobre GoReleaser; este ticket entrega archives + checksums.
- Firma/notarización de binarios (code signing) y SBOM: futuro.
- La **decisión del primer tag** (`v0.1.0`) y su push: ocurre tras pasar la validación manual de 10-01; este ticket deja el pipeline listo para que ese tag lo dispare.
- La sección de instalación de la guía (10-01) se actualiza con la URL/tag real una vez exista el primer release.

## Referencias

- PRD.md — RNF-3 (portabilidad Windows/macOS/Linux), §8.1 (portabilidad).
- CLAUDE.md — §1 (inglés), §5 (nunca tocar `git config`; sin atribución a IA en notas/commits).
- `internal/buildinfo` (versión embebida; hoy `0.1.0-dev`, expuesta por `--version`).
- GoReleaser (`.goreleaser.yaml`, `goreleaser check`, `--snapshot --clean`) + GitHub Actions (release por tag).
- epic-10-distribution-and-docs/epic.md — alcance (archives multiplataforma + checksums, versión por tag, deploy automatizado).
