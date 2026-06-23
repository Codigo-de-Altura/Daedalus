# Validación automática — Ticket 00-03

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- Tickets 00-01 y 00-02 aprobados (módulo, layout y bootstrap existentes).
- Herramientas disponibles: `make`, `docker`, `docker compose` y el toolchain de Go.
- Se ejecuta desde la raíz del repositorio (idealmente un clon limpio para validar onboarding).

## Checks

1. **Archivos de scaffolding presentes** — Comando: `test -f Makefile && test -f Dockerfile && test -f docker-compose.yml && test -f .dockerignore` · Esperado: los cuatro archivos existen.
2. **Targets del Makefile** — Comando: `grep -E '^(build|test|lint|run|fmt|tidy):' Makefile` · Esperado: están definidos los targets estándar.
3. **make build** — Comando: `make build` · Esperado: compila sin errores, código de salida 0.
4. **make test** — Comando: `make test` · Esperado: ejecuta los tests sin fallos, código de salida 0.
5. **make lint** — Comando: `make lint` · Esperado: corre el linter, código de salida 0.
6. **Build de imagen Docker** — Comando: `docker build -t daedalus:dev .` · Esperado: la imagen se construye sin errores.
7. **Contenedor ejecuta el binario** — Comando: `docker run --rm daedalus:dev --help` (o ejecución no interactiva equivalente) · Esperado: el binario corre dentro del contenedor y retorna código de salida 0.
8. **Compose válido** — Comando: `docker compose config` · Esperado: la configuración es válida y define el servicio de dev de Daedalus.
9. **Comando de onboarding documentado** — Comando: `grep -iE 'make .*(setup|bootstrap|onboard|dev)|onboarding' README.md` · Esperado: el README referencia un comando único de onboarding.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1 |
| CA2 | 2, 3, 4, 5 |
| CA3 | 6, 7 |
| CA4 | 8 |
| CA5 | 9 |
| CA6 | 1 |
| CA7 | 9 |

## Verdict

- **APPROVED** si todos los checks pasan.
- **REJECTED** si alguno falla → hallazgos con severidad, comportamiento observado y esperado (para `observations.md`).
