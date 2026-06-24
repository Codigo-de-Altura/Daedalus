# Validación manual — Prompt Preview (TUI)

> Para alguien **sin experiencia en testing**. Primero preparás unos prompts de ejemplo con comandos, después abrís la interfaz y seguís los casos. Si algo no coincide con "Esperás ver", no intentes arreglarlo: avisá al **orquestador** describiendo qué hiciste y qué viste (Workflow B).

## Preparación

1. Necesitás el binario `daedalus` disponible (probá `daedalus --version`).
2. Abrí una terminal **PowerShell** en una carpeta de prueba vacía e inicializá el workspace:

   ```powershell
   cd $env:TEMP
   Remove-Item -Recurse -Force preview-test -ErrorAction SilentlyContinue
   mkdir preview-test; cd preview-test
   daedalus init
   ```

3. Cargá unos prompts de ejemplo: un fragmento compartido, un prompt **largo que lo incluye** (para probar el scroll y la composición) y un prompt **"roto"** (con una referencia que no existe):

   ```powershell
   daedalus prompt create glossary --kind shared --title "Glossary" --body "**Daedalus**: the scaffolding tool.`n**Prompt**: a reusable text fragment.`n**Agent**: a backend-agnostic AI role."

   daedalus prompt create onboarding --kind global --title "Onboarding guide" --body "# Onboarding`n`n## Glossary`n`n{{include: glossary}}`n`n## Steps`n`n1. Read the project README.`n2. Install the toolchain.`n3. Run the test suite.`n4. Open your first ticket.`n5. Ask questions early.`n`n## Conventions`n`n- Write everything in English.`n- Keep commits small and focused.`n- Prefer clarity over cleverness.`n`n## More`n`nRepeat the steps until comfortable. Welcome aboard!"

   daedalus prompt create broken --kind global --title "Broken prompt" --body "This one points to a fragment that does not exist:`n`n{{include: ghost}}"
   ```

4. Arrancá la interfaz (sin subcomando):

   ```powershell
   daedalus
   ```

   **Esperás ver** la interfaz abrirse en la terminal, con una **lista de prompts** (cada uno muestra su id, su tipo `global`/`shared` y su título). Abajo hay una línea de **ayuda** con los atajos.

> Si en vez de la interfaz ves un mensaje tipo "run in an interactive terminal", es porque la terminal no es interactiva (por ejemplo, salida redirigida). Abrí una terminal normal y volvé a intentar.

## Casos

### Caso 1 — Abrir la previsualización

- **Hacé:** con las flechas `↑`/`↓` (o `k`/`j`) movete hasta **`onboarding`** y presioná **Enter**.
- **Esperás ver:** una pantalla de **preview** que muestra el texto **formateado** (el título `# Onboarding` se ve como un encabezado destacado, la lista numerada y las viñetas se ven prolijas, las palabras en **negrita** resaltan) — no como símbolos sueltos de Markdown.

### Caso 2 — El texto aparece completo y armado (composición resuelta)

- **Hacé:** en la preview de `onboarding`, mirá la sección "Glossary".
- **Esperás ver:** los términos del glosario (**Daedalus**, **Prompt**, **Agent**) **ya metidos adentro** del prompt. **No** debe aparecer ninguna línea cruda tipo `{{include: glossary}}` sin resolver.

### Caso 3 — Desplazarse en un prompt largo

- **Hacé:** seguí en la preview de `onboarding` y usá `↓`/`↑`, o `PgDn`/`PgUp` (también `Espacio`/`b`), para bajar y subir. Probá `G` para ir al final y `g` para volver al principio.
- **Esperás ver:** el contenido se desplaza; podés llegar al final ("Welcome aboard!") y volver al principio. Suele haber un indicador de posición (porcentaje) que cambia al desplazarte.

### Caso 4 — Atajos y ayuda

- **Hacé:** presioná **`?`** para ver la ayuda con todos los atajos. Después presioná **`Esc`** para **cerrar la preview** y volver a la lista.
- **Esperás ver:** la ayuda lista los atajos (moverse, abrir, volver, salir); al presionar `Esc` volvés a la **lista de prompts**. Sin callejones sin salida.

### Caso 5 — No se puede editar desde la preview

- **Hacé:** entrá de nuevo a la preview de cualquier prompt e intentá **escribir o borrar** texto (tecleá letras al azar).
- **Esperás ver:** **no pasa nada** sobre el prompt; la preview es **solo para mirar** (las teclas que no son de navegación se ignoran y no modifican nada). Nota: dentro de la preview, `q` no hace nada — para volver usá `Esc`, y para salir de la app usá `Ctrl+C`.

### Caso 6 — Un prompt con error de composición

- **Hacé:** volvé a la lista (`Esc`), movete hasta **`broken`** y presioná **Enter**.
- **Esperás ver:** un **mensaje de error claro** explicando que no se pudo armar el prompt porque la referencia `ghost` no existe. La aplicación **no** se cierra sola ni muestra texto corrupto: seguís pudiendo presionar `Esc` para volver a la lista.

### Cierre

- **Hacé:** desde la lista, presioná **`q`** (o `Ctrl+C`).
- **Esperás ver:** la aplicación se cierra y volvés a la terminal normal.

## Si algo no coincide

Si algún resultado no coincide con lo esperado, no intentes arreglarlo: avisá al **orquestador** describiendo qué hiciste y qué viste (Workflow B).
