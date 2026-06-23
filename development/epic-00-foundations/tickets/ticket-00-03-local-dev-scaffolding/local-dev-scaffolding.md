# Ticket 00-03 — Scaffolding de desarrollo local

> **Epic:** epic-00-foundations · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** HANDOFF §5 (terreno de Obi-Wan: scaffolding de dev local) · epic.md (ticket-00-03) · CLAUDE.md §2 (Docker/scripts donde tenga sentido) · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Uno de los criterios del epic es que un desarrollador nuevo levante el entorno local con un solo comando documentado. Sobre el módulo y el bootstrap ya creados, este ticket aporta el andamiaje de desarrollo local: Docker, Docker Compose, un Makefile y scripts de onboarding que estandarizan build, test, run y formateo. El objetivo es onboarding *seamless* y reproducible en Windows, macOS y Linux (RNF-3).

## Feature / Qué se construye

El conjunto de herramientas de desarrollo local que envuelve al proyecto Go: un `Dockerfile` que construye/ejecuta Daedalus de forma reproducible, un `docker-compose.yml` para levantar el entorno de dev, un `Makefile` con targets estándar (build, test, lint, run, fmt, tidy) y, si aplica, scripts de onboarding. Estas piezas no agregan lógica de producto; orquestan el toolchain existente para que cualquier dev arranque con un comando.

## Requerimientos

- R1 — Existe un `Makefile` con targets estándar y nombres convencionales: como mínimo `build`, `test`, `lint`, `run`, `fmt` y `tidy`, cada uno delegando en el toolchain de Go correspondiente.
- R2 — Existe un `Dockerfile` que construye el binario de Daedalus de forma reproducible (preferentemente multi-stage) y produce una imagen ejecutable.
- R3 — Existe un `docker-compose.yml` que define el servicio de desarrollo de Daedalus y permite levantarlo con un solo comando.
- R4 — Existe un único comando documentado de onboarding (target del Makefile o script) que prepara el entorno (p. ej. dependencias + build) desde un clon limpio.
- R5 — Los scripts de soporte (si los hay) son portables y residen en una ubicación convencional (p. ej. `scripts/`); el conjunto funciona en Windows, macOS y Linux (RNF-3).
- R6 — El scaffolding reutiliza el toolchain de Go ya presente; no duplica ni reimplementa lógica de build.
- R7 — El README (o doc base) referencia el comando único de onboarding, manteniendo la documentación de producto en inglés (CLAUDE.md §1).
- R8 — El `.dockerignore` evita copiar artefactos innecesarios (binarios, `.git`, etc.) al contexto de build.

## Criterios de aceptación

- [ ] CA1 — Existen `Makefile`, `Dockerfile` y `docker-compose.yml` en las ubicaciones esperadas.
- [ ] CA2 — `make build` compila el proyecto; `make test` ejecuta los tests; `make lint` corre el linter; `make run` ejecuta el binario.
- [ ] CA3 — El `Dockerfile` construye una imagen sin errores y el contenedor ejecuta el binario de Daedalus.
- [ ] CA4 — `docker compose` levanta el servicio de dev de Daedalus.
- [ ] CA5 — Existe un comando único de onboarding documentado que, desde un clon limpio, deja el entorno listo para build/test.
- [ ] CA6 — Existe `.dockerignore` que excluye artefactos innecesarios del contexto de build.
- [ ] CA7 — El README referencia el comando de onboarding.

## Fuera de alcance

- Pipeline de CI (ticket-00-05); aunque CI puede reutilizar estos targets, su definición vive en su propio ticket.
- Configuración del linter en sí más allá de invocarlo (la elección/config del linter se materializa donde corresponda; CI en ticket-00-05).
- Logging estructurado (ticket-00-04) y cualquier feature de producto.

## Referencias

- HANDOFF §5 — Terreno de Obi-Wan (Docker, Makefile, scripts)
- epic-00-foundations/epic.md (ticket-00-03; criterio "un solo comando")
- PRD RNF-3 (portabilidad Windows/macOS/Linux), RNF-1 (binario único)
- CLAUDE.md §2 (Docker/scripts donde tenga sentido), §1 (idioma)
