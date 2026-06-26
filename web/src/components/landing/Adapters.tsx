import { Container } from "../ui";
import { Reveal } from "../motion";

const adapters = [
  { name: "Claude Code", target: ".claude/", status: "available" },
  { name: "Cursor", target: ".cursor/", status: "planned" },
  { name: "Copilot", target: ".github/", status: "planned" },
  { name: "Windsurf", target: ".windsurf/", status: "planned" },
  { name: "Aider", target: ".aider/", status: "planned" },
] as const;

export default function Adapters() {
  return (
    <section className="border-y border-ink-800/60 bg-ink-900/30 py-12">
      <Container className="flex flex-col items-center gap-7">
        <Reveal>
          <p className="text-center text-xs font-semibold uppercase tracking-[0.2em] text-ink-400">
            One canonical model · a pluggable adapter for every tool
          </p>
        </Reveal>
        <Reveal delay={0.05}>
          <ul className="flex flex-wrap items-center justify-center gap-3">
            {adapters.map((a) => (
              <li
                key={a.name}
                className="ring-hairline flex items-center gap-2.5 rounded-full bg-ink-800/50 py-2 pl-4 pr-3 backdrop-blur"
              >
                <span className="font-display text-sm font-medium text-ink-100">
                  {a.name}
                </span>
                <code className="font-mono text-xs text-ink-500">
                  {a.target}
                </code>
                {a.status === "available" ? (
                  <span className="rounded-full bg-forge-500/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-forge-300">
                    live
                  </span>
                ) : (
                  <span className="rounded-full bg-ink-700/60 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-ink-400">
                    soon
                  </span>
                )}
              </li>
            ))}
          </ul>
        </Reveal>
      </Container>
    </section>
  );
}
