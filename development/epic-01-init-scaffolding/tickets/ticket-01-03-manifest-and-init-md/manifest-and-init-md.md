# Ticket 01-03 — Generar `daedalus.yaml` (manifiesto) e `init.md` base

> **Epic:** epic-01-init-scaffolding · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RF-1.3 (PRD) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

El workspace `.daedalus/` necesita dos artefactos raíz con **contenido** real, no solo presencia: el **manifiesto** `daedalus.yaml` (nombre del proyecto, backend(s), versión, convenciones) y el `init.md` base (lineamiento maestro del proyecto target). El ticket 01-01 garantiza que estos archivos existan dentro del scaffolding; este ticket define **qué contienen** y exige que se generen de forma **determinista** (RNF-5), con formatos diff-friendly (RNF-6, init.md §7).

El manifiesto es la pieza de configuración central del workspace: lo leen las operaciones posteriores (gestión de agentes, compilación al backend). El `init.md` base es el punto de entrada documental del proyecto target, análogo al `init.md` de Daedalus pero instanciado para el repo del usuario.

## Feature / Qué se construye

Generación determinista del contenido de los dos artefactos raíz del workspace durante `daedalus init`:

- **`.daedalus/daedalus.yaml`** — manifiesto con, al menos: `name` (nombre del proyecto), `version` (versión del workspace/esquema Daedalus), `backends` (lista de backend(s) objetivo; el valor concreto lo aporta el ticket 01-04) y `conventions` (convenciones del proyecto: naming kebab-case, etc.). Claves **estables y ordenadas** para diffs limpios.
- **`.daedalus/init.md`** — documento base del proyecto target: lineamiento maestro instanciado (visión/propósito placeholder, mapa de la estructura `.daedalus/` §4.2, convenciones, referencias). Sirve como punto de entrada para personas y agentes del proyecto del usuario.

Ambos se generan de manera **determinista**: el mismo input (nombre de proyecto, backend) produce exactamente el mismo archivo. El ticket 01-04 define cómo se rellena el campo de backend; aquí el manifiesto contempla el campo y un valor por defecto coherente con el MVP (Claude Code).

## Requerimientos

- R1. `daedalus init` genera el contenido de `.daedalus/daedalus.yaml` con, como mínimo, las claves: `name`, `version`, `backends`, `conventions`.
- R2. `daedalus init` genera el contenido de `.daedalus/init.md` base del proyecto target, alineado en estructura con el `init.md` de Daedalus (visión, mapa de estructura §4.2, convenciones, referencias) e instanciado para el proyecto.
- R3. Ambos artefactos se generan de forma **determinista**: mismo input → mismo output byte a byte (apto para golden files, RNF-5).
- R4. El `daedalus.yaml` usa claves **ordenadas y estables** (YAML legible y diff-friendly, RNF-6 / init.md §7).
- R5. El manifiesto describe la estructura `.daedalus/` de init.md §4.2 de forma coherente (no contradice carpetas/artefactos del scaffolding).
- R6. El campo `backends` del manifiesto existe y admite un valor por defecto coherente con el MVP (Claude Code); su selección efectiva es responsabilidad del ticket 01-04.
- R7. El `name` del proyecto se deriva de forma determinista (p. ej. del nombre del directorio del repo objetivo) o de un valor provisto; el criterio elegido es estable y documentado en la spec.
- R8. La generación respeta la no destrucción del ticket 01-02: si los artefactos ya existen con contenido manual, no se sobrescriben.

## Criterios de aceptación

- [ ] CA1. Tras `init`, `.daedalus/daedalus.yaml` contiene al menos `name`, `version`, `backends` y `conventions`, en YAML válido.
- [ ] CA2. Tras `init`, `.daedalus/init.md` contiene el lineamiento base del proyecto, incluyendo el mapa de estructura `.daedalus/` (§4.2) y convenciones.
- [ ] CA3. Ejecutar la generación dos veces con el mismo input produce archivos idénticos byte a byte (determinismo / golden file).
- [ ] CA4. Las claves del `daedalus.yaml` están en un orden estable y reproducible entre corridas.
- [ ] CA5. El campo `backends` está presente en el manifiesto con un valor por defecto coherente con el MVP.
- [ ] CA6. Si `daedalus.yaml` o `init.md` ya existían con contenido, la generación no los sobrescribe (coherencia con 01-02).

## Fuera de alcance

- Creación del esqueleto de carpetas y presencia de archivos (ticket 01-01).
- Detección/upgrade no destructivo del workspace (ticket 01-02), del que este ticket solo respeta la regla de no sobrescritura.
- Mecánica de **selección** interactiva del backend y su escritura efectiva en el manifiesto (ticket 01-04).
- Contenido de agentes/prompts/workflows/backlog (epics 02–05) y compilación (epic 06).

## Referencias

- PRD.md — RF-1.3, §8.2 (manifiesto: nombre, backend(s), versión, convenciones), RNF-5 (determinismo), RNF-6 (git-friendly).
- init.md — §4.2 (artefactos raíz `daedalus.yaml` e `init.md`), §7 (convenciones, YAML ordenado/determinista), documento mismo como referencia de estructura del `init.md` base.
- epic-01-init-scaffolding/epic.md — criterio: "el manifiesto y el init.md base se generan de forma determinista".
- Tickets 01-01 (scaffolding), 01-02 (no destrucción), 01-04 (backend).
