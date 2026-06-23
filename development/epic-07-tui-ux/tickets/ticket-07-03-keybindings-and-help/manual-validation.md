# Validación manual — Atajos de teclado y ayuda contextual

> Para alguien **sin experiencia en testing**. Seguí los pasos tal cual. Si algo no coincide con lo esperado, no intentes arreglarlo: avisá al **orquestador** (Workflow B).

## Preparación

1. Abrí una terminal en el repositorio del proyecto.
2. Arrancá la TUI de Daedalus (el orquestador te dará el comando si no lo sabés).
3. Quedate en la pantalla inicial para empezar.

## Casos

### Caso 1 — Abrir la ayuda
- **Hacé:** desde la pantalla inicial, abrí la ayuda (la tecla suele estar indicada en pantalla; probá la que diga "ayuda" o "?").
- **Esperá ver:** un listado de atajos disponibles en ese momento.

### Caso 2 — La ayuda funciona en todas partes
- **Hacé:** entrá a un área cualquiera y abrí la ayuda con la misma tecla que usaste antes.
- **Esperá ver:** la ayuda se abre igual, y muestra los atajos que aplican a esa área.

### Caso 3 — Misma tecla, mismo efecto
- **Hacé:** anotá qué tecla usás para "volver" en un área. Entrá a otras áreas y probá esa misma tecla.
- **Esperá ver:** "volver" hace siempre lo mismo con la misma tecla, sin importar el área. Lo mismo debería pasar con moverte, entrar y salir.

### Caso 4 — Lo que dice la ayuda es lo que pasa
- **Hacé:** elegí un atajo que aparezca en la ayuda y usalo.
- **Esperá ver:** ocurre exactamente la acción que la ayuda describía.

### Caso 5 — Ayuda dentro de un formulario
- **Hacé:** abrí un formulario y abrí la ayuda ahí.
- **Esperá ver:** la ayuda muestra cómo enviar y cancelar el formulario, además de lo básico de navegación.

## Si algo no coincide

Si cualquier caso no se comporta como dice "Esperá ver", reportalo al **orquestador** (Workflow B) describiendo qué hiciste y qué viste.
