import { createFileRoute, Link, notFound } from "@tanstack/react-router";
import { ProblemWorkspace } from "@/components/ProblemWorkspace";
import { ApiError } from "@/lib/api";
import { getProblemBySlug } from "@/lib/problems";
import type { Problem } from "@/lib/problems";

export const Route = createFileRoute("/problems/$problemSlug")({
  loader: async ({ params }) => {
    try {
      const problem = await getProblemBySlug(params.problemSlug);
      return { problem };
    } catch (e) {
      if (e instanceof ApiError && e.status === 404) {
        throw notFound();
      }
      throw e;
    }
  },
  notFoundComponent: ProblemNotFound,
  component: RouteComponent,
});

function RouteComponent() {
  const { problem } = Route.useLoaderData() as { problem: Problem };

  return <ProblemWorkspace problem={problem} />;
}

function ProblemNotFound() {
  return (
    <main className="mx-auto flex min-h-[calc(100dvh-3.5rem)] max-w-lg flex-col items-center justify-center gap-4 px-4 text-center">
      <h1 className="text-2xl font-semibold tracking-tight">Problem not found</h1>
      <p className="text-sm text-muted-foreground">
        This problem does not exist or is not available.
      </p>
      <Link
        to="/"
        search={{
          q: undefined,
          difficulty: undefined,
          tags: undefined,
          page: 1,
        }}
        className="text-sm font-medium text-primary underline-offset-4 hover:underline"
      >
        Back to problems
      </Link>
    </main>
  );
}
