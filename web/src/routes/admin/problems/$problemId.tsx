import { createFileRoute, Link } from "@tanstack/react-router";
import { useCallback, useEffect, useState, useSyncExternalStore } from "react";
import { ChevronLeft, Pencil, Plus, Trash2, X } from "lucide-react";
import MDEditor from "@uiw/react-md-editor";
import { toast } from "sonner";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { Textarea } from "@/components/ui/textarea";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError } from "@/lib/api";
import {
  ALL_PARAM_TYPES,
  createTag,
  createTestCase,
  deleteTestCase,
  getProblemById,
  listTags,
  listTestCases,
  updateProblem,
  updateProblemTags,
  updateTestCase,
  type CreateProblemRequest,
  type Difficulty,
  type FunctionSpec,
  type Parameter,
  type ParamType,
  type Problem,
  type Tag,
  type TestCase,
  type Visibility,
} from "@/lib/problems";
import {
  listInternalProblemLanguages,
  listLanguages,
  updateProblemLanguages,
  type Language,
} from "@/lib/languages";

function subscribeHtmlClass(callback: () => void) {
  const el = document.documentElement;
  const mo = new MutationObserver(callback);
  mo.observe(el, { attributes: true, attributeFilter: ["class"] });
  return () => mo.disconnect();
}

function useDocumentDarkMode(): "light" | "dark" {
  return useSyncExternalStore(
    subscribeHtmlClass,
    () => (document.documentElement.classList.contains("dark") ? "dark" : "light"),
    () => "light",
  );
}

export const Route = createFileRoute("/admin/problems/$problemId")({
  component: AdminProblemEditPage,
});

function AdminProblemEditPage() {
  const { problemId } = Route.useParams();
  const [problem, setProblem] = useState<Problem | null>(null);
  const [loading, setLoading] = useState(true);
  const [err, setErr] = useState<string | null>(null);
  const [testCasesRefreshKey, setTestCasesRefreshKey] = useState(0);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setErr(null);
    getProblemById(problemId)
      .then((p) => { if (!cancelled) setProblem(p); })
      .catch((e: unknown) => {
        if (!cancelled) setErr(e instanceof ApiError ? e.message : "Failed to load problem");
      })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [problemId]);

  const refreshTestCases = useCallback(() => setTestCasesRefreshKey((k) => k + 1), []);
  const onProblemUpdated = useCallback((updated: Problem) => setProblem(updated), []);

  if (loading) return <EditPageSkeleton />;
  if (err || !problem) {
    return (
      <main className="mx-auto max-w-4xl px-4 py-12 sm:px-6">
        <p className="text-destructive">{err ?? "Problem not found."}</p>
        <Link to="/admin/problems" params={{}} search={{ page: 1, q: undefined, visibility: undefined }} className="mt-4 inline-block text-sm text-primary underline-offset-4 hover:underline">
          ← Back to problems
        </Link>
      </main>
    );
  }

  return (
    <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6">
      <div className="mb-6 flex items-center gap-3">
        <Link to="/admin/problems" params={{}} search={{ page: 1, q: undefined, visibility: undefined }} className="text-muted-foreground hover:text-foreground transition-colors">
          <ChevronLeft className="h-5 w-5" />
        </Link>
        <div>
          <h1 className="text-xl font-bold tracking-tight">{problem.title}</h1>
          <p className="text-xs text-muted-foreground font-mono">{problem.slug}</p>
        </div>
      </div>

      <Tabs defaultValue="details">
        <TabsList className="mb-6">
          <TabsTrigger value="details">Details</TabsTrigger>
          <TabsTrigger value="testcases">Test Cases</TabsTrigger>
        </TabsList>

        <TabsContent value="details">
          <DetailsForm problem={problem} onUpdated={onProblemUpdated} />
        </TabsContent>

        <TabsContent value="testcases">
          <TestCasesTab
            problem={problem}
            refreshKey={testCasesRefreshKey}
            onRefresh={refreshTestCases}
          />
        </TabsContent>
      </Tabs>
    </main>
  );
}

// ---------------------------------------------------------------------------
// Details tab
// ---------------------------------------------------------------------------

function DetailsForm({
  problem,
  onUpdated,
}: {
  problem: Problem;
  onUpdated: (p: Problem) => void;
}) {
  const colorMode = useDocumentDarkMode();

  const [title, setTitle] = useState(problem.title);
  const [summary, setSummary] = useState(problem.summary ?? "");
  const [statement, setStatement] = useState(problem.statement_markdown);
  const [timeLimit, setTimeLimit] = useState(String(problem.time_limit_ms));
  const [memoryLimit, setMemoryLimit] = useState(String(problem.memory_limit_mb));
  const [testsRef, setTestsRef] = useState(problem.tests_ref ?? "");
  const [visibility, setVisibility] = useState<Visibility>(problem.visibility);
  const [difficulty, setDifficulty] = useState<Difficulty>(problem.difficulty);
  const [submitting, setSubmitting] = useState(false);

  const [tagCatalog, setTagCatalog] = useState<Tag[]>([]);
  const [selectedTags, setSelectedTags] = useState<Tag[]>(problem.tags ?? []);
  const [tagInput, setTagInput] = useState("");
  const [tagSuggestOpen, setTagSuggestOpen] = useState(false);
  const [languageCatalog, setLanguageCatalog] = useState<Language[]>([]);
  const [selectedLanguageIds, setSelectedLanguageIds] = useState<string[]>([]);

  // function_spec
  const [enableFuncSpec, setEnableFuncSpec] = useState(!!problem.function_spec);
  const [funcName, setFuncName] = useState(problem.function_spec?.function_name ?? "");
  const [returnType, setReturnType] = useState<ParamType>(problem.function_spec?.return_type ?? "int");
  const [parameters, setParameters] = useState<Parameter[]>(problem.function_spec?.parameters ?? []);

  useEffect(() => {
    listTags().then(setTagCatalog).catch(() => { toast.error("Failed to load tags."); });
    listLanguages()
      .then(setLanguageCatalog)
      .catch(() => { toast.error("Failed to load languages."); });
    listInternalProblemLanguages(problem.id)
      .then((languages) => setSelectedLanguageIds(languages.map((language) => language.id)))
      .catch(() => { toast.error("Failed to load problem languages."); });
  }, []);

  const selectedTagIds = new Set(selectedTags.map((t) => t.id));

  const tagSuggestions = (() => {
    const q = tagInput.trim().toLowerCase();
    const pool = tagCatalog.filter((t) => !selectedTagIds.has(t.id));
    if (!q) return pool.slice(0, 8);
    return pool.filter((t) => t.name.toLowerCase().includes(q)).sort((a, b) => a.name.localeCompare(b.name)).slice(0, 8);
  })();

  function mergeIntoCatalog(tag: Tag) {
    setTagCatalog((prev) => prev.some((t) => t.id === tag.id) ? prev : [...prev, tag]);
  }

  function addSelectedTag(tag: Tag) {
    setSelectedTags((prev) => prev.some((t) => t.id === tag.id) ? prev : [...prev, tag]);
    setTagInput("");
  }

  async function commitTagInput() {
    const q = tagInput.trim();
    if (!q) return;
    const pool = tagCatalog.filter((t) => !selectedTagIds.has(t.id));
    const qLower = q.toLowerCase();
    const matches = pool.filter((t) => t.name.toLowerCase().includes(qLower)).sort((a, b) => a.name.localeCompare(b.name));
    if (matches.length > 0) { addSelectedTag(matches.find((t) => t.name.toLowerCase() === qLower) ?? matches[0]!); return; }
    try {
      const created = await createTag(q);
      mergeIntoCatalog(created);
      addSelectedTag(created);
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to create tag.");
    }
  }

  function addParameter() {
    setParameters((prev) => [...prev, { name: "", type: "int" }]);
  }

  function updateParameter(index: number, field: keyof Parameter, value: string) {
    setParameters((prev) => prev.map((p, i) => i === index ? { ...p, [field]: value } : p));
  }

  function removeParameter(index: number) {
    setParameters((prev) => prev.filter((_, i) => i !== index));
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    const time = Number(timeLimit);
    const mem = Number(memoryLimit);
    if (!title.trim()) { toast.error("Title is required."); return; }
    if (!statement.trim()) { toast.error("Statement is required."); return; }
    if (summary.length > 500) { toast.error("Summary must be at most 500 characters."); return; }
    if (selectedLanguageIds.length === 0) { toast.error("Select at least one supported language."); return; }
    if (!Number.isFinite(time) || time <= 0 || !Number.isFinite(mem) || mem <= 0) {
      toast.error("Time and memory limits must be positive numbers."); return;
    }

    let function_spec: FunctionSpec | undefined;
    if (enableFuncSpec) {
      if (!funcName.trim()) { toast.error("Function name is required."); return; }
      for (const p of parameters) {
        if (!p.name.trim()) { toast.error("All parameter names are required."); return; }
      }
      function_spec = { function_name: funcName.trim(), parameters, return_type: returnType };
    }

    const body: Partial<CreateProblemRequest> = {
      title: title.trim(),
      summary: summary.trim() || undefined,
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
      const updated = await updateProblem(problem.id, body);
      const currentTagIds = selectedTags.map((t) => t.id);
      await updateProblemTags(problem.id, currentTagIds).catch(() => {
        toast.error("Problem saved, but tags could not be updated.");
      });
      await updateProblemLanguages(problem.id, selectedLanguageIds).catch(() => {
        toast.error("Problem saved, but supported languages could not be updated.");
      });
      updated.tags = selectedTags;
      onUpdated(updated);
      toast.success("Problem updated.");
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to update problem");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <form onSubmit={handleSubmit} className="grid gap-4">
      <div className="grid gap-2">
        <Label htmlFor="ep-title">Title</Label>
        <Input id="ep-title" value={title} onChange={(e) => setTitle(e.target.value)} required />
      </div>

      <div className="grid gap-2">
        <Label htmlFor="ep-summary">List summary (plain text, optional)</Label>
        <Textarea
          id="ep-summary"
          value={summary}
          onChange={(e) => setSummary(e.target.value)}
          rows={3}
          maxLength={500}
          className="resize-y min-h-18"
        />
        <p className="text-xs text-muted-foreground">{summary.length}/500</p>
      </div>

      <div className="grid gap-2">
        <Label>Statement (Markdown / HTML)</Label>
        <div data-color-mode={colorMode}>
          <MDEditor value={statement} onChange={(v) => setStatement(v ?? "")} height={300} visibleDragbar={false} />
        </div>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        <div className="grid gap-2">
          <Label htmlFor="ep-time">Time limit (ms)</Label>
          <Input id="ep-time" type="number" min={1} value={timeLimit} onChange={(e) => setTimeLimit(e.target.value)} required />
        </div>
        <div className="grid gap-2">
          <Label htmlFor="ep-mem">Memory limit (MB)</Label>
          <Input id="ep-mem" type="number" min={1} value={memoryLimit} onChange={(e) => setMemoryLimit(e.target.value)} required />
        </div>
      </div>

      <div className="grid gap-2">
        <Label htmlFor="ep-tests">Tests reference (optional)</Label>
        <Input
          id="ep-tests"
          value={testsRef}
          onChange={(e) => setTestsRef(e.target.value)}
          placeholder="s3://bucket/tests/my-problem"
        />
      </div>

      {/* Tags */}
      <div className="grid gap-2">
        <Label htmlFor="ep-tags">Tags</Label>
        {selectedTags.length > 0 && (
          <div className="flex flex-wrap gap-1.5">
            {selectedTags.map((t) => (
              <Badge key={t.id} variant="secondary" className="gap-1 pr-1 font-normal">
                {t.name}
                <button
                  type="button"
                  className="rounded-full p-0.5 hover:bg-muted-foreground/20"
                  aria-label={`Remove ${t.name}`}
                  onClick={() => setSelectedTags((prev) => prev.filter((x) => x.id !== t.id))}
                >
                  <X className="size-3" />
                </button>
              </Badge>
            ))}
          </div>
        )}
        <div className="relative">
          <Input
            id="ep-tags"
            value={tagInput}
            onChange={(e) => { setTagInput(e.target.value); setTagSuggestOpen(true); }}
            onKeyDown={async (e) => { if (e.key === "Enter") { e.preventDefault(); await commitTagInput(); } }}
            onFocus={() => setTagSuggestOpen(true)}
            onBlur={() => { window.setTimeout(() => setTagSuggestOpen(false), 150); }}
            placeholder="Type to search tags…"
            autoComplete="off"
          />
          {tagSuggestOpen && (tagSuggestions.length > 0 || tagInput.trim().length > 0) && (
            <ul className="absolute z-100 mt-1 max-h-48 w-full overflow-auto rounded-md border border-border bg-popover py-1 text-sm shadow-md" role="listbox">
              {tagSuggestions.map((t) => (
                <li key={t.id} role="option">
                  <button
                    type="button"
                    className="flex w-full cursor-pointer px-3 py-2 text-left hover:bg-muted"
                    onMouseDown={(e) => e.preventDefault()}
                    onClick={() => { addSelectedTag(t); setTagSuggestOpen(false); }}
                  >
                    {t.name}
                  </button>
                </li>
              ))}
              {tagInput.trim().length > 0 && tagSuggestions.length === 0 && (
                <li className="px-3 py-2 text-muted-foreground">
                  Press <kbd className="rounded border bg-muted px-1">Enter</kbd> to create &quot;{tagInput.trim()}&quot;
                </li>
              )}
            </ul>
          )}
        </div>
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

      {/* Visibility / Difficulty */}
      <div className="grid gap-4 sm:grid-cols-2">
        <div className="grid gap-2">
          <Label>Visibility</Label>
          <Select value={visibility} onValueChange={(v) => setVisibility(v as Visibility)}>
            <SelectTrigger className="w-full min-w-0"><SelectValue /></SelectTrigger>
            <SelectContent>
              <SelectItem value="draft">Draft</SelectItem>
              <SelectItem value="published">Published</SelectItem>
              <SelectItem value="archived">Archived</SelectItem>
            </SelectContent>
          </Select>
        </div>
        <div className="grid gap-2">
          <Label>Difficulty</Label>
          <Select value={difficulty} onValueChange={(v) => setDifficulty(v as Difficulty)}>
            <SelectTrigger className="w-full min-w-0"><SelectValue /></SelectTrigger>
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
        <div className="flex items-center gap-2">
          <input
            id="ep-func-enable"
            type="checkbox"
            checked={enableFuncSpec}
            onChange={(e) => setEnableFuncSpec(e.target.checked)}
            className="h-4 w-4 rounded border-border"
          />
          <Label htmlFor="ep-func-enable" className="cursor-pointer">Enable function-call mode</Label>
        </div>
        <p className="text-xs text-muted-foreground -mt-1">
          Test cases will use typed parameter inputs instead of raw JSON.
        </p>

        {enableFuncSpec && (
          <div className="grid gap-3 pt-1">
            <div className="grid gap-2 sm:grid-cols-2">
              <div className="grid gap-2">
                <Label htmlFor="ep-func-name">Function name</Label>
                <Input
                  id="ep-func-name"
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
        )}
      </div>

      <div className="flex justify-end gap-2 pt-2">
        <Button type="submit" disabled={submitting}>
          {submitting ? "Saving…" : "Save changes"}
        </Button>
      </div>
    </form>
  );
}

// ---------------------------------------------------------------------------
// Test Cases tab
// ---------------------------------------------------------------------------

function TestCasesTab({
  problem,
  refreshKey,
  onRefresh,
}: {
  problem: Problem;
  refreshKey: number;
  onRefresh: () => void;
}) {
  const [testCases, setTestCases] = useState<TestCase[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [editingTc, setEditingTc] = useState<TestCase | null>(null);
  const [showForm, setShowForm] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setErr(null);
    listTestCases(problem.id)
      .then((tcs) => { if (!cancelled) setTestCases(tcs); })
      .catch((e: unknown) => { if (!cancelled) setErr(e instanceof ApiError ? e.message : "Failed to load test cases"); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [problem.id, refreshKey]);

  async function handleDelete(tc: TestCase) {
    if (!window.confirm(`Delete test case #${tc.order_index + 1}?`)) return;
    try {
      await deleteTestCase(problem.id, tc.id);
      setTestCases((prev) => prev.filter((t) => t.id !== tc.id));
      toast.success("Test case deleted.");
      onRefresh();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to delete test case");
    }
  }

  function openAdd() {
    setEditingTc(null);
    setShowForm(true);
  }

  function openEdit(tc: TestCase) {
    setEditingTc(tc);
    setShowForm(true);
  }

  function onSaved() {
    setShowForm(false);
    setEditingTc(null);
    onRefresh();
  }

  return (
    <div className="grid gap-4">
      <div className="flex items-center justify-between">
        <p className="text-sm text-muted-foreground">{testCases.length} test case{testCases.length !== 1 ? "s" : ""}</p>
        <Button size="sm" onClick={openAdd}>
          <Plus className="mr-1 h-4 w-4" /> Add test case
        </Button>
      </div>

      {err && <p className="text-sm text-destructive">{err}</p>}

      <div className="rounded-lg border border-border bg-card">
        {loading ? (
          <div className="p-6"><Skeleton className="h-24 w-full" /></div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-12">#</TableHead>
                <TableHead className="w-20">Hidden</TableHead>
                <TableHead>Input</TableHead>
                <TableHead>Expected</TableHead>
                <TableHead className="w-24 text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {testCases.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={5} className="text-center text-muted-foreground py-8">
                    No test cases yet. Add one above.
                  </TableCell>
                </TableRow>
              ) : (
                testCases.map((tc) => (
                  <TableRow key={tc.id}>
                    <TableCell className="font-mono text-xs">{tc.order_index}</TableCell>
                    <TableCell>
                      {tc.is_hidden ? (
                        <Badge variant="secondary" className="text-xs">hidden</Badge>
                      ) : (
                        <span className="text-muted-foreground text-xs">—</span>
                      )}
                    </TableCell>
                    <TableCell className="font-mono text-xs max-w-[200px] truncate">
                      {JSON.stringify(tc.input_data)}
                    </TableCell>
                    <TableCell className="font-mono text-xs max-w-[200px] truncate">
                      {JSON.stringify(tc.expected_data)}
                    </TableCell>
                    <TableCell className="text-right">
                      <div className="flex justify-end gap-1">
                        <Button variant="ghost" size="icon" onClick={() => openEdit(tc)}>
                          <Pencil className="h-3.5 w-3.5" />
                        </Button>
                        <Button variant="ghost" size="icon" onClick={() => handleDelete(tc)}>
                          <Trash2 className="h-3.5 w-3.5 text-destructive" />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        )}
      </div>

      {showForm && (
        <TestCaseForm
          problem={problem}
          testCase={editingTc}
          totalCount={testCases.length}
          onSaved={onSaved}
          onCancel={() => { setShowForm(false); setEditingTc(null); }}
        />
      )}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Test case add/edit form
// ---------------------------------------------------------------------------

function isComplexType(t: ParamType): boolean {
  return t.endsWith("[]") || t === "ListNode" || t === "TreeNode";
}

function TestCaseForm({
  problem,
  testCase,
  totalCount,
  onSaved,
  onCancel,
}: {
  problem: Problem;
  testCase: TestCase | null;
  totalCount: number;
  onSaved: () => void;
  onCancel: () => void;
}) {
  const spec = problem.function_spec;
  const isEdit = testCase !== null;

  const [orderIndex, setOrderIndex] = useState(
    testCase ? String(testCase.order_index) : String(totalCount),
  );
  const [isHidden, setIsHidden] = useState(testCase?.is_hidden ?? false);
  const [submitting, setSubmitting] = useState(false);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  // Raw mode state
  const [rawInput, setRawInput] = useState(
    testCase ? JSON.stringify(testCase.input_data, null, 2) : "",
  );
  const [rawExpected, setRawExpected] = useState(
    testCase ? JSON.stringify(testCase.expected_data, null, 2) : "",
  );

  // Function-call mode state: one string per param + one for expected
  const [paramValues, setParamValues] = useState<Record<string, string>>(() => {
    if (!spec || !testCase) return {};
    const data = testCase.input_data as Record<string, unknown>;
    return Object.fromEntries(
      spec.parameters.map((p) => [p.name, JSON.stringify(data[p.name] ?? "", null, 2)]),
    );
  });
  const [expectedValue, setExpectedValue] = useState(
    testCase ? JSON.stringify(testCase.expected_data, null, 2) : "",
  );

  function setParamValue(name: string, val: string) {
    setParamValues((prev) => ({ ...prev, [name]: val }));
  }

  function clearErrors() {
    setFieldErrors({});
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    clearErrors();

    const errors: Record<string, string> = {};
    let input_data: unknown;
    let expected_data: unknown;

    if (spec) {
      // Function-call mode
      const inputObj: Record<string, unknown> = {};
      for (const p of spec.parameters) {
        const raw = paramValues[p.name] ?? "";
        try {
          inputObj[p.name] = JSON.parse(raw);
        } catch {
          errors[`param_${p.name}`] = "Invalid JSON";
        }
      }
      try {
        expected_data = JSON.parse(expectedValue);
      } catch {
        errors["expected"] = "Invalid JSON";
      }
      input_data = inputObj;
    } else {
      // Raw mode
      try {
        input_data = JSON.parse(rawInput);
      } catch {
        errors["raw_input"] = "Invalid JSON";
      }
      try {
        expected_data = JSON.parse(rawExpected);
      } catch {
        errors["raw_expected"] = "Invalid JSON";
      }
    }

    if (Object.keys(errors).length > 0) {
      setFieldErrors(errors);
      return;
    }

    const order = Number(orderIndex);
    if (!Number.isFinite(order) || order < 0) {
      toast.error("Order index must be a non-negative number.");
      return;
    }

    setSubmitting(true);
    try {
      if (isEdit && testCase) {
        await updateTestCase(problem.id, testCase.id, {
          input_data,
          expected_data,
          order_index: Math.floor(order),
          is_hidden: isHidden,
        });
        toast.success("Test case updated.");
      } else {
        await createTestCase(problem.id, {
          input_data,
          expected_data,
          order_index: Math.floor(order),
          is_hidden: isHidden,
        });
        toast.success("Test case created.");
      }
      onSaved();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Failed to save test case");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="rounded-lg border border-border bg-card p-4 grid gap-4">
      <div className="flex items-center justify-between">
        <h3 className="font-medium text-sm">{isEdit ? "Edit test case" : "Add test case"}</h3>
        <Button type="button" variant="ghost" size="icon" onClick={onCancel}>
          <X className="h-4 w-4" />
        </Button>
      </div>

      <form onSubmit={handleSubmit} className="grid gap-4">
        {spec ? (
          // Function-call mode
          <>
            {spec.parameters.map((p) => (
              <div key={p.name} className="grid gap-1.5">
                <Label className="font-mono text-xs">
                  {p.name} <span className="text-muted-foreground">({p.type})</span>
                </Label>
                {isComplexType(p.type) ? (
                  <Textarea
                    value={paramValues[p.name] ?? ""}
                    onChange={(e) => setParamValue(p.name, e.target.value)}
                    className="font-mono text-xs resize-y min-h-16"
                    placeholder={`JSON value for ${p.name}`}
                  />
                ) : (
                  <Input
                    value={paramValues[p.name] ?? ""}
                    onChange={(e) => setParamValue(p.name, e.target.value)}
                    className="font-mono text-xs"
                    placeholder={`JSON value for ${p.name}`}
                  />
                )}
                {fieldErrors[`param_${p.name}`] && (
                  <p className="text-xs text-destructive">{fieldErrors[`param_${p.name}`]}</p>
                )}
              </div>
            ))}

            <div className="grid gap-1.5">
              <Label className="font-mono text-xs">
                expected <span className="text-muted-foreground">({spec.return_type})</span>
              </Label>
              {isComplexType(spec.return_type) ? (
                <Textarea
                  value={expectedValue}
                  onChange={(e) => setExpectedValue(e.target.value)}
                  className="font-mono text-xs resize-y min-h-16"
                  placeholder="JSON value"
                />
              ) : (
                <Input
                  value={expectedValue}
                  onChange={(e) => setExpectedValue(e.target.value)}
                  className="font-mono text-xs"
                  placeholder="JSON value"
                />
              )}
              {fieldErrors["expected"] && (
                <p className="text-xs text-destructive">{fieldErrors["expected"]}</p>
              )}
            </div>
          </>
        ) : (
          // Raw mode
          <>
            <div className="grid gap-1.5">
              <Label>Input JSON</Label>
              <Textarea
                value={rawInput}
                onChange={(e) => setRawInput(e.target.value)}
                className="font-mono text-xs resize-y min-h-24"
                placeholder='{"nums": [2, 7, 11, 15], "target": 9}'
              />
              {fieldErrors["raw_input"] && (
                <p className="text-xs text-destructive">{fieldErrors["raw_input"]}</p>
              )}
            </div>
            <div className="grid gap-1.5">
              <Label>Expected JSON</Label>
              <Textarea
                value={rawExpected}
                onChange={(e) => setRawExpected(e.target.value)}
                className="font-mono text-xs resize-y min-h-16"
                placeholder="[0, 1]"
              />
              {fieldErrors["raw_expected"] && (
                <p className="text-xs text-destructive">{fieldErrors["raw_expected"]}</p>
              )}
            </div>
          </>
        )}

        <div className="grid gap-4 sm:grid-cols-2">
          <div className="grid gap-1.5">
            <Label htmlFor="tc-order">Order index</Label>
            <Input
              id="tc-order"
              type="number"
              min={0}
              value={orderIndex}
              onChange={(e) => setOrderIndex(e.target.value)}
            />
          </div>
          <div className="flex items-end gap-2 pb-0.5">
            <input
              id="tc-hidden"
              type="checkbox"
              checked={isHidden}
              onChange={(e) => setIsHidden(e.target.checked)}
              className="h-4 w-4 rounded border-border"
            />
            <Label htmlFor="tc-hidden" className="cursor-pointer">Hidden test case</Label>
          </div>
        </div>

        <div className="flex justify-end gap-2">
          <Button type="button" variant="outline" onClick={onCancel} disabled={submitting}>
            Cancel
          </Button>
          <Button type="submit" disabled={submitting}>
            {submitting ? "Saving…" : isEdit ? "Save changes" : "Add test case"}
          </Button>
        </div>
      </form>
    </div>
  );
}

function EditPageSkeleton() {
  return (
    <main className="mx-auto max-w-4xl px-4 py-8 sm:px-6">
      <Skeleton className="mb-2 h-6 w-40" />
      <Skeleton className="mb-6 h-4 w-24" />
      <Skeleton className="h-96 w-full" />
    </main>
  );
}
