import { useEffect, useState } from "react";
import { ChevronDown } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Skeleton } from "@/components/ui/skeleton";
import { listSubmissions, type Submission } from "@/lib/submissions";
import { ApiError } from "@/lib/api";
import { toast } from "sonner";

interface SubmissionsTabProps {
  problemId: string;
}

const STATUS_COLORS: Record<string, string> = {
  accepted: "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
  wrong_answer:
    "bg-rose-500/15 text-rose-700 dark:text-rose-300",
  time_limit_exceeded:
    "bg-orange-500/15 text-orange-700 dark:text-orange-300",
  memory_limit_exceeded:
    "bg-amber-500/15 text-amber-700 dark:text-amber-300",
  runtime_error:
    "bg-red-500/15 text-red-700 dark:text-red-300",
  compilation_error:
    "bg-yellow-500/15 text-yellow-700 dark:text-yellow-300",
  pending: "bg-slate-500/15 text-slate-700 dark:text-slate-300",
  queued: "bg-blue-500/15 text-blue-700 dark:text-blue-300",
  running: "bg-cyan-500/15 text-cyan-700 dark:text-cyan-300",
  internal_error:
    "bg-red-500/15 text-red-700 dark:text-red-300",
};

const STATUS_LABELS: Record<string, string> = {
  pending: "Pending",
  queued: "Queued",
  running: "Running",
  accepted: "Accepted",
  wrong_answer: "Wrong Answer",
  time_limit_exceeded: "TLE",
  memory_limit_exceeded: "MLE",
  runtime_error: "Runtime Error",
  compilation_error: "Compilation Error",
  internal_error: "Internal Error",
};

export function SubmissionsTab({ problemId }: SubmissionsTabProps) {
  const [submissions, setSubmissions] = useState<Submission[]>([]);
  const [loading, setLoading] = useState(true);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [showAll, setShowAll] = useState(false);

  useEffect(() => {
    let cancelled = false;

    (async () => {
      try {
        setLoading(true);
        const list = await listSubmissions({
          problemId,
          limit: 50,
        });
        if (!cancelled) {
          setSubmissions(list.submissions || []);
        }
      } catch (err) {
        if (!cancelled) {
          toast.error(
            err instanceof ApiError
              ? err.message
              : "Failed to load submissions"
          );
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [problemId]);

  if (loading) {
    return (
      <div className="space-y-2 p-4">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-10 w-full" />
        ))}
      </div>
    );
  }

  if (submissions.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center gap-2 py-8 text-center">
        <p className="text-sm font-medium">No submissions yet</p>
        <p className="text-xs text-muted-foreground">
          Test run or submit your solution to see history
        </p>
      </div>
    );
  }

  // Filter to show only submits by default, unless showAll is true
  const filteredSubmissions = showAll
    ? submissions
    : submissions.filter((s) => s.kind === "submit");

  const displaySubmissions = filteredSubmissions.slice(0, 20);
  const hasMore = filteredSubmissions.length > 20;

  return (
    <div className="flex h-full flex-col min-h-0 bg-card">
      <div className="flex shrink-0 items-center justify-between border-b border-border px-4 py-2">
        <label className="flex items-center gap-2 text-xs font-medium">
          <input
            type="checkbox"
            checked={showAll}
            onChange={(e) => setShowAll(e.target.checked)}
            className="rounded border-border"
          />
          <span>Show test runs</span>
        </label>
        <span className="text-xs text-muted-foreground">
          {displaySubmissions.length} / {filteredSubmissions.length}
        </span>
      </div>

      <div className="flex-1 min-h-0 overflow-auto">
        {displaySubmissions.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-2 py-8 text-center">
            <p className="text-sm font-medium">No test runs</p>
            <p className="text-xs text-muted-foreground">
              Check "Show test runs" to see them
            </p>
          </div>
        ) : (
          <div className="divide-y divide-border/50">
            {displaySubmissions.map((sub) => {
              const expanded = expandedId === sub.id;
              const time = new Date(sub.created_at);
              const timeStr = time.toLocaleTimeString([], {
                hour: "2-digit",
                minute: "2-digit",
              });

              return (
                <div key={sub.id}>
                  <button
                    className="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-muted/40"
                    onClick={() =>
                      setExpandedId(expanded ? null : sub.id)
                    }
                  >
                    <span className="text-xs text-muted-foreground w-12 tabular-nums">
                      {timeStr}
                    </span>
                    <Badge
                      className={`shrink-0 text-xs font-medium ${
                        STATUS_COLORS[sub.status] ||
                        STATUS_COLORS.pending
                      }`}
                    >
                      {STATUS_LABELS[sub.status] || sub.status}
                    </Badge>
                    <span className="text-xs text-muted-foreground">
                      {sub.language_key?.toUpperCase()}
                    </span>
                    <div className="ml-auto flex items-center gap-3 text-xs tabular-nums text-muted-foreground">
                      {sub.result?.total_time_ms && (
                        <span>{sub.result.total_time_ms}ms</span>
                      )}
                      {sub.result?.max_memory_kb && (
                        <span>
                          {(sub.result.max_memory_kb / 1024).toFixed(1)}MB
                        </span>
                      )}
                      <ChevronDown
                        className={`size-3.5 transition-transform ${
                          expanded ? "rotate-180" : ""
                        }`}
                      />
                    </div>
                  </button>

                  {expanded && sub.source_text && (
                    <div className="border-t border-border/50 bg-muted/20 px-4 py-3">
                      <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground mb-2">
                        Source Code
                      </p>
                      <pre className="rounded border border-border bg-zinc-950 p-2 font-mono text-[10px] leading-relaxed overflow-x-auto text-emerald-400 max-h-40">
                        {sub.source_text}
                      </pre>
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
      </div>

      {hasMore && (
        <div className="border-t border-border bg-muted/20 px-4 py-2 text-center text-xs text-muted-foreground">
          +{filteredSubmissions.length - displaySubmissions.length} more submissions
        </div>
      )}
    </div>
  );
}
