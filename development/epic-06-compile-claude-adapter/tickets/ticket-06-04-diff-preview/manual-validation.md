# Validación manual — Reporte de diff / preview

> Para alguien **sin background de testing**. Seguí los pasos tal cual. Si algo no coincide con lo esperado, **avisá al orquestador (Workflow B)**.

---

## Preparación

1. Conseguí el binario `daedalus` (preguntá al equipo dónde está o cómo compilarlo).
2. Abrí una terminal con buen soporte de colores.
3. Ubicate en un repo de prueba con una carpeta `.daedalus/` válida (pedísela al orquestador si no la tenés).
4. Asegurate de empezar **sin** una carpeta `.claude/` previa para el primer caso (si existe, pedile al orquestador un repo limpio).

## Casos

### Caso 1 — Ver el preview antes de escribir
- **Hacé:** Lanzá el build con preview (el orquestador te dirá el comando exacto).
- **Esperá ver:** Una pantalla que lista los archivos que se van a generar. Como no había `.claude/`, todos deberían figurar como **nuevos**. Todavía **no** se escribió nada.

### Caso 2 — Confirmar escribe
- **Hacé:** Confirmá (usá el atajo que muestra la pantalla).
- **Esperá ver:** Recién ahora se crean los archivos. Aparece la carpeta `.claude/` con lo que viste en el preview.

### Caso 3 — Sin cambios
- **Hacé:** Volvé a lanzar el preview sin tocar nada.
- **Esperá ver:** El reporte dice claramente que **no hay cambios** (no hay nada nuevo ni modificado).

### Caso 4 — Un cambio se muestra como "modificado"
- **Hacé:** Pedile al orquestador que cambie algo en la definición (o seguí sus instrucciones) y volvé a lanzar el preview.
- **Esperá ver:** El archivo afectado figura como **modificado**, con un detalle del cambio legible. Los demás siguen sin cambios.

### Caso 5 — Cancelar no escribe nada
- **Hacé:** En el preview con cambios, **cancelá** en vez de confirmar.
- **Esperá ver:** Nada se escribe; los archivos quedan como estaban antes de lanzar el preview.

### Caso 6 — Solo-preview
- **Hacé:** Lanzá el modo solo-preview (el orquestador te dirá cómo).
- **Esperá ver:** Se muestra el diff y el programa **sale sin escribir**.

## Si algo no coincide

Si cualquier caso no se comporta como dice "Esperá ver", **reportáselo al orquestador (Workflow B)** con: qué comando corriste, qué esperabas y qué pasó en realidad.
