import type { ReactNode } from "react";
import { Link } from "react-router-dom";
import { cn } from "../lib/cn";
import { Icon, type IconName } from "./Icon";

export function Container({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("mx-auto w-full max-w-6xl px-5 sm:px-8", className)}>
      {children}
    </div>
  );
}

export function Section({
  children,
  className,
  id,
}: {
  children: ReactNode;
  className?: string;
  id?: string;
}) {
  return (
    <section
      id={id}
      className={cn("relative scroll-mt-24 py-20 sm:py-28", className)}
    >
      {children}
    </section>
  );
}

export function Eyebrow({ children }: { children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-2 text-xs font-semibold uppercase tracking-[0.18em] text-forge-400">
      <span className="h-px w-6 bg-forge-500/60" />
      {children}
    </span>
  );
}

export function SectionHeading({
  eyebrow,
  title,
  subtitle,
  align = "left",
  className,
}: {
  eyebrow?: string;
  title: ReactNode;
  subtitle?: ReactNode;
  align?: "left" | "center";
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col gap-4",
        align === "center" && "items-center text-center",
        className,
      )}
    >
      {eyebrow && <Eyebrow>{eyebrow}</Eyebrow>}
      <h2 className="max-w-2xl text-balance text-3xl font-semibold text-ink-50 sm:text-4xl">
        {title}
      </h2>
      {subtitle && (
        <p className="max-w-2xl text-pretty text-base leading-relaxed text-ink-300 sm:text-lg">
          {subtitle}
        </p>
      )}
    </div>
  );
}

type ButtonProps = {
  children: ReactNode;
  variant?: "primary" | "secondary" | "ghost";
  size?: "md" | "lg";
  to?: string;
  href?: string;
  icon?: IconName;
  iconRight?: IconName;
  className?: string;
  onClick?: () => void;
};

const buttonBase =
  "group inline-flex select-none items-center justify-center gap-2 rounded-xl font-medium transition-all duration-200 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-forge-400";

const buttonVariants: Record<NonNullable<ButtonProps["variant"]>, string> = {
  primary:
    "bg-forge-500 text-ink-950 shadow-[0_8px_30px_-8px_rgba(232,133,27,0.55)] hover:bg-forge-400 hover:-translate-y-0.5 active:translate-y-0",
  secondary:
    "ring-hairline bg-ink-800/60 text-ink-100 backdrop-blur hover:bg-ink-700/70 hover:-translate-y-0.5",
  ghost: "text-ink-200 hover:text-ink-50 hover:bg-ink-800/60",
};

const buttonSizes: Record<NonNullable<ButtonProps["size"]>, string> = {
  md: "h-10 px-4 text-sm",
  lg: "h-12 px-6 text-[15px]",
};

export function Button({
  children,
  variant = "primary",
  size = "md",
  to,
  href,
  icon,
  iconRight,
  className,
  onClick,
}: ButtonProps) {
  const classes = cn(
    buttonBase,
    buttonVariants[variant],
    buttonSizes[size],
    className,
  );
  const inner = (
    <>
      {icon && <Icon name={icon} size={18} />}
      {children}
      {iconRight && (
        <Icon
          name={iconRight}
          size={18}
          className="transition-transform duration-200 group-hover:translate-x-0.5"
        />
      )}
    </>
  );

  if (to) {
    return (
      <Link to={to} className={classes} onClick={onClick}>
        {inner}
      </Link>
    );
  }
  if (href) {
    return (
      <a
        href={href}
        target="_blank"
        rel="noreferrer"
        className={classes}
        onClick={onClick}
      >
        {inner}
      </a>
    );
  }
  return (
    <button type="button" className={classes} onClick={onClick}>
      {inner}
    </button>
  );
}

export function Badge({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "ring-hairline inline-flex items-center gap-2 rounded-full bg-ink-800/70 px-3 py-1 text-xs font-medium text-ink-200 backdrop-blur",
        className,
      )}
    >
      {children}
    </span>
  );
}

export function Card({
  children,
  className,
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "ring-hairline surface relative overflow-hidden rounded-2xl",
        className,
      )}
    >
      {children}
    </div>
  );
}
