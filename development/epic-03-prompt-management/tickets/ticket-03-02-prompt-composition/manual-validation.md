# Validación manual — Prompt Composition (inclusiones)

> Para alguien **sin experiencia en testing**. Copiá y pegá cada comando tal cual, en orden, y compará lo que ves con el bloque **"Esperás ver"**. Si algo no coincide, no intentes arreglarlo: avisá al **orquestador** describiendo qué hiciste y qué viste (Workflow B).

## Preparación

1. Necesitás el binario `daedalus` disponible (probá `daedalus --version`).
2. Abrí una terminal **PowerShell** en una carpeta de prueba vacía e inicializá el workspace:

   ```powershell
   cd $env:TEMP
   Remove-Item -Recurse -Force compose-test -ErrorAction SilentlyContinue
   mkdir compose-test; cd compose-test
   daedalus init
   ```

3. Creá un fragmento compartido y un prompt que lo **incluye**. La inclusión se escribe con una línea propia `{{include: <id>}}`:

   ```powershell
   daedalus prompt create glossary --kind shared --title "Glossary" --body "Term: Daedalus is the scaffolding tool.`nTerm: a prompt is a reusable text fragment."
   daedalus prompt create onboarding --kind global --title "Onboarding" --body "# Onboarding`n`n{{include: glossary}}`n`nWelcome aboard."
   ```

   (El `` `n `` de PowerShell agrega saltos de línea.)

> Nota: cada comando imprime también unas líneas técnicas en formato JSON (los *logs*); podés **ignorarlas**.

## Casos

### Caso 1 — Ver el prompt "crudo" (sin resolver)

```powershell
daedalus prompt show onboarding
```

**Esperás ver** el contenido tal como se guardó, **con la directiva sin resolver** (`{{include: glossary}}` aparece literal):

```
---
id: onboarding
kind: global
title: Onboarding
---
# Onboarding

{{include: glossary}}

Welcome aboard.
```

### Caso 2 — Ver el prompt "compuesto" (inclusiones resueltas)

```powershell
daedalus prompt render onboarding
```

**Esperás ver** el texto final **ensamblado**: en el lugar de `{{include: glossary}}` aparece el contenido del glosario, y **no** queda ninguna línea `{{include: ...}}` sin resolver:

```
# Onboarding

Term: Daedalus is the scaffolding tool.
Term: a prompt is a reusable text fragment.

Welcome aboard.
```

> La diferencia clave: `show` = crudo (con la directiva), `render` = compuesto (resuelto). El archivo en disco **no** cambia: `render` arma el texto al vuelo sin reescribir nada.

### Caso 3 — El mismo render dos veces da lo mismo (determinismo)

```powershell
daedalus prompt render onboarding
daedalus prompt render onboarding
```

**Esperás ver** exactamente la **misma** salida las dos veces.

### Caso 4 — Una referencia que no existe falla con un error claro

```powershell
daedalus prompt create broken --kind global --title "Broken" --body "{{include: ghost}}"
daedalus prompt render broken
```

**Esperás ver** un error que **nombra** el id faltante (`ghost`) y desde dónde se lo referencia (`broken`):

```
daedalus: included prompt "ghost" not found (referenced by "broken")
```

Y el código de salida debe ser **2**:

```powershell
echo $LASTEXITCODE
```

### Caso 5 — Un bucle de inclusiones se detecta (no se cuelga)

```powershell
daedalus prompt create loop-a --kind shared --title "Loop A" --body "A start`n{{include: loop-b}}"
daedalus prompt create loop-b --kind shared --title "Loop B" --body "B start`n{{include: loop-a}}"
daedalus prompt render loop-a
```

**Esperás ver** un error que muestra el **ciclo** (la cadena `loop-a -> loop-b -> loop-a`). El programa **termina enseguida**; no se queda colgado ni se cierra de golpe:

```
daedalus: include cycle detected: loop-a -> loop-b -> loop-a
```

(Código de salida **2**.)

### Caso 6 — DRY: un fragmento compartido vive en un solo archivo

```powershell
daedalus prompt create role-policy --kind shared --title "Policy" --body "Always write commit messages in English."
daedalus prompt create agent-one --kind global --title "Agent one" --body "{{include: role-policy}}"
daedalus prompt create agent-two --kind global --title "Agent two" --body "{{include: role-policy}}"
daedalus prompt render agent-one
daedalus prompt render agent-two
```

**Esperás ver** que **ambos** renders muestran el texto de la política, aunque el fragmento existe **una sola vez** en disco. Confirmalo listando los archivos:

```powershell
Get-ChildItem .daedalus\prompts -Name
```

**Esperás ver** un único `role-policy.md` (no hay copias duplicadas del fragmento).

## Si algo no coincide

Si algún resultado no coincide con lo esperado, no intentes arreglarlo: avisá al **orquestador** describiendo qué comando corriste y qué viste (Workflow B).
