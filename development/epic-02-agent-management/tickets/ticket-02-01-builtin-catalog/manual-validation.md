# Validación manual — Ticket 02-01 — Catálogo built-in de agentes

> Para alguien **sin experiencia en testing**. Seguí los pasos tal cual. Cada caso te dice qué comando correr y qué deberías ver.
>
> Los comandos están en **PowerShell** (Windows). El binario `daedalus` debe estar instalado y disponible en tu PATH (`daedalus --version` debería responder). Los logs en formato JSON salen por **stderr** y no afectan el resultado; si te molestan en consola, agregá `2>$null` al final del comando.

## Preparación

1. Conseguí el binario de Daedalus ya instalado (o pedíle al orquestador cómo obtenerlo). Verificá con `daedalus --version`.
2. Abrí una terminal en una carpeta de prueba **vacía** (no uses un repo importante):

   ```powershell
   cd $env:TEMP
   mkdir prueba-daedalus; cd prueba-daedalus
   ```

3. Asegurate de que en esa carpeta todavía **no** exista una carpeta `.daedalus/`:

   ```powershell
   Test-Path .daedalus   # debe imprimir: False
   ```

## Casos

### Caso 1 — Ver la lista de agentes built-in
- **Hacé:**

  ```powershell
  daedalus agent list
  ```

- **Esperá ver:** una lista que incluye al menos estos cinco nombres, cada uno con una breve descripción de su rol:

  ```
  Built-in agents (5):
    analyst     Turns a brief into a spec/PRD.
    architect   Defines the architecture from the spec.
    documenter  Produces derived documentation.
    planner     Derives epics and tickets from spec and architecture.
    validator   Verifies artifacts and implementation against gates and criteria.
  ```

### Caso 2 — Materializar un agente en el workspace
- **Hacé:**

  ```powershell
  daedalus agent add analyst
  Get-ChildItem -Recurse .daedalus\agents
  ```

- **Esperá ver:** un mensaje de confirmación y los archivos del agente creados:

  ```
  Materialized agent "analyst" at .daedalus/agents/analyst (created 2 files).
  ```

  Y dentro de `.daedalus\agents\analyst\` deben aparecer dos archivos: `agent.yaml` (la definición) y `prompt.md` (su prompt).

### Caso 3 — Intentar materializar el mismo agente otra vez (no destructivo)
- **Hacé:** primero editá el archivo a mano (para comprobar que no lo pisa), después volvé a materializar:

  ```powershell
  Add-Content .daedalus\agents\analyst\agent.yaml "# MI EDICION MANUAL"
  daedalus agent add analyst
  Get-Content .daedalus\agents\analyst\agent.yaml | Select-String "MI EDICION MANUAL"
  ```

- **Esperá ver:** que Daedalus **no** lo pisa en silencio — te avisa que ya existe y que no sobreescribió:

  ```
  Agent "analyst" already exists at .daedalus/agents/analyst — not overwritten (skipped 2 files).
  ```

  Y la última línea debe **encontrar** tu edición (`# MI EDICION MANUAL`), confirmando que tu cambio manual sobrevivió. **No** debería sobreescribir sin avisar.

## Casos extra (opcionales)

### Preview — ver qué haría sin escribir nada
- **Hacé:**

  ```powershell
  daedalus agent add planner --preview
  Test-Path .daedalus\agents\planner
  ```

- **Esperá ver:** un preview de los archivos que se crearían, y que `Test-Path` imprime `False` (no se escribió nada).

### Error accionable — agente inexistente
- **Hacé:**

  ```powershell
  daedalus agent add bogus
  echo $LASTEXITCODE
  ```

- **Esperá ver:** un mensaje de error claro (`agent not found in catalog: "bogus"`) con una pista para correr `daedalus agent list`, y un código de salida `2`.

## Limpieza

```powershell
cd $env:TEMP; Remove-Item -Recurse -Force prueba-daedalus
```

## Si algo no coincide

Si algo no coincide con lo esperado → **reportar al orquestador (Workflow B)**, describiendo qué hiciste, qué esperabas ver y qué viste en su lugar.
