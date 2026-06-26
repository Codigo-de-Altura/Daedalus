/**
 * Documentation content layer.
 *
 * The manual is authored as plain markdown in the repo's top-level docs/ (the
 * same files C-3PO maintains). We bundle them as raw strings at build time and
 * render them inside the SPA, so the source of truth stays in markdown while the
 * site styles it with the Daedalus theme.
 */

const modules = import.meta.glob("../../../docs/**/*.md", {
  query: "?raw",
  import: "default",
  eager: true,
}) as Record<string, string>;

const PREFIX = "../../../docs/";

/** Map every markdown file to a docs slug ("" for the index, README.md). */
export const docsBySlug: Record<string, string> = {};
for (const [path, content] of Object.entries(modules)) {
  const rel = path.slice(PREFIX.length).replace(/\.md$/, "");
  const slug = rel === "README" ? "" : rel;
  docsBySlug[slug] = content;
}

export function getDoc(slug: string): string | undefined {
  return docsBySlug[normalizeSlug(slug)];
}

export function normalizeSlug(slug: string): string {
  const s = slug.replace(/^\/+|\/+$/g, "");
  return s === "README" ? "" : s;
}

/** Navigation, mirroring mkdocs.yml so both stay in lockstep. */
export interface NavItem {
  label: string;
  slug: string;
}
export interface NavSection {
  title: string;
  items: NavItem[];
}

export const nav: NavSection[] = [
  {
    title: "Overview",
    items: [{ label: "Introduction", slug: "" }],
  },
  {
    title: "Getting started",
    items: [
      { label: "Installation", slug: "getting-started/installation" },
      { label: "Quickstart", slug: "getting-started/quickstart" },
    ],
  },
  {
    title: "Guide",
    items: [
      { label: "Concepts", slug: "guide/concepts" },
      { label: "Core workflow", slug: "guide/core-workflow" },
      { label: "Command reference", slug: "guide/command-reference" },
      { label: "Command line", slug: "guide/command-line" },
      { label: "Navigating the interface", slug: "guide/navigating-the-tui" },
      { label: "Initializing a workspace", slug: "guide/initializing-a-workspace" },
      { label: "Managing agents", slug: "guide/managing-agents" },
      { label: "Managing prompts", slug: "guide/managing-prompts" },
      { label: "Managing workflows", slug: "guide/managing-workflows" },
      { label: "Managing specs", slug: "guide/managing-specs" },
      { label: "Managing architecture", slug: "guide/managing-architecture" },
      { label: "Managing epics and tickets", slug: "guide/managing-epics-and-tickets" },
      { label: "Tracing the backlog", slug: "guide/tracing-the-backlog" },
      { label: "Validating conventions", slug: "guide/validating-conventions" },
      { label: "Compiling to a backend", slug: "guide/compiling-to-a-backend" },
      { label: "Configuration", slug: "guide/configuration" },
      { label: "Examples", slug: "guide/examples" },
      { label: "Troubleshooting", slug: "guide/troubleshooting" },
    ],
  },
  {
    title: "Contributing",
    items: [
      { label: "Development environment", slug: "contributing/development-environment" },
      { label: "Continuous integration", slug: "contributing/continuous-integration" },
      { label: "Testing and golden files", slug: "contributing/testing-and-golden-files" },
    ],
  },
];

/** Flattened nav order, for prev/next links. */
export const flatNav: NavItem[] = nav.flatMap((s) => s.items);

export function neighbors(slug: string): { prev?: NavItem; next?: NavItem } {
  const norm = normalizeSlug(slug);
  const i = flatNav.findIndex((item) => item.slug === norm);
  if (i === -1) return {};
  return { prev: flatNav[i - 1], next: flatNav[i + 1] };
}

export function labelForSlug(slug: string): string {
  const norm = normalizeSlug(slug);
  return flatNav.find((i) => i.slug === norm)?.label ?? "Documentation";
}

/**
 * Resolve a relative markdown link (as written in the docs) to an in-app docs
 * route. Returns null for external links, which the renderer opens directly.
 */
export function resolveDocLink(
  currentSlug: string,
  href: string,
): { to: string; hash: string } | null {
  if (/^(https?:)?\/\//.test(href) || href.startsWith("mailto:")) return null;

  const [rawPath, rawHash] = href.split("#");
  const hash = rawHash ? `#${rawHash}` : "";

  // Pure in-page anchor.
  if (!rawPath) return { to: docRoute(currentSlug), hash };

  const dirSegments = normalizeSlug(currentSlug).split("/").slice(0, -1);
  const segments = [...dirSegments];
  for (const part of rawPath.replace(/\.md$/, "").split("/")) {
    if (part === "" || part === ".") continue;
    if (part === "..") segments.pop();
    else segments.push(part);
  }
  let target = segments.join("/");
  if (target === "README") target = "";
  return { to: docRoute(target), hash };
}

export function docRoute(slug: string): string {
  const norm = normalizeSlug(slug);
  return norm ? `/docs/${norm}` : "/docs";
}

export interface TocEntry {
  id: string;
  text: string;
  level: 2 | 3;
}

/** Pull H2/H3 headings (outside fenced code) for the on-page table of contents. */
export function extractToc(markdown: string): TocEntry[] {
  const toc: TocEntry[] = [];
  let inFence = false;
  for (const line of markdown.split("\n")) {
    if (/^\s*```/.test(line)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;
    const m = /^(#{2,3})\s+(.*)$/.exec(line);
    if (!m) continue;
    const level = m[1].length as 2 | 3;
    const text = m[2].replace(/\s*#*\s*$/, "").replace(/[*`_]/g, "").trim();
    toc.push({ id: slugify(text), text, level });
  }
  return toc;
}

/** Mirror GitHub-style heading id generation (matches rehype-slug defaults). */
export function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, "")
    .trim()
    .replace(/\s+/g, "-");
}

export function firstHeading(markdown: string): string | undefined {
  const m = /^#\s+(.*)$/m.exec(markdown);
  return m?.[1]?.trim();
}
