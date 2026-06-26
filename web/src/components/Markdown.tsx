import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeSlug from "rehype-slug";
import rehypeHighlight from "rehype-highlight";
import { Link } from "react-router-dom";
import { resolveDocLink } from "../lib/docs";
import "../docs.css";

/** Render a docs markdown string with theme-aware components. */
export function Markdown({ source, slug }: { source: string; slug: string }) {
  return (
    <div className="markdown">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeSlug, rehypeHighlight]}
        components={{
          a({ node: _node, href, children }) {
            if (!href) return <>{children}</>;
            const resolved = resolveDocLink(slug, href);
            if (!resolved) {
              return (
                <a href={href} target="_blank" rel="noreferrer">
                  {children}
                </a>
              );
            }
            return <Link to={`${resolved.to}${resolved.hash}`}>{children}</Link>;
          },
        }}
      >
        {source}
      </ReactMarkdown>
    </div>
  );
}
