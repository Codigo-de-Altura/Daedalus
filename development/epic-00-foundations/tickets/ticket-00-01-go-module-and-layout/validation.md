# Validación automática — Ticket 00-01

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- Go toolchain instalado y disponible en el `PATH` (versión compatible con la declarada en `go.mod`).
- Se ejecuta desde la raíz del repositorio de Daedalus.

## Checks

1. **Módulo declarado** — Comando: `test -f go.mod && grep -E '^module ' go.mod && grep -E '^go ' go.mod` · Esperado: existe `go.mod` con una línea `module <path>` válida y una línea `go <versión>`.
2. **Layout de directorios** — Comando: `test -d cmd/daedalus && test -d internal` · Esperado: ambos directorios existen.
3. **Paquete main del binario** — Comando: `grep -R '^package main' cmd/daedalus/` · Esperado: al menos un archivo en `cmd/daedalus/` declara `package main`.
4. **Compilación** — Comando: `go build ./...` · Esperado: salida sin errores, código de salida 0.
5. **Vet** — Comando: `go vet ./...` · Esperado: sin hallazgos, código de salida 0.
6. **Binario ejecutable** — Comando: `go build -o daedalus ./cmd/daedalus && ./daedalus --help` (o ejecución no interactiva equivalente) · Esperado: el binario arranca, no entra en pánico y retorna código de salida 0.
7. **README base** — Comando: `test -f README.md && grep -iE 'build|run' README.md` · Esperado: existe `README.md` con instrucciones de build y run.
8. **gitignore** — Comando: `test -f .gitignore && grep -E 'daedalus' .gitignore` · Esperado: existe `.gitignore` que ignora el binario compilado.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 4 |
| CA2 | 1 |
| CA3 | 2, 3 |
| CA4 | 6 |
| CA5 | 5 |
| CA6 | 7 |
| CA7 | 8 |

## Verdict

- **APPROVED** si todos los checks pasan.
- **REJECTED** si alguno falla → hallazgos con severidad, comportamiento observado y esperado (para `observations.md`).
