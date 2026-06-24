# Validación manual — Ticket 02-03 — Importar agentes desde archivos locales

> Para alguien **sin experiencia en testing**. Seguí los pasos tal cual. Cada caso te dice qué comando correr y qué deberías ver.
>
> Los comandos están en **PowerShell** (Windows). El binario `daedalus` debe estar instalado y disponible en tu PATH (`daedalus --version` debería responder). Los logs en formato JSON salen por **stderr** y no afectan el resultado; si te molestan en consola, agregá `2>$null` al final del comando.

## Preparación

1. Conseguí el binario de Daedalus ya instalado (o pedíle al orquestador cómo obtenerlo). Verificá con `daedalus --version`.
2. Abrí una terminal en una carpeta de prueba **vacía** (no uses un repo importante):

   ```powershell
   cd $env:TEMP
   mkdir prueba-daedalus-0203; cd prueba-daedalus-0203
   ```

3. Creá un par de agentes de **Claude Code** de prueba (archivos Markdown con frontmatter) en una carpeta `.claude\agents`. Estos son los orígenes que vamos a importar:

   ```powershell
   mkdir .claude\agents -Force | Out-Null

   @'
---
name: reviewer
description: Reviews pull requests for correctness and style
tools: Read, Grep
model: opus
color: blue
---

# Reviewer

You review changes and report issues, one finding at a time.
'@ | Set-Content -Encoding utf8 .claude\agents\reviewer.md

   @'
---
name: drafter
description: Drafts release notes from merged changes
model: sonnet
---

# Drafter

You turn a list of merged changes into clear release notes.
'@ | Set-Content -Encoding utf8 .claude\agents\drafter.md
   ```

4. Asegurate de que todavía **no** exista un workspace `.daedalus/` en esta carpeta:

   ```powershell
   Test-Path .daedalus   # debe imprimir: False
   ```

## Casos

### Caso 1 — Importar un solo archivo de Claude Code
- **Hacé:**

  ```powershell
  daedalus agent import .claude\agents\reviewer.md
  Get-ChildItem -Recurse .daedalus\agents\reviewer
  ```

- **Esperá ver:** una línea de importación y un resumen:

  ```
    + reviewer imported to .daedalus/agents/reviewer (created 2 files).
  Import summary: 1 imported, 0 already existed, 0 failed.
  ```

  Y dentro de `.daedalus\agents\reviewer\` deben aparecer dos archivos: `agent.yaml` y `prompt.md`.

### Caso 2 — Verificar la conversión Claude Code → canónico (campos mapeados y descartados)
- **Hacé:**

  ```powershell
  Get-Content .daedalus\agents\reviewer\agent.yaml
  Get-Content .daedalus\agents\reviewer\prompt.md
  ```

- **Esperá ver:** en `agent.yaml`, que los campos se mapearon así:
  - `id: reviewer` (de `name`, en kebab-case)
  - `role: Reviews pull requests for correctness and style` (de `description`)
  - bajo `parameters:`, `model: opus` (de `model`)
  - **no** debe aparecer `tools` ni `color` por ningún lado (se descartan a propósito: son específicos de Claude Code).

  Y en `prompt.md`, el cuerpo Markdown del archivo original (el texto que iba después del segundo `---`).

### Caso 3 — Importar un directorio completo (`.claude/agents/`)
- **Hacé:** primero borrá lo importado en el Caso 1 para partir limpio, después importá toda la carpeta:

  ```powershell
  Remove-Item -Recurse -Force .daedalus
  daedalus agent import .claude\agents
  ```

- **Esperá ver:** una línea por cada agente válido y el resumen (el orden de las líneas puede variar):

  ```
    + drafter imported to .daedalus/agents/drafter (created 2 files).
    + reviewer imported to .daedalus/agents/reviewer (created 2 files).
  Import summary: 2 imported, 0 already existed, 0 failed.
  ```

### Caso 4 — Importar otra vez sobre lo ya importado (no destructivo)
- **Hacé:** sin borrar nada, volvé a importar la misma carpeta:

  ```powershell
  daedalus agent import .claude\agents
  ```

- **Esperá ver:** que Daedalus **no** pisa lo existente: marca cada agente como ya existente y no sobreescribe:

  ```
    = drafter already exists at .daedalus/agents/drafter — not overwritten (skipped 2 files).
    = reviewer already exists at .daedalus/agents/reviewer — not overwritten (skipped 2 files).
  Import summary: 0 imported, 2 already existed, 0 failed.
  ```

## Casos extra (opcionales)

### Preview — ver qué se importaría sin escribir nada
- **Hacé:** partí de un workspace limpio y corré el preview:

  ```powershell
  Remove-Item -Recurse -Force .daedalus
  daedalus agent import .claude\agents --preview
  Test-Path .daedalus
  ```

- **Esperá ver:** un preview de los agentes que se importarían, y que `Test-Path` imprime `False` (no se escribió nada):

  ```
  Preview of importing 2 agent(s):
    + drafter -> .daedalus/agents/drafter
    + reviewer -> .daedalus/agents/reviewer
  ```

### Origen inválido — se reporta y no se importa a medias
- **Hacé:** creá un agente inválido (sin `description`, por lo que el rol queda vacío) e intentá importar la carpeta:

  ```powershell
  @'
---
name: broken
---

# Broken

This agent has no description, so its role is empty.
'@ | Set-Content -Encoding utf8 .claude\agents\broken.md

  Remove-Item -Recurse -Force .daedalus -ErrorAction SilentlyContinue
  daedalus agent import .claude\agents
  echo $LASTEXITCODE
  ```

- **Esperá ver:** que los agentes válidos **sí** se importan, pero el inválido se reporta con `!` y su motivo, y el resumen marca 1 fallo. El código de salida debe ser `2`:

  ```
    + drafter imported to .daedalus/agents/drafter (created 2 files).
    + reviewer imported to .daedalus/agents/reviewer (created 2 files).
    ! .claude/agents/broken.md: agent "broken" has an empty role
  Import summary: 2 imported, 0 already existed, 1 failed.
  ```

  (El orden de las líneas `+` y `!` puede variar; lo importante es que aparezca el fallo del inválido y que los válidos se hayan importado.)

## Limpieza

```powershell
cd $env:TEMP; Remove-Item -Recurse -Force prueba-daedalus-0203
```

## Si algo no coincide

Si algo no coincide con lo esperado → **reportar al orquestador (Workflow B)**, describiendo qué hiciste, qué esperabas ver y qué viste en su lugar.
