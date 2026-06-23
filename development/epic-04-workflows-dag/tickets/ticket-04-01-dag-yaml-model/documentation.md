# Workflow DAG YAML Model — Usage Guide

> Authored and maintained by C-3PO, technical writer for Daedalus. Part of the growing end-user usage guide. Written for the person who uses the Daedalus product, not for internals.

## Overview

_To be completed by C-3PO after implementation._

## How to use

Daedalus represents a workflow as a declarative DAG stored as a YAML file under `.daedalus/workflows/<name>.yaml`. Each workflow is an ordered list of **phases**; every phase references the agent that runs it, the artifacts it consumes and produces, a validation gate, and the phases it depends on.

A phase looks like this:

```yaml
phases:
  - id: spec
    agent: analyst
    inputs:  [brief]
    outputs: [spec]
    gate: spec-gate
    depends_on: [brief]
```

You author and edit workflows by editing these YAML files (or through Daedalus' editing operations). Daedalus loads the file into its canonical model and writes it back as clean, deterministic YAML.

## Options / flags

_To be completed by C-3PO after implementation (commands, flags and editing operations exposed for workflows)._

## Notes & limitations

- **Phase 1: Daedalus configures the AI structure; it does not execute agents.** This feature models and edits the workflow definition only — it does not run the pipeline or invoke any agent.
- Workflow definitions are backend-agnostic; they are not tied to Claude Code or any specific agent runtime.
- Serialization is deterministic (stable, ordered keys) to keep git diffs clean.
- Semantic graph validation (cycles, missing artifacts, unknown agents) is covered separately by the DAG validation feature.
