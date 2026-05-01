import { useRouter } from "@tanstack/react-router";
import { CheckCircle2, Circle } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { DifficultyBadge } from "@/components/DifficultyBadge";
import type { Problem, Tag } from "@/lib/problems";

interface ProblemTableProps {
  problems: Problem[];
  startIndex: number;
  sort?: string;
}

function getProblemStatus(problem: Problem): "solved" | "unsolved" {
  return problem.is_solved ? "solved" : "unsolved";
}

export function ProblemTable({
  problems,
  startIndex,
  sort,
}: ProblemTableProps) {
  const router = useRouter();

  const handleRowClick = (slug: string) => {
    router.navigate({ to: `/problems/${slug}` });
  };

  return (
    <div className="overflow-x-auto rounded-lg border border-border">
      <table className="w-full">
        <thead>
          <tr className="border-b border-border bg-muted/40">
            <th className="w-10 px-3 py-3 text-left">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Status
              </span>
            </th>
            <th className="w-12 px-3 py-3 text-left">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                #
              </span>
            </th>
            <th className="px-4 py-3 text-left">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Title
              </span>
            </th>
            <th className="px-3 py-3 text-left">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Tags
              </span>
            </th>
            <th className="w-24 px-3 py-3 text-left">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Difficulty
              </span>
            </th>
            <th className="w-24 px-3 py-3 text-right">
              <span className="text-xs font-semibold uppercase tracking-wider text-muted-foreground">
                Acceptance
              </span>
            </th>
          </tr>
        </thead>
        <tbody>
          {problems.map((problem, idx) => {
            const status = getProblemStatus(problem);
            const isSolved = status === "solved";
            const number = startIndex + idx;

            return (
              <tr
                key={problem.id}
                onClick={() => handleRowClick(problem.slug)}
                className="cursor-pointer border-b border-border/50 transition-colors hover:bg-muted/40 active:bg-muted/60"
              >
                {/* Status */}
                <td className="px-3 py-3">
                  {isSolved ? (
                    <CheckCircle2 className="h-4 w-4 text-emerald-500" />
                  ) : (
                    <Circle className="h-4 w-4 text-muted-foreground/30" />
                  )}
                </td>

                {/* Number */}
                <td className="px-3 py-3 text-sm font-medium text-muted-foreground">
                  {number}
                </td>

                {/* Title */}
                <td className="px-4 py-3">
                  <span className="text-sm font-medium text-foreground">
                    {problem.title}
                  </span>
                </td>

                {/* Tags */}
                <td className="px-3 py-3">
                  <div className="flex flex-wrap gap-1">
                    {(problem.tags ?? []).slice(0, 2).map((tag: Tag) => (
                      <Badge
                        key={tag.id}
                        variant="outline"
                        className="text-xs font-normal"
                      >
                        {tag.name}
                      </Badge>
                    ))}
                    {(problem.tags ?? []).length > 2 && (
                      <Badge
                        variant="outline"
                        className="text-xs font-normal text-muted-foreground"
                      >
                        +{(problem.tags ?? []).length - 2}
                      </Badge>
                    )}
                  </div>
                </td>

                {/* Difficulty */}
                <td className="px-3 py-3">
                  <DifficultyBadge difficulty={problem.difficulty} />
                </td>

                {/* Acceptance */}
                <td className="px-3 py-3 text-right">
                  <span className="text-sm tabular-nums text-muted-foreground">
                    {problem.acceptance_rate !== undefined
                      ? `${problem.acceptance_rate.toFixed(1)}%`
                      : "—"}
                  </span>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
