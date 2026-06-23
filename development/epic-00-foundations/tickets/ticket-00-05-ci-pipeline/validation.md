# Validación automática — Ticket 00-05

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- Tickets 00-01 a 00-04 aprobados (módulo, bootstrap, scaffolding local y logging existentes).
- Go toolchain y el linter definido disponibles localmente.
- Un parser de YAML disponible para validar el workflow.
- Se ejecuta desde la raíz del repositorio.

## Checks

1. **Workflow de CI presente** — Comando: `ls .github/workflows/*.yml .github/workflows/*.yaml` · Esperado: existe al menos un archivo de workflow de CI.
2. **Triggers** — Comando: `grep -E 'on:|push:|pull_request:' .github/workflows/*.y*ml` · Esperado: el workflow se dispara en `push` (rama principal) y `pull_request`.
3. **Etapas build/test/lint** — Comando: `grep -E 'go build|go test|golangci-lint|go vet|gofmt' .github/workflows/*.y*ml` · Esperado: aparecen comandos de build, test y lint (directos o vía targets de Makefile).
4. **Versión de Go coherente** — Comando: `grep -E 'go-version|^go ' .github/workflows/*.y*ml go.mod` · Esperado: la versión de Go del workflow coincide con la de `go.mod`.
5. **YAML válido** — Comando: validar el/los archivo(s) con un parser YAML (p. ej. `python -c "import sys,yaml; [yaml.safe_load(open(f)) for f in sys.argv[1:]]" .github/workflows/*.y*ml`) · Esperado: parsea sin errores.
6. **Gate alcanzable localmente** — Comando: `go build ./... && go test ./... && make lint` (o el lint definido) · Esperado: las tres etapas que CI ejecutará pasan localmente, código de salida 0.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1, 5 |
| CA2 | 2 |
| CA3 | 3 |
| CA4 | 4 |
| CA5 | 6 |
| CA6 | 5 |

## Verdict

- **APPROVED** si todos los checks pasan.
- **REJECTED** si alguno falla → hallazgos con severidad, comportamiento observado y esperado (para `observations.md`).
