import { Link } from "react-router-dom";
import { Container } from "./ui";
import { Icon } from "./Icon";
import { Labyrinth } from "./Labyrinth";
import { footerColumns, GITHUB_URL } from "../lib/site";

export default function Footer() {
  return (
    <footer className="relative mt-10 border-t border-ink-800/70 bg-ink-950">
      <Container className="py-14">
        <div className="grid gap-10 md:grid-cols-[1.4fr_repeat(3,1fr)]">
          <div className="flex flex-col gap-4">
            <Link to="/" className="flex items-center gap-2.5" aria-label="Daedalus home">
              <span className="grid h-9 w-9 place-items-center rounded-lg bg-ink-800/80 ring-hairline">
                <Labyrinth className="h-5 w-5" animate={false} strokeWidth={3} />
              </span>
              <span className="font-display text-lg font-semibold text-ink-50">
                Daedalus
              </span>
            </Link>
            <p className="max-w-xs text-sm leading-relaxed text-ink-400">
              Design your project's AI scaffolding once. Compile it to your
              tool's native format.
            </p>
            <a
              href={GITHUB_URL}
              target="_blank"
              rel="noreferrer"
              className="inline-flex w-fit items-center gap-2 text-sm text-ink-300 transition-colors hover:text-ink-50"
            >
              <Icon name="github" size={18} />
              Codigo-de-Altura/Daedalus
            </a>
          </div>

          {footerColumns.map((col) => (
            <div key={col.title} className="flex flex-col gap-3">
              <h3 className="font-display text-sm font-semibold text-ink-200">
                {col.title}
              </h3>
              <ul className="flex flex-col gap-2.5">
                {col.links.map((link) => (
                  <li key={link.label}>
                    {"external" in link && link.external ? (
                      <a
                        href={link.href}
                        target="_blank"
                        rel="noreferrer"
                        className="text-sm text-ink-400 transition-colors hover:text-forge-300"
                      >
                        {link.label}
                      </a>
                    ) : (
                      <Link
                        to={link.href}
                        className="text-sm text-ink-400 transition-colors hover:text-forge-300"
                      >
                        {link.label}
                      </Link>
                    )}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="mt-12 flex flex-col items-start justify-between gap-4 border-t border-ink-800/70 pt-6 sm:flex-row sm:items-center">
          <p className="text-xs text-ink-500">
            © {new Date().getFullYear()} Daedalus · Código de Altura. Built with
            care.
          </p>
          <p className="font-mono text-xs text-ink-600">
            do, or do not — there is no try
          </p>
        </div>
      </Container>
    </footer>
  );
}
