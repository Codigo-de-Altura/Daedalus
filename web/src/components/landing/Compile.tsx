import { useState } from "react";
import { AnimatePresence, motion } from "framer-motion";
import { Container, Section, SectionHeading } from "../ui";
import { Reveal } from "../motion";
import { Window } from "../CodeWindow";
import { Icon } from "../Icon";
import { cn } from "../../lib/cn";

const canonical = `# .daedalus/workflows/sdd-default.yaml
name: sdd-default
nodes:
  - id: spec
    agent: planner
  - id: build
    agent: engineer
    needs: [spec]
  - id: review
    agent: reviewer
    needs: [build]`;

type Backend = {
  id: string;
  label: string;
  path: string;
  available: boolean;
  output: string;
};

const backends: Backend[] = [
  {
    id: "claude",
    label: "Claude Code",
    path: ".claude/settings.json",
    available: true,
    output: `{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "daedalus": {
    "managed": true,
    "generator": "daedalus"
  }
}`,
  },
  {
    id: "cursor",
    label: "Cursor",
    path: ".cursor/",
    available: false,
    output: "",
  },
  {
    id: "copilot",
    label: "Copilot",
    path: ".github/",
    available: false,
    output: "",
  },
];

export default function Compile() {
  const [active, setActive] = useState(backends[0]);

  return (
    <Section id="compile" className="overflow-hidden">
      <div className="pointer-events-none absolute inset-x-0 top-1/2 -z-10 h-[420px] -translate-y-1/2 glow-forge opacity-60" />
      <Container className="flex flex-col gap-12">
        <SectionHeading
          align="center"
          className="mx-auto items-center"
          eyebrow="Compile"
          title="Author once. Target any backend."
          subtitle="Your canonical definitions never change. Switch the adapter and Daedalus emits the native files your tool expects — with a preview diff before a single byte is written."
        />

        {/* Backend tabs */}
        <Reveal>
          <div className="mx-auto flex flex-wrap items-center justify-center gap-2">
            {backends.map((b) => (
              <button
                key={b.id}
                type="button"
                disabled={!b.available}
                onClick={() => b.available && setActive(b)}
                className={cn(
                  "rounded-full px-4 py-2 text-sm font-medium transition-colors",
                  active.id === b.id
                    ? "bg-forge-500 text-ink-950"
                    : "ring-hairline bg-ink-800/50 text-ink-300 hover:text-ink-50",
                  !b.available && "cursor-not-allowed opacity-50",
                )}
              >
                {b.label}
                {!b.available && (
                  <span className="ml-2 text-[10px] uppercase tracking-wide">
                    soon
                  </span>
                )}
              </button>
            ))}
          </div>
        </Reveal>

        <div className="grid items-center gap-4 lg:grid-cols-[1fr_auto_1fr]">
          <Reveal>
            <Window title=".daedalus/ · canonical">
              <pre className="overflow-x-auto p-5 font-mono text-[12.5px] leading-relaxed text-ink-200">
                <code>{canonical}</code>
              </pre>
            </Window>
          </Reveal>

          {/* Compile arrow */}
          <Reveal delay={0.1}>
            <div className="flex items-center justify-center py-2 lg:flex-col">
              <div className="flex h-12 w-12 items-center justify-center rounded-full bg-forge-500/15 text-forge-300 ring-1 ring-forge-500/30">
                <Icon name="bolt" size={22} />
              </div>
            </div>
          </Reveal>

          <Reveal delay={0.15}>
            <AnimatePresence mode="wait">
              <motion.div
                key={active.id}
                initial={{ opacity: 0, y: 8 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: -8 }}
                transition={{ duration: 0.25 }}
              >
                <Window title={`${active.path} · compiled`} accent>
                  <pre className="overflow-x-auto p-5 font-mono text-[12.5px] leading-relaxed text-ink-200">
                    <code>{active.output}</code>
                  </pre>
                </Window>
              </motion.div>
            </AnimatePresence>
          </Reveal>
        </div>

        <Reveal>
          <p className="text-center text-sm text-ink-500">
            Re-running with no changes is an idempotent no-op. Output is
            deterministic and verified against golden files.
          </p>
        </Reveal>
      </Container>
    </Section>
  );
}
