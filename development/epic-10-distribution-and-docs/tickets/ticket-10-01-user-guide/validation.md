# Validación automática — Ticket 10-01

> La corre **Yoda** (validación automática previa a la validación manual del usuario). Solo reporta verdict (APPROVED/REJECTED); no implementa. Nota: el juicio editorial/UX de la guía es del usuario vía `manual-validation.md`; esta validación cubre lo objetivamente verificable.

## Precondiciones

- El binario `daedalus` está compilado y disponible (`--version` responde).
- El árbol `docs/` está presente; se puede listar y abrir cada archivo enlazado desde `docs/README.md`.

## Checks

1. **Índice con recorrido de adopción** — Comando: abrir `docs/README.md` · Esperado: ofrece un recorrido lineal (instalar → primeros pasos → conceptos → flujo → referencia → ejemplos → troubleshooting) y enlaza la referencia por comando; separa "guide" (uso) de "contributing" (contribución).
2. **Cobertura de comandos** — Comando: revisar la sección de referencia · Esperado: hay un apartado por comando real (`init`, `build`/`sync`, `validate`, `trace verify`/`show`, TUI, `--version`) con propósito, flags/parámetros, exit codes y ≥1 ejemplo.
3. **Fidelidad de la referencia a la CLI** — Comando: contrastar cada flag/exit-code documentado contra `daedalus <cmd> --help` para `init`, `build`, `validate`, `trace` · Esperado: coinciden exactamente; no hay flags ni comandos documentados que no existan, ni flags reales sin documentar.
4. **Sin enlaces rotos ni huérfanos** — Comando: verificar que todo enlace interno de `docs/` resuelve a un archivo existente y que todo archivo de `docs/` está alcanzable desde el índice · Esperado: cero enlaces rotos, cero archivos huérfanos.
5. **Quickstart reproducible** — Comando: ejecutar al pie de la letra los comandos del quickstart en un workspace throwaway · Esperado: producen un workspace inicializado y compilado, con las salidas/exit codes que la guía indica.
6. **Separación de audiencias** — Comando: revisar `guide/` vs `contributing/` · Esperado: la guía de uso no mezcla tareas de desarrollo del proyecto; `contributing/` permanece separado y referenciado.
7. **Idioma y formato** — Comando: revisar los archivos · Esperado: inglés, markdown legible, sin BOM, encabezados/listas/bloques de código bien formados.
8. **Existe `manual-validation.md`** — Comando: comprobar el archivo del ticket · Esperado: presente y recorre los comandos/flujos a través de la guía.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1 |
| CA2 | 2 |
| CA3 | 3 |
| CA4 | 5 |
| CA5 | 4, 6 |
| CA6 | 8 |
| CA7 | 7 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
- Tras APPROVED, el workflow **se detiene** en el `manual-validation.md`: el usuario recorre la herramienta siguiendo la guía antes de avanzar a 10-02/10-03.
