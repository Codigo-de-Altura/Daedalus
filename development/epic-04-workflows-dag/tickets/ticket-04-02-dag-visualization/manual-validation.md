# Validación manual — Ticket 04-02

> Para alguien **sin background de testing**. Seguí los pasos tal cual y compará lo que ves con lo que se espera. Es una vista visual de la TUI, así que conviene validarla con los ojos.

## Preparación

1. Asegurate de tener Daedalus instalado y de poder abrir la TUI (seguí la guía de uso del proyecto / `documentation.md`).
2. Trabajá sobre un repo que ya tenga el workspace `.daedalus/` inicializado y al menos un workflow disponible (el `sdd-default` viene de fábrica).
3. Abrí la TUI de Daedalus.

## Casos

### Caso 1 — Abrir la vista del DAG

- **Hacé:** navegá hasta el área de **workflows** y seleccioná el workflow `sdd-default`.
- **Esperá ver:** un grafo dibujado en la terminal, con varios recuadros/nodos conectados por líneas/flechas. No debería verse el YAML crudo.

### Caso 2 — Leer los nodos

- **Hacé:** mirá cada nodo del grafo.
- **Esperá ver:** en cada nodo, el nombre de la **fase** (por ejemplo `spec`, `architecture`, `tickets`) y el **agente** que la ejecuta (por ejemplo `analyst`, `architect`, `planner`).

### Caso 3 — Leer el orden del pipeline

- **Hacé:** seguí las líneas/flechas entre nodos de principio a fin.
- **Esperá ver:** el orden del pipeline SDD: **brief → spec → arquitectura → epics → tickets → validación → docs**. Cada paso depende del anterior.

### Caso 4 — Confirmar que es solo lectura

- **Hacé:** intentá modificar o "correr" el workflow desde esta vista (probá las teclas habituales de editar/ejecutar).
- **Esperá ver:** la vista **no** te deja editar ni ejecutar nada; solo muestra el grafo. (En Fase 1, Daedalus configura la estructura de IA; no ejecuta agentes.)

### Caso 5 — Estética y legibilidad

- **Hacé:** observá colores, bordes y alineación del grafo; probá redimensionar un poco la terminal.
- **Esperá ver:** una vista prolija, con estilo consistente con el resto de la TUI, sin texto roto, superpuesto ni caracteres raros.

### Caso 6 — Caso vacío

- **Hacé:** si el proyecto te permite seleccionar un workflow vacío o que no cargue, abrilo.
- **Esperá ver:** un mensaje/estado claro (por ejemplo "workflow vacío" o "no se pudo cargar"), nunca un cuelgue ni un error que cierre la app.

---

**Si algo no coincide con lo esperado → reportáselo al orquestador (Workflow B).** Describí qué caso falló, qué viste y qué esperabas ver.
