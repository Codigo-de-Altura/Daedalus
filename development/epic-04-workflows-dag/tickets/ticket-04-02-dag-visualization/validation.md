# Validación automática — Ticket 04-02

> La corre **Leia**. Solo reporta verdict (APPROVED/REJECTED); no implementa.

## Precondiciones

- El binario `daedalus` está compilado y la TUI es lanzable (o la vista es ejercitable vía la suite de tests de la TUI / golden render del proyecto).
- Existe al menos un workflow cargable, incluido el `sdd-default` (brief→spec→arquitectura→epics→tickets→validación→docs), disponible en `.daedalus/workflows/`.
- Existe un workflow vacío o no cargable para el check de degradación.

## Checks

1. **Vista del DAG disponible** — Comando: navegar en la TUI hasta seleccionar un workflow y abrir su vista de DAG · Esperado: se renderiza una vista de grafo con nodos y aristas.
2. **Nodos muestran fase y agente** — Comando: inspeccionar el render de los nodos · Esperado: cada nodo muestra el `id` de la fase y el `agent` asociado.
3. **Aristas reflejan dependencias** — Comando: comparar las aristas dibujadas con las dependencias (`depends_on`) del workflow · Esperado: cada dependencia aparece como arista, con la dirección/orden del pipeline.
4. **Render del `sdd-default`** — Comando: visualizar el `sdd-default` · Esperado: las fases se ven en el orden brief→spec→arquitectura→epics→tickets→validación→docs, de forma legible.
5. **Solo lectura** — Comando: intentar editar o ejecutar desde la vista · Esperado: la vista no ofrece editar ni ejecutar; es solo de presentación.
6. **Consistencia de estilo** — Comando: revisar el render (golden/captura) contra el tema de la TUI · Esperado: estilo Lipgloss consistente; la vista no se rompe con un DAG de tamaño moderado.
7. **Degradación con gracia** — Comando: abrir la vista con un workflow vacío o no cargable · Esperado: se muestra un estado claro, sin panic ni glitches.

## Mapeo a criterios de aceptación

| Criterio | Check |
|---|---|
| CA1 | 1 |
| CA2 | 2 |
| CA3 | 3 |
| CA4 | 4 |
| CA5 | 5 |
| CA6 | 6 |
| CA7 | 7 |

## Verdict

- APPROVED si todos los checks pasan; si no, REJECTED con hallazgos (severidad, observado, esperado).
