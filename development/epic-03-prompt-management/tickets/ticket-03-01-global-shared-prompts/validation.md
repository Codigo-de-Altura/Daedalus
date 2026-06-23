# Validación — Global & Shared Prompts

> La corre **Yoda** (validador backend). Solo reporta; no implementa ni arregla.

## Precondiciones

- Repo con el core de Daedalus compilable (`go build ./...` sin errores).
- Workspace `.daedalus/` inicializable en un directorio temporal de prueba.
- Carpeta `.daedalus/prompts/` disponible (creada por el feature o por `init`).
- Suite de tests del core ejecutable.

## Checks

### Check 1 — Compila el core
- **Comando:** `go build ./...`
- **Esperado:** compila sin errores ni warnings que rompan el build.

### Check 2 — Crear prompt global persiste archivo
- **Comando:** ejecutar el caso/test que crea un prompt `global` con `id` válido y lista el contenido de `.daedalus/prompts/`.
- **Esperado:** existe un archivo `.md` en `.daedalus/prompts/` con nombre `kebab-case` derivado del `id`, `kind: global` en sus metadatos.

### Check 3 — Crear prompt shared persiste archivo
- **Comando:** ejecutar el caso/test que crea un prompt `shared`.
- **Esperado:** existe el archivo `.md` correspondiente con `kind: shared`.

### Check 4 — Listar y filtrar por kind
- **Comando:** ejecutar la operación de listado total y con filtro `global` / `shared`.
- **Esperado:** el listado total incluye ambos prompts; el filtro `global` devuelve solo globales y `shared` solo compartidos.

### Check 5 — Editar no destruye otros archivos
- **Comando:** editar título/cuerpo de un prompt y comparar el resto de los archivos del workspace antes/después.
- **Esperado:** solo cambia el archivo editado; los demás permanecen byte-idénticos.

### Check 6 — Id duplicado falla
- **Comando:** intentar crear dos prompts con el mismo `id`.
- **Esperado:** la segunda creación retorna error explícito; el archivo original no se sobrescribe.

### Check 7 — Determinismo
- **Comando:** crear (o regenerar) el mismo prompt con idéntico input dos veces y comparar la salida.
- **Esperado:** archivos byte-idénticos (mismo orden de metadatos y formato).

### Check 8 — Slug/id inválido rechazado
- **Comando:** crear un prompt con `id` vacío o con caracteres no `kebab-case`.
- **Esperado:** error explícito; no se crea archivo.

### Check 9 — Eliminar remueve solo su archivo
- **Comando:** eliminar un prompt y verificar el contenido de `.daedalus/prompts/`.
- **Esperado:** desaparece solo el archivo del prompt eliminado.

### Check 10 — Suite de tests
- **Comando:** `go test ./...`
- **Esperado:** todos los tests del área de prompts pasan.

## Mapeo a criterios

| Check | Criterio de aceptación |
|---|---|
| 2 | Crear prompt global persiste archivo |
| 3 | Crear prompt shared persiste archivo |
| 4 | Listar y filtrar por kind |
| 5 | Editar sin alterar otros archivos |
| 9 | Eliminar remueve solo su archivo |
| 6 | Id duplicado falla sin sobrescribir |
| 2, 9 | Nombre de archivo kebab-case derivado del id |
| 7 | Determinismo |
| 8 | Slug/id inválido rechazado |
| 1, 10 | Build y tests verdes |

## Verdict

**Estado:** _APPROVED / REJECTED_ (a completar por Yoda al ejecutar).

**Hallazgos** (uno por ítem):

| # | Severidad | Observado | Esperado |
|---|---|---|---|
| | | | |
