# Built-in `sdd-default` Workflow — Usage Guide

> Authored and maintained by C-3PO, technical writer for Daedalus. Part of the growing end-user usage guide. Written for the person who uses the Daedalus product, not for internals.

## Overview

_To be completed by C-3PO after implementation._

## How to use

Daedalus ships with a factory workflow, `sdd-default.yaml`, available in your workspace under `.daedalus/workflows/`. It encodes the default SDD pipeline so you do not have to write it by hand:

```
brief → spec → architecture → epics → tickets → (external implementation) → validation → docs
```

Each phase names the agent that runs it (analyst, architect, planner, planner, validator, documenter), the artifacts it consumes and produces, and a validation gate. You can view it like any other workflow, and use it as the starting point for your project's pipeline.

## Options / flags

_To be completed by C-3PO after implementation (how the default workflow is provided and referenced)._

## Notes & limitations

- **Phase 1: Daedalus configures the AI structure; it does not execute agents.** `sdd-default.yaml` is a definition of the pipeline — Daedalus provides and validates it, but does not run the agents.
- The **implementation** step is external: a developer or agent performs it in the backend, outside Daedalus; the workflow only reflects where the implementation artifact enters the pipeline to be validated.
- The workflow is backend-agnostic and is written in the canonical DAG YAML format.
- It is provided as a deterministic, git-friendly file (stable, ordered keys) and passes Daedalus' DAG validation (no cycles, no missing artifacts, no unknown agents).
