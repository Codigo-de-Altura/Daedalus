# Adaptador Claude Code (`Compiler`)

> **Epic:** epic-06-compile-claude-adapter · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-6.2 · **Estilo:** SDD

---

## Contexto

El comando `daedalus build` (RF-6.1) orquesta la compilación pero delega el **mapeo canónico → formato nativo** a un **adaptador**. Este ticket define la **interfaz `Compiler`** (extensible, para añadir backends sin tocar el núcleo) y su **primera implementación: Claude Code**.

El adaptador Claude Code traduce la definición canónica (agentes, prompts/comandos, settings) a la estructura `.claude/` que consume Claude Code: `.claude/agents/*.md` con frontmatter, `.claude/commands/*.md` y los settings relevantes.

Referencias: PRD §8.1 (compilación por adaptador), RF-6.2, RNF-5 (determinismo), RNF-7 (extensibilidad: la interfaz de adaptador permite añadir backends sin tocar el núcleo). La idempotencia/no-destrucción profunda se especifica en RF-6.3; aquí se exige que el output sea **determinista** para habilitar golden files.

## Feature / Qué se construye

1. **Interfaz `Compiler`** — Un contrato agnóstico que todo adaptador de backend implementa. Permite que `daedalus build` invoque cualquier backend de forma uniforme y que se registren nuevos adaptadores sin modificar el núcleo.

2. **Adaptador Claude Code** — Implementación de `Compiler` que mapea la definición canónica al formato nativo de Claude Code:
   - **`.claude/agents/*.md`**: un archivo por agente, con **frontmatter** (metadatos: nombre/id, rol, parámetros relevantes) seguido del prompt.
   - **`.claude/commands/*.md`**: comandos derivados de la definición canónica.
   - **Settings relevantes**: la configuración que Claude Code requiere, derivada del manifiesto/definición.

El adaptador produce salida **determinista**: el mismo input canónico genera byte-a-byte el mismo conjunto de archivos (orden estable de claves, formato estable), habilitando **golden files** en validación.

## Requerimientos

- **REQ-1.** Existe una **interfaz `Compiler`** que define el contrato de compilación (entrada: definición canónica; salida: artefactos nativos para un backend).
- **REQ-2.** La interfaz permite **registrar** múltiples adaptadores; añadir uno nuevo **no requiere modificar** el comando `build` ni el núcleo (RNF-7).
- **REQ-3.** Existe una implementación `Compiler` para **Claude Code**.
- **REQ-4.** El adaptador genera **`.claude/agents/*.md`**: un archivo por agente canónico, con **frontmatter** válido + cuerpo (prompt).
- **REQ-5.** El frontmatter mapea los campos canónicos del agente a las claves esperadas por Claude Code (nombre/id, descripción/rol, parámetros relevantes).
- **REQ-6.** El adaptador genera **`.claude/commands/*.md`** desde la definición canónica de comandos.
- **REQ-7.** El adaptador genera los **settings** relevantes para Claude Code.
- **REQ-8.** La salida es **determinista** (RNF-5): mismo input → mismo output byte-a-byte (orden de claves estable, formato estable, sin timestamps ni datos volátiles).
- **REQ-9.** Los nombres de archivo siguen `kebab-case` y derivan del id canónico de forma estable.
- **REQ-10.** El adaptador reporta los artefactos que produciría/produjo, para que el comando arme el resumen y el preview (RF-6.4) pueda consumirlo.

## Criterios de aceptación

- [ ] Existe la interfaz `Compiler` y un mecanismo de registro de adaptadores.
- [ ] Agregar un backend nuevo no obliga a modificar el comando `build` ni el núcleo (la interfaz lo permite).
- [ ] El adaptador Claude Code genera `.claude/agents/*.md` con frontmatter válido (uno por agente).
- [ ] El adaptador genera `.claude/commands/*.md` desde la definición canónica.
- [ ] El adaptador genera los settings relevantes de Claude Code.
- [ ] El mismo input canónico produce **exactamente el mismo output** (golden files): determinismo verificable.
- [ ] Los nombres de archivo son `kebab-case` y estables respecto al id canónico.

## Fuera de alcance

- El **comando** `build`/`sync` y su orquestación → RF-6.1 (ticket-06-01).
- La estrategia de **escritura idempotente y no destructiva** (merge con cambios manuales) → RF-6.3 (ticket-06-03).
- El **diff/preview** y su UI → RF-6.4 (ticket-06-04).
- Adaptadores para otros backends (Codex, Gemini CLI…): la **interfaz** queda lista, pero la implementación es post-MVP.

## Referencias

- `PRD.md` — RF-6.2, §8.1, RNF-5 (determinismo), RNF-7 (extensibilidad).
- `init.md` — §5 (artefactos nativos `.claude/…`), §7 (naming kebab-case, YAML determinista), §8 (catálogo de agentes built-in).
- `CLAUDE.md` — §2 (Claude Code → `.claude/`), §6, §7 (determinismo / golden files).
- `epic.md` — Epic 06.
