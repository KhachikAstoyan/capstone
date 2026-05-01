import { createFileRoute, Link, useRouter } from "@tanstack/react-router";
import { useCallback, useEffect, useState, useSyncExternalStore } from "react";
import { CheckCircle2, ChevronLeft, ChevronRight, Plus, Search, Trash2, X, XCircle } from "lucide-react";
import MDEditor from "@uiw/react-md-editor";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { useAuth } from "@/lib/auth";
import { ProtectedRoute } from "@/components/ProtectedRoute";
import { ApiError } from "@/lib/api";
import {
  canManageProblems,
  getMyPermissions,
  type Permission,
} from "@/lib/permissions";
import {
  ADMIN_PAGE_SIZE,
  ALL_PARAM_TYPES,
  createProblem,
  createTag,
  deleteProblem,
  listProblems,
  listTags,
  updateProblemTags,
  type CreateProblemRequest,
  type Difficulty,
  type Parameter,
  type ParamType,
  type Problem,
  type Tag,
  type Visibility,
} from "@/lib/problems";
import {
  createLanguage,
  listLanguages,
  updateProblemLanguages,
  type Language,
} from "@/lib/languages";

export type AdminProblemsSearch = {
  page: number;
  q: string | undefined;
  visibility: string | undefined;
};

function subscribeHtmlClass(callback: () => void) {
  const el = document.documentElement;
  const mo = new MutationObserver(callback);
  mo.observe(el, { attributes: true, attributeFilter: ["class"] });
  return () => mo.disconnect();
}

function getHtmlIsDark(): boolean {
  return document.documentElement.classList.contains("dark");
}

function useDocumentDarkMode(): "light" | "dark" {
  return useSyncExternalStore(
    subscribeHtmlClass,
    () => (getHtmlIsDark() ? "dark" : "light"),
    () => "light",
  );
}

export const Route = createFileRoute("/admin/problems/")({
  validateSearch: (s: Record<string, unknown>): AdminProblemsSearch => ({
    page: Number(s.page) > 0 ? Number(s.page) : 1,
    q: typeof s.q === "string" && s.q ? s.q : undefined,
    visibility:
      typeof s.visibility === "string" && s.visibility
        ? s.visibility
        : undefined,
  }),
  component: AdminProblemsPage,
});

function AdminProblemsPage() {
  const { user, loading: authLoading, accessToken } = useAuth();
  const search = Route.useSearch();
  const router = useRouter();

  const [perms, setPerms] = useState<Permission[] | null>(null);
  const [permsErr, setPermsErr] = useState(false);

  const [listLoading, setListLoading] = useState(false);
  const [listErr, setListErr] = useState<string | null>(null);
  const [problems, setProblems] = useState<Problem[]>([]);
  const [total, setTotal] = useState(0);

  const [refreshKey, setRefreshKey] = useState(0);
  const [problemToDelete, setProblemToDelete] = useState<Problem | null>(null);
  const [deleting, setDeleting] = useState(false);

  const allowed = perms !== null && canManageProblems(perms);

  useEffect(() => {
    if (authLoading || !user || !accessToken) {
      setPerms(null);
      setPermsErr(false);
      return;
    }
    let cancelled = false;
    setPermsErr(false);
    getMyPermissions()
      .then((p) => {
        if (!cancelled) setPerms(p);
      })
      .catch(() => {
        if (!cancelled) {
          setPermsErr(true);
          setPerms([]);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [authLoading, user, accessToken]);

  useEffect(() => {
    if (!allowed) return;
    let cancelled = false;
    setListLoading(true);
    setListErr(null);
    listProblems({
      page: search.page,
      limit: ADMIN_PAGE_SIZE,
      q: search.q,
      visibility: search.visibility,
    })
      .then((res) => {
        if (!cancelled) {
          setProblems(res.problems);
          setTotal(res.total);
        }
      })
      .catch((e: unknown) => {
        if (!cancelled) {
          setListErr(e instanceof ApiError ? e.message : "Failed to load");
        }
      })
      .finally(() => {
        if (!cancelled) setListLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [allowed, search.page, search.q, search.visibility, refreshKey]);

  const totalPages = Math.max(1, Math.ceil(total / ADMIN_PAGE_SIZE));

  const goToPage = useCallback(
    (next: number) => {
      router.navigate({
        to: "/admin/problems",
        search: {
          page: next,
          q: search.q,
          visibility: search.visibility,
        },
      });
    },
    [router, search.q, search.visibility],
  );

  const refetchList = useCallback(() => {
    setRefreshKey((k) => k + 1);
  }, []);

  async function handleDeleteProblem() {
    if (!problemToDelete) return;

    setDeleting(true);
    try {
      await deleteProblem(problemToDelete.id);
      toast.success("Problem deleted.");
      setProblemToDelete(null);
      refetchList();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to delete problem.");
    } finally {
      setDeleting(false);
    }
  }

  const isLoading = authLoading || (!!user && perms === null && !permsErr);
  const isAllowed = !!user && !permsErr && allowed;

  return (
    <ProtectedRoute
      loading={isLoading}
      allowed={isAllowed}
      loadingFallback={<AdminProblemsSkeleton />}
    >
      <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            Admin — Problems
          </h1>
          <p className="text-sm text-muted-foreground">
            {total} problem{total !== 1 ? "s" : ""} total
          </p>
        </div>
        <CreateProblemButton onCreated={refetchList} />
      </div>

      <LanguageManager />

      <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-end">
        <div className="grid flex-1 gap-2">
          <Label htmlFor="admin-q">Search</Label>
          <form
            className="flex gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              const fd = new FormData(e.currentTarget);
              const q = (fd.get("q") as string)?.trim() || undefined;
              router.navigate({
                to: "/admin/problems",
                search: { page: 1, q, visibility: search.visibility },
              });
            }}
          >
            <Input
              id="admin-q"
              name="q"
              defaultValue={search.q ?? ""}
              placeholder="Title or slug…"
              className="max-w-md"
            />
            <Button type="submit" variant="secondary">
              Search
            </Button>
          </form>
        </div>
        <div className="grid gap-2 sm:w-48">
          <Label>Visibility</Label>
          <Select
            value={search.visibility ?? "all"}
            onValueChange={(v) => {
              router.navigate({
                to: "/admin/problems",
                search: {
                  page: 1,
                  q: search.q,
                  visibility: v === "all" ? undefined : v,
                },
              });
            }}
          >
            <SelectTrigger className="w-full min-w-0">
              <SelectValue placeholder="All" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All</SelectItem>
              <SelectItem value="draft">Draft</SelectItem>
              <SelectItem value="published">Published</SelectItem>
              <SelectItem value="archived">Archived</SelectItem>
            </SelectContent>
          </Select>
        </div>
      </div>

      {listErr && <p className="mb-4 text-sm text-destructive">{listErr}</p>}

      <div className="rounded-lg border border-border bg-card">
        {listLoading ? (
          <div className="p-8">
            <Skeleton className="h-48 w-full" />
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Title</TableHead>
                <TableHead>Slug</TableHead>
                <TableHead>Difficulty</TableHead>
                <TableHead>Visibility</TableHead>
                <TableHead className="text-right">Limits</TableHead>
                <TableHead>Updated</TableHead>
                <TableHead />
                <TableHead className="w-12 text-right">Delete</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {problems.length === 0 ? (
                <TableRow>
                  <TableCell
                    colSpan={8}
                    className="text-center text-muted-foreground"
                  >
                    No problems match the current filters.
                  </TableCell>
                </TableRow>
              ) : (
                problems.map((p) => (
                  <TableRow key={p.id}>
                    <TableCell className="max-w-[200px] font-medium">
                      <Link
                        to="/problems/$problemSlug"
                        params={{ problemSlug: p.slug }}
                        className="text-primary underline-offset-4 hover:underline"
                      >
                        {p.title}
                      </Link>
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {p.slug}
                    </TableCell>
                    <TableCell>{p.difficulty}</TableCell>
                    <TableCell>{p.visibility}</TableCell>
                    <TableCell className="text-right text-muted-foreground text-xs">
                      {p.time_limit_ms} ms / {p.memory_limit_mb} MB
                    </TableCell>
                    <TableCell className="text-muted-foreground text-xs whitespace-nowrap">
                      {new Date(p.updated_at).toLocaleString()}
                    </TableCell>
                    <TableCell>
                      <Link
                        to="/admin/problems/$problemId"
                        params={{ problemId: p.id }}
                        className="text-sm text-primary underline-offset-4 hover:underline"
                      >
                        Edit
                      </Link>
                    </TableCell>
                    <TableCell className="text-right">
                      <Button
                        type="button"
                        variant="destructive"
                        size="icon-sm"
                        aria-label={`Delete ${p.title}`}
                        onClick={() => setProblemToDelete(p)}
                      >
                        <Trash2 className="size-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        )}
      </div>

      {totalPages > 1 && (
        <div className="mt-6 flex items-center justify-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => goToPage(search.page - 1)}
            disabled={search.page <= 1 || listLoading}
          >
            <ChevronLeft className="h-4 w-4" />
            Prev
          </Button>
          <span className="text-sm text-muted-foreground">
            Page {search.page} of {totalPages}
          </span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => goToPage(search.page + 1)}
            disabled={search.page >= totalPages || listLoading}
          >
            Next
            <ChevronRight className="h-4 w-4" />
          </Button>
        </div>
      )}

      <Dialog
        open={problemToDelete !== null}
        onOpenChange={(open) => {
          if (!open && !deleting) setProblemToDelete(null);
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>Delete problem?</DialogTitle>
            <DialogDescription>
              This will permanently delete{" "}
              <span className="font-medium text-foreground">
                {problemToDelete?.title}
              </span>
              . This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={deleting}
              onClick={() => setProblemToDelete(null)}
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="destructive"
              disabled={deleting}
              onClick={handleDeleteProblem}
            >
              <Trash2 className="size-4" />
              {deleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </main>
    </ProtectedRoute>
  );
}

function LanguageManager() {
  const [languages, setLanguages] = useState<Language[]>([]);
  const [search, setSearch] = useState("");
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [key, setKey] = useState("");
  const [displayName, setDisplayName] = useState("");

  const loadLanguages = useCallback((q: string) => {
    setLoading(true);
    setError(null);
    listLanguages(q)
      .then(setLanguages)
      .catch((err: unknown) => {
        setError(err instanceof ApiError ? err.message : "Failed to load languages.");
      })
      .finally(() => setLoading(false));
  }, []);

  useEffect(() => {
    loadLanguages("");
  }, [loadLanguages]);

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault();
    loadLanguages(search);
  }

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    const languageKey = key.trim().toLowerCase();
    const name = displayName.trim();
    if (!languageKey || !name) {
      toast.error("Language key and display name are required.");
      return;
    }

    setSaving(true);
    try {
      const language = await createLanguage({
        key: languageKey,
        display_name: name,
        is_enabled: true,
      });
      toast.success("Language saved.");
      setKey("");
      setDisplayName("");
      setLanguages((prev) => {
        const next = prev.filter((l) => l.id !== language.id && l.key !== language.key);
        return [...next, language].sort((a, b) => a.key.localeCompare(b.key));
      });
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to save language.");
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="mb-6 rounded-lg border border-border bg-card p-4">
      <div className="mb-4 flex flex-col gap-3 md:flex-row md:items-end md:justify-between">
        <div>
          <h2 className="text-base font-semibold tracking-tight">Languages</h2>
          <p className="text-sm text-muted-foreground">
            Manage enabled runtime keys used by submissions and workers.
          </p>
        </div>
        <form className="flex w-full gap-2 md:max-w-sm" onSubmit={handleSearch}>
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search languages..."
            className="min-w-0"
          />
          <Button type="submit" variant="secondary" disabled={loading}>
            <Search className="mr-2 h-4 w-4" />
            Search
          </Button>
        </form>
      </div>

      <form className="mb-4 grid gap-3 md:grid-cols-[minmax(120px,180px)_1fr_auto]" onSubmit={handleCreate}>
        <div className="grid gap-2">
          <Label htmlFor="language-key">Key</Label>
          <Input
            id="language-key"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="javascript"
            autoComplete="off"
          />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="language-display-name">Display name</Label>
          <Input
            id="language-display-name"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            placeholder="JavaScript"
            autoComplete="off"
          />
        </div>
        <div className="flex items-end">
          <Button type="submit" disabled={saving} className="w-full md:w-auto">
            <Plus className="mr-2 h-4 w-4" />
            {saving ? "Saving..." : "Add language"}
          </Button>
        </div>
      </form>

      {error && <p className="mb-3 text-sm text-destructive">{error}</p>}

      <div className="overflow-hidden rounded-md border border-border">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Key</TableHead>
              <TableHead>Name</TableHead>
              <TableHead>Status</TableHead>
              <TableHead>Updated</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {loading ? (
              <TableRow>
                <TableCell colSpan={4}>
                  <Skeleton className="h-10 w-full" />
                </TableCell>
              </TableRow>
            ) : languages.length === 0 ? (
              <TableRow>
                <TableCell colSpan={4} className="text-center text-muted-foreground">
                  No languages match the current search.
                </TableCell>
              </TableRow>
            ) : (
              languages.map((language) => (
                <TableRow key={language.id}>
                  <TableCell className="font-mono text-xs">{language.key}</TableCell>
                  <TableCell>{language.display_name}</TableCell>
                  <TableCell>
                    {language.is_enabled ? (
                      <Badge className="gap-1 bg-emerald-500/15 text-emerald-700 dark:text-emerald-300">
                        <CheckCircle2 className="h-3 w-3" />
                        Enabled
                      </Badge>
                    ) : (
                      <Badge variant="secondary" className="gap-1">
                        <XCircle className="h-3 w-3" />
                        Disabled
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
                    {new Date(language.updated_at).toLocaleString()}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>
    </section>
  );
}

function AdminProblemsSkeleton() {
  return (
    <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
      <Skeleton className="mb-2 h-8 w-48" />
      <Skeleton className="mb-6 h-4 w-32" />
      <Skeleton className="h-64 w-full" />
    </main>
  );
}

function CreateProblemButton({ onCreated }: { onCreated: () => void }) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <Button onClick={() => setOpen(true)}>
        <Plus className="mr-2 h-4 w-4" />
        New problem
      </Button>
      <CreateProblemDialog
        open={open}
        onOpenChange={setOpen}
        onCreated={() => {
          setOpen(false);
          onCreated();
        }}
      />
    </>
  );
}

function CreateProblemDialog({
  open,
  onOpenChange,
  onCreated,
}: {
  open: boolean;
  onOpenChange: (o: boolean) => void;
  onCreated: () => void;
}) {
  const colorMode = useDocumentDarkMode();
  const router = useRouter();
  const [title, setTitle] = useState("");
  const [summary, setSummary] = useState("");
  const [statement, setStatement] = useState("");
  const [timeLimit, setTimeLimit] = useState("1000");
  const [memoryLimit, setMemoryLimit] = useState("256");
  const [testsRef, setTestsRef] = useState("");
  const [visibility, setVisibility] = useState<Visibility>("draft");
  const [difficulty, setDifficulty] = useState<Difficulty>("easy");
  const [submitting, setSubmitting] = useState(false);
  const [editorMounted, setEditorMounted] = useState(false);

  const [tagCatalog, setTagCatalog] = useState<Tag[]>([]);
  const [selectedTags, setSelectedTags] = useState<Tag[]>([]);
  const [tagInput, setTagInput] = useState("");
  const [tagSuggestOpen, setTagSuggestOpen] = useState(false);
  const [languageCatalog, setLanguageCatalog] = useState<Language[]>([]);
  const [selectedLanguageIds, setSelectedLanguageIds] = useState<string[]>([]);

  const [funcName, setFuncName] = useState("");
  const [returnType, setReturnType] = useState<ParamType>("int");
  const [parameters, setParameters] = useState<Parameter[]>([]);

  useEffect(() => {
    if (open) setEditorMounted(true);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    listTags()
      .then(setTagCatalog)
      .catch(() => {
        toast.error("Failed to load tags.");
      });
    listLanguages()
      .then((languages) => {
        setLanguageCatalog(languages);
        setSelectedLanguageIds(languages.filter((l) => l.is_enabled).map((l) => l.id));
      })
      .catch(() => {
        toast.error("Failed to load languages.");
      });
  }, [open]);

  useEffect(() => {
    if (!open) {
      setTitle("");
      setSummary("");
      setStatement("");
      setTimeLimit("1000");
      setMemoryLimit("256");
      setTestsRef("");
      setVisibility("draft");
      setDifficulty("easy");
      setSelectedTags([]);
      setTagInput("");
      setTagSuggestOpen(false);
      setLanguageCatalog([]);
      setSelectedLanguageIds([]);
      setFuncName("");
      setReturnType("int");
      setParameters([]);
    }
  }, [open]);

  function addParameter() {
    setParameters((prev) => [...prev, { name: "", type: "int" as ParamType }]);
  }

  function updateParameter(index: number, field: keyof Parameter, value: string) {
    setParameters((prev) => prev.map((p, i) => i === index ? { ...p, [field]: value } : p));
  }

  function removeParameter(index: number) {
    setParameters((prev) => prev.filter((_, i) => i !== index));
  }

  const selectedTagIds = new Set(selectedTags.map((t) => t.id));

  const tagSuggestions = (() => {
    const q = tagInput.trim().toLowerCase();
    const pool = tagCatalog.filter((t) => !selectedTagIds.has(t.id));
    if (!q) return pool.slice(0, 8);
    return pool
      .filter((t) => t.name.toLowerCase().includes(q))
      .sort((a, b) => a.name.localeCompare(b.name))
      .slice(0, 8);
  })();

  function mergeIntoCatalog(tag: Tag) {
    setTagCatalog((prev) =>
      prev.some((t) => t.id === tag.id) ? prev : [...prev, tag],
    );
  }

  function addSelectedTag(tag: Tag) {
    setSelectedTags((prev) =>
      prev.some((t) => t.id === tag.id) ? prev : [...prev, tag],
    );
    setTagInput("");
  }

  async function commitTagInput() {
    const q = tagInput.trim();
    if (!q) return;

    const pool = tagCatalog.filter((t) => !selectedTagIds.has(t.id));
    const qLower = q.toLowerCase();
    const matches = pool
      .filter((t) => t.name.toLowerCase().includes(qLower))
      .sort((a, b) => a.name.localeCompare(b.name));

    if (matches.length > 0) {
      const exact = matches.find((t) => t.name.toLowerCase() === qLower);
      addSelectedTag(exact ?? matches[0]!);
      return;
    }

    try {
      const created = await createTag(q);
      mergeIntoCatalog(created);
      addSelectedTag(created);
    } catch (err) {
      toast.error(
        err instanceof ApiError ? err.message : "Failed to create tag.",
      );
    }
  }

  async function onTagInputKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key !== "Enter") return;
    e.preventDefault();
    await commitTagInput();
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const time = Number(timeLimit);
    const mem = Number(memoryLimit);
    if (!title.trim() || !statement.trim()) {
      toast.error("Title and statement are required.");
      return;
    }
    if (selectedLanguageIds.length === 0) {
      toast.error("Select at least one supported language.");
      return;
    }
    if (summary.length > 500) {
      toast.error("List summary must be at most 500 characters.");
      return;
    }
    if (
      !Number.isFinite(time) ||
      time <= 0 ||
      !Number.isFinite(mem) ||
      mem <= 0
    ) {
      toast.error("Time and memory limits must be positive numbers.");
      return;
    }

    if (!funcName.trim()) {
      toast.error("Function name is required.");
      return;
    }
    for (const p of parameters) {
      if (!p.name.trim()) {
        toast.error("All parameter names are required.");
        return;
      }
    }
    const function_spec: CreateProblemRequest["function_spec"] = {
      function_name: funcName.trim(),
      parameters,
      return_type: returnType,
    };

    const body: CreateProblemRequest = {
      title: title.trim(),
      ...(summary.trim() ? { summary: summary.trim() } : {}),
      statement_markdown: statement,
      time_limit_ms: Math.floor(time),
      memory_limit_mb: Math.floor(mem),
      ...(testsRef.trim() ? { tests_ref: testsRef.trim() } : {}),
      visibility,
      difficulty,
      function_spec,
    };

    setSubmitting(true);
    try {
      const problem = await createProblem(body);
      try {
        await updateProblemLanguages(problem.id, selectedLanguageIds);
      } catch {
        toast.error("Problem created, but supported languages could not be saved.");
      }
      if (selectedTags.length > 0) {
        try {
          await updateProblemTags(
            problem.id,
            selectedTags.map((t) => t.id),
          );
        } catch {
          toast.error("Problem created, but tags could not be saved.");
        }
      }
      toast.success("Problem created.");
      onCreated();
      router.navigate({
        to: "/admin/problems/$problemId",
        params: { problemId: problem.id },
      });
    } catch (err) {
      const msg =
        err instanceof ApiError ? err.message : "Failed to create problem";
      toast.error(msg);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        className="max-h-[min(90vh,900px)] overflow-y-auto sm:max-w-2xl"
        showCloseButton
      >
        <DialogHeader>
          <DialogTitle>Create problem</DialogTitle>
          <DialogDescription>
            Slug is generated from the title. Statement supports Markdown and
            HTML. The list summary is plain text for the problems grid.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="grid gap-4">
          <div className="grid gap-2">
            <Label htmlFor="cp-title">Title</Label>
            <Input
              id="cp-title"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="Two Sum"
              required
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="cp-summary">List summary (plain text, optional)</Label>
            <Textarea
              id="cp-summary"
              value={summary}
              onChange={(e) => setSummary(e.target.value)}
              placeholder="One or two sentences shown on the problems list."
              rows={3}
              maxLength={500}
              className="resize-y min-h-18"
            />
            <p className="text-xs text-muted-foreground">
              {summary.length}/500 — no markdown; line breaks are preserved on cards.
            </p>
          </div>

          <div className="grid gap-2">
            <Label>Statement (Markdown / HTML)</Label>
            {editorMounted ? (
              <div data-color-mode={colorMode}>
                <MDEditor
                  value={statement}
                  onChange={(v) => setStatement(v ?? "")}
                  height={280}
                  visibleDragbar={false}
                />
              </div>
            ) : (
              <Skeleton className="h-[280px] w-full" />
            )}
          </div>

          <div className="grid gap-2">
            <Label>Supported languages</Label>
            <p className="text-xs text-muted-foreground">
              These are the languages users can run for this problem.
            </p>
            <div className="grid gap-2 rounded-md border border-border p-3 sm:grid-cols-3">
              {languageCatalog.filter((language) => language.is_enabled).length === 0 ? (
                <p className="text-sm text-muted-foreground sm:col-span-3">
                  Add enabled languages from the Languages panel first.
                </p>
              ) : (
                languageCatalog
                  .filter((language) => language.is_enabled)
                  .map((language) => (
                    <label
                      key={language.id}
                      className="flex items-center gap-2 text-sm"
                    >
                      <input
                        type="checkbox"
                        checked={selectedLanguageIds.includes(language.id)}
                        onChange={(e) => {
                          setSelectedLanguageIds((prev) =>
                            e.target.checked
                              ? [...prev, language.id]
                              : prev.filter((id) => id !== language.id),
                          );
                        }}
                        className="h-4 w-4 rounded border-border"
                      />
                      <span>{language.display_name}</span>
                      <span className="font-mono text-xs text-muted-foreground">
                        {language.key}
                      </span>
                    </label>
                  ))
              )}
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="cp-time">Time limit (ms)</Label>
              <Input
                id="cp-time"
                type="number"
                min={1}
                value={timeLimit}
                onChange={(e) => setTimeLimit(e.target.value)}
                required
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="cp-mem">Memory limit (MB)</Label>
              <Input
                id="cp-mem"
                type="number"
                min={1}
                value={memoryLimit}
                onChange={(e) => setMemoryLimit(e.target.value)}
                required
              />
            </div>
          </div>

          <div className="grid gap-2">
            <Label htmlFor="cp-tests">Tests reference</Label>
            <Input
              id="cp-tests"
              value={testsRef}
              onChange={(e) => setTestsRef(e.target.value)}
              placeholder="s3://bucket/tests/my-problem"
              required
            />
          </div>

          <div className="grid gap-2">
            <Label htmlFor="cp-tags">Tags</Label>
            <p className="text-xs text-muted-foreground">
              Search existing tags; press Enter to add a match or create a new
              tag.
            </p>
            {selectedTags.length > 0 && (
              <div className="flex flex-wrap gap-1.5">
                {selectedTags.map((t) => (
                  <Badge
                    key={t.id}
                    variant="secondary"
                    className="gap-1 pr-1 font-normal"
                  >
                    {t.name}
                    <button
                      type="button"
                      className="rounded-full p-0.5 hover:bg-muted-foreground/20"
                      aria-label={`Remove ${t.name}`}
                      onClick={() =>
                        setSelectedTags((prev) =>
                          prev.filter((x) => x.id !== t.id),
                        )
                      }
                    >
                      <X className="size-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            )}
            <div className="relative">
              <Input
                id="cp-tags"
                value={tagInput}
                onChange={(e) => {
                  setTagInput(e.target.value);
                  setTagSuggestOpen(true);
                }}
                onKeyDown={onTagInputKeyDown}
                onFocus={() => setTagSuggestOpen(true)}
                onBlur={() => {
                  window.setTimeout(() => setTagSuggestOpen(false), 150);
                }}
                placeholder="Type to search tags…"
                autoComplete="off"
              />
              {tagSuggestOpen &&
                (tagSuggestions.length > 0 ||
                  tagInput.trim().length > 0) && (
                  <ul
                    className="absolute z-100 mt-1 max-h-48 w-full overflow-auto rounded-md border border-border bg-popover py-1 text-sm shadow-md"
                    role="listbox"
                  >
                    {tagSuggestions.map((t) => (
                      <li key={t.id} role="option">
                        <button
                          type="button"
                          className="flex w-full cursor-pointer px-3 py-2 text-left hover:bg-muted"
                          onMouseDown={(e) => e.preventDefault()}
                          onClick={() => {
                            addSelectedTag(t);
                            setTagSuggestOpen(false);
                          }}
                        >
                          {t.name}
                        </button>
                      </li>
                    ))}
                    {tagInput.trim().length > 0 && tagSuggestions.length === 0 && (
                      <li className="px-3 py-2 text-muted-foreground">
                        No matches — press{" "}
                        <kbd className="rounded border bg-muted px-1">Enter</kbd>{" "}
                        to create &quot;{tagInput.trim()}&quot;
                      </li>
                    )}
                  </ul>
                )}
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label>Visibility</Label>
              <Select
                value={visibility}
                onValueChange={(v) => setVisibility(v as Visibility)}
              >
                <SelectTrigger className="w-full min-w-0" id="cp-vis">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="draft">Draft</SelectItem>
                  <SelectItem value="published">Published</SelectItem>
                  <SelectItem value="archived">Archived</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="grid gap-2">
              <Label>Difficulty</Label>
              <Select
                value={difficulty}
                onValueChange={(v) => setDifficulty(v as Difficulty)}
              >
                <SelectTrigger className="w-full min-w-0" id="cp-diff">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="easy">Easy</SelectItem>
                  <SelectItem value="medium">Medium</SelectItem>
                  <SelectItem value="hard">Hard</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Function spec */}
          <div className="rounded-lg border border-border p-4 grid gap-3">
            <p className="text-sm font-medium">Function signature</p>
            <p className="text-xs text-muted-foreground -mt-1">
              Test cases will use typed parameter inputs. You can configure them after creation.
            </p>

            <div className="grid gap-3 pt-1">
                <div className="grid gap-2 sm:grid-cols-2">
                  <div className="grid gap-2">
                    <Label htmlFor="cp-func-name">Function name</Label>
                    <Input
                      id="cp-func-name"
                      value={funcName}
                      onChange={(e) => setFuncName(e.target.value)}
                      placeholder="twoSum"
                    />
                  </div>
                  <div className="grid gap-2">
                    <Label>Return type</Label>
                    <Select value={returnType} onValueChange={(v) => setReturnType(v as ParamType)}>
                      <SelectTrigger className="w-full min-w-0"><SelectValue /></SelectTrigger>
                      <SelectContent>
                        {ALL_PARAM_TYPES.map((t) => (
                          <SelectItem key={t} value={t}>{t}</SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                </div>

                <div className="grid gap-2">
                  <Label>Parameters</Label>
                  {parameters.map((p, i) => (
                    <div key={i} className="flex gap-2 items-center">
                      <Input
                        value={p.name}
                        onChange={(e) => updateParameter(i, "name", e.target.value)}
                        placeholder="name"
                        className="flex-1"
                      />
                      <Select value={p.type} onValueChange={(v) => updateParameter(i, "type", v)}>
                        <SelectTrigger className="w-36 min-w-0"><SelectValue /></SelectTrigger>
                        <SelectContent>
                          {ALL_PARAM_TYPES.map((t) => (
                            <SelectItem key={t} value={t}>{t}</SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                      <Button type="button" variant="ghost" size="icon" onClick={() => removeParameter(i)}>
                        <X className="h-4 w-4" />
                      </Button>
                    </div>
                  ))}
                  <Button type="button" variant="outline" size="sm" className="w-fit" onClick={addParameter}>
                    <Plus className="mr-1 h-3 w-3" /> Add parameter
                  </Button>
                </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? "Creating…" : "Create"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
