import { useState, useEffect, type ReactNode } from "react";
import { createFileRoute, Link, notFound } from "@tanstack/react-router";
import {
  ArrowLeft,
  Calendar,
  CheckCircle2,
  ChevronDown,
  Code2,
  Copy,
  Mail,
  Trophy,
  User as UserIcon,
} from "lucide-react";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { Skeleton } from "@/components/ui/skeleton";
import { DifficultyBadge } from "@/components/DifficultyBadge";
import { useAuth } from "@/lib/auth";
import { ApiError } from "@/lib/api";
import { getPublicProfile, type PublicUserProfile } from "@/lib/users";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

function getInitials(displayName: string | undefined, handle: string): string {
  const base = displayName?.trim() || handle;
  return base
    .split(/\s+/)
    .map((w) => w[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString(undefined, {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}

function formatDateTime(value: string): string {
  return new Date(value).toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export const Route = createFileRoute("/users/$userRef")({
  loader: async ({ params }) => {
    try {
      const profile = await getPublicProfile(params.userRef);
      return { profile };
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        throw notFound();
      }
      throw e;
    }
  },
  pendingComponent: ProfileSkeleton,
  notFoundComponent: ProfileNotFound,
  component: UserProfilePage,
});

function UserProfilePage() {
  const { profile } = Route.useLoaderData() as { profile: PublicUserProfile };
  const { user: currentUser } = useAuth();
  const isSelf = currentUser?.id === profile.id;

  const display = profile.display_name?.trim() || profile.handle;
  const solved = profile.solved_problems ?? [];

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
            }}
          >
            <ArrowLeft className="size-3.5" />
            Problems
          </Link>
        </Button>
      </div>

      {/* Profile header */}
      <div className="rounded-lg border bg-card p-6 sm:p-8">
        <div className="flex flex-col gap-6 sm:flex-row sm:items-start">
          <Avatar className="size-20 shrink-0 sm:size-24">
            <AvatarImage src={profile.avatar_url} alt={display} />
            <AvatarFallback className="text-xl font-semibold">
              {getInitials(profile.display_name, profile.handle)}
            </AvatarFallback>
          </Avatar>

          <div className="min-w-0 flex-1 space-y-3">
            <div className="flex flex-wrap items-center gap-2">
              <h1 className="text-2xl font-bold tracking-tight sm:text-3xl">
                {display}
              </h1>
              {isSelf && (
                <Badge variant="secondary" className="text-xs">
                  You
                </Badge>
              )}
            </div>

            <div className="flex flex-wrap gap-x-4 gap-y-1.5 text-sm text-muted-foreground">
              <Meta icon={<UserIcon className="size-3.5" />}>
                @{profile.handle}
              </Meta>
              {isSelf && currentUser?.email && (
                <Meta icon={<Mail className="size-3.5" />}>
                  {currentUser.email}
                </Meta>
              )}
              <Meta icon={<Calendar className="size-3.5" />}>
                Joined {formatDate(profile.created_at)}
              </Meta>
              <Meta icon={<StatusDot status={profile.status} />}>
                <span className="capitalize">{profile.status.toLowerCase()}</span>
              </Meta>
            </div>
          </div>

          <div className="flex gap-3 sm:flex-col sm:gap-2">
            <StatTile
              icon={<Trophy className="size-4" />}
              label="Solved"
              value={solved.length}
            />
            <StatTile
              icon={<Code2 className="size-4" />}
              label="Solutions"
              value={solved.length}
            />
          </div>
        </div>
      </div>

      {/* Solved problems */}
      <div className="mt-8">
        <div className="mb-4">
          <h2 className="text-lg font-semibold">Solved problems</h2>
          <p className="mt-0.5 text-sm text-muted-foreground">
            {solved.length === 0
              ? "Accepted submissions will appear here."
              : `${solved.length} accepted submission${solved.length === 1 ? "" : "s"}`}
          </p>
        </div>

        {solved.length === 0 ? (
          <div className="flex flex-col items-center justify-center gap-2 rounded-lg border border-dashed px-6 py-16 text-center">
            <Trophy className="size-8 text-muted-foreground/40" />
            <p className="text-sm font-medium">No accepted submissions yet</p>
            <p className="max-w-sm text-xs text-muted-foreground">
              Solve a problem and the accepted code will be archived here.
            </p>
          </div>
        ) : (
          <div className="grid gap-3">
            {solved.map((item) => (
              <SolvedProblemCard key={item.solution.id} item={item} />
            ))}
          </div>
        )}
      </div>
    </main>
  );
}

function Meta({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      {icon}
      {children}
    </span>
  );
}

function StatusDot({ status }: { status: string }) {
  const ok = status.toLowerCase() === "active";
  return (
    <span
      className={cn(
        "inline-block size-2 rounded-full",
        ok ? "bg-green-500" : "bg-red-500",
      )}
    />
  );
}

function StatTile({
  icon,
  label,
  value,
}: {
  icon: ReactNode;
  label: string;
  value: number;
}) {
  return (
    <div className="rounded-lg border bg-muted/40 px-4 py-3 text-left min-w-28">
      <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground">
        {icon}
        {label}
      </div>
      <div className="mt-1 text-2xl font-bold tabular-nums">{value}</div>
    </div>
  );
}

type SolvedItem = NonNullable<PublicUserProfile["solved_problems"]>[number];

function SolvedProblemCard({ item }: { item: SolvedItem }) {
  const [open, setOpen] = useState(false);
  const source = item.solution.source_text?.trim() ?? "";
  const langLabel =
    item.solution.language_display_name || item.solution.language_key;

  function copyCode() {
    if (!source) return;
    navigator.clipboard.writeText(source);
    toast.success("Code copied.");
  }

  return (
    <div className="rounded-lg border bg-card overflow-hidden">
      <div className="flex flex-col gap-3 p-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0 flex-1 space-y-1.5">
          <div className="flex flex-wrap items-center gap-2">
            <Link
              to="/problems/$problemSlug"
              params={{ problemSlug: item.slug }}
              className="min-w-0 truncate text-base font-semibold hover:underline"
            >
              {item.title}
            </Link>
            <DifficultyBadge difficulty={item.difficulty} />
          </div>

          {item.summary && (
            <p className="line-clamp-2 max-w-2xl text-sm text-muted-foreground">
              {item.summary}
            </p>
          )}

          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted-foreground">
            <Badge variant="outline" className="gap-1 font-normal">
              <Code2 className="size-3" />
              {langLabel}
            </Badge>
            <span className="inline-flex items-center gap-1 text-green-600 dark:text-green-500 font-medium">
              <CheckCircle2 className="size-3" />
              Accepted
            </span>
            <span>{formatDateTime(item.solved_at)}</span>
          </div>
        </div>

        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={() => setOpen((v) => !v)}
          className="h-8 shrink-0 gap-1.5 text-xs"
        >
          {open ? "Hide solution" : "View solution"}
          <ChevronDown
            className={cn("size-3.5 transition-transform", open && "rotate-180")}
          />
        </Button>
      </div>

      {open && (
        <>
          <Separator />
          <div className="flex items-center justify-between px-4 py-2">
            <span className="text-xs text-muted-foreground">
              {langLabel} · solution
            </span>
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-7"
              onClick={copyCode}
              disabled={!source}
              title="Copy code"
            >
              <Copy className="size-3.5" />
            </Button>
          </div>
          <Separator />
          {source ? (
            <CodeBlock code={source} langKey={item.solution.language_key} />
          ) : (
            <p className="px-4 py-3 text-xs text-muted-foreground">
              Source code is not available for this submission.
            </p>
          )}
        </>
      )}
    </div>
  );
}

function langKeyToShiki(key: string): string {
  const map: Record<string, string> = {
    python3: "python",
    javascript: "javascript",
    go: "go",
    cpp: "cpp",
    java: "java",
    rust: "rust",
    typescript: "typescript",
  };
  return map[key.toLowerCase()] ?? "text";
}

const capstoneDarkTheme = {
  name: "capstone-dark",
  type: "dark" as const,
  colors: {
    "editor.background": "#0d0d0d",
    "editor.foreground": "#e5e7eb",
  },
  tokenColors: [
    {
      scope: ["comment", "punctuation.definition.comment", "comment.block", "comment.line"],
      settings: { foreground: "#6b7280", fontStyle: "italic" },
    },
    {
      scope: ["keyword", "storage.type", "storage.modifier", "keyword.control", "keyword.operator.new"],
      settings: { foreground: "#a78bfa" },
    },
    {
      scope: ["string", "string.quoted", "string.template"],
      settings: { foreground: "#6ee7b7" },
    },
    {
      scope: ["constant.numeric", "constant.language.boolean"],
      settings: { foreground: "#fb923c" },
    },
    {
      scope: ["entity.name.type", "support.type", "storage.type.class", "entity.name.class"],
      settings: { foreground: "#67e8f9" },
    },
    {
      scope: ["entity.name.function", "support.function", "meta.function-call"],
      settings: { foreground: "#818cf8" },
    },
  ],
};

function CodeBlock({ code, langKey }: { code: string; langKey: string }) {
  const [html, setHtml] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    import("shiki").then(({ codeToHtml }) =>
      codeToHtml(code, {
        lang: langKeyToShiki(langKey),
        theme: capstoneDarkTheme,
      }),
    ).then((result) => {
      if (!cancelled) setHtml(result);
    }).catch(() => {});
    return () => { cancelled = true; };
  }, [code, langKey]);

  if (!html) {
    return (
      <pre className="max-h-96 overflow-auto bg-muted/30 px-4 py-3 font-mono text-[12.5px] leading-relaxed">
        <code>{code}</code>
      </pre>
    );
  }

  return (
    <div
      className="max-h-96 overflow-auto text-[12.5px] leading-relaxed [&>pre]:p-4 [&>pre]:h-full [&>pre]:m-0"
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
}

function ProfileSkeleton() {
  return (
    <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <Skeleton className="mb-6 h-8 w-24" />
      <div className="rounded-lg border bg-card p-6 sm:p-8">
        <div className="flex flex-col gap-6 sm:flex-row sm:items-start">
          <Skeleton className="size-20 shrink-0 rounded-full sm:size-24" />
          <div className="flex flex-col gap-2.5">
            <Skeleton className="h-8 w-48" />
            <Skeleton className="h-4 w-72 max-w-full" />
          </div>
          <div className="flex gap-3 sm:flex-col sm:gap-2">
            <Skeleton className="h-16 w-28 rounded-lg" />
            <Skeleton className="h-16 w-28 rounded-lg" />
          </div>
        </div>
      </div>
      <div className="mt-8 grid gap-3">
        <Skeleton className="h-24 rounded-lg" />
        <Skeleton className="h-24 rounded-lg" />
        <Skeleton className="h-24 rounded-lg" />
      </div>
    </main>
  );
}

function ProfileNotFound() {
  return (
    <main className="mx-auto max-w-5xl flex min-h-[60vh] flex-col items-center justify-center gap-4 px-4 py-12 text-center">
      <h1 className="text-2xl font-bold tracking-tight">User not found</h1>
      <p className="max-w-md text-sm text-muted-foreground">
        There is no profile for this link. The account may not exist or may be
        unavailable.
      </p>
      <Button asChild variant="outline" size="sm">
        <Link
          to="/"
          search={{
            q: undefined,
            difficulty: undefined,
            tags: undefined,
            page: 1,
          }}
        >
          <ArrowLeft className="size-3.5" />
          Back to problems
        </Link>
      </Button>
    </main>
  );
}
