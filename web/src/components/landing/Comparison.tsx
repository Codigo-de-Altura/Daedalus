import { Container, Section, SectionHeading } from "../ui";
import { StaggerGroup, motion, staggerItem } from "../motion";
import { Icon } from "../Icon";
import { comparison } from "../../lib/site";

export default function Comparison() {
  return (
    <Section>
      <Container className="flex flex-col gap-12">
        <SectionHeading
          eyebrow="Why Daedalus"
          title="The same job, without the toil"
          subtitle="Hand-rolling AI scaffolding per repo is slow, drifty, and risky. Daedalus makes it a single, versioned, reproducible step."
        />

        <StaggerGroup className="ring-hairline surface overflow-hidden rounded-2xl">
          {/* Header row */}
          <div className="grid grid-cols-1 border-b border-ink-700/60 sm:grid-cols-[1.2fr_1fr_1fr]">
            <div className="hidden p-5 sm:block" />
            <div className="flex items-center gap-2 p-5 text-sm font-semibold text-ink-300">
              By hand
            </div>
            <div className="flex items-center gap-2 bg-forge-500/[0.06] p-5 text-sm font-semibold text-forge-200">
              <Icon name="bolt" size={16} />
              With Daedalus
            </div>
          </div>

          {comparison.map((row) => (
            <motion.div
              key={row.label}
              variants={staggerItem}
              className="grid grid-cols-1 border-b border-ink-800/60 last:border-0 sm:grid-cols-[1.2fr_1fr_1fr]"
            >
              <div className="px-5 pb-1 pt-5 font-display text-sm font-semibold text-ink-100 sm:py-5">
                {row.label}
              </div>
              <div className="flex items-start gap-2.5 px-5 pb-4 pt-1 text-sm text-ink-400 sm:py-5">
                <Icon
                  name="close"
                  size={16}
                  className="mt-0.5 shrink-0 text-ink-600"
                />
                {row.byHand}
              </div>
              <div className="flex items-start gap-2.5 bg-forge-500/[0.04] px-5 pb-5 pt-1 text-sm text-ink-200 sm:py-5">
                <Icon
                  name="check"
                  size={16}
                  className="mt-0.5 shrink-0 text-forge-400"
                />
                {row.daedalus}
              </div>
            </motion.div>
          ))}
        </StaggerGroup>
      </Container>
    </Section>
  );
}
