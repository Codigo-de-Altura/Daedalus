# Build idempotente y no destructivo

> **Epic:** epic-06-compile-claude-adapter · **Tipo:** backend · **Implementador:** Obi-Wan · **Validador:** Yoda · **Origen:** RF-6.3 · **Estilo:** SDD

---

## Contexto

La compilación (`daedalus build`, RF-6.1) escribe artefactos nativos al backend (p. ej. `.claude/`, RF-6.2). Esa escritura debe ser **segura**: re-compilar no puede romper el trabajo del usuario. Este ticket especifica dos garantías:

1. **Idempotencia** — Correr `build` N veces sobre el mismo input produce el mismo resultado y no genera ruido (sin cambios espurios entre corridas).
2. **No destrucción** — La compilación **no destruye cambios manuales** que vivan **fuera del área gestionada** por Daedalus.

Estas garantías dependen del **determinismo** del adaptador (RF-6.2, RNF-5) y son la base de la seguridad de escritura (RNF-8: operaciones no destructivas por defecto; preview/confirm en RF-6.4).

Referencias: PRD RF-6.3, RNF-5 (determinismo / golden files), RNF-8 (safety), §13 (riesgo "compilación destructiva" → mitigación: build idempotente, áreas gestionadas marcadas, preview/diff).

## Feature / Qué se construye

La **estrategia de escritura** del build que garantiza idempotencia y no-destrucción:

- **Área gestionada marcada.** Daedalus delimita claramente qué porción del output es **gestionada** (generada y sobre-escribible por el build) frente a contenido manual del usuario. Solo el área gestionada se regenera.
- **Idempotencia.** Re-ejecutar el build sin cambios en la definición canónica deja el output **sin cambios** (mismo contenido byte-a-byte; ninguna escritura espuria).
- **No-destrucción fuera del área gestionada.** Cambios manuales del usuario fuera del área gestionada **se preservan** a través de re-compilaciones.
- **Determinismo verificable.** Apoyado en golden files: mismo input → mismo output, condición necesaria para que la idempotencia sea comprobable.

## Requerimientos

- **REQ-1.** El build define y respeta un **área gestionada**: solo regenera lo que produce, sin tocar contenido fuera de ella.
- **REQ-2.** **Idempotencia:** correr `build` dos o más veces sobre la misma definición canónica produce un output **idéntico** y no introduce cambios espurios (verificable por diff vacío entre corridas).
- **REQ-3.** **No-destrucción:** cambios manuales del usuario **fuera del área gestionada** se conservan tras re-compilar.
- **REQ-4.** **Determinismo (golden files):** el output es reproducible byte-a-byte para un input dado (RNF-5); el build no introduce datos volátiles (timestamps, rutas absolutas, orden no determinista).
- **REQ-5.** El comportamiento por defecto es **seguro** (RNF-8): no se destruye nada inadvertidamente; la confirmación previa a escribir se cubre vía el preview/diff (RF-6.4).
- **REQ-6.** Si una re-compilación **sí** debe modificar contenido del área gestionada (porque cambió la definición canónica), el cambio es **acotado al área gestionada** y reproducible.
- **REQ-7.** El criterio de qué es "área gestionada" es **explícito y estable** (no depende del orden de ejecución ni del entorno).

## Criterios de aceptación

- [ ] Correr `daedalus build` dos veces sin cambios deja el output idéntico (idempotente; diff vacío).
- [ ] Un cambio manual fuera del área gestionada **sobrevive** a una re-compilación.
- [ ] El mismo input canónico produce el mismo output byte-a-byte (determinismo / golden files).
- [ ] El build no escribe datos volátiles que rompan el determinismo.
- [ ] El área gestionada está claramente delimitada y solo ella se regenera.
- [ ] El comportamiento por defecto no es destructivo.

## Fuera de alcance

- El **comando** y su orquestación → RF-6.1 (ticket-06-01).
- El **mapeo canónico → Claude Code** y el formato concreto de los artefactos → RF-6.2 (ticket-06-02).
- La **UI del diff/preview** y el flujo de confirmación del usuario → RF-6.4 (ticket-06-04). (Este ticket garantiza la seguridad de la escritura; el preview interactivo es de RF-6.4.)
- Estrategia de migración/upgrade de un `.daedalus/` existente → fuera de este epic.

## Referencias

- `PRD.md` — RF-6.3, RNF-5 (determinismo / golden files), RNF-8 (safety), §13 (mitigación de compilación destructiva).
- `init.md` — §7 (idempotencia: `build`/`sync` no destruye cambios manuales fuera del área gestionada; YAML determinista).
- `CLAUDE.md` — §7 (idempotencia y no destrucción; determinismo / golden files).
- `epic.md` — Epic 06, criterios "mismo input → mismo output" y "re-compilar no destruye cambios manuales".
