import type { SVGProps } from "react";

type IconName =
  | "cube"
  | "graph"
  | "layers"
  | "diff"
  | "shield"
  | "bolt"
  | "arrow"
  | "arrowUpRight"
  | "github"
  | "check"
  | "copy"
  | "terminal"
  | "menu"
  | "close"
  | "book";

const paths: Record<IconName, JSX.Element> = {
  cube: (
    <>
      <path d="M12 2.5 21 7v10l-9 4.5L3 17V7l9-4.5Z" />
      <path d="m3 7 9 4.5L21 7M12 11.5V21" />
    </>
  ),
  graph: (
    <>
      <circle cx="6" cy="6" r="2.4" />
      <circle cx="18" cy="6" r="2.4" />
      <circle cx="12" cy="18" r="2.4" />
      <path d="M7.8 7.6 11 15.8M16.2 7.6 13 15.8M8.4 6h7.2" />
    </>
  ),
  layers: (
    <>
      <path d="m12 3 9 5-9 5-9-5 9-5Z" />
      <path d="m3 13 9 5 9-5M3 16.5l9 5 9-5" />
    </>
  ),
  diff: (
    <>
      <path d="M12 4v6M9 7h6" />
      <path d="M9 17h6" />
      <rect x="3.5" y="2.5" width="17" height="19" rx="2.5" />
    </>
  ),
  shield: (
    <>
      <path d="M12 2.5 20 6v6c0 5-3.5 7.6-8 9.5C7.5 19.6 4 17 4 12V6l8-3.5Z" />
      <path d="m8.8 12 2.2 2.2 4.2-4.4" />
    </>
  ),
  bolt: <path d="M13 2 4.5 13.5H11l-1 8.5 8.5-12H12l1-8Z" />,
  arrow: <path d="M5 12h14M13 6l6 6-6 6" />,
  arrowUpRight: <path d="M7 17 17 7M8 7h9v9" />,
  github: (
    <path d="M12 2C6.48 2 2 6.58 2 12.25c0 4.53 2.87 8.37 6.84 9.73.5.1.68-.22.68-.49l-.01-1.9c-2.78.62-3.37-1.2-3.37-1.2-.46-1.18-1.11-1.5-1.11-1.5-.91-.64.07-.62.07-.62 1 .07 1.53 1.06 1.53 1.06.9 1.57 2.36 1.12 2.94.85.09-.67.35-1.12.63-1.38-2.22-.26-4.55-1.14-4.55-5.07 0-1.12.39-2.03 1.03-2.75-.1-.26-.45-1.3.1-2.71 0 0 .84-.28 2.75 1.05a9.3 9.3 0 0 1 5 0c1.91-1.33 2.75-1.05 2.75-1.05.55 1.41.2 2.45.1 2.71.64.72 1.03 1.63 1.03 2.75 0 3.94-2.34 4.8-4.57 5.06.36.32.68.94.68 1.9l-.01 2.82c0 .27.18.6.69.49A10.27 10.27 0 0 0 22 12.25C22 6.58 17.52 2 12 2Z" />
  ),
  check: <path d="m4.5 12.5 5 5 10-11" />,
  copy: (
    <>
      <rect x="8.5" y="8.5" width="12" height="12" rx="2.2" />
      <path d="M15.5 8.5V5.5a2 2 0 0 0-2-2h-8a2 2 0 0 0-2 2v8a2 2 0 0 0 2 2h3" />
    </>
  ),
  terminal: (
    <>
      <path d="m5 8 4 4-4 4M12 16h6" />
      <rect x="2.5" y="3.5" width="19" height="17" rx="2.5" />
    </>
  ),
  menu: <path d="M4 7h16M4 12h16M4 17h16" />,
  close: <path d="M6 6l12 12M18 6 6 18" />,
  book: (
    <>
      <path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H20v15H6.5A2.5 2.5 0 0 0 4 20.5V5.5Z" />
      <path d="M4 20.5A2.5 2.5 0 0 1 6.5 18H20v3H6.5A2.5 2.5 0 0 1 4 20.5Z" />
    </>
  ),
};

export function Icon({
  name,
  size = 24,
  ...props
}: { name: IconName; size?: number } & SVGProps<SVGSVGElement>) {
  return (
    <svg
      viewBox="0 0 24 24"
      width={size}
      height={size}
      fill={name === "github" ? "currentColor" : "none"}
      stroke={name === "github" ? "none" : "currentColor"}
      strokeWidth={1.6}
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      {...props}
    >
      {paths[name]}
    </svg>
  );
}

export type { IconName };
