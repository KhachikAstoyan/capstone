import { CheckCircle2 } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import type { Difficulty, Problem, Tag } from "@/lib/problems";
// index param kept for API compatibility but no longer rendered
import { DifficultyBadge } from "./DifficultyBadge";
import { cn } from "@/lib/utils";

const difficultyBarColor: Record<Difficulty, string> = {
  easy: "bg-emerald-500",
  medium: "bg-amber-500",
  hard: "bg-rose-500",
};

function stripHtml(html: string): string {
  return html
    .replace(/<[^>]*>/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function excerpt(text: string, maxLen = 120): string {
  const t = text.trim();
  if (!t) return "";
  return t.length > maxLen ? t.slice(0, maxLen).trimEnd() + "…" : t;
}

function listPreview(problem: Problem, maxLen = 120): string {
  const s = problem.summary?.trim();
  if (s) return excerpt(s, maxLen);
  return excerpt(stripHtml(problem.statement_markdown), maxLen);
}

interface ProblemCardProps {
  problem: Problem;
  index: number; // kept for API compatibility
}

export function ProblemCard({ problem }: ProblemCardProps) {
  return (
    <div
      className={cn(
        "group relative flex flex-col gap-3 overflow-hidden rounded-lg border bg-card p-4 text-card-foreground shadow-sm transition-all duration-150 hover:-translate-y-px hover:shadow-md",
        problem.is_solved && "ring-1 ring-emerald-500/30",
      )}
    >
      {/* Difficulty left-border accent */}
      <div
        className={cn(
          "absolute inset-y-0 left-0 w-0.75",
          difficultyBarColor[problem.difficulty],
        )}
      />

      <div className="flex items-start justify-between gap-2">
        <a
          href={`/problems/${problem.slug}`}
          className="min-w-0 truncate font-semibold text-foreground no-underline hover:text-primary hover:underline"
        >
          {problem.title}
        </a>
        <DifficultyBadge difficulty={problem.difficulty} className="shrink-0" />
      </div>

      {(problem.summary?.trim() || problem.statement_markdown) && (
        <p className="line-clamp-2 text-sm text-muted-foreground whitespace-pre-line">
          {listPreview(problem)}
        </p>
      )}

      {problem.tags && problem.tags.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {problem.tags.map((tag: Tag) => (
            <Badge
              key={tag.id}
              variant="outline"
              className="text-xs font-normal"
            >
              {tag.name}
            </Badge>
          ))}
        </div>
      )}

      <div className="mt-auto flex items-center justify-between text-xs text-muted-foreground">
        {problem.acceptance_rate !== undefined ? (
          <span>{problem.acceptance_rate.toFixed(1)}% acceptance</span>
        ) : (
          <span />
        )}
        {problem.is_solved && (
          <span className="flex items-center gap-1 text-green-600 dark:text-green-400">
            <CheckCircle2 className="h-3.5 w-3.5" />
            Solved
          </span>
        )}
      </div>
    </div>
  );
}

export function ProblemCardSkeleton() {
  return (
    <div className="flex flex-col gap-3 rounded-lg border bg-card p-4 shadow-sm">
      <div className="flex items-start justify-between gap-2">
        <div className="flex flex-1 items-baseline gap-2">
          <Skeleton className="h-4 w-6" />
          <Skeleton className="h-4 flex-1" />
        </div>
        <Skeleton className="h-5 w-14 rounded-full" />
      </div>
      <div className="flex flex-col gap-1.5">
        <Skeleton className="h-3 w-full" />
        <Skeleton className="h-3 w-4/5" />
      </div>
      <div className="flex gap-1">
        <Skeleton className="h-5 w-16 rounded-full" />
        <Skeleton className="h-5 w-20 rounded-full" />
      </div>
      <div className="flex justify-between">
        <Skeleton className="h-3 w-24" />
      </div>
    </div>
  );
}
