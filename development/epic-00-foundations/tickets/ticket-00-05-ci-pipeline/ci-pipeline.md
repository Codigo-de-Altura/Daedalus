# Ticket 00-05 — Pipeline de CI (build, test, lint)

> **Epic:** epic-00-foundations · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** RNF-2 (performance/UX) · RNF-5 (determinismo/reproducibilidad) · epic.md (ticket-00-05) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

El último cimiento del epic es la integración continua: cada cambio debe validarse de forma reproducible con build, test y lint. Sobre el módulo, el bootstrap, el scaffolding local y el baseline de logging ya creados, este ticket define un pipeline de CI que ejecuta esas validaciones automáticamente en cada push/PR. Garantiza que el repo se mantiene compilable, testeado y limpio, y que la base es reproducible (RNF-5) para el resto de los epics.

## Feature / Qué se construye

Una definición de pipeline de CI (p. ej. GitHub Actions) que, ante cada cambio, ejecuta de forma reproducible: build del proyecto, suite de tests y lint. El pipeline reutiliza el toolchain y, donde tenga sentido, los targets del Makefile (ticket-00-03), fija la versión de Go y reporta el resultado como gate de calidad. No introduce lógica de producto; es infraestructura de validación.

## Requerimientos

- R1 — Existe una definición de workflow de CI en la ubicación convencional del proveedor (p. ej. `.github/workflows/`), en formato declarativo.
- R2 — El pipeline se dispara en eventos relevantes: al menos `push` a la rama principal y `pull_request`.
- R3 — El pipeline ejecuta, como mínimo, tres etapas verificables: **build** (`go build ./...`), **test** (`go test ./...`) y **lint** (linter de Go, p. ej. `golangci-lint` o `go vet` + `gofmt -l`).
- R4 — La versión de Go usada en CI queda fijada y es coherente con la declarada en `go.mod` (reproducibilidad, RNF-5).
- R5 — El pipeline falla (no verde) si build, test o lint fallan, actuando como gate de calidad.
- R6 — En la medida de lo posible, las etapas reutilizan los targets del Makefile (ticket-00-03) para evitar divergencia entre CI y dev local.
- R7 — El pipeline usa caché de dependencias de Go donde sea razonable, sin comprometer la reproducibilidad.
- R8 — La definición es válida sintácticamente (YAML del proveedor) y no referencia secretos para las etapas básicas de build/test/lint.

## Criterios de aceptación

- [ ] CA1 — Existe un archivo de workflow de CI en `.github/workflows/` (o equivalente del proveedor) con YAML válido.
- [ ] CA2 — El workflow define triggers de `push` (rama principal) y `pull_request`.
- [ ] CA3 — El workflow incluye etapas de build, test y lint con sus comandos correspondientes.
- [ ] CA4 — La versión de Go en el workflow coincide con la de `go.mod`.
- [ ] CA5 — Localmente, `go build ./...`, `go test ./...` y el lint definido pasan (lo que CI ejecutará), confirmando que el gate es alcanzable.
- [ ] CA6 — El YAML del workflow parsea sin errores.

## Fuera de alcance

- Etapas de release/publicación de binarios, firma o distribución (Fase posterior).
- Cobertura de tests obligatoria, badges, matrices multi-OS exhaustivas (pueden añadirse luego; el baseline es build+test+lint).
- Despliegue o entrega continua (CD).
- Definición del scaffolding local en sí (ticket-00-03); aquí solo se reutiliza.

## Referencias

- PRD RNF-2 (performance/UX sin regresiones), RNF-5 (determinismo/reproducibilidad)
- epic-00-foundations/epic.md (ticket-00-05; "CI valida build + test + lint de forma reproducible")
- ticket-00-03-local-dev-scaffolding (targets del Makefile reutilizables)
- CLAUDE.md §2 (CI en el terreno de backend/Obi-Wan), §1 (idioma)
