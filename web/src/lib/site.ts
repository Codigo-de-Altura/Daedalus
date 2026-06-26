/** Central place for copy, links, and structured landing-page data. */

export const GITHUB_URL = "https://github.com/Codigo-de-Altura/Daedalus";

/** One-line install command shown in the hero (Linux/macOS). */
export const INSTALL_CMD =
  "curl -fsSL https://raw.githubusercontent.com/Codigo-de-Altura/Daedalus/main/scripts/install.sh | sh";

export const DOCS_HOME = "/docs";
export const QUICKSTART = "/docs/getting-started/quickstart";
export const INSTALL = "/docs/getting-started/installation";

export const navLinks = [
  { label: "How it works", href: "/#how" },
  { label: "Features", href: "/#features" },
  { label: "Compile", href: "/#compile" },
  { label: "Docs", href: DOCS_HOME },
] as const;

export interface Step {
  command: string;
  title: string;
  body: string;
}

export const steps: Step[] = [
  {
    command: "daedalus init",
    title: "Define",
    body: "Scaffold a backend-agnostic .daedalus/ workspace — the single source of truth for agents, prompts, DAG workflows and an SDD backlog.",
  },
  {
    command: "daedalus validate",
    title: "Validate",
    body: "Check conventions and that every agent, workflow and the manifest are well-formed — before anything is generated.",
  },
  {
    command: "daedalus build --preview",
    title: "Preview",
    body: "See the exact diff of what will be written. Preview writes nothing, so you stay in control of every file.",
  },
  {
    command: "daedalus build --yes",
    title: "Compile",
    body: "Compile the canonical model to your tool's native format — the first adapter targets Claude Code → .claude/. Idempotent and deterministic.",
  },
];

export interface Feature {
  title: string;
  body: string;
  icon: string; // key consumed by the Icon component
  span?: "wide";
}

export const features: Feature[] = [
  {
    title: "One canonical model",
    body: "Describe agents, prompts, workflows and the backlog once in .daedalus/. Daedalus keeps them versioned in git and backend-agnostic.",
    icon: "cube",
    span: "wide",
  },
  {
    title: "DAG workflows in YAML",
    body: "Express your SDD pipeline as a directed graph. The seeded sdd-default workflow gets you productive on the first run.",
    icon: "graph",
  },
  {
    title: "SDD backlog, built in",
    body: "Epics, tickets, specs and architecture live alongside your agents — traceable from requirement to artifact.",
    icon: "layers",
  },
  {
    title: "Preview every write",
    body: "build --preview shows a precise diff and writes nothing. No surprise overwrites, ever.",
    icon: "diff",
  },
  {
    title: "Idempotent & deterministic",
    body: "Re-running a build with no changes is a no-op. Output is reproducible and verified against golden files.",
    icon: "shield",
  },
  {
    title: "Compiles to your tool",
    body: "A pluggable adapter layer targets native formats. Claude Code → .claude/ ships first; the model stays portable.",
    icon: "bolt",
    span: "wide",
  },
];

export interface CompareRow {
  label: string;
  byHand: string;
  daedalus: string;
}

export const comparison: CompareRow[] = [
  {
    label: "Set up a new repo",
    byHand: "Copy-paste prompts & agents by hand, every time",
    daedalus: "daedalus init — seeded, versioned, consistent",
  },
  {
    label: "Source of truth",
    byHand: "Scattered across tool-specific config files",
    daedalus: "One canonical .daedalus/ model in git",
  },
  {
    label: "Switching AI tools",
    byHand: "Re-author everything for the new format",
    daedalus: "Recompile with a different adapter",
  },
  {
    label: "Safety of changes",
    byHand: "Hope you didn't clobber a manual edit",
    daedalus: "Preview diff, idempotent, non-destructive",
  },
  {
    label: "Team consistency",
    byHand: "Each dev drifts in their own direction",
    daedalus: "Shared spec, deterministic builds",
  },
];

export const footerColumns = [
  {
    title: "Product",
    links: [
      { label: "How it works", href: "/#how" },
      { label: "Features", href: "/#features" },
      { label: "Compile", href: "/#compile" },
    ],
  },
  {
    title: "Docs",
    links: [
      { label: "Installation", href: INSTALL },
      { label: "Quickstart", href: QUICKSTART },
      { label: "Concepts", href: "/docs/guide/concepts" },
      { label: "Command reference", href: "/docs/guide/command-reference" },
    ],
  },
  {
    title: "Project",
    links: [
      { label: "GitHub", href: GITHUB_URL, external: true },
      { label: "Releases", href: `${GITHUB_URL}/releases`, external: true },
      { label: "Issues", href: `${GITHUB_URL}/issues`, external: true },
    ],
  },
] as const;
