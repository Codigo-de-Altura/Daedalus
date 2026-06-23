# Prompt Preview — Usage Guide

> Authored and maintained by C-3PO, technical writer for Daedalus.
> _To be completed by C-3PO after implementation._

## Overview

_To be completed by C-3PO after implementation._

This guide explains how to preview a prompt's **final rendered text** in the Daedalus TUI, with all inclusions resolved, before it is compiled to your agent backend.

## How to use

_To be completed by C-3PO after implementation._

- Select a prompt in the TUI.
- Open its preview to see the composed, rendered Markdown.
- Scroll through long content.
- Close the preview to return.

## Options / flags

_To be completed by C-3PO after implementation._

- Keyboard shortcuts to open/close the preview and scroll (shown in contextual help).
- The preview is read-only.

## Notes & limitations

- The preview shows the composed prompt (inclusions resolved), rendered as Markdown via Glamour.
- The preview is read-only; it does not edit or persist the prompt.
- If a prompt has a composition error (cycle or missing reference), the preview shows a clear error message instead of broken content.
- Phase 1: Daedalus configures the AI structure; it does not execute agents.
