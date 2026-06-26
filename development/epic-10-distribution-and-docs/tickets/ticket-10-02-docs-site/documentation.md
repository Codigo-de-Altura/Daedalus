# Documentation pointer — Ticket 10-02

This ticket publishes the product manual under `docs/` as a documentation **site**.
This file is only a **pointer**; the manual itself is the content.

## What this ticket delivers

The `docs/` manual is built as an **MkDocs Material** site and deployed to
**GitHub Pages**:

- `mkdocs.yml` — the site configuration (theme, navigation, the chapters under
  `docs/`).
- `.github/workflows/docs.yml` — builds the site with `mkdocs build --strict`
  and deploys it to GitHub Pages when `docs/` or `mkdocs.yml` changes on `main`.

For an end user, the published manual lives at the repository's **GitHub Pages**
URL.

## Where to read more

- The manual itself: [`docs/README.md`](../../../../docs/README.md) (the index).
- The CI detail for the docs pipeline (and how to preview the site locally with
  `mkdocs serve` / `mkdocs build --strict`):
  [`docs/contributing/continuous-integration.md`](../../../../docs/contributing/continuous-integration.md).

> Note: the manual is built with `--strict`, so every internal link must resolve
> and every page must be reachable from the navigation. Keep the index in sync
> when adding or renaming chapters.
