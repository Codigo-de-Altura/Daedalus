# Validación manual — Ticket 02-02 — Importar / clonar y editar un agente

> Para alguien **sin experiencia en testing**. Seguí los pasos tal cual. Cada caso te dice qué comando correr y qué deberías ver.
>
> Los comandos están en **PowerShell** (Windows). El binario `daedalus` debe estar instalado y disponible en tu PATH (`daedalus --version` debería responder). Los logs en formato JSON salen por **stderr** y no afectan el resultado; si te molestan en consola, agregá `2>$null` al final del comando.

## Preparación

1. Conseguí el binario de Daedalus ya instalado (o pedíle al orquestador cómo obtenerlo). Verificá con `daedalus --version`.
2. Abrí una terminal en una carpeta de prueba **vacía** (no uses un repo importante):

   ```powershell
   cd $env:TEMP
   mkdir prueba-daedalus-0202; cd prueba-daedalus-0202
   ```

3. Asegurate de que en esa carpeta todavía **no** exista una carpeta `.daedalus/`:

   ```powershell
   Test-Path .daedalus   # debe imprimir: False
   ```

## Casos

### Caso 1 — Clonar un agente del catálogo a un id nuevo
- **Hacé:**

  ```powershell
  daedalus agent clone analyst analyst-custom
  Get-ChildItem -Recurse .daedalus\agents\analyst-custom
  ```

- **Esperá ver:** un mensaje de confirmación y los archivos del clon creados:

  ```
  Materialized agent "analyst-custom" at .daedalus/agents/analyst-custom (created 2 files).
  ```

  Y dentro de `.daedalus\agents\analyst-custom\` deben aparecer dos archivos: `agent.yaml` (la definición) y `prompt.md` (su prompt).

### Caso 2 — Editar el clon (rol y un parámetro) y que el cambio persista
- **Hacé:**

  ```powershell
  daedalus agent edit analyst-custom --role "Drafts specs for the mobile team" --set-param temperature=0.2
  Get-Content .daedalus\agents\analyst-custom\agent.yaml
  ```

- **Esperá ver:** una confirmación de la edición:

  ```
  Edited agent "analyst-custom" at .daedalus/agents/analyst-custom.
  ```

  Y al mostrar el `agent.yaml`, la línea `role:` debe decir `Drafts specs for the mobile team` y bajo `parameters:` debe aparecer `temperature: "0.2"` (los parámetros editados por CLI se guardan como texto, por eso va entre comillas).

### Caso 3 — El clon es independiente del original (editarlo no toca el catálogo)
- **Hacé:** materializá el agente original `analyst` (el del catálogo) y compará su rol con el del clon que editaste en el Caso 2:

  ```powershell
  daedalus agent add analyst
  Get-Content .daedalus\agents\analyst\agent.yaml | Select-String "role:"
  Get-Content .daedalus\agents\analyst-custom\agent.yaml | Select-String "role:"
  ```

- **Esperá ver:** que el `analyst` original conserva su rol del catálogo (`Turns a brief into a spec/PRD.`) y el clon `analyst-custom` tiene tu rol editado (`Drafts specs for the mobile team`). Editar el clon **no** cambió el original.

### Caso 4 — Clonar sobre un id que ya existe (no destructivo)
- **Hacé:** intentá clonar otro agente del catálogo usando el id `analyst-custom`, que ya existe del Caso 1:

  ```powershell
  daedalus agent clone architect analyst-custom
  Get-Content .daedalus\agents\analyst-custom\agent.yaml | Select-String "role:"
  ```

- **Esperá ver:** que Daedalus **no** lo pisa en silencio — avisa que ya existe y que no sobreescribió:

  ```
  Agent "analyst-custom" already exists at .daedalus/agents/analyst-custom — not overwritten (skipped 2 files).
  ```

  Y la última línea debe seguir mostrando tu rol editado del Caso 2 (`Drafts specs for the mobile team`), confirmando que el clon existente se preservó y no fue reemplazado por el de `architect`.

## Casos extra (opcionales)

### Edición inválida — rechazada sin romper el archivo
- **Hacé:** intentá poner un rol vacío (edición inválida) y verificá que el archivo siguió intacto:

  ```powershell
  daedalus agent edit analyst-custom --role ""
  echo $LASTEXITCODE
  Get-Content .daedalus\agents\analyst-custom\agent.yaml | Select-String "role:"
  ```

- **Esperá ver:** un error accionable (campo / observado / esperado) y código de salida `2`:

  ```
  daedalus: agent "analyst-custom" is invalid; the edit was not applied:
    - role: observed empty; expected a non-empty role/description
  ```

  Después de eso, `echo $LASTEXITCODE` debe imprimir `2`, y la línea `role:` debe seguir mostrando tu rol del Caso 2: la edición inválida **no** dejó el archivo a medio escribir ni lo vació.

### Editar sin ningún flag de edición
- **Hacé:**

  ```powershell
  daedalus agent edit analyst-custom
  echo $LASTEXITCODE
  ```

- **Esperá ver:** un error de uso (no un no-op silencioso) y código de salida `2`:

  ```
  daedalus: agent edit requires at least one edit flag (--role, --prompt, --prompt-file, --set-param, --remove-param)
  ```

### Editar un agente que no existe en el workspace
- **Hacé:**

  ```powershell
  daedalus agent edit ghost --role "anything"
  echo $LASTEXITCODE
  ```

- **Esperá ver:** un error que dice que el agente no existe y que lo clones o lo agregues primero, con código de salida `2`:

  ```
  daedalus: agent not found in catalog: "ghost"
  the agent must already exist in the workspace; clone or add it first
  ```

### Preview de un clon — ver qué haría sin escribir nada
- **Hacé:**

  ```powershell
  daedalus agent clone planner planner-custom --preview
  Test-Path .daedalus\agents\planner-custom
  ```

- **Esperá ver:** un preview de los archivos que se crearían, y que `Test-Path` imprime `False` (no se escribió nada).

### Id de destino inválido (no kebab-case)
- **Hacé:**

  ```powershell
  daedalus agent clone analyst Bad_Id
  echo $LASTEXITCODE
  ```

- **Esperá ver:** un error claro y código de salida `2`:

  ```
  daedalus: destination agent id "Bad_Id" is not valid kebab-case
  ```

## Limpieza

```powershell
cd $env:TEMP; Remove-Item -Recurse -Force prueba-daedalus-0202
```

## Si algo no coincide

Si algo no coincide con lo esperado → **reportar al orquestador (Workflow B)**, describiendo qué hiciste, qué esperabas ver y qué viste en su lugar.
