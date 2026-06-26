import { useEffect, useState } from "react";
import { Link, useLocation } from "react-router-dom";
import { AnimatePresence, motion } from "framer-motion";
import { Container, Button } from "./ui";
import { Icon } from "./Icon";
import { Labyrinth } from "./Labyrinth";
import { GITHUB_URL, navLinks, QUICKSTART } from "../lib/site";
import { cn } from "../lib/cn";

function Wordmark() {
  return (
    <Link
      to="/"
      className="group flex items-center gap-2.5"
      aria-label="Daedalus home"
    >
      <span className="grid h-9 w-9 place-items-center rounded-lg bg-ink-800/80 ring-hairline transition-transform duration-300 group-hover:rotate-[-6deg]">
        <Labyrinth className="h-5 w-5" animate={false} strokeWidth={3} />
      </span>
      <span className="font-display text-lg font-semibold tracking-tight text-ink-50">
        Daedalus
      </span>
    </Link>
  );
}

export default function Navbar() {
  const [scrolled, setScrolled] = useState(false);
  const [open, setOpen] = useState(false);
  const { pathname } = useLocation();

  useEffect(() => {
    const onScroll = () => setScrolled(window.scrollY > 8);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);

  useEffect(() => setOpen(false), [pathname]);

  return (
    <header className="sticky top-0 z-50">
      <div
        className={cn(
          "transition-colors duration-300",
          scrolled || open
            ? "border-b border-ink-700/60 bg-ink-950/80 backdrop-blur-xl"
            : "border-b border-transparent",
        )}
      >
        <Container className="flex h-16 items-center justify-between">
          <Wordmark />

          <nav className="hidden items-center gap-1 md:flex">
            {navLinks.map((link) => (
              <Link
                key={link.href}
                to={link.href}
                className="rounded-lg px-3 py-2 text-sm text-ink-300 transition-colors hover:text-ink-50"
              >
                {link.label}
              </Link>
            ))}
          </nav>

          <div className="hidden items-center gap-2 md:flex">
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noreferrer"
              aria-label="Daedalus on GitHub"
              className="grid h-10 w-10 place-items-center rounded-lg text-ink-300 transition-colors hover:bg-ink-800/60 hover:text-ink-50"
            >
              <Icon name="github" size={20} />
            </a>
            <Button to={QUICKSTART} size="md" iconRight="arrow">
              Get started
            </Button>
          </div>

          <button
            type="button"
            className="grid h-10 w-10 place-items-center rounded-lg text-ink-200 md:hidden"
            aria-label={open ? "Close menu" : "Open menu"}
            aria-expanded={open}
            onClick={() => setOpen((v) => !v)}
          >
            <Icon name={open ? "close" : "menu"} size={22} />
          </button>
        </Container>
      </div>

      <AnimatePresence>
        {open && (
          <motion.div
            initial={{ opacity: 0, height: 0 }}
            animate={{ opacity: 1, height: "auto" }}
            exit={{ opacity: 0, height: 0 }}
            transition={{ duration: 0.25, ease: [0.16, 1, 0.3, 1] }}
            className="overflow-hidden border-b border-ink-700/60 bg-ink-950/95 backdrop-blur-xl md:hidden"
          >
            <Container className="flex flex-col gap-1 py-4">
              {navLinks.map((link) => (
                <Link
                  key={link.href}
                  to={link.href}
                  className="rounded-lg px-3 py-3 text-sm text-ink-200 hover:bg-ink-800/60"
                >
                  {link.label}
                </Link>
              ))}
              <div className="mt-2 flex items-center gap-2">
                <Button to={QUICKSTART} className="flex-1" iconRight="arrow">
                  Get started
                </Button>
                <Button href={GITHUB_URL} variant="secondary" icon="github">
                  GitHub
                </Button>
              </div>
            </Container>
          </motion.div>
        )}
      </AnimatePresence>
    </header>
  );
}
