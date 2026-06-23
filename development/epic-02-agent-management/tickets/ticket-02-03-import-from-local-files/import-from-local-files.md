# Ticket 02-03 — Importar agentes desde archivos locales

> **Epic:** epic-02-agent-management · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-2.3 · **Estilo:** SDD

## Contexto

Muchos proyectos ya tienen agentes definidos fuera de Daedalus, en particular estructuras `.claude/agents/` de Claude Code (PRD §8.1, init.md §5). Para reducir la fricción de adopción, Daedalus debe **absorber** esas definiciones existentes y convertirlas en definiciones canónicas agnósticas dentro del workspace, sin obligar a reescribirlas a mano. Esto materializa el principio de "una fuente de verdad canónica" (PRD §3) a partir de artefactos que ya viven en el repo.

## Feature / Qué se construye

La capacidad de **importar agentes desde archivos locales** al workspace `.daedalus/agents/`, convirtiéndolos a la definición canónica agnóstica. El import reconoce explícitamente las estructuras **`.claude/agents/`** existentes (agentes de Claude Code, con su frontmatter) además de definiciones canónicas en archivos locales. Cada agente importado queda como una definición canónica válida en el workspace.

## Requerimientos

- R1 — Se puede importar uno o varios agentes desde una ruta local (archivo o directorio) al workspace `.daedalus/agents/`.
- R2 — El import reconoce estructuras **`.claude/agents/`** existentes (formato Claude Code: archivos con frontmatter) y las convierte a la definición canónica agnóstica.
- R3 — El import también acepta definiciones de agente ya canónicas desde archivos locales.
- R4 — Cada agente importado se **valida** contra el esquema canónico (ticket-02-04); una definición que no valida produce un error accionable y no se importa en estado inválido sin avisar.
- R5 — El import es **no destructivo** (RNF-8): un identificador que ya existe en el workspace no se sobreescribe silenciosamente; se informa/ofrece preview o confirmación.
- R6 — Los identificadores resultantes siguen `kebab-case` (init.md §7); si el origen no lo cumple, se normaliza o se reporta de forma accionable.
- R7 — La importación es **determinista** (RNF-5): el mismo archivo de origen produce la misma definición canónica.
- R8 — El import **no ejecuta** agentes ni invoca el backend; solo transforma definiciones (Fase 1, D5).

## Criterios de aceptación

- [ ] CA1 — Importar desde un archivo local de agente crea su definición canónica bajo `.daedalus/agents/`.
- [ ] CA2 — Importar desde una estructura `.claude/agents/` existente convierte los agentes (frontmatter Claude Code) a definiciones canónicas.
- [ ] CA3 — Un archivo de origen inválido contra el esquema canónico produce un error accionable y no se importa silenciosamente.
- [ ] CA4 — Importar sobre un identificador ya existente no sobreescribe silenciosamente (operación no destructiva).
- [ ] CA5 — Los identificadores importados quedan en `kebab-case`.
- [ ] CA6 — Importar el mismo origen dos veces produce la misma definición canónica (determinismo).
- [ ] CA7 — Trazabilidad a RF-2.3.

## Fuera de alcance

- Catálogo built-in y su materialización (ticket-02-01).
- Clonado y edición de agentes (ticket-02-02).
- Definición del esquema canónico y sus errores accionables (ticket-02-04); este ticket lo consume.
- Compilación de canónico → `.claude/` (dirección inversa, epic-06).
- Edición visual avanzada en TUI (epic-07).

## Referencias

- PRD §9 — RF-2.3 (importar agentes desde archivos locales, incl. `.claude/agents`)
- PRD §8.1 (canónico ↔ formato nativo), §6 — D5 (Daedalus gestiona config, no ejecuta)
- init.md §5 (agente como YAML + prompt MD), §7 (convenciones)
- CLAUDE.md §6 (estructura de tickets)
- epic-02-agent-management/epic.md (ticket-02-03)
