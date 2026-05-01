import { createFileRoute, Link } from "@tanstack/react-router";
import { ArrowLeft, BarChart3, TrendingUp, Zap, Code2 } from "lucide-react";
import React, { useEffect, useState } from "react";
import { getUserStats, type UserStats } from "@/lib/users";
import { useAuth } from "@/lib/auth";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

export const Route = createFileRoute("/dashboard")({
  component: DashboardPage,
});

function DashboardPage() {
  const { user } = useAuth();
  const [stats, setStats] = useState<UserStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!user) return;

    let cancelled = false;
    getUserStats()
      .then((s) => {
        if (!cancelled) setStats(s);
      })
      .catch(() => {})
      .finally(() => {
        if (!cancelled) setLoading(false);
      });

    return () => {
      cancelled = true;
    };
  }, [user]);

  if (!user) {
    return (
      <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
        <p>Please log in to view your dashboard.</p>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <div className="mb-6">
        <Button variant="ghost" size="sm" className="-ml-2 gap-1.5" asChild>
          <Link
            to="/"
            search={{
              q: undefined,
              difficulty: undefined,
              tags: undefined,
              page: 1,
              sort: undefined,
            }}
          >
            <ArrowLeft className="size-3.5" />
            Problems
          </Link>
        </Button>
      </div>

      <div className="mb-8 border-b border-border pb-6">
        <h1 className="text-3xl font-bold tracking-tight">Dashboard</h1>
        <p className="mt-1 max-w-lg text-sm text-muted-foreground">
          Track your progress and statistics
        </p>
      </div>

      {loading ? (
        <DashboardSkeleton />
      ) : stats ? (
        <DashboardContent stats={stats} />
      ) : (
        <div className="text-center py-12">
          <p className="text-muted-foreground">Failed to load dashboard</p>
        </div>
      )}
    </main>
  );
}

function DashboardContent({ stats }: { stats: UserStats }) {
  return (
    <div className="space-y-8">
      {/* Overview Cards */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Problems Solved"
          value={stats.total_solved}
          icon={<TrendingUp className="h-5 w-5" />}
        />
        <StatCard
          label="Total Submissions"
          value={stats.submission_stats.total_submissions}
          icon={<BarChart3 className="h-5 w-5" />}
        />
        <StatCard
          label="Acceptance Rate"
          value={`${stats.submission_stats.acceptance_rate.toFixed(1)}%`}
          icon={<Zap className="h-5 w-5" />}
        />
        <StatCard
          label="Test Runs"
          value={stats.submission_stats.total_test_runs}
          icon={<Code2 className="h-5 w-5" />}
        />
      </div>

      {/* Difficulty Breakdown */}
      <div className="rounded-lg border bg-card p-6">
        <h2 className="text-lg font-semibold mb-4">Difficulty Breakdown</h2>
        <div className="space-y-3">
          {["easy", "medium", "hard"].map((diff) => {
            const count = stats.solved_by_difficulty[diff] ?? 0;
            return (
              <DifficultyBar
                key={diff}
                difficulty={diff}
                count={count}
                total={stats.total_solved}
              />
            );
          })}
        </div>
      </div>

      {/* Tags */}
      {stats.solved_by_tag.length > 0 && (
        <div className="rounded-lg border bg-card p-6">
          <h2 className="text-lg font-semibold mb-4">Solved by Tag</h2>
          <div className="flex flex-wrap gap-2">
            {stats.solved_by_tag.map((tag) => (
              <Badge key={tag.tag_id} variant="secondary">
                {tag.tag_name}{" "}
                <span className="ml-1 text-xs opacity-75">({tag.count})</span>
              </Badge>
            ))}
          </div>
        </div>
      )}

      {/* Languages */}
      {stats.submission_stats.most_used_languages.length > 0 && (
        <div className="rounded-lg border bg-card p-6">
          <h2 className="text-lg font-semibold mb-4">Most Used Languages</h2>
          <div className="space-y-2">
            {stats.submission_stats.most_used_languages.map((lang) => (
              <div key={lang.language_key} className="flex items-center justify-between text-sm">
                <span className="font-medium">{lang.language_name}</span>
                <Badge variant="outline">{lang.count}</Badge>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Recent Submissions */}
      {stats.recent_submissions.length > 0 && (
        <div className="rounded-lg border bg-card p-6">
          <h2 className="text-lg font-semibold mb-4">Recent Submissions</h2>
          <div className="space-y-3">
            {stats.recent_submissions.slice(0, 5).map((sub) => (
              <div
                key={sub.problem_id}
                className="flex items-center justify-between border-b border-border/50 pb-3 last:border-0"
              >
                <div className="min-w-0 flex-1">
                  <p className="font-medium truncate">{sub.problem_title}</p>
                  <p className="text-xs text-muted-foreground">
                    {new Date(sub.created_at).toLocaleDateString()}
                  </p>
                </div>
                <div className="ml-2 flex items-center gap-2">
                  <Badge variant="outline" className="text-xs">
                    {sub.language_key}
                  </Badge>
                  <StatusBadge status={sub.status} />
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function StatCard({
  label,
  value,
  icon,
}: {
  label: string;
  value: string | number;
  icon: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border bg-card p-4">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-xs font-medium text-muted-foreground">{label}</p>
          <p className="mt-2 text-2xl font-bold tabular-nums">{value}</p>
        </div>
        <div className="text-muted-foreground/40">{icon}</div>
      </div>
    </div>
  );
}

function DifficultyBar({
  difficulty,
  count,
  total,
}: {
  difficulty: string;
  count: number;
  total: number;
}) {
  const percentage = total > 0 ? (count / total) * 100 : 0;
  const colors: Record<string, string> = {
    easy: "bg-emerald-500",
    medium: "bg-amber-500",
    hard: "bg-rose-500",
  };
  const labels: Record<string, string> = {
    easy: "Easy",
    medium: "Medium",
    hard: "Hard",
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-2 text-sm">
        <span className="font-medium capitalize">{labels[difficulty]}</span>
        <span className="text-muted-foreground">{count}</span>
      </div>
      <div className="h-2 w-full overflow-hidden rounded-full bg-muted/40">
        <div
          className={cn("h-full transition-all duration-500", colors[difficulty])}
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const statusMap: Record<string, { label: string; color: string }> = {
    accepted: { label: "Accepted", color: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-300" },
    wrong_answer: { label: "Wrong Answer", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300" },
    time_limit_exceeded: { label: "TLE", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300" },
    memory_limit_exceeded: { label: "MLE", color: "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300" },
    runtime_error: { label: "Runtime Error", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300" },
    compilation_error: { label: "Compile Error", color: "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300" },
  };

  const config = statusMap[status] || { label: status, color: "bg-gray-100 text-gray-800" };

  return (
    <span className={cn("inline-flex items-center rounded px-2 py-1 text-xs font-medium", config.color)}>
      {config.label}
    </span>
  );
}

function DashboardSkeleton() {
  return (
    <div className="space-y-8">
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-24 rounded-lg" />
        ))}
      </div>
      <Skeleton className="h-64 rounded-lg" />
      <Skeleton className="h-40 rounded-lg" />
    </div>
  );
}
