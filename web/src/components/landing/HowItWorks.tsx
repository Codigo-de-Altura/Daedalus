import { Container, Section, SectionHeading } from "../ui";
import { StaggerGroup, motion, staggerItem } from "../motion";
import { steps } from "../../lib/site";

export default function HowItWorks() {
  return (
    <Section id="how">
      <Container className="flex flex-col gap-14">
        <SectionHeading
          eyebrow="The loop"
          title="From a blank repo to compiled scaffolding in four commands"
          subtitle="Daedalus keeps a clean edit → validate → preview → build loop. Every step is explicit, reversible, and safe to re-run."
        />

        <StaggerGroup className="relative grid gap-5 md:grid-cols-2 lg:grid-cols-4">
          {/* Connecting thread behind the cards on desktop */}
          <div
            aria-hidden
            className="pointer-events-none absolute left-0 right-0 top-[2.15rem] hidden h-px bg-gradient-to-r from-transparent via-forge-600/40 to-transparent lg:block"
          />
          {steps.map((step, i) => (
            <motion.div
              key={step.title}
              variants={staggerItem}
              className="ring-hairline surface relative flex flex-col gap-4 rounded-2xl p-6"
            >
              <div className="flex items-center justify-between">
                <span className="grid h-9 w-9 place-items-center rounded-lg bg-forge-500/15 font-display text-sm font-semibold text-forge-300 ring-1 ring-forge-500/20">
                  {String(i + 1).padStart(2, "0")}
                </span>
                <span className="h-1.5 w-1.5 rounded-full bg-forge-500/50" />
              </div>
              <code className="block w-fit rounded-md bg-ink-950/60 px-2.5 py-1 font-mono text-xs text-forge-200 ring-hairline">
                {step.command}
              </code>
              <div className="flex flex-col gap-2">
                <h3 className="font-display text-lg font-semibold text-ink-50">
                  {step.title}
                </h3>
                <p className="text-sm leading-relaxed text-ink-400">
                  {step.body}
                </p>
              </div>
            </motion.div>
          ))}
        </StaggerGroup>
      </Container>
    </Section>
  );
}
