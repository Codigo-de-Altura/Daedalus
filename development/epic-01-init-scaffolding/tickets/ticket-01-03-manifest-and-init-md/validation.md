# Validación automática — Ticket 01-03

> La corre **Yoda**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y disponible.
- Directorio temporal vacío como repo objetivo.
- Un parser YAML disponible para validar el manifiesto.

## Checks

1. **Manifiesto es YAML válido** — Comando: ejecutar `daedalus init` y parsear `.daedalus/daedalus.yaml` · Esperado: el archivo es YAML válido y carga sin errores.
2. **Claves mínimas del manifiesto** — Comando: inspeccionar las claves de nivel raíz del manifiesto · Esperado: presentes `name`, `version`, `backends`, `conventions`.
3. **Campo `backends` con default** — Comando: leer la clave `backends` · Esperado: presente, con un valor por defecto coherente con el MVP (Claude Code).
4. **`init.md` base generado** — Comando: verificar el contenido de `.daedalus/init.md` · Esperado: contiene el lineamiento base con el mapa de estructura `.daedalus/` (§4.2) y una sección de convenciones.
5. **Determinismo (golden)** — Comando: generar el workspace dos veces con el mismo input y comparar `daedalus.yaml` e `init.md` byte a byte · Esperado: archivos idénticos en ambas corridas.
6. **Orden estable de claves** — Comando: comparar el orden de claves del `daedalus.yaml` entre dos corridas · Esperado: orden idéntico y estable.
7. **No sobrescritura** — Comando: editar `daedalus.yaml` con una marca conocida, re-ejecutar `init`, releer · Esperado: la marca persiste (no se sobrescribe contenido existente).

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1, 2 |
| CA2 | 4 |
| CA3 | 5 |
| CA4 | 6 |
| CA5 | 3 |
| CA6 | 7 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
