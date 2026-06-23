# Prompt Composition — Usage Guide

> Authored and maintained by C-3PO, technical writer for Daedalus.
> _To be completed by C-3PO after implementation._

## Overview

_To be completed by C-3PO after implementation._

This guide explains how to compose prompts by **including reusable fragments** (conventions, glossary, style) so you keep your prompts DRY and consistent.

## How to use

_To be completed by C-3PO after implementation._

- Reference a shared fragment from inside another prompt using the inclusion syntax.
- Resolve a prompt to obtain its final composed text.
- Nest inclusions (a fragment can include other fragments).

## Options / flags

_To be completed by C-3PO after implementation._

- Inclusion directive: references a shared prompt by its `id` / slug.
- Resolution is recursive and deterministic.

## Notes & limitations

- Inclusion resolution is deterministic: the same set of prompts always yields the same composed text.
- Inclusion cycles and references to non-existent prompts are reported as explicit errors.
- Composition never rewrites the source prompt files.
- Phase 1: Daedalus configures the AI structure; it does not execute agents.
