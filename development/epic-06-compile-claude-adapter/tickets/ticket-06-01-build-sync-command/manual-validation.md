# Validación manual — `daedalus build` / `sync`

> Para alguien **sin background de testing**. Seguí los pasos tal cual. Si algo no coincide con lo esperado, **avisá al orquestador (Workflow B)**.

---

## Preparación

1. Conseguí el binario `daedalus` (preguntá al equipo dónde está o cómo compilarlo).
2. Abrí una terminal.
3. Ubicate en un repositorio de prueba que **ya tenga** una carpeta `.daedalus/` con su `daedalus.yaml` configurado para Claude Code. Si no lo tenés, pedíselo al orquestador.
4. Tené a mano un segundo directorio **vacío** (sin `.daedalus/`) para una de las pruebas.

## Casos

### Caso 1 — La ayuda del comando aparece
- **Hacé:** `daedalus build --help`
- **Esperá ver:** Un texto de ayuda que describe el comando `build`. Sin errores.

### Caso 2 — El alias `sync` funciona igual
- **Hacé:** `daedalus sync --help`
- **Esperá ver:** Una ayuda equivalente a la del Caso 1 (es el mismo comando con otro nombre).

### Caso 3 — Compilar el proyecto
- **Hacé:** Parado dentro del repo con `.daedalus/`, ejecutá `daedalus build`.
- **Esperá ver:** Un mensaje de resumen que indica el backend (Claude Code), qué archivos se crearon o actualizaron y un estado final de éxito. Debería aparecer una carpeta `.claude/` con los artefactos.

### Caso 4 — Sin workspace
- **Hacé:** Movete al directorio vacío y ejecutá `daedalus build`.
- **Esperá ver:** Un mensaje de error claro diciendo que no encontró un workspace `.daedalus/`. **No** debería crear ningún archivo.

### Caso 5 — Nada se escribe si hay un error
- **Hacé:** Pedile al orquestador que introduzca un error a propósito en la definición (o seguí sus instrucciones), y volvé a correr `daedalus build`.
- **Esperá ver:** Un mensaje que describe el problema de validación y **ningún archivo nuevo escrito** en `.claude/`.

## Si algo no coincide

Si cualquier caso no se comporta como dice "Esperá ver", **reportáselo al orquestador (Workflow B)** con: qué comando corriste, qué esperabas y qué pasó en realidad.
