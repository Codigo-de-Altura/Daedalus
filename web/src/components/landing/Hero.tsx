import { motion, Reveal } from "../motion";
import { Container, Button, Badge } from "../ui";
import { Window, Terminal, CommandPill, type Line } from "../CodeWindow";
import { Labyrinth } from "../Labyrinth";
import { GITHUB_URL, INSTALL_CMD, QUICKSTART } from "../../lib/site";

const heroLines: Line[] = [
  { kind: "cmd", text: "daedalus init" },
  { kind: "out", text: "Created Daedalus workspace at .daedalus from scratch." },
  { kind: "dim", text: 'Seeded factory workflow "sdd-default".' },
  { kind: "out", text: "" },
  { kind: "cmd", text: "daedalus build --yes" },
  { kind: "out", text: "Compiled .:" },
  { kind: "out", text: "  claude-code: 1 created, 0 updated (of 1 artifact)" },
  { kind: "added", text: "    + .claude/settings.json" },
];

export default function Hero() {
  return (
    <section className="relative overflow-hidden">
      {/* Background motifs */}
      <div className="pointer-events-none absolute inset-0 -z-10">
        <div className="absolute inset-0 bg-grid bg-grid-fade opacity-70" />
        <div className="absolute inset-x-0 top-0 h-[640px] glow-forge" />
      </div>

      <Container className="grid items-center gap-14 pb-20 pt-16 sm:pt-24 lg:grid-cols-[1.05fr_0.95fr] lg:pb-28">
        {/* Copy */}
        <div className="flex flex-col items-start gap-7">
          <Reveal>
            <Badge>
              <span className="h-1.5 w-1.5 rounded-full bg-forge-400" />
              v0.1.0 · foundations
              <span className="text-ink-500">— backend-agnostic by design</span>
            </Badge>
          </Reveal>

          <Reveal delay={0.05}>
            <h1 className="text-balance text-4xl font-semibold leading-[1.05] tracking-tight text-ink-50 sm:text-5xl lg:text-6xl">
              Build your AI scaffolding once.{" "}
              <span className="text-gradient-amber">Compile it anywhere.</span>
            </h1>
          </Reveal>

          <Reveal delay={0.1}>
            <p className="max-w-xl text-pretty text-lg leading-relaxed text-ink-300">
              Daedalus is a lightweight TUI/CLI that designs your project's
              agents, prompts, DAG workflows and SDD backlog in one canonical
              model — then compiles them to your tool's native format.
            </p>
          </Reveal>

          <Reveal delay={0.15}>
            <div className="flex flex-wrap items-center gap-3">
              <Button to={QUICKSTART} size="lg" iconRight="arrow">
                Get started
              </Button>
              <Button
                href={GITHUB_URL}
                size="lg"
                variant="secondary"
                icon="github"
              >
                Star on GitHub
              </Button>
            </div>
          </Reveal>

          <Reveal delay={0.2}>
            <CommandPill command={INSTALL_CMD} />
          </Reveal>
        </div>

        {/* Visual */}
        <Reveal delay={0.15} className="relative">
          <motion.div
            aria-hidden
            className="pointer-events-none absolute -right-10 -top-16 hidden h-44 w-44 opacity-30 lg:block"
            animate={{ rotate: [0, 6, 0] }}
            transition={{ duration: 12, repeat: Infinity, ease: "easeInOut" }}
          >
            <Labyrinth className="h-full w-full" strokeWidth={1.2} />
          </motion.div>

          <Window title="daedalus — bash" accent>
            <Terminal lines={heroLines} />
          </Window>

          <motion.div
            aria-hidden
            className="pointer-events-none absolute -bottom-10 -left-8 hidden h-28 w-28 opacity-20 lg:block"
            animate={{ y: [0, -10, 0] }}
            transition={{ duration: 7, repeat: Infinity, ease: "easeInOut" }}
          >
            <Labyrinth className="h-full w-full" animate={false} strokeWidth={1.4} />
          </motion.div>
        </Reveal>
      </Container>
    </section>
  );
}
