import { Container, Section, Button } from "../ui";
import { Reveal } from "../motion";
import { CommandPill } from "../CodeWindow";
import { Labyrinth } from "../Labyrinth";
import { GITHUB_URL, QUICKSTART } from "../../lib/site";

export default function CTA() {
  return (
    <Section>
      <Container>
        <Reveal>
          <div className="ring-hairline relative overflow-hidden rounded-3xl bg-gradient-to-b from-ink-800/70 to-ink-900/80 px-6 py-16 text-center sm:px-12 sm:py-20">
            <div className="pointer-events-none absolute inset-0 bg-grid opacity-40" />
            <div className="pointer-events-none absolute inset-x-0 top-0 h-64 glow-forge" />
            <div
              aria-hidden
              className="pointer-events-none absolute -bottom-20 -right-16 h-64 w-64 opacity-[0.12]"
            >
              <Labyrinth className="h-full w-full" animate={false} strokeWidth={1} />
            </div>

            <div className="relative mx-auto flex max-w-2xl flex-col items-center gap-6">
              <h2 className="text-balance text-3xl font-semibold text-ink-50 sm:text-4xl">
                Give every repo the same well-built foundation
              </h2>
              <p className="text-pretty text-lg text-ink-300">
                Initialize a workspace, validate it, preview the diff, and
                compile. It takes minutes — and it's the same every time.
              </p>
              <div className="flex flex-wrap items-center justify-center gap-3">
                <Button to={QUICKSTART} size="lg" iconRight="arrow">
                  Read the quickstart
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
              <div className="pt-2">
                <CommandPill command="daedalus init" />
              </div>
            </div>
          </div>
        </Reveal>
      </Container>
    </Section>
  );
}
