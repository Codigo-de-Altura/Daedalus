# Validación manual — Ticket 01-02

> Para alguien sin experiencia en testing. Qué probar, cómo correrlo y qué esperar.

## Preparación

- Tené el binario `daedalus` listo.
- Partí de una carpeta donde ya corriste `daedalus init` una vez (es decir, ya tiene la subcarpeta `.daedalus/`).

## Casos

### Caso 1 — No pisa tus ediciones manuales

1. Hacé: abrí el archivo `.daedalus/init.md`, escribí en algún lado una frase reconocible como `EDICION DE PRUEBA 123` y guardá. Después ejecutá `daedalus init` otra vez.
2. Esperá ver: un mensaje que indica que se trata de una actualización (upgrade) de un workspace existente, y que tu frase `EDICION DE PRUEBA 123` sigue estando en el archivo, sin haberse borrado.

### Caso 2 — Completa lo que falta sin tocar el resto

1. Hacé: borrá a mano una de las carpetas de adentro, por ejemplo `.daedalus/docs`. Luego ejecutá `daedalus init`.
2. Esperá ver: antes de aplicar, un preview que lista lo que va a crear (la carpeta `docs` faltante). Después de aplicar, la carpeta `docs` vuelve a existir y el resto de tus archivos quedan igual.

### Caso 3 — Correr de nuevo no cambia nada

1. Hacé: con el workspace ya completo, ejecutá `daedalus init` dos veces seguidas.
2. Esperá ver: en la segunda corrida no se proponen cambios (preview vacío o "nada que actualizar").

## Si algo no coincide

Reportalo al orquestador (Workflow B del CLAUDE.md).
