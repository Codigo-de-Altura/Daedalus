import { useState, type ReactNode } from "react";
import { cn } from "../lib/cn";
import { Icon } from "./Icon";

/** Window chrome with traffic-light dots and an optional title. */
export function Window({
  title,
  children,
  className,
  accent,
}: {
  title?: ReactNode;
  children: ReactNode;
  className?: string;
  accent?: boolean;
}) {
  return (
    <div
      className={cn(
        "ring-hairline overflow-hidden rounded-xl bg-ink-900/90 shadow-card backdrop-blur",
        accent && "shadow-glow",
        className,
      )}
    >
      <div className="flex items-center gap-2 border-b border-ink-700/60 bg-ink-800/60 px-4 py-3">
        <span className="h-3 w-3 rounded-full bg-ink-600" />
        <span className="h-3 w-3 rounded-full bg-ink-600" />
        <span className="h-3 w-3 rounded-full bg-ink-600" />
        {title && (
          <span className="ml-2 font-mono text-xs text-ink-400">{title}</span>
        )}
      </div>
      {children}
    </div>
  );
}

export function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      type="button"
      aria-label="Copy command"
      onClick={() => {
        void navigator.clipboard.writeText(text).then(() => {
          setCopied(true);
          window.setTimeout(() => setCopied(false), 1600);
        });
      }}
      className="inline-flex h-8 w-8 items-center justify-center rounded-lg text-ink-400 transition-colors hover:bg-ink-700/60 hover:text-ink-100"
    >
      <Icon name={copied ? "check" : "copy"} size={16} />
    </button>
  );
}

type Line =
  | { kind: "cmd"; text: string }
  | { kind: "out"; text: string }
  | { kind: "added"; text: string }
  | { kind: "dim"; text: string };

/** A faux terminal that renders prompt commands and their output. */
export function Terminal({
  lines,
  className,
}: {
  lines: Line[];
  className?: string;
}) {
  return (
    <pre
      className={cn(
        "overflow-x-auto p-5 font-mono text-[13px] leading-relaxed",
        className,
      )}
    >
      <code className="block">
        {lines.map((line, i) => {
          if (line.kind === "cmd") {
            return (
              <span key={i} className="block text-ink-50">
                <span className="select-none text-forge-400">$ </span>
                {line.text}
              </span>
            );
          }
          if (line.kind === "added") {
            return (
              <span key={i} className="block text-emerald-400/90">
                {line.text}
              </span>
            );
          }
          if (line.kind === "dim") {
            return (
              <span key={i} className="block text-ink-500">
                {line.text}
              </span>
            );
          }
          return (
            <span key={i} className="block text-ink-300">
              {line.text}
            </span>
          );
        })}
      </code>
    </pre>
  );
}

/** An inline, copyable command pill. */
export function CommandPill({ command }: { command: string }) {
  return (
    <div className="ring-hairline flex items-center gap-3 rounded-xl bg-ink-900/80 py-1.5 pl-4 pr-1.5 backdrop-blur">
      <span className="select-none font-mono text-sm text-forge-400">$</span>
      <code className="font-mono text-sm text-ink-100">{command}</code>
      <CopyButton text={command} />
    </div>
  );
}

export type { Line };
