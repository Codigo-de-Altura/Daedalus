# Manifest & Project `init.md` — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated. This is the initial outline to be filled with real behavior once the ticket passes validation.

## Overview

_To be completed by C-3PO after implementation._

When you initialize a workspace, Daedalus generates two root artifacts: `daedalus.yaml` (the manifest holding your project name, target backend(s), version, and conventions) and a project `init.md` (the master guideline for your target project). Both are generated deterministically.

## How to use

_Steps the end user follows._

1. Run `daedalus init` in your repository.
2. Open `.daedalus/daedalus.yaml` to review or adjust project name, backend(s), and conventions.
3. Open `.daedalus/init.md` as the entry point for your project's AI structure.

## Options / flags

_If applicable._

## Notes & limitations

- The manifest is a human-readable, diff-friendly YAML file you can version in git.
- Phase 1: Daedalus configures the AI structure; it does not execute agents.
