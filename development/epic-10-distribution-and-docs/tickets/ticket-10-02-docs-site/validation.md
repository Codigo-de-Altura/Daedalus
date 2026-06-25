# Validación automática — Ticket 10-02

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El árbol `docs/` y el contenido de la guía (10-01) están presentes.
- Las herramientas del sitio (MkDocs + tema Material + plugins) se pueden instalar con las versiones fijadas declaradas en el repo.

## Checks

1. **Build estricto del sitio** — Comando: instalar las herramientas fijadas y correr el build en modo estricto (enlaces rotos = error) sobre `docs/` · Esperado: construye sin errores ni warnings de enlaces rotos.
2. **Navegación válida y completa** — Comando: revisar `mkdocs.yml` y contrastar cada entrada de `nav` contra los archivos de `docs/` · Esperado: toda entrada resuelve a una página existente; toda página relevante está alcanzable; no hay huérfanos; refleja el recorrido de adopción.
3. **Workflow de deploy** — Comando: revisar el workflow de GitHub Actions · Esperado: construye el sitio y despliega a GitHub Pages al actualizar la documentación en `main`; usa permisos mínimos de Pages; no interfiere con otros workflows.
4. **Reproducibilidad** — Comando: revisar que las versiones de MkDocs/tema/plugins estén fijadas · Esperado: versiones pinneadas; el build no depende de un entorno local concreto.
5. **Separación de audiencias preservada** — Comando: revisar la nav · Esperado: uso (guide) y contribución (contributing) quedan distinguidos, como en el índice del manual.
6. **Idioma** — Comando: revisar config/nav/comentarios · Esperado: inglés.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1 |
| CA2 | 2 |
| CA3 | 3 |
| CA4 | 2, 5 |
| CA5 | 4 |
| CA6 | 3 |
| CA7 | 6 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
- Nota: la verificación del **deploy real** a Pages (que la URL sirva el sitio) puede requerir el push a `main` y la activación de Pages en el repo; Yoda valida el build estricto local y la corrección del workflow. El deploy efectivo se confirma tras el merge.
