# Compilación: comando `daedalus build` / `sync`

> **Epic:** epic-06-compile-claude-adapter · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-6.1 · **Estilo:** SDD

---

## Contexto

Daedalus mantiene una **definición canónica** (agnóstica del backend) en `.daedalus/` —agentes, prompts, workflows y backlog SDD— como fuente de verdad. Para que esa definición sea utilizable por un backend de agentes (p. ej. Claude Code), debe **compilarse** a su formato nativo. Este ticket define el **comando** que dispara esa compilación: `daedalus build` (alias `sync`).

El comando es el punto de entrada del pipeline de compilación. Orquesta: carga de la definición canónica → validación → invocación del/los adaptador(es) registrados → escritura de artefactos nativos. La implementación concreta del adaptador Claude Code (RF-6.2), la idempotencia/no-destrucción (RF-6.3) y el diff/preview (RF-6.4) viven en tickets hermanos; este ticket establece la **superficie del comando** y su orquestación.

Referencias: PRD §8.1 (fuente de verdad → compilación por adaptador), RF-6.1, RNF-5 (determinismo), RNF-7 (extensibilidad), RNF-8 (seguridad/safety).

## Feature / Qué se construye

Un comando de CLI `daedalus build` (con alias `sync`) que compila la definición canónica del workspace `.daedalus/` al formato nativo del backend configurado en `daedalus.yaml`.

Responsabilidades del comando:

- Localizar el workspace `.daedalus/` (a partir del directorio actual o del repo objetivo).
- Cargar y validar la definición canónica antes de compilar (falla temprano si es inválida).
- Resolver el/los backend(s) objetivo desde el manifiesto `daedalus.yaml`.
- Seleccionar el adaptador (`Compiler`) correspondiente desde un registro de adaptadores.
- Invocar la compilación y reportar el resultado (artefactos generados, resumen).
- Devolver códigos de salida claros (éxito / error de validación / error de escritura).

> El comando es el **orquestador** de la compilación; delega el mapeo canónico→nativo al adaptador (RF-6.2), la estrategia de escritura no destructiva a RF-6.3, y la previsualización a RF-6.4.

## Requerimientos

- **REQ-1.** Existe un comando `daedalus build` invocable desde la CLI, con **alias `sync`** equivalente.
- **REQ-2.** El comando detecta el workspace `.daedalus/` y falla con mensaje claro si no existe.
- **REQ-3.** Antes de compilar, **valida** la definición canónica; si es inválida, **aborta sin escribir** y reporta los errores.
- **REQ-4.** Lee el/los backend(s) objetivo desde `daedalus.yaml` (MVP: Claude Code) y selecciona el adaptador correspondiente vía un registro.
- **REQ-5.** Si el backend solicitado no tiene adaptador registrado, falla con mensaje claro (no escribe nada).
- **REQ-6.** El comando expone, como mínimo: una opción para previsualizar sin escribir (delegada a RF-6.4) y la operación normal de escritura.
- **REQ-7.** Reporta un **resumen** del resultado: backend objetivo, artefactos creados/actualizados, y estado final.
- **REQ-8.** Códigos de salida diferenciados: `0` éxito; distinto de `0` para error de validación vs. error de compilación/escritura.
- **REQ-9.** El comando es **determinista**: el mismo workspace produce el mismo resultado (consistente con RNF-5; la garantía profunda se valida en RF-6.3).
- **REQ-10.** La orquestación es **agnóstica del backend**: agregar un backend nuevo no debe requerir modificar el comando, solo registrar un adaptador (consistente con RNF-7; la interfaz se define en RF-6.2).

## Criterios de aceptación

- [ ] `daedalus build` compila la definición canónica al formato nativo del backend configurado.
- [ ] `daedalus sync` se comporta como alias de `daedalus build`.
- [ ] Con un workspace ausente o inválido, el comando **aborta sin escribir** y muestra un error accionable.
- [ ] El backend objetivo se resuelve desde `daedalus.yaml` y se enruta al adaptador correcto.
- [ ] Un backend sin adaptador registrado produce un error claro y ninguna escritura.
- [ ] Al finalizar, se muestra un resumen con backend, artefactos y estado.
- [ ] Los códigos de salida distinguen éxito, error de validación y error de compilación/escritura.

## Fuera de alcance

- El **mapeo canónico → Claude Code** (frontmatter, comandos, settings) → RF-6.2 (ticket-06-02).
- La estrategia de **idempotencia y no-destrucción** y los golden files → RF-6.3 (ticket-06-03).
- La **UI/render del diff/preview** → RF-6.4 (ticket-06-04).
- Adaptadores para otros backends (Codex, Gemini CLI…) → post-MVP.
- Ejecución/orquestación de agentes en vivo → Fase 2+.

## Referencias

- `PRD.md` — RF-6.1, §8.1 (fuente de verdad → compilación por adaptador), RNF-5, RNF-7, RNF-8.
- `init.md` — §5 (contrato de artefactos: artefactos nativos producidos por `daedalus build`), §7 (convenciones: idempotencia/determinismo).
- `CLAUDE.md` — §6 (estructura `development/`), §7 (filosofía SDD).
- `epic.md` — Epic 06, criterios de aceptación del epic.
