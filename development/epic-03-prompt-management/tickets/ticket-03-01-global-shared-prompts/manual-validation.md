# Validación manual — Global & Shared Prompts

> Para alguien **sin experiencia en testing**. Copiá y pegá cada comando tal cual, en orden, y compará lo que ves con el bloque **"Esperás ver"**. Si algo no coincide, no intentes arreglarlo: avisá al **orquestador** describiendo qué hiciste y qué viste (Workflow B).

## Preparación

1. Necesitás el binario `daedalus` ya instalado y disponible en tu terminal (escribí `daedalus --version` y debería responder algo como `daedalus 0.1.0-dev`). Si no, pedíselo a quien te ayuda.
2. Abrí una terminal **PowerShell** en una carpeta de prueba vacía (no uses un proyecto real). Por ejemplo:

   ```powershell
   cd $env:TEMP
   Remove-Item -Recurse -Force prompts-test -ErrorAction SilentlyContinue
   mkdir prompts-test; cd prompts-test
   ```

3. Inicializá el workspace de Daedalus:

   ```powershell
   daedalus init
   ```

   **Esperás ver** (al final, la línea de texto plano) y una carpeta `.daedalus/` recién creada:

   ```
   Created Daedalus workspace at .daedalus from scratch.
   ```

> Nota: además del texto importante, cada comando imprime unas líneas técnicas en formato JSON (los *logs*). Podés **ignorarlas**: lo que tenés que mirar es la línea de texto plano (y el "código de salida" cuando el caso lo pida).

## Casos

### Caso 1 — Crear un prompt global

```powershell
daedalus prompt create project-style --kind global --title "Project style" --body "Write in clear, concise English."
```

**Esperás ver:**

```
Created prompt "project-style" (global) at .daedalus/prompts/project-style.md.
```

Y dentro de `.daedalus/prompts/` un archivo nuevo `project-style.md` (nombre en minúsculas y guiones, derivado del id).

### Caso 2 — Crear un prompt compartido (shared)

```powershell
daedalus prompt create glossary --kind shared --title "Glossary" --body "Term: Daedalus is the scaffolding tool."
```

**Esperás ver:**

```
Created prompt "glossary" (shared) at .daedalus/prompts/glossary.md.
```

Otro archivo `glossary.md` dentro de `.daedalus/prompts/`, **sin** que el del Caso 1 desaparezca o cambie.

### Caso 3 — Listar los prompts (y filtrar por tipo)

```powershell
daedalus prompt list
```

**Esperás ver** los dos prompts, cada uno con su **tipo** (`global` o `shared`):

```
Prompts (2):
  glossary	shared	Glossary
  project-style	global	Project style
```

Ahora filtrá solo los compartidos:

```powershell
daedalus prompt list --kind shared
```

**Esperás ver** solo `glossary`:

```
Prompts (1, kind=shared):
  glossary	shared	Glossary
```

### Caso 4 — Editar un prompt

```powershell
daedalus prompt edit glossary --body "Term: Daedalus is the scaffolding tool.`nTerm: a prompt is a reusable text fragment."
```

(El `` `n `` de PowerShell agrega un salto de línea dentro del cuerpo.)

**Esperás ver:**

```
Edited prompt "glossary" at .daedalus/prompts/glossary.md.
```

El cambio queda guardado en `glossary.md`. El otro prompt (`project-style`) debe quedar **igual que antes**. Para confirmar el contenido editado:

```powershell
daedalus prompt show glossary
```

**Esperás ver** el archivo con su encabezado y el cuerpo con las dos líneas:

```
---
id: glossary
kind: shared
title: Glossary
---
Term: Daedalus is the scaffolding tool.
Term: a prompt is a reusable text fragment.
```

### Caso 5 — Intentar duplicar un id

```powershell
daedalus prompt create glossary --kind shared --title "Dup"
```

**Esperás ver** un **mensaje de error** claro avisando que ya existe (y el `glossary` original **no** cambia):

```
daedalus: prompt already exists: "glossary" — not overwritten
```

Para confirmar que fue tratado como error, mirá el código de salida (debe ser **2**):

```powershell
echo $LASTEXITCODE
```

### Caso 6 — Un id inválido es rechazado

```powershell
daedalus prompt create Bad_ID --kind global --title "X" --body "y"
```

**Esperás ver** un error que explica que el id debe ser *kebab-case* (minúsculas y guiones), y **no** se crea ningún archivo:

```
daedalus: prompt "Bad_ID" is invalid; it was not created:
  - id: observed "Bad_ID"; expected kebab-case: lowercase letters/digits in dash-separated segments (e.g. my-prompt)
```

### Caso 7 — Eliminar un prompt

```powershell
daedalus prompt remove project-style
```

**Esperás ver:**

```
Removed prompt "project-style" from .daedalus/prompts/project-style.md.
```

Desaparece **solo** `project-style.md` de `.daedalus/prompts/`; `glossary.md` sigue ahí. Confirmalo:

```powershell
daedalus prompt list
```

**Esperás ver** que `project-style` ya no está en la lista.

## Si algo no coincide

Si algún resultado no coincide con lo esperado, no intentes arreglarlo: avisá al **orquestador** describiendo qué comando corriste y qué viste (Workflow B).
