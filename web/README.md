# Daedalus — website

The marketing landing page and documentation site for Daedalus, built as a
single-page app and published to GitHub Pages.

- **Stack:** Vite + React + TypeScript + Tailwind CSS v4 + Framer Motion.
- **Routing:** one SPA serves the landing (`/`) and the docs (`/docs/*`).
- **Docs source:** the manual is authored as markdown in the repo's top-level
  [`docs/`](../docs). The site bundles those files at build time and renders
  them with the Daedalus theme — so the markdown stays the single source of
  truth (maintained by C-3PO) and the site styles it.

## Develop

```sh
cd web
npm install
npm run dev
```

The dev server runs at the configured base path (`/Daedalus/`). Open the URL it
prints.

## Build

```sh
npm run build      # type-check + production build into web/dist
npm run preview    # serve the production build locally
```

## Deploy

Pushing changes to `web/**` or `docs/**` on `main` triggers
[`.github/workflows/pages.yml`](../.github/workflows/pages.yml), which builds the
SPA and deploys `web/dist` to GitHub Pages.

The site is served from the project Pages path
(`https://<org>.github.io/Daedalus/`), so the build uses `VITE_BASE=/Daedalus/`.
If the site moves to a custom domain at the root, set `VITE_BASE=/` (in the
workflow and locally) and update `APP_BASE` in [`public/404.html`](public/404.html).

## Project layout

```
web/
  public/
    404.html        # SPA deep-link fallback for GitHub Pages
    favicon.svg     # the labyrinth mark
  src/
    components/      # UI primitives, motion helpers, landing sections
    pages/           # Landing, Docs, NotFound
    lib/             # site copy/data, docs content layer, helpers
    index.css        # design tokens (theme) + base styles + motifs
    docs.css         # markdown prose + syntax theme
```
