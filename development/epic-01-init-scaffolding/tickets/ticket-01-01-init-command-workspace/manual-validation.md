# Validación manual — Ticket 01-01

> Para alguien sin experiencia en testing. Qué probar, cómo correrlo y qué esperar.

## Preparación

- Tené el binario `daedalus` listo (compilado o instalado).
- Creá una carpeta vacía nueva en tu disco, por ejemplo `mi-repo-prueba`, y abrí una terminal dentro de ella.

## Casos

### Caso 1 — Crear el workspace en una carpeta vacía

1. Hacé: dentro de la carpeta vacía, ejecutá `daedalus init`.
2. Esperá ver: un mensaje que dice que el workspace fue creado, indicando la ruta. Al mirar la carpeta, aparece una nueva subcarpeta `.daedalus/` que contiene: `daedalus.yaml`, `init.md` y las carpetas `agents`, `prompts`, `workflows`, `specs`, `architecture`, `epics`, `tickets`, `docs` y `.state`.

### Caso 2 — No tocar tus archivos existentes

1. Hacé: en otra carpeta, poné primero un par de archivos tuyos (por ejemplo un `README.md` con algo de texto y una carpeta `src` con un archivo). Después ejecutá `daedalus init`.
2. Esperá ver: la subcarpeta `.daedalus/` recién creada, **y** tus archivos originales (`README.md`, `src/...`) intactos, con el mismo contenido de antes.

### Caso 3 — Mismo resultado siempre (determinismo)

1. Hacé: ejecutá `daedalus init` en dos carpetas vacías distintas.
2. Esperá ver: la estructura interna de `.daedalus/` es idéntica en ambas (mismas carpetas, mismos nombres).

## Si algo no coincide

Reportalo al orquestador (Workflow B del CLAUDE.md).
