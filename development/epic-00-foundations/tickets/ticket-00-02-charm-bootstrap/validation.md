# Validación automática — Ticket 00-02

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- Ticket-00-01 aprobado (módulo Go y layout existentes).
- Go toolchain disponible; acceso a red para resolver dependencias (o módulos ya en caché).
- Se ejecuta desde la raíz del repositorio.

## Checks

1. **Dependencias Charm declaradas** — Comando: `grep -E 'charmbracelet/(bubbletea|lipgloss)' go.mod` · Esperado: Bubble Tea y Lipgloss presentes en `go.mod` (verificar también el resto del set previsto: bubbles, glamour, huh).
2. **Modelo Bubble Tea** — Comando: `grep -RE 'func .*Init\(|func .*Update\(|func .*View\(' cmd/ internal/` · Esperado: existen los tres métodos del modelo Elm (`Init`, `Update`, `View`).
3. **Compilación** — Comando: `go build ./...` · Esperado: compila sin errores, código de salida 0.
4. **Arranque/cierre limpio no interactivo** — Comando: `go build -o daedalus ./cmd/daedalus && printf 'q' | ./daedalus` (o el modo/flag no interactivo definido) · Esperado: el programa arranca, cierra limpio y retorna código de salida 0 sin pánico.
5. **Consistencia de dependencias** — Comando: `go mod verify && go mod tidy && git diff --exit-code go.mod go.sum` · Esperado: `go mod verify` exitoso y sin cambios pendientes tras `tidy`.
6. **Vet** — Comando: `go vet ./...` · Esperado: sin hallazgos, código de salida 0.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1 |
| CA2 | 3 |
| CA3 | 2 |
| CA4 | 4 |
| CA5 | 5 |
| CA6 | 6 |

## Verdict

- **APPROVED** si todos los checks pasan.
- **REJECTED** si alguno falla → hallazgos con severidad, comportamiento observado y esperado (para `observations.md`).
