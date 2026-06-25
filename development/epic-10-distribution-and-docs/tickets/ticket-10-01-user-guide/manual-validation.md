# Validación manual — Ticket 10-01 (recorrido end-to-end de la herramienta)

> Para alguien sin background de testing. El objetivo es **recorrer toda la herramienta siguiendo la guía paso a paso** y confirmar que cada paso documentado coincide con el comportamiento real. Es el **gate previo al release**: hasta que esto pase, no se publica.
>
> Cómo reportar: por cada paso, anotá OK o, si falla, qué esperabas vs. qué viste. Las discrepancias guía↔herramienta se enrutan al orquestador (fix de doc → C-3PO; fix de comportamiento → Obi-Wan/Padmé).
>
> Las partes de **CLI** las puede correr el orquestador; la **TUI** la operás vos (necesita terminal interactiva).

## Preparación

- Trabajá en una carpeta **descartable** (no el repo de Daedalus). Por ejemplo creá una carpeta vacía `demo-proyecto` y entrá ahí.
- Tené `daedalus` disponible en el PATH.

## Pasos

1. **Versión** — Seguí la sección "Installation" de la guía y corré `daedalus --version`.
   - Esperado: imprime `daedalus <versión>` (hoy `0.1.0-dev`) y termina. Coincide con lo que dice la guía.

2. **Quickstart de cero** — Seguí el "Quickstart" tal cual.
   - `daedalus init` en la carpeta vacía.
   - Esperado: crea `.daedalus/` con la estructura documentada (agents, prompts, workflows, etc., y el manifiesto `daedalus.yaml`). El mensaje final coincide con el de la guía ("creado desde cero").

3. **Inspección del workspace** — Mirá `.daedalus/` siguiendo "Conceptos".
   - Esperado: lo que la guía describe (modelo canónico, manifiesto, workflow por defecto sembrado) está presente.

4. **Preview de init (idempotencia)** — Corré `daedalus init --preview` otra vez sobre el mismo workspace.
   - Esperado: indica que no hay cambios que escribir (o solo los faltantes), no destruye nada; coincide con lo que la guía dice sobre upgrade no destructivo.

5. **Validación de un workspace conforme** — Corré `daedalus validate`.
   - Esperado: exit 0; reporta ambos ejes (`Conventions:` y `Definitions:`) como válidos. La guía explica cómo leer este reporte.

6. **Validación con un error inyectado** — Seguí el ejemplo de "Troubleshooting": introducí una definición inválida (p. ej. un manifiesto con un backend inexistente, o un agente al que le falta un campo requerido) y volvé a correr `daedalus validate`.
   - Esperado: exit 1; un hallazgo **accionable** que nombra el archivo, el lugar (campo/fase/clave) y qué se esperaba vs. qué se encontró, tal como lo describe la guía. Revertí el cambio después.

7. **Build en preview** — Corré `daedalus build --preview` (alias `daedalus sync`).
   - Esperado: muestra el diff/los artefactos que se compilarían a `.claude/` sin escribir nada; coincide con la guía.

8. **Build con escritura** — Corré `daedalus build` (gate interactivo) o `daedalus build --yes` (sin gate).
   - Esperado: escribe `.claude/` con los artefactos compilados; el resumen (creados/actualizados/sin cambios) coincide con lo documentado. Una segunda corrida muestra un no-op idempotente.

9. **Trazabilidad** — Si la guía documenta `trace`, corré `daedalus trace verify` y `daedalus trace show <id>` con un id de ejemplo de la guía.
   - Esperado: el comportamiento (verify reporta consistencia; show navega la cadena) coincide con lo descrito.

10. **Logging / configuración** — Seguí "Configuration": corré un comando con `DAEDALUS_LOG_LEVEL=debug` y observá los eventos estructurados en stderr; probá `--version` y el redireccionamiento de logs que la guía muestre.
    - Esperado: los niveles y el formato JSON a stderr coinciden con la guía; nada sensible en los logs.

11. **TUI (la operás vos)** — Lanzá `daedalus` sin subcomando en una terminal interactiva y navegá siguiendo el capítulo de la TUI.
    - Esperado: la TUI abre y los atajos/navegación documentados funcionan tal como se describen.

12. **Recorrido de adopción completo** — Releé el índice (`docs/README.md`) y confirmá que, siguiéndolo de arriba a abajo, una persona nueva llega de "instalar" a "workspace compilado" sin baches ni saltos.
    - Esperado: el recorrido es seguible de punta a punta; no faltan pasos ni hay referencias a cosas inexistentes.

## Resultado

- **PASA** si los 12 pasos coinciden con la guía y con el comportamiento real. Recién entonces se habilita avanzar a 10-02 (sitio) y 10-03 (release) y publicar `v0.1.0`.
- **NO PASA** si algún paso diverge: se documenta la discrepancia, el orquestador la enruta (doc o comportamiento) y se repite el paso afectado tras el fix.
