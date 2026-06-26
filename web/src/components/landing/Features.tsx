import { Container, Section, SectionHeading } from "../ui";
import { StaggerGroup, motion, staggerItem } from "../motion";
import { Icon, type IconName } from "../Icon";
import { Labyrinth } from "../Labyrinth";
import { features } from "../../lib/site";
import { cn } from "../../lib/cn";

export default function Features() {
  return (
    <Section id="features">
      <Container className="flex flex-col gap-14">
        <SectionHeading
          eyebrow="What you get"
          title="A source of truth for your AI scaffolding"
          subtitle="Stop re-authoring prompts and agents in every repo. Describe them once, keep them in git, and let Daedalus do the careful, deterministic work."
        />

        <StaggerGroup className="grid gap-5 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((f) => (
            <motion.div
              key={f.title}
              variants={staggerItem}
              className={cn(
                "group ring-hairline surface relative overflow-hidden rounded-2xl p-6 transition-colors duration-300 hover:bg-ink-800/40",
                f.span === "wide" && "sm:col-span-2",
              )}
            >
              <div className="mb-5 inline-flex h-11 w-11 items-center justify-center rounded-xl bg-forge-500/12 text-forge-300 ring-1 ring-forge-500/20 transition-transform duration-300 group-hover:scale-110">
                <Icon name={f.icon as IconName} size={22} />
              </div>
              <h3 className="mb-2 font-display text-lg font-semibold text-ink-50">
                {f.title}
              </h3>
              <p className="max-w-md text-sm leading-relaxed text-ink-400">
                {f.body}
              </p>
              <div
                aria-hidden
                className="pointer-events-none absolute -right-16 -top-16 h-40 w-40 rounded-full bg-forge-500/0 blur-3xl transition-colors duration-500 group-hover:bg-forge-500/10"
              />
            </motion.div>
          ))}

          {/* Decorative closing card — the maze mark */}
          <motion.div
            variants={staggerItem}
            className="ring-hairline relative flex flex-col items-center justify-center gap-3 overflow-hidden rounded-2xl bg-ink-900/60 p-6 text-center"
          >
            <Labyrinth className="h-16 w-16" strokeWidth={1.6} />
            <p className="font-mono text-xs leading-relaxed text-ink-500">
              one thread through
              <br />
              the labyrinth
            </p>
          </motion.div>
        </StaggerGroup>
      </Container>
    </Section>
  );
}
