import { createFileRoute, useRouter } from "@tanstack/react-router";
import { useCallback, useEffect, useState } from "react";
import {
  AlertTriangle,
  ChevronDown,
  ChevronLeft,
  ChevronRight,
  ChevronUp,
  Code2,
  Search,
  Shield,
  ShieldAlert,
  User as UserIcon,
} from "lucide-react";
import { toast } from "sonner";
import { formatDistanceToNow } from "date-fns";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useAuth } from "@/lib/auth";
import { ProtectedRoute } from "@/components/ProtectedRoute";
import {
  canManageProblems,
  getMyPermissions,
  type Permission,
} from "@/lib/permissions";
import {
  listAdminUsers,
  getUserSecurityEvents,
  type AdminUser,
  type AdminUserSortKey,
  type SecurityEvent,
} from "@/lib/admin";

const PAGE_SIZE = 20;

export type AdminUsersSearch = {
  page: number;
  q: string | undefined;
  sort: AdminUserSortKey | undefined;
};

export const Route = createFileRoute("/admin/users/")({
  validateSearch: (s: Record<string, unknown>): AdminUsersSearch => ({
    page: Number(s.page) > 0 ? Number(s.page) : 1,
    q: typeof s.q === "string" && s.q ? s.q : undefined,
    sort:
      typeof s.sort === "string" &&
      ["violation_count", "submissions", "handle", "created_at"].includes(
        s.sort,
      )
        ? (s.sort as AdminUserSortKey)
        : undefined,
  }),
  component: AdminUsersPage,
});

function severityColor(severity: string) {
  switch (severity) {
    case "block":
      return "destructive";
    case "high":
      return "destructive";
    case "warn":
      return "secondary";
    default:
      return "outline";
  }
}

function UserDetailSheet({
  user,
  open,
  onOpenChange,
}: {
  user: AdminUser | null;
  open: boolean;
  onOpenChange: (v: boolean) => void;
}) {
  const [events, setEvents] = useState<SecurityEvent[]>([]);
  const [eventsTotal, setEventsTotal] = useState(0);
  const [eventsLoading, setEventsLoading] = useState(false);
  const [expandedCode, setExpandedCode] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!user || !open) return;
    let cancelled = false;
    setEventsLoading(true);
    setEvents([]);
    getUserSecurityEvents(user.id, { limit: 50 })
      .then((res) => {
        if (!cancelled) {
          setEvents(res.events);
          setEventsTotal(res.total);
        }
      })
      .catch(() => {
        if (!cancelled) toast.error("Failed to load security events");
      })
      .finally(() => {
        if (!cancelled) setEventsLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [user, open]);

  if (!user) return null;

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full sm:max-w-lg flex flex-col gap-0 p-0 overflow-hidden">
        <SheetHeader className="px-6 py-5 border-b">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-muted">
              <UserIcon className="h-5 w-5 text-muted-foreground" />
            </div>
            <div>
              <SheetTitle className="text-left">@{user.handle}</SheetTitle>
              <SheetDescription className="text-left">
                {user.email ?? "no email"}
              </SheetDescription>
            </div>
          </div>
        </SheetHeader>

        <ScrollArea className="flex-1 min-h-0 px-6 py-4">
          {/* Stats */}
          <div className="grid grid-cols-3 gap-3 mb-6">
            <div className="rounded-lg border bg-card p-3 text-center">
              <p className="text-2xl font-bold text-destructive">
                {user.violation_count}
              </p>
              <p className="text-xs text-muted-foreground mt-1">Violations</p>
            </div>
            <div className="rounded-lg border bg-card p-3 text-center">
              <p className="text-2xl font-bold">{user.submission_count}</p>
              <p className="text-xs text-muted-foreground mt-1">Submissions</p>
            </div>
            <div className="rounded-lg border bg-card p-3 text-center">
              <Badge
                variant={user.status === "ACTIVE" ? "outline" : "destructive"}
                className="text-xs"
              >
                {user.status}
              </Badge>
              <p className="text-xs text-muted-foreground mt-1">Status</p>
            </div>
          </div>

          <p className="text-xs text-muted-foreground mb-4">
            Member since{" "}
            {formatDistanceToNow(new Date(user.created_at), {
              addSuffix: true,
            })}
          </p>

          {/* Security Events */}
          <div className="flex items-center justify-between mb-3">
            <h3 className="text-sm font-semibold flex items-center gap-1.5">
              <ShieldAlert className="h-4 w-4 text-destructive" />
              Security Events
            </h3>
            {eventsTotal > 0 && (
              <Badge variant="destructive" className="text-xs">
                {eventsTotal}
              </Badge>
            )}
          </div>

          {eventsLoading ? (
            <div className="space-y-2">
              {[...Array(3)].map((_, i) => (
                <Skeleton key={i} className="h-16 w-full rounded-lg" />
              ))}
            </div>
          ) : events.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-8 text-center">
              <Shield className="h-8 w-8 text-muted-foreground/40" />
              <p className="text-sm text-muted-foreground">
                No security events
              </p>
            </div>
          ) : (
            <div className="space-y-2">
              {events.map((ev) => {
                const codeOpen = expandedCode.has(ev.id);
                return (
                  <div
                    key={ev.id}
                    className="rounded-lg border bg-card p-3 space-y-1.5"
                  >
                    <div className="flex items-center justify-between">
                      <Badge variant={severityColor(ev.severity) as "destructive" | "secondary" | "outline"} className="text-xs capitalize">
                        {ev.severity}
                      </Badge>
                      <span className="text-xs text-muted-foreground">
                        {formatDistanceToNow(new Date(ev.created_at), {
                          addSuffix: true,
                        })}
                      </span>
                    </div>
                    {ev.detail_json?.reason && (
                      <p className="text-xs text-foreground/80 leading-relaxed">
                        {String(ev.detail_json.reason)}
                      </p>
                    )}
                    {ev.detail_json?.language_key && (
                      <p className="text-xs text-muted-foreground">
                        Language: {String(ev.detail_json.language_key)}
                      </p>
                    )}
                    {ev.source_text && (
                      <div>
                        <button
                          onClick={() => {
                            setExpandedCode((prev) => {
                              const next = new Set(prev);
                              if (next.has(ev.id)) next.delete(ev.id);
                              else next.add(ev.id);
                              return next;
                            });
                          }}
                          className="flex items-center gap-1 text-xs text-muted-foreground hover:text-foreground transition-colors"
                        >
                          <Code2 className="h-3.5 w-3.5" />
                          {codeOpen ? "Hide code" : "View submitted code"}
                          {codeOpen ? (
                            <ChevronUp className="h-3 w-3" />
                          ) : (
                            <ChevronDown className="h-3 w-3" />
                          )}
                        </button>
                        {codeOpen && (
                          <pre className="mt-2 overflow-x-auto rounded-md bg-muted p-3 text-xs leading-relaxed font-mono whitespace-pre-wrap break-all max-h-64">
                            {ev.source_text}
                          </pre>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
              {eventsTotal > events.length && (
                <p className="text-xs text-muted-foreground text-center pt-2">
                  Showing {events.length} of {eventsTotal}
                </p>
              )}
            </div>
          )}
        </ScrollArea>
      </SheetContent>
    </Sheet>
  );
}

function AdminUsersPage() {
  const { user, loading: authLoading, accessToken } = useAuth();
  const search = Route.useSearch();
  const router = useRouter();

  const [perms, setPerms] = useState<Permission[] | null>(null);
  const [permsErr, setPermsErr] = useState(false);

  const [listLoading, setListLoading] = useState(false);
  const [users, setUsers] = useState<AdminUser[]>([]);
  const [total, setTotal] = useState(0);

  const [selectedUser, setSelectedUser] = useState<AdminUser | null>(null);
  const [sheetOpen, setSheetOpen] = useState(false);

  const [inputQ, setInputQ] = useState(search.q ?? "");

  const allowed = perms !== null && canManageProblems(perms);
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  useEffect(() => {
    if (authLoading || !user || !accessToken) {
      setPerms(null);
      setPermsErr(false);
      return;
    }
    let cancelled = false;
    setPermsErr(false);
    getMyPermissions()
      .then((p) => { if (!cancelled) setPerms(p); })
      .catch(() => { if (!cancelled) setPermsErr(true); });
    return () => { cancelled = true; };
  }, [user, authLoading, accessToken]);

  const fetchUsers = useCallback(() => {
    if (!allowed) return;
    setListLoading(true);
    listAdminUsers({
      q: search.q,
      sort: search.sort,
      page: search.page,
      limit: PAGE_SIZE,
    })
      .then((res) => {
        setUsers(res.users);
        setTotal(res.total);
      })
      .catch(() => toast.error("Failed to load users"))
      .finally(() => setListLoading(false));
  }, [allowed, search.q, search.sort, search.page]);

  useEffect(() => { fetchUsers(); }, [fetchUsers]);

  function navigate(updates: Partial<AdminUsersSearch>) {
    router.navigate({
      to: "/admin/users",
      search: { ...search, page: 1, ...updates },
    });
  }

  function handleSearchSubmit(e: React.FormEvent) {
    e.preventDefault();
    navigate({ q: inputQ || undefined, page: 1 });
  }

  function openUser(u: AdminUser) {
    setSelectedUser(u);
    setSheetOpen(true);
  }

  return (
    <ProtectedRoute
      loading={authLoading || perms === null}
      allowed={!permsErr && allowed}
    >
      <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
        <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <h1 className="text-xl font-semibold tracking-tight">
              User Management
            </h1>
            <p className="text-sm text-muted-foreground">
              {total} user{total !== 1 ? "s" : ""}
            </p>
          </div>

          <div className="flex flex-col gap-2 sm:flex-row">
            <form onSubmit={handleSearchSubmit} className="flex gap-2">
              <div className="relative">
                <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                <Input
                  className="pl-8 w-56"
                  placeholder="Search handle or email…"
                  value={inputQ}
                  onChange={(e) => setInputQ(e.target.value)}
                />
              </div>
              <Button type="submit" variant="outline" size="icon">
                <Search className="h-4 w-4" />
              </Button>
            </form>

            <Select
              value={search.sort ?? "violation_count"}
              onValueChange={(v) =>
                navigate({ sort: v as AdminUserSortKey })
              }
            >
              <SelectTrigger className="w-44">
                <SelectValue placeholder="Sort by" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="violation_count">Most violations</SelectItem>
                <SelectItem value="submissions">Most submissions</SelectItem>
                <SelectItem value="handle">Handle A–Z</SelectItem>
                <SelectItem value="created_at">Newest</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <div className="rounded-lg border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>User</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>Status</TableHead>
                <TableHead className="text-right">Violations</TableHead>
                <TableHead className="text-right">Submissions</TableHead>
                <TableHead>Joined</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {listLoading ? (
                Array.from({ length: 8 }).map((_, i) => (
                  <TableRow key={i}>
                    {Array.from({ length: 6 }).map((_, j) => (
                      <TableCell key={j}>
                        <Skeleton className="h-4 w-full" />
                      </TableCell>
                    ))}
                  </TableRow>
                ))
              ) : users.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={6}
                    className="py-12 text-center text-sm text-muted-foreground"
                  >
                    No users found
                  </TableCell>
                </TableRow>
              ) : (
                users.map((u) => (
                  <TableRow
                    key={u.id}
                    className="cursor-pointer"
                    onClick={() => openUser(u)}
                  >
                    <TableCell>
                      <div className="flex items-center gap-2">
                        <div className="flex h-7 w-7 items-center justify-center rounded-full bg-muted shrink-0">
                          <UserIcon className="h-3.5 w-3.5 text-muted-foreground" />
                        </div>
                        <span className="font-medium">@{u.handle}</span>
                        {u.display_name && (
                          <span className="text-xs text-muted-foreground">
                            {u.display_name}
                          </span>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {u.email ?? "—"}
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={
                          u.status === "ACTIVE" ? "outline" : "destructive"
                        }
                        className="text-xs"
                      >
                        {u.status}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-right">
                      {u.violation_count > 0 ? (
                        <span className="inline-flex items-center gap-1 font-semibold text-destructive">
                          <AlertTriangle className="h-3.5 w-3.5" />
                          {u.violation_count}
                        </span>
                      ) : (
                        <span className="text-muted-foreground">0</span>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      {u.submission_count}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {formatDistanceToNow(new Date(u.created_at), {
                        addSuffix: true,
                      })}
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>

        {/* Pagination */}
        {totalPages > 1 && (
          <div className="mt-4 flex items-center justify-between">
            <p className="text-sm text-muted-foreground">
              Page {search.page} of {totalPages}
            </p>
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                disabled={search.page <= 1}
                onClick={() => navigate({ page: search.page - 1 })}
              >
                <ChevronLeft className="h-4 w-4" />
                Prev
              </Button>
              <Button
                variant="outline"
                size="sm"
                disabled={search.page >= totalPages}
                onClick={() => navigate({ page: search.page + 1 })}
              >
                Next
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          </div>
        )}
      </main>

      <UserDetailSheet
        user={selectedUser}
        open={sheetOpen}
        onOpenChange={setSheetOpen}
      />
    </ProtectedRoute>
  );
}
