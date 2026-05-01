import { createFileRoute, useRouter } from "@tanstack/react-router";
import { ChevronLeft, ChevronRight } from "lucide-react";
import {
  listProblems,
  listTags,
  PAGE_SIZE,
  type Problem,
  type Tag,
} from "@/lib/problems";
import { ProblemFilters } from "@/components/ProblemFilters";
import { HomeStats } from "@/components/HomeStats";
import { ProblemTable } from "@/components/ProblemTable";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";

// The full validated search shape — all keys present, optional ones may be undefined.
export type HomeSearch = {
  q: string | undefined;
  difficulty: string | undefined;
  tags: string[] | undefined;
  page: number;
  sort: string | undefined;
};

export const Route = createFileRoute("/")({
  validateSearch: (s: Record<string, unknown>): HomeSearch => {
    const rawTags = Array.isArray(s.tags)
      ? (s.tags as string[])
      : typeof s.tags === "string" && s.tags
        ? [s.tags]
        : [];
    return {
      q: typeof s.q === "string" && s.q ? s.q : undefined,
      difficulty:
        typeof s.difficulty === "string" && s.difficulty
          ? s.difficulty
          : undefined,
      tags: rawTags.length > 0 ? rawTags : undefined,
      page: Number(s.page) > 0 ? Number(s.page) : 1,
      sort: typeof s.sort === "string" && s.sort ? s.sort : undefined,
    };
  },
  loaderDeps: ({ search }) => search,
  loader: async ({ deps }) => {
    const [data, tags] = await Promise.all([
      listProblems({
        q: deps.q,
        difficulty: deps.difficulty,
        tags: deps.tags ?? [],
        page: deps.page,
        limit: PAGE_SIZE,
        visibility: "published",
      }),
      listTags(),
    ]);
    return { problems: data.problems, total: data.total, tags };
  },
  pendingComponent: HomePageSkeleton,
  component: HomePage,
});

function sortProblems(problems: Problem[], sortBy?: string): Problem[] {
  if (!sortBy || sortBy === "default") return problems;

  const copy = [...problems];
  if (sortBy === "easiest") {
    return copy.sort((a, b) => {
      const order = { easy: 0, medium: 1, hard: 2 };
      return order[a.difficulty] - order[b.difficulty];
    });
  }
  if (sortBy === "hardest") {
    return copy.sort((a, b) => {
      const order = { easy: 2, medium: 1, hard: 0 };
      return order[a.difficulty] - order[b.difficulty];
    });
  }
  if (sortBy === "acceptance-high") {
    return copy.sort(
      (a, b) => (b.acceptance_rate ?? 0) - (a.acceptance_rate ?? 0)
    );
  }
  if (sortBy === "acceptance-low") {
    return copy.sort(
      (a, b) => (a.acceptance_rate ?? 0) - (b.acceptance_rate ?? 0)
    );
  }
  if (sortBy === "a-z") {
    return copy.sort((a, b) => a.title.localeCompare(b.title));
  }
  return copy;
}

function HomePage() {
  const { problems, total, tags } = Route.useLoaderData() as {
    problems: Problem[];
    total: number;
    tags: Tag[];
  };
  const search = Route.useSearch();
  const router = useRouter();
  const { q, difficulty, tags: selectedTags = [], page, sort } = search;

  const sortedProblems = sortProblems(problems, sort);
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const startIndex = (page - 1) * PAGE_SIZE;

  function goToPage(next: number) {
    router.navigate({
      to: "/",
      search: {
        q,
        difficulty,
        tags: selectedTags.length > 0 ? selectedTags : undefined,
        page: next,
        sort,
      },
    });
  }

  return (
    <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <div className="mb-8 border-b border-border pb-6">
        <h1 className="text-3xl font-bold tracking-tight">Problems</h1>
        <p className="mt-1 max-w-lg text-sm text-muted-foreground">
          Solve algorithmic challenges in a sandboxed environment.
          {total > 0 && (
            <>
              {" · "}
              <span className="tabular-nums">{total} available</span>
            </>
          )}
        </p>
      </div>

      <HomeStats total={total} />

      <div className="mb-6">
        <ProblemFilters
          tags={tags}
          currentSearch={{ q, difficulty, tags: selectedTags, page, sort }}
        />
      </div>

      {problems.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-24 text-center">
          <p className="text-lg font-medium">No problems found</p>
          <p className="text-sm text-muted-foreground">
            Try adjusting your filters or search query.
          </p>
        </div>
      ) : (
        <>
          <ProblemTable
            problems={sortedProblems}
            startIndex={startIndex}
            sort={sort}
          />

          {totalPages > 1 && (
            <div className="mt-8 flex items-center justify-center gap-3">
              <Button
                variant="outline"
                size="sm"
                onClick={() => goToPage(page - 1)}
                disabled={page <= 1}
              >
                <ChevronLeft className="h-4 w-4" />
                Prev
              </Button>
              <Badge variant="outline" className="tabular-nums px-3 py-1 text-xs font-normal">
                {page} / {totalPages}
              </Badge>
              <Button
                variant="outline"
                size="sm"
                onClick={() => goToPage(page + 1)}
                disabled={page >= totalPages}
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
        </>
      )}
    </main>
  );
}

function HomePageSkeleton() {
  return (
    <main className="mx-auto max-w-5xl px-4 py-8 sm:px-6">
      <div className="mb-8 border-b border-border pb-6">
        <div className="h-8 w-32 animate-pulse rounded-md bg-muted" />
        <div className="mt-1 h-4 w-40 animate-pulse rounded bg-muted" />
      </div>
      <div className="mb-6 flex gap-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="h-20 w-24 animate-pulse rounded-lg bg-muted" />
        ))}
      </div>
      <div className="mb-6 flex flex-wrap gap-2">
        <div className="h-9 flex-1 animate-pulse rounded-md bg-muted sm:max-w-72" />
        <div className="h-9 w-36 animate-pulse rounded-md bg-muted" />
        <div className="h-9 w-40 animate-pulse rounded-md bg-muted" />
      </div>
      <div className="space-y-2 rounded-lg border border-border bg-card">
        {Array.from({ length: 6 }).map((_, i) => (
          <div
            key={i}
            className="flex items-center gap-4 border-b border-border/50 px-4 py-3 last:border-b-0"
          >
            <div className="h-4 w-4 animate-pulse rounded bg-muted" />
            <div className="h-4 w-8 animate-pulse rounded bg-muted" />
            <div className="h-4 flex-1 animate-pulse rounded bg-muted" />
            <div className="h-4 w-20 animate-pulse rounded bg-muted" />
          </div>
        ))}
      </div>
    </main>
  );
}
