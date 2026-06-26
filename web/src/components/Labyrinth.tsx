import { useReducedMotion } from "framer-motion";
import { motion } from "./motion";
import { cn } from "../lib/cn";

/**
 * The Daedalus mark: a unicursal, squared labyrinth that draws itself in.
 * Used large as the hero's signature visual and faint as a background motif.
 * Paths are concentric squared rings joined into one continuous route.
 */
const RINGS = [
  "M50 8 H92 V92 H8 V8 H50",
  "M50 20 H80 V80 H20 V20 H50",
  "M50 32 H68 V68 H32 V32 H50",
  "M50 44 H56 V56 H44 V44 H50",
];

export function Labyrinth({
  className,
  animate = true,
  strokeWidth = 1.4,
}: {
  className?: string;
  animate?: boolean;
  strokeWidth?: number;
}) {
  const reduce = useReducedMotion();
  const shouldAnimate = animate && !reduce;

  return (
    <svg
      viewBox="0 0 100 100"
      fill="none"
      className={cn(className)}
      aria-hidden="true"
    >
      <defs>
        <linearGradient id="lab-stroke" x1="0" y1="0" x2="1" y2="1">
          <stop offset="0%" stopColor="#f1b057" />
          <stop offset="100%" stopColor="#cb6a12" />
        </linearGradient>
      </defs>
      {RINGS.map((d, i) => (
        <motion.path
          key={d}
          d={d}
          stroke="url(#lab-stroke)"
          strokeWidth={strokeWidth}
          strokeLinecap="square"
          initial={shouldAnimate ? { pathLength: 0, opacity: 0 } : false}
          whileInView={shouldAnimate ? { pathLength: 1, opacity: 1 } : undefined}
          viewport={{ once: true }}
          transition={{
            duration: 1.6,
            ease: [0.16, 1, 0.3, 1],
            delay: i * 0.18,
          }}
        />
      ))}
      {/* The single thread out of the maze */}
      <motion.path
        d="M50 56 V44 M50 68 V32 M50 80 V20 M50 92 V8"
        stroke="url(#lab-stroke)"
        strokeWidth={strokeWidth * 0.7}
        strokeOpacity={0.35}
        initial={shouldAnimate ? { pathLength: 0 } : false}
        whileInView={shouldAnimate ? { pathLength: 1 } : undefined}
        viewport={{ once: true }}
        transition={{ duration: 1.4, ease: "easeInOut", delay: 0.6 }}
      />
    </svg>
  );
}
