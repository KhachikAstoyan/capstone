"use client";

import type { Components } from "react-markdown";
import ReactMarkdown from "react-markdown";
import rehypeRaw from "rehype-raw";
import remarkGfm from "remark-gfm";

import { cn } from "@/lib/utils";

/**
 * Typography aligned with shadcn/ui guidance:
 * https://ui.shadcn.com/docs/components/radix/typography
 */
const components: Components = {
  h1: ({ className, ...props }) => (
    <h1
      className={cn(
        "scroll-m-20 text-balance text-2xl font-extrabold tracking-tight text-foreground",
        "first:mt-0 not-first:mt-8",
        className,
      )}
      {...props}
    />
  ),
  h2: ({ className, ...props }) => (
    <h2
      className={cn(
        "scroll-m-20 border-b border-border pb-2 text-xl font-semibold tracking-tight text-foreground first:mt-0",
        "not-first:mt-8",
        className,
      )}
      {...props}
    />
  ),
  h3: ({ className, ...props }) => (
    <h3
      className={cn(
        "scroll-m-20 text-lg font-semibold tracking-tight text-foreground first:mt-0",
        "not-first:mt-6",
        className,
      )}
      {...props}
    />
  ),
  h4: ({ className, ...props }) => (
    <h4
      className={cn(
        "scroll-m-20 text-base font-semibold tracking-tight text-foreground first:mt-0",
        "not-first:mt-6",
        className,
      )}
      {...props}
    />
  ),
  p: ({ className, ...props }) => (
    <p
      className={cn(
        "leading-7 text-foreground first:mt-0 not-first:mt-4",
        className,
      )}
      {...props}
    />
  ),
  blockquote: ({ className, ...props }) => (
    <blockquote
      className={cn(
        "mt-6 border-l-2 border-border pl-4 italic text-muted-foreground",
        className,
      )}
      {...props}
    />
  ),
  ul: ({ className, ...props }) => (
    <ul
      className={cn(
        "my-4 ml-6 list-disc marker:text-muted-foreground [&>li]:mt-2",
        className,
      )}
      {...props}
    />
  ),
  ol: ({ className, ...props }) => (
    <ol
      className={cn(
        "my-4 ml-6 list-decimal marker:text-muted-foreground [&>li]:mt-2",
        className,
      )}
      {...props}
    />
  ),
  li: ({ className, ...props }) => (
    <li className={cn("leading-7 text-foreground", className)} {...props} />
  ),
  strong: ({ className, ...props }) => (
    <strong className={cn("font-semibold", className)} {...props} />
  ),
  a: ({ className, ...props }) => (
    <a
      className={cn(
        "font-medium text-primary underline underline-offset-4 hover:text-primary/90",
        className,
      )}
      {...props}
    />
  ),
  hr: ({ className, ...props }) => (
    <hr className={cn("my-6 border-border", className)} {...props} />
  ),
  img: ({ className, alt, ...props }) => (
    <img
      className={cn("my-4 max-w-full rounded-md border border-border", className)}
      alt={alt ?? ""}
      {...props}
    />
  ),
  code: ({ className, children, ...props }) => {
    const isBlock = Boolean(className?.includes("language-"));
    if (isBlock) {
      return (
        <code className={cn("font-mono text-sm", className)} {...props}>
          {children}
        </code>
      );
    }
    return (
      <code
        className={cn(
          "relative rounded-md bg-muted px-[0.3rem] py-[0.2rem] font-mono text-[0.9em] font-semibold text-foreground",
          className,
        )}
        {...props}
      >
        {children}
      </code>
    );
  },
  pre: ({ className, children, ...props }) => (
    <pre
      className={cn(
        "my-4 overflow-x-auto rounded-lg border border-border bg-muted p-4 font-mono text-sm leading-relaxed text-foreground",
        "[&_code]:bg-transparent [&_code]:p-0 [&_code]:font-normal",
        className,
      )}
      {...props}
    >
      {children}
    </pre>
  ),
  table: ({ className, children, ...props }) => (
    <div className="my-6 w-full overflow-x-auto rounded-lg border border-border">
      <table
        className={cn("w-full border-collapse text-sm", className)}
        {...props}
      >
        {children}
      </table>
    </div>
  ),
  thead: ({ className, ...props }) => (
    <thead className={cn("border-b border-border bg-muted/60", className)} {...props} />
  ),
  tbody: ({ className, ...props }) => (
    <tbody className={cn("[&_tr:last-child]:border-0", className)} {...props} />
  ),
  tr: ({ className, ...props }) => (
    <tr
      className={cn("border-b border-border transition-colors hover:bg-muted/40", className)}
      {...props}
    />
  ),
  th: ({ className, ...props }) => (
    <th
      className={cn(
        "h-10 px-3 text-left align-middle font-medium text-foreground",
        className,
      )}
      {...props}
    />
  ),
  td: ({ className, ...props }) => (
    <td
      className={cn("px-3 py-2 align-middle text-foreground", className)}
      {...props}
    />
  ),
};

/** Legacy statements saved as HTML fragments (e.g. `<p>…</p>`) — keep typography via Tailwind prose. */
function isLikelyHtmlFragment(s: string): boolean {
  const t = s.trimStart();
  return t.startsWith("<") && /^<[a-zA-Z!?]/.test(t);
}

export function StatementMarkdown({ source }: { source: string }) {
  if (isLikelyHtmlFragment(source)) {
    return (
      <article
        className={cn(
          "prose prose-sm max-w-none text-foreground dark:prose-invert",
          "prose-headings:scroll-m-20 prose-headings:font-semibold prose-headings:tracking-tight",
          "prose-h1:text-2xl prose-h1:font-extrabold prose-h2:border-b prose-h2:pb-2 prose-h2:text-xl",
          "prose-p:leading-7 prose-p:first:mt-0",
          "prose-blockquote:border-l-2 prose-blockquote:border-border prose-blockquote:pl-4 prose-blockquote:italic prose-blockquote:text-muted-foreground",
          "prose-code:rounded-md prose-code:bg-muted prose-code:px-[0.3rem] prose-code:py-[0.2rem] prose-code:font-mono prose-code:text-[0.9em] prose-code:font-semibold prose-code:before:content-none prose-code:after:content-none",
          "prose-pre:rounded-lg prose-pre:border prose-pre:border-border prose-pre:bg-muted",
          "prose-a:font-medium prose-a:text-primary prose-a:underline prose-a:underline-offset-4",
          "prose-img:rounded-md prose-img:border prose-img:border-border",
          "prose-table:text-sm prose-th:border prose-th:border-border prose-td:border prose-td:border-border",
        )}
        data-slot="statement-markdown"
        dangerouslySetInnerHTML={{ __html: source }}
      />
    );
  }

  return (
    <article
      className="text-sm text-foreground"
      data-slot="statement-markdown"
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeRaw]}
        components={components}
      >
        {source}
      </ReactMarkdown>
    </article>
  );
}
