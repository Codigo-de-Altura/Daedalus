# Ticket 00-01 — Módulo Go y layout del repositorio

> **Epic:** epic-00-foundations · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda
> **Origen:** init.md §4.1 · PRD §11 (arquitectura: un solo proyecto Go) · CLAUDE.md §2 · **Estilo:** SDD (plano, no guía de implementación).

## Contexto

Daedalus es un único proyecto Go (TUI/CLI con Charm) que separa conceptualmente *frontend* (capa TUI) de *backend* (core: dominio, adaptadores, compilación, persistencia, logging). Antes de poder construir cualquier feature de producto (init, agentes, prompts, workflows, build) hace falta un módulo Go inicializado y un layout estándar que todos los epics posteriores asuman como base. Este ticket es el cimiento del repositorio: sin él, ningún otro ticket compila.

## Feature / Qué se construye

La estructura base del repositorio Go de Daedalus: el módulo Go declarado, el layout de directorios que separa el punto de entrada del binario (`cmd/`) del código no exportable (`internal/`), y un README base que orienta a un desarrollador nuevo. El binario mínimo de Daedalus debe compilar y ejecutarse (aunque no haga nada de producto todavía).

## Requerimientos

- R1 — El repositorio declara un módulo Go con un *module path* canónico y coherente (estilo `github.com/<owner>/daedalus`) y fija la versión de Go soportada.
- R2 — Existe un punto de entrada de binario bajo `cmd/daedalus/` (paquete `main`) que compila y produce un ejecutable nombrado `daedalus`.
- R3 — Existe un directorio `internal/` reservado para el core (dominio, adaptadores, compilación, persistencia, logging), de modo que el código del core no sea importable desde fuera del módulo.
- R4 — El binario mínimo arranca, no entra en pánico y termina con código de salida 0 en una ejecución no interactiva (p. ej. imprimiendo versión/identificación o un mensaje base).
- R5 — Existe un README base en la raíz que describe qué es Daedalus, cómo compilar y cómo ejecutar el binario, en inglés (documentación de producto/código en inglés por CLAUDE.md §1).
- R6 — El layout sigue convenciones idiomáticas de Go y no incluye aún código de producto (init, agentes, TUI completa, etc.), que pertenecen a epics posteriores.
- R7 — Existe un `.gitignore` adecuado para Go que excluye binarios compilados y artefactos de build.

## Criterios de aceptación

- [ ] CA1 — `go build ./...` compila sin errores desde la raíz del repo.
- [ ] CA2 — Existe `go.mod` con un module path válido y una versión de Go declarada.
- [ ] CA3 — Existen los directorios `cmd/daedalus/` (con paquete `main`) e `internal/`.
- [ ] CA4 — El binario compilado se ejecuta y retorna código de salida 0 en modo no interactivo.
- [ ] CA5 — `go vet ./...` no reporta problemas.
- [ ] CA6 — Existe un `README.md` en la raíz con secciones de descripción, build y run.
- [ ] CA7 — Existe `.gitignore` que ignora el binario compilado.

## Fuera de alcance

- Dependencias del ecosistema Charm y bootstrap de Bubble Tea (ticket-00-02).
- Scaffolding de dev local: Docker, Compose, Makefile, scripts (ticket-00-03).
- Baseline de logging estructurado (ticket-00-04).
- Pipeline de CI (ticket-00-05).
- Cualquier feature de producto (init, build, agentes, prompts, workflows).

## Referencias

- init.md §4.1 (layout del repo)
- PRD §11 — Arquitectura técnica (un solo proyecto Go; núcleo vs. UI)
- CLAUDE.md §2 (qué es Daedalus), §1 (idioma: código y docs de producto en inglés)
- epic-00-foundations/epic.md (ticket-00-01)
