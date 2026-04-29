import type { ReactNode } from "react";
import { Link } from "@tanstack/react-router";

interface ProtectedRouteProps {
  loading: boolean;
  allowed: boolean;
  loadingFallback?: ReactNode;
  children: ReactNode;
}

function NotAuthorized() {
  return (
    <main className="mx-auto flex min-h-[calc(100dvh-3.5rem)] max-w-lg flex-col items-center justify-center gap-4 px-4 text-center">
      <h1 className="text-2xl font-semibold tracking-tight">Page not found</h1>
      <p className="text-sm text-muted-foreground">
        This page does not exist or you do not have permission to view it.
      </p>
      <Link
        to="/"
        search={{ q: undefined, difficulty: undefined, tags: undefined, page: 1 }}
        className="text-sm font-medium text-primary underline-offset-4 hover:underline"
      >
        Back to problems
      </Link>
    </main>
  );
}

export function ProtectedRoute({
  loading,
  allowed,
  loadingFallback = null,
  children,
}: ProtectedRouteProps) {
  if (loading) return <>{loadingFallback}</>;
  if (!allowed) return <NotAuthorized />;
  return <>{children}</>;
}
