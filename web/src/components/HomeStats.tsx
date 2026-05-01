import { useEffect, useState } from "react";
import { useAuth } from "@/lib/auth";
import { getUserStats, type UserStats } from "@/lib/users";
import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface HomeStatsProps {
  total: number;
}

const difficultyOrder = ["easy", "medium", "hard"];
const difficultyColors: Record<string, string> = {
  easy: "border-emerald-200 bg-emerald-50 text-emerald-900 dark:border-emerald-500/30 dark:bg-emerald-500/10 dark:text-emerald-300",
  medium:
    "border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300",
  hard: "border-rose-200 bg-rose-50 text-rose-900 dark:border-rose-500/30 dark:bg-rose-500/10 dark:text-rose-300",
};

const difficultyLabels: Record<string, string> = {
  easy: "Easy",
  medium: "Medium",
  hard: "Hard",
};

function StatPill({
  difficulty,
  solved,
  total,
}: {
  difficulty: string;
  solved: number;
  total: number;
}) {
  const percentage = total > 0 ? Math.round((solved / total) * 100) : 0;

  return (
    <div
      className={cn(
        "rounded-lg border px-3 py-2 text-sm font-medium",
        difficultyColors[difficulty]
      )}
    >
      <div className="flex items-baseline gap-1">
        <span className="font-semibold">{difficultyLabels[difficulty]}</span>
        <span className="text-xs opacity-75">
          {solved}/{total}
        </span>
      </div>
      <div className="mt-1 h-1.5 w-20 overflow-hidden rounded-full bg-black/10 dark:bg-white/10">
        <div
          className={cn(
            "h-full transition-all duration-500",
            difficulty === "easy"
              ? "bg-emerald-500"
              : difficulty === "medium"
                ? "bg-amber-500"
                : "bg-rose-500"
          )}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}

export function HomeStats({ total }: HomeStatsProps) {
  const { user } = useAuth();
  const [stats, setStats] = useState<UserStats | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!user) return;

    let cancelled = false;
    setLoading(true);

    getUserStats()
      .then((s) => {
        if (!cancelled) {
          setStats(s);
        }
      })
      .catch(() => {
        // Non-fatal: stats fetch failed
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [user]);

  if (!user) return null;

  const solvedByDifficulty: Record<string, number> = {
    easy: stats?.solved_by_difficulty?.easy ?? 0,
    medium: stats?.solved_by_difficulty?.medium ?? 0,
    hard: stats?.solved_by_difficulty?.hard ?? 0,
  };

  const perDifficulty = Math.ceil(total / 3);

  return (
    <div className="mb-6 flex gap-3">
      {difficultyOrder.map((diff) => (
        <div key={diff}>
          {loading ? (
            <Skeleton className="h-20 w-24 rounded-lg" />
          ) : (
            <StatPill
              difficulty={diff}
              solved={solvedByDifficulty[diff] || 0}
              total={perDifficulty}
            />
          )}
        </div>
      ))}
    </div>
  );
}
