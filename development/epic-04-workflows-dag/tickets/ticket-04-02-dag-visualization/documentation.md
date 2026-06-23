# Workflow DAG Visualization — Usage Guide

> Authored and maintained by C-3PO, technical writer for Daedalus. Part of the growing end-user usage guide. Written for the person who uses the Daedalus product, not for internals.

## Overview

_To be completed by C-3PO after implementation._

## How to use

In the Daedalus TUI you can open a workflow and see it drawn as a graph (DAG) right in your terminal, instead of reading the raw YAML. Each **node** is a phase of the workflow and shows which **agent** runs it; each **edge** is a dependency between phases, so you can read the pipeline from start to finish (for example: brief → spec → architecture → epics → tickets → validation → docs).

Navigate to the workflows area, select a workflow, and the DAG view renders it.

## Options / flags

_To be completed by C-3PO after implementation (navigation entry points and any keyboard shortcuts for the DAG view)._

## Notes & limitations

- **Phase 1: Daedalus configures the AI structure; it does not execute agents.** The DAG view is read-only presentation — it does not run the workflow or invoke any agent.
- The view shows the workflow as defined; it does not edit it. Editing is done on the workflow YAML.
- Semantic validation (cycles, missing artifacts, unknown agents) is a separate feature; the view shows the graph, not the validation report.
- Designed for workflows of moderate size, such as the built-in `sdd-default`.
