# Epic 06 — Compile to Backend (Claude Code adapter)

> **Origen:** EPIC-6 del PRD (RF-6.1 … RF-6.4). **Estilo:** SDD (plano, no guía de implementación).

## Objetivo

Compilar la definición canónica del workspace al formato nativo del backend mediante `daedalus build`/`sync`. El primer adaptador es **Claude Code**: genera `.claude/agents/*.md` (con frontmatter), `.claude/commands/*.md` y los settings relevantes. La compilación es **idempotente**, no destructiva de cambios manuales fuera del área gestionada, y ofrece reporte de diff/preview antes de escribir.

## Alcance

**Incluye:** comando `build`/`sync`, interfaz `Compiler` (adaptador) con implementación Claude Code, idempotencia y no-destrucción, diff/preview.

**No incluye:** adaptadores adicionales (Codex, Gemini CLI…) — la interfaz queda preparada pero la implementación es post-MVP.

## Tickets

| Ticket | Tipo | Foco | Origen |
|---|---|---|---|
| `ticket-06-01-build-sync-command` | backend | `daedalus build`/`sync` compila la definición canónica al formato nativo. | RF-6.1 |
| `ticket-06-02-claude-code-adapter` | backend | Adaptador Claude Code: `.claude/agents/*.md`, `.claude/commands/*.md`, settings. | RF-6.2 |
| `ticket-06-03-idempotent-nondestructive-build` | backend | Build idempotente y no destructivo de cambios manuales fuera del área gestionada. | RF-6.3 |
| `ticket-06-04-diff-preview` | frontend | Reporte de diff/preview antes de escribir. | RF-6.4 |

## Criterios de aceptación del epic

- `daedalus build` genera artefactos Claude Code válidos desde la definición canónica.
- El mismo input produce el mismo output (determinismo / golden files).
- Re-compilar no destruye cambios manuales fuera del área gestionada.
- El usuario ve un diff/preview y confirma antes de que se escriba.
- La interfaz `Compiler` permite añadir backends sin tocar el núcleo.
- Trazabilidad a RF-6.x.
