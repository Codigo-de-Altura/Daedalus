import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";

// The site is published to GitHub Pages as a project site, so it lives under
// /Daedalus/. Override with VITE_BASE (e.g. "/" for a custom domain) without
// touching the code.
const base = process.env.VITE_BASE ?? "/Daedalus/";

export default defineConfig({
  base,
  plugins: [react(), tailwindcss()],
  server: {
    // The manual lives in ../docs (repo root). Allow the dev server to read it
    // so import.meta.glob can bundle the markdown during development.
    fs: { allow: [".."] },
  },
  build: {
    target: "es2020",
    cssCodeSplit: true,
  },
});
