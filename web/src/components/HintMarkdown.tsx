import type { Components } from "react-markdown";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import "katex/dist/katex.min.css";

import { cn } from "@/lib/utils";

/**
 * LLM hints arrive with LaTeX in the `\( … \)` / `\[ … \]` delimiters that
 * OpenAI-style models emit. `remark-math` only recognizes `$ … $` / `$$ … $$`,
 * so normalize the delimiters before parsing.
 */
function normalizeMath(src: string): string {
  return src
    .replace(/\\\[([\s\S]+?)\\\]/g, (_, body) => `$$${body}$$`)
    .replace(/\\\(([\s\S]+?)\\\)/g, (_, body) => `$${body}$`);
}

/** Compact typography sized for the hint chat bubble (text-xs context). */
const components: Components = {
  p: ({ className, ...props }) => (
    <p className={cn("first:mt-0 not-first:mt-2", className)} {...props} />
  ),
  ul: ({ className, ...props }) => (
    <ul
      className={cn("my-2 ml-4 list-disc marker:text-muted-foreground [&>li]:mt-1", className)}
      {...props}
    />
  ),
  ol: ({ className, ...props }) => (
    <ol
      className={cn("my-2 ml-4 list-decimal marker:text-muted-foreground [&>li]:mt-1", className)}
      {...props}
    />
  ),
  li: ({ className, ...props }) => (
    <li className={cn("leading-relaxed", className)} {...props} />
  ),
  strong: ({ className, ...props }) => (
    <strong className={cn("font-semibold", className)} {...props} />
  ),
  a: ({ className, ...props }) => (
    <a
      className={cn("font-medium underline underline-offset-2", className)}
      target="_blank"
      rel="noreferrer"
      {...props}
    />
  ),
  code: ({ className, children, ...props }) => {
    const isBlock = Boolean(className?.includes("language-"));
    if (isBlock) {
      return (
        <code className={cn("font-mono text-[0.85em]", className)} {...props}>
          {children}
        </code>
      );
    }
    return (
      <code
        className={cn(
          "rounded bg-background/60 px-1 py-[0.1rem] font-mono text-[0.85em]",
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
        "my-2 overflow-x-auto rounded-md border border-border bg-background/60 p-2 font-mono text-[0.85em] leading-relaxed",
        "[&_code]:bg-transparent [&_code]:p-0",
        className,
      )}
      {...props}
    >
      {children}
    </pre>
  ),
};

export function HintMarkdown({ content }: { content: string }) {
  return (
    <div className="[&_.katex]:text-[1em]">
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath]}
        rehypePlugins={[rehypeKatex]}
        components={components}
      >
        {normalizeMath(content)}
      </ReactMarkdown>
    </div>
  );
}
