import { useEffect, useMemo, useState } from "react";
import { Link, NavLink, useParams } from "react-router-dom";
import { Markdown } from "../components/Markdown";
import { Icon } from "../components/Icon";
import { Button } from "../components/ui";
import {
  docRoute,
  extractToc,
  firstHeading,
  getDoc,
  labelForSlug,
  nav,
  neighbors,
  normalizeSlug,
} from "../lib/docs";
import { cn } from "../lib/cn";

function Sidebar({ slug }: { slug: string }) {
  return (
    <nav className="flex flex-col gap-7">
      {nav.map((section) => (
        <div key={section.title} className="flex flex-col gap-2">
          <p className="px-3 text-xs font-semibold uppercase tracking-[0.16em] text-ink-500">
            {section.title}
          </p>
          <ul className="flex flex-col gap-0.5">
            {section.items.map((item) => {
              const active = normalizeSlug(item.slug) === slug;
              return (
                <li key={item.slug}>
                  <NavLink
                    to={docRoute(item.slug)}
                    className={cn(
                      "block rounded-lg px-3 py-1.5 text-sm transition-colors",
                      active
                        ? "bg-forge-500/12 font-medium text-forge-200"
                        : "text-ink-400 hover:bg-ink-800/50 hover:text-ink-100",
                    )}
                  >
                    {item.label}
                  </NavLink>
                </li>
              );
            })}
          </ul>
        </div>
      ))}
    </nav>
  );
}

function useActiveHeading(ids: string[]) {
  const [active, setActive] = useState<string>("");
  useEffect(() => {
    if (ids.length === 0) return;
    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((e) => e.isIntersecting)
          .sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
        if (visible[0]) setActive(visible[0].target.id);
      },
      { rootMargin: "-80px 0px -70% 0px", threshold: 0 },
    );
    for (const id of ids) {
      const el = document.getElementById(id);
      if (el) observer.observe(el);
    }
    return () => observer.disconnect();
  }, [ids]);
  return active;
}

export default function Docs() {
  const params = useParams();
  const slug = normalizeSlug(params["*"] ?? "");
  const source = getDoc(slug);

  const toc = useMemo(() => (source ? extractToc(source) : []), [source]);
  const tocIds = useMemo(() => toc.map((t) => t.id), [toc]);
  const activeId = useActiveHeading(tocIds);
  const { prev, next } = neighbors(slug);

  useEffect(() => {
    const title = source ? firstHeading(source) ?? labelForSlug(slug) : "Not found";
    document.title = `${title} · Daedalus docs`;
  }, [source, slug]);

  return (
    <div className="border-t border-ink-800/60 bg-grid bg-grid-fade">
      <div className="mx-auto grid w-full max-w-[88rem] gap-10 px-5 py-10 sm:px-8 lg:grid-cols-[15rem_minmax(0,1fr)] xl:grid-cols-[15rem_minmax(0,1fr)_14rem]">
        {/* Sidebar (desktop) */}
        <aside className="hidden lg:block">
          <div className="sticky top-24 max-h-[calc(100vh-7rem)] overflow-y-auto pr-2">
            <Sidebar slug={slug} />
          </div>
        </aside>

        {/* Mobile section picker */}
        <details className="ring-hairline group rounded-xl bg-ink-900/60 lg:hidden">
          <summary className="flex cursor-pointer items-center justify-between px-4 py-3 text-sm font-medium text-ink-100">
            <span className="flex items-center gap-2">
              <Icon name="book" size={16} /> Documentation
            </span>
            <Icon
              name="arrow"
              size={16}
              className="rotate-90 transition-transform group-open:rotate-[270deg]"
            />
          </summary>
          <div className="border-t border-ink-800/60 p-4">
            <Sidebar slug={slug} />
          </div>
        </details>

        {/* Content */}
        <article className="min-w-0">
          {source ? (
            <>
              <nav className="mb-6 flex items-center gap-2 text-xs text-ink-500">
                <Link to="/docs" className="hover:text-ink-200">
                  Docs
                </Link>
                <span>/</span>
                <span className="text-ink-300">{labelForSlug(slug)}</span>
              </nav>

              <Markdown source={source} slug={slug} />

              <div className="mt-14 grid gap-3 border-t border-ink-800/60 pt-8 sm:grid-cols-2">
                {prev ? (
                  <Link
                    to={docRoute(prev.slug)}
                    className="ring-hairline group rounded-xl bg-ink-900/50 p-4 transition-colors hover:bg-ink-800/50"
                  >
                    <span className="text-xs text-ink-500">Previous</span>
                    <span className="mt-1 flex items-center gap-1.5 font-medium text-ink-100">
                      <Icon name="arrow" size={15} className="rotate-180" />
                      {prev.label}
                    </span>
                  </Link>
                ) : (
                  <span />
                )}
                {next && (
                  <Link
                    to={docRoute(next.slug)}
                    className="ring-hairline group rounded-xl bg-ink-900/50 p-4 text-right transition-colors hover:bg-ink-800/50 sm:col-start-2"
                  >
                    <span className="text-xs text-ink-500">Next</span>
                    <span className="mt-1 flex items-center justify-end gap-1.5 font-medium text-ink-100">
                      {next.label}
                      <Icon name="arrow" size={15} />
                    </span>
                  </Link>
                )}
              </div>
            </>
          ) : (
            <div className="flex flex-col items-start gap-4 py-16">
              <h1 className="font-display text-2xl font-semibold text-ink-50">
                Page not found
              </h1>
              <p className="text-ink-400">
                That documentation page doesn't exist (yet).
              </p>
              <Button to="/docs" icon="book">
                Back to the docs
              </Button>
            </div>
          )}
        </article>

        {/* On this page (TOC) */}
        {toc.length > 1 && (
          <aside className="hidden xl:block">
            <div className="sticky top-24">
              <p className="mb-3 text-xs font-semibold uppercase tracking-[0.16em] text-ink-500">
                On this page
              </p>
              <ul className="flex flex-col gap-1 border-l border-ink-800/70">
                {toc.map((entry) => (
                  <li key={entry.id}>
                    <a
                      href={`#${entry.id}`}
                      className={cn(
                        "-ml-px block border-l py-1 text-sm transition-colors",
                        entry.level === 3 ? "pl-6" : "pl-4",
                        activeId === entry.id
                          ? "border-forge-400 text-forge-200"
                          : "border-transparent text-ink-500 hover:text-ink-200",
                      )}
                    >
                      {entry.text}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          </aside>
        )}
      </div>
    </div>
  );
}
