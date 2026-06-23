# Brief to Spec — Usage Guide

> Audience: end users of Daedalus. Authored and maintained by **C-3PO** as the feature is implemented and validated. This is the initial outline to be filled with real behavior once the ticket passes validation.

## Overview

_To be completed by C-3PO after implementation._

Capture a project brief and turn it into a spec/PRD that lives in `.daedalus/specs/`. Daedalus manages the definition (brief, the link to the *analyst* agent, and where the spec lands); you then refine the spec by hand.

## How to use

_Steps the end user follows._

1. Capture your brief as the entry artifact of the SDD pipeline.
2. Daedalus links the brief to the *analyst* agent definition (the `brief → spec/PRD` step of `sdd-default.yaml`).
3. Run the *analyst* agent in your own backend to generate the spec/PRD into `.daedalus/specs/<slug>.md`.
4. Refine the spec by hand — it is yours to edit.

## Options / flags

_If applicable._

## Notes & limitations

- Phase 1: Daedalus configures the AI structure; it does not execute agents. Generating the spec content from the brief is done by running the *analyst* agent in your backend, outside Daedalus.
- Specs are markdown and editable; Daedalus will not destructively overwrite your manual edits.
