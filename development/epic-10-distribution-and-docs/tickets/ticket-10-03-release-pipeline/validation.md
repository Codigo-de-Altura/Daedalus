# Validación automática — Ticket 10-03

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario se compila (`go build ./...` verde) y `--version` funciona.
- GoReleaser está disponible (versión fijada) para `check` y un build snapshot local. Si GoReleaser no está instalado en el entorno, instalarlo a la versión pinneada del workflow antes de validar.

## Checks

1. **Config válida** — Comando: `goreleaser check` sobre `.goreleaser.yaml` · Esperado: pasa sin errores.
2. **Snapshot multiplataforma** — Comando: `goreleaser release --snapshot --clean` (sin publicar) · Esperado: genera en `dist/` archives para los 6 targets (windows/darwin/linux × amd64/arm64) — zip para Windows, tar.gz para Unix — más un archivo de checksums.
3. **Versión inyectada** — Comando: desempaquetar el binario del snapshot (con una versión simulada por el snapshot) y correr `--version` · Esperado: reporta la versión derivada del tag/snapshot, no `0.1.0-dev`; confirma que los ldflags inyectan en `internal/buildinfo`.
4. **Workflow por tag** — Comando: revisar el workflow de GitHub Actions · Esperado: se dispara con tags `vX.Y.Z`, corre GoReleaser, publica en GitHub Releases; Go y GoReleaser con versiones fijadas; permisos mínimos; sin `git config`; sin secretos en logs.
5. **Reproducibilidad** — Comando: revisar pins de versión (Go, GoReleaser) y que el build no dependa de estado local · Esperado: versiones fijadas; proceso reproducible.
6. **Idioma** — Comando: revisar config/workflow/notas · Esperado: inglés.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1 |
| CA2 | 2 |
| CA3 | 3 |
| CA4 | 4 |
| CA5 | 4, 5 |
| CA6 | 4 |
| CA7 | 6 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
- Nota: el **release real** (publicación en GitHub Releases) se confirma recién al pushear el primer tag `v0.1.0`, lo cual ocurre tras pasar la validación manual de 10-01. Yoda valida `check` + snapshot local + corrección del workflow; la publicación efectiva es post-merge/tag.
