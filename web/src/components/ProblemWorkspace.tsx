import { useCallback, useEffect, useRef, useState } from "react";
import { Link } from "@tanstack/react-router";
import {
  ArrowLeft,
  CheckCircle2,
  ChevronDown,
  Clock,
  Copy,
  HardDrive,
  Loader2,
  Lock,
  Play,
  RotateCcw,
  Send,
  XCircle,
} from "lucide-react";
import Editor, { type BeforeMount, type OnMount } from "@monaco-editor/react";
import { toast } from "sonner";
import { SubmissionsTab } from "@/components/SubmissionsTab";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from "@/components/ui/resizable";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ScrollArea } from "@/components/ui/scroll-area";
import { Separator } from "@/components/ui/separator";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";
import type { FunctionSpec, ParamType, Problem } from "@/lib/problems";
import { ApiError } from "@/lib/api";
import {
  getSubmission,
  isTerminalSubmissionStatus,
  listSubmissions,
  runSolution,
  submitSolution,
  type Submission,
  type SubmissionStatus,
  type TestcaseResultEntry,
} from "@/lib/submissions";
import { listProblemLanguages, type Language } from "@/lib/languages";
import { DifficultyBadge } from "@/components/DifficultyBadge";
import { StatementMarkdown } from "@/components/StatementMarkdown";

const STARTER_CODE: Record<string, string> = {
  python: "# Write your solution here\n\ndef solve():\n    pass\n",
  javascript:
    "// Write your solution here\n\nfunction solve() {\n  return null;\n}\n",
  go: "package main\n\nfunc solve() {\n\t// Write your solution here\n}\n",
  java: "// Write your solution here\n\npublic int solve() {\n    return 0;\n}\n",
};

const MONACO_LANG: Record<string, string> = {
  python: "python",
  javascript: "javascript",
  go: "go",
  java: "java",
};

const STATUS_LABEL: Record<SubmissionStatus, string> = {
  pending: "Pending",
  queued: "Queued",
  running: "Running",
  accepted: "Accepted",
  wrong_answer: "Wrong answer",
  time_limit_exceeded: "Time limit exceeded",
  memory_limit_exceeded: "Memory limit exceeded",
  runtime_error: "Runtime error",
  compilation_error: "Compilation error",
  internal_error: "Internal error",
  blocked: "Blocked",
};

function formatVerdict(verdict: string | undefined): string {
  if (!verdict) return "—";
  return verdict
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .replace(/_/g, " ")
    .replace(/^./, (c) => c.toUpperCase());
}


function prettyData(value: unknown): string {
  if (value === undefined || value === null) return "";
  if (typeof value === "string") return value;
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function prettyActual(raw: string): string {
  try {
    return prettyData(JSON.parse(raw));
  } catch {
    return raw;
  }
}

function DataBlock({
  label,
  value,
  highlight,
}: {
  label: string;
  value: string;
  highlight?: "ok" | "err" | "neutral";
}) {
  const colorClass =
    highlight === "ok"
      ? "border-emerald-500/20 bg-emerald-500/5 text-emerald-400"
      : highlight === "err"
        ? "border-rose-500/20 bg-rose-500/5 text-rose-400"
        : "border-border bg-muted/40 text-foreground";
  return (
    <div className="space-y-1">
      <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
        {label}
      </p>
      <pre
        className={`rounded border p-2 font-mono text-[11px] leading-relaxed whitespace-pre-wrap ${colorClass}`}
      >
        {value}
      </pre>
    </div>
  );
}

type DiffSegment = { type: "same" | "add" | "del"; text: string };

function diffChars(expected: string, actual: string): DiffSegment[] {
  const m = expected.length;
  const n = actual.length;
  if (m === 0 && n === 0) return [];
  if (m === 0) return [{ type: "add", text: actual }];
  if (n === 0) return [{ type: "del", text: expected }];

  const dp: Uint32Array[] = Array.from(
    { length: m + 1 },
    () => new Uint32Array(n + 1),
  );
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      if (expected[i - 1] === actual[j - 1]) {
        dp[i][j] = dp[i - 1][j - 1] + 1;
      } else {
        dp[i][j] = Math.max(dp[i - 1][j], dp[i][j - 1]);
      }
    }
  }

  const segs: DiffSegment[] = [];
  const push = (type: DiffSegment["type"], ch: string) => {
    const last = segs[segs.length - 1];
    if (last && last.type === type) last.text += ch;
    else segs.push({ type, text: ch });
  };

  let i = m;
  let j = n;
  const stack: DiffSegment[] = [];
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && expected[i - 1] === actual[j - 1]) {
      stack.push({ type: "same", text: expected[i - 1] });
      i--;
      j--;
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      stack.push({ type: "add", text: actual[j - 1] });
      j--;
    } else {
      stack.push({ type: "del", text: expected[i - 1] });
      i--;
    }
  }
  for (let k = stack.length - 1; k >= 0; k--) {
    push(stack[k].type, stack[k].text);
  }
  return segs;
}

function DiffPanels({
  expected,
  actual,
}: {
  expected: string;
  actual: string;
}) {
  const segs = diffChars(expected, actual);
  return (
    <>
      <div className="space-y-1">
        <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
          Expected Output
        </p>
        <pre className="rounded border border-border bg-muted/40 p-2 font-mono text-[11px] leading-relaxed whitespace-pre-wrap text-foreground">
          {segs.map((s, idx) => {
            if (s.type === "add") return null;
            if (s.type === "same") return <span key={idx}>{s.text}</span>;
            return (
              <span
                key={idx}
                className="rounded-sm bg-rose-100 text-rose-700 dark:bg-rose-500/30 dark:text-rose-300"
              >
                {s.text}
              </span>
            );
          })}
        </pre>
      </div>
      <div className="space-y-1">
        <p className="text-[10px] font-medium uppercase tracking-wider text-muted-foreground">
          Your Output
        </p>
        <pre className="rounded border border-border bg-muted/40 p-2 font-mono text-[11px] leading-relaxed whitespace-pre-wrap text-foreground">
          {segs.map((s, idx) => {
            if (s.type === "del") return null;
            if (s.type === "same") return <span key={idx}>{s.text}</span>;
            return (
              <span
                key={idx}
                className="rounded-sm bg-rose-100 text-rose-700 dark:bg-rose-500/30 dark:text-rose-300"
              >
                {s.text}
              </span>
            );
          })}
        </pre>
      </div>
    </>
  );
}

function isAcceptedVerdict(verdict: string): boolean {
  return verdict === "Accepted" || verdict === "accepted";
}

function goType(type: ParamType): string {
  const types: Record<ParamType, string> = {
    int: "int",
    float: "float64",
    string: "string",
    bool: "bool",
    "int[]": "[]int",
    "float[]": "[]float64",
    "string[]": "[]string",
    "bool[]": "[]bool",
    "int[][]": "[][]int",
    "string[][]": "[][]string",
    ListNode: "*ListNode",
    TreeNode: "*TreeNode",
  };
  return types[type] ?? "any";
}

function goZeroValue(type: ParamType): string {
  if (type.endsWith("[]") || type.includes("[][]")) return "nil";
  const values: Partial<Record<ParamType, string>> = {
    int: "0",
    float: "0",
    string: `""`,
    bool: "false",
    ListNode: "nil",
    TreeNode: "nil",
  };
  return values[type] ?? "nil";
}

function javaType(type: ParamType): string {
  const types: Record<ParamType, string> = {
    int: "int",
    float: "double",
    string: "String",
    bool: "boolean",
    "int[]": "int[]",
    "float[]": "double[]",
    "string[]": "String[]",
    "bool[]": "boolean[]",
    "int[][]": "int[][]",
    "string[][]": "String[][]",
    ListNode: "ListNode",
    TreeNode: "TreeNode",
  };
  return types[type] ?? "Object";
}

function javaZeroValue(type: ParamType): string {
  if (type.endsWith("[]") || type.includes("[][]")) return "null";
  const values: Partial<Record<ParamType, string>> = {
    int: "0",
    float: "0.0",
    string: `""`,
    bool: "false",
    ListNode: "null",
    TreeNode: "null",
  };
  return values[type] ?? "null";
}

function buildFunctionStarter(spec: FunctionSpec, language: string): string {
  const names = spec.parameters.map((p) => p.name);

  if (language === "python") {
    const args = names.join(", ");
    return `# Write your solution here\n\ndef ${spec.function_name}(${args}):\n    pass\n`;
  }

  if (language === "javascript") {
    const args = names.join(", ");
    return `// Write your solution here\n\nfunction ${spec.function_name}(${args}) {\n  return null;\n}\n`;
  }

  if (language === "go") {
    const args = spec.parameters
      .map((p) => `${p.name} ${goType(p.type)}`)
      .join(", ");
    const returnType = goType(spec.return_type);
    const zero = goZeroValue(spec.return_type);
    return `func ${spec.function_name}(${args}) ${returnType} {\n\treturn ${zero}\n}\n`;
  }

  if (language === "java") {
    const args = spec.parameters
      .map((p) => `${javaType(p.type)} ${p.name}`)
      .join(", ");
    const returnType = javaType(spec.return_type);
    const zero = javaZeroValue(spec.return_type);
    return `public ${returnType} ${spec.function_name}(${args}) {\n    return ${zero};\n}\n`;
  }

  return STARTER_CODE[language] ?? "";
}

function starterCodeFor(problem: Problem, language: string): string {
  if (problem.function_spec) {
    return buildFunctionStarter(problem.function_spec, language);
  }
  return STARTER_CODE[language] ?? "";
}

// ─── Monaco theme registration (called once via beforeMount) ──────────────────

const defineMonacoThemes: BeforeMount = (monaco) => {
  monaco.editor.defineTheme("capstone-dark", {
    base: "vs-dark",
    inherit: true,
    rules: [
      { token: "comment", foreground: "6b7280", fontStyle: "italic" },
      { token: "keyword", foreground: "a78bfa" },
      { token: "string", foreground: "6ee7b7" },
      { token: "number", foreground: "fb923c" },
      { token: "type", foreground: "67e8f9" },
      { token: "function", foreground: "818cf8" },
    ],
    colors: {
      "editor.background": "#0d0d0d",
      "editor.foreground": "#e5e7eb",
      "editorLineNumber.foreground": "#374151",
      "editorLineNumber.activeForeground": "#6b7280",
      "editor.selectionBackground": "#3b1d8a55",
      "editor.lineHighlightBackground": "#ffffff08",
      "editorCursor.foreground": "#a78bfa",
      "editorIndentGuide.background1": "#1f2937",
      "editorIndentGuide.activeBackground1": "#374151",
      "editor.inactiveSelectionBackground": "#3b1d8a33",
    },
  });
  monaco.editor.defineTheme("capstone-light", {
    base: "vs",
    inherit: true,
    rules: [
      { token: "comment", foreground: "9ca3af", fontStyle: "italic" },
      { token: "keyword", foreground: "7c3aed" },
      { token: "string", foreground: "059669" },
      { token: "number", foreground: "ea580c" },
      { token: "type", foreground: "0891b2" },
      { token: "function", foreground: "4f46e5" },
    ],
    colors: {
      "editor.background": "#fafafa",
      "editor.foreground": "#111827",
      "editorLineNumber.foreground": "#d1d5db",
      "editorLineNumber.activeForeground": "#9ca3af",
      "editor.selectionBackground": "#ede9fe",
      "editor.lineHighlightBackground": "#f5f3ff",
      "editorCursor.foreground": "#7c3aed",
    },
  });
};

// ─── sub-components ───────────────────────────────────────────────────────────

function ProblemMeta({ problem }: { problem: Problem }) {
  const tags = (problem.tags ?? []).map((t, i) =>
    typeof t === "string"
      ? { key: `t-${i}`, label: t }
      : { key: t.id, label: t.name },
  );

  return (
    <div className="space-y-3">
      <div className="flex flex-wrap gap-2">
        <div className="flex items-center gap-1.5 rounded-md border border-border bg-muted/40 px-2.5 py-1.5">
          <Clock className="size-3.5 shrink-0 text-muted-foreground" />
          <span className="text-xs font-medium tabular-nums">
            {problem.time_limit_ms.toLocaleString()} ms
          </span>
        </div>
        <div className="flex items-center gap-1.5 rounded-md border border-border bg-muted/40 px-2.5 py-1.5">
          <HardDrive className="size-3.5 shrink-0 text-muted-foreground" />
          <span className="text-xs font-medium tabular-nums">
            {problem.memory_limit_mb} MB
          </span>
        </div>
        {problem.acceptance_rate !== undefined && (
          <div className="flex items-center gap-1.5 rounded-md border border-border bg-muted/40 px-2.5 py-1.5">
            <CheckCircle2 className="size-3.5 shrink-0 text-muted-foreground" />
            <span className="text-xs font-medium tabular-nums">
              {problem.acceptance_rate.toFixed(1)}% accepted
            </span>
          </div>
        )}
      </div>

      {tags.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {tags.map((t) => (
            <Badge
              key={t.key}
              variant="outline"
              className="text-xs font-normal"
            >
              {t.label}
            </Badge>
          ))}
        </div>
      )}
    </div>
  );
}

function TestResultsPane({
  submission,
  isExecuting,
  error,
}: {
  submission: Submission | null;
  isExecuting: boolean;
  error: string | null;
}) {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const result = submission?.result;
  const rows = result?.testcase_results ?? [];
  const isCompilationError = submission?.status === "compilation_error";

  const stdoutSegments = rows
    .map((row, i) => {
      const out = row.stdout_output?.trim();
      if (!out) return null;
      return rows.length > 1 ? `[Test ${i + 1}]\n${out}` : out;
    })
    .filter(Boolean)
    .join("\n\n");

  const consoleOutput = result?.compiler_output ?? stdoutSegments ?? null;

  // Calculate pass/fail stats
  const passedCount = rows.filter(r => isAcceptedVerdict(r.verdict)).length;
  const totalCount = rows.length;

  return (
    <div className="flex h-full min-h-0 flex-col bg-card">
      <Tabs
        defaultValue="results"
        className="flex min-h-0 flex-1 flex-col gap-0"
      >
        <div className="flex shrink-0 items-center justify-between border-b border-border px-4 py-0">
          <TabsList
            variant="line"
            className="h-10 w-auto justify-start gap-5 bg-transparent p-0"
          >
            <TabsTrigger value="results" className="px-0 text-xs font-medium">
              Results
            </TabsTrigger>
            <TabsTrigger value="console" className="px-0 text-xs font-medium">
              Console
            </TabsTrigger>
          </TabsList>
          <div className="flex items-center gap-2">
            {totalCount > 0 && (
              <Badge
                variant="outline"
                className="shrink-0 text-[10px] font-normal"
              >
                {passedCount} / {totalCount} passed
              </Badge>
            )}
            <Badge
              variant="secondary"
              className="shrink-0 text-[10px] font-normal"
            >
              {isExecuting
                ? "Executing"
                : submission
                  ? STATUS_LABEL[submission.status]
                  : "No run yet"}
            </Badge>
          </div>
        </div>

        <TabsContent
          value="results"
          className="mt-0 min-h-0 flex-1 overflow-auto p-0 data-[state=inactive]:hidden"
        >
          {error && (
            <p className="px-4 py-5 text-xs text-rose-600 dark:text-rose-400">
              {error}
            </p>
          )}
          {submission?.validation && !submission.validation.is_allowed && (
            <div className="border-b border-border bg-destructive/5 p-4">
              <div className="flex items-start gap-3">
                <XCircle className="mt-0.5 h-4 w-4 shrink-0 text-destructive" />
                <div className="min-w-0 flex-1">
                  <h3 className="text-sm font-semibold text-destructive">
                    Security Validation Failed
                  </h3>
                  <p className="mt-1 text-xs text-muted-foreground">
                    {submission.validation.reason || "Your code violates security constraints and cannot be executed."}
                  </p>
                  {submission.validation.details && (
                    <details className="mt-2">
                      <summary className="cursor-pointer text-xs font-medium text-muted-foreground hover:text-foreground">
                        Details
                      </summary>
                      <div className="mt-2 space-y-2">
                        {Array.isArray(submission.validation.details.violations) &&
                          (submission.validation.details.violations as string[]).length > 0 && (
                          <div>
                            <p className="text-xs font-medium text-destructive">Violations</p>
                            <ul className="mt-1 list-disc pl-4 space-y-0.5">
                              {(submission.validation.details.violations as string[]).map((v, i) => (
                                <li key={i} className="text-xs text-muted-foreground">{v}</li>
                              ))}
                            </ul>
                          </div>
                        )}
                      </div>
                    </details>
                  )}
                </div>
              </div>
            </div>
          )}
          {!error && isCompilationError && result?.compiler_output && (
            <div className="p-3">
              <DataBlock
                label="Compiler Output"
                value={result.compiler_output}
                highlight="err"
              />
            </div>
          )}
          {!error && !isCompilationError && rows.length === 0 && (
            <p className="px-4 py-5 text-xs text-muted-foreground">
              {isExecuting
                ? "Waiting for execution results..."
                : "Run the code to see test case results."}
            </p>
          )}
          {!error &&
            !isCompilationError &&
            rows.map((row: TestcaseResultEntry, index: number) => {
              const accepted = isAcceptedVerdict(row.verdict);
              const expanded = expandedId === row.testcase_id;
              const isHidden =
                row.input_data === undefined && row.expected_data === undefined;
              return (
                <div
                  key={row.testcase_id}
                  className="border-b border-border last:border-b-0"
                >
                  <button
                    className="flex w-full items-center gap-3 px-4 py-2.5 text-left transition-colors hover:bg-muted/40"
                    onClick={() =>
                      setExpandedId(expanded ? null : row.testcase_id)
                    }
                  >
                    <span className="flex min-w-0 flex-1 items-center gap-2">
                      {accepted ? (
                        <CheckCircle2 className="size-3.5 shrink-0 text-emerald-500" />
                      ) : (
                        <XCircle className="size-3.5 shrink-0 text-rose-500" />
                      )}
                      <span className="text-xs font-medium">
                        Test {index + 1}
                      </span>
                      {isHidden && (
                        <Badge
                          variant="outline"
                          className="h-4 gap-1 px-1.5 py-0 text-[9px] font-medium uppercase tracking-wider text-muted-foreground"
                        >
                          <Lock className="size-2.5" />
                          Hidden
                        </Badge>
                      )}
                      <span
                        className={`text-xs ${accepted ? "text-emerald-600 dark:text-emerald-400" : "text-rose-600 dark:text-rose-400"}`}
                      >
                        {formatVerdict(row.verdict)}
                      </span>
                    </span>
                    <span className="flex shrink-0 items-center gap-3 text-xs tabular-nums text-muted-foreground">
                      {row.time_ms != null && <span>{row.time_ms} ms</span>}
                      {row.memory_kb != null && (
                        <span>{(row.memory_kb / 1024).toFixed(1)} MB</span>
                      )}
                      <ChevronDown
                        className={`size-3.5 transition-transform ${expanded ? "rotate-180" : ""}`}
                      />
                    </span>
                  </button>
                  {expanded && (
                    <div className="space-y-2.5 border-t border-border/50 bg-muted/20 px-4 pb-3.5 pt-3">
                      {isHidden && (
                        <p className="flex items-center gap-1.5 text-[11px] italic text-muted-foreground">
                          <Lock className="size-3" />
                          Hidden test case — input and expected output not shown
                        </p>
                      )}
                      {row.input_data !== undefined && (
                        <DataBlock
                          label="Input"
                          value={prettyData(row.input_data)}
                          highlight="neutral"
                        />
                      )}
                      {!accepted &&
                      row.expected_data !== undefined &&
                      row.actual_output !== undefined ? (
                        <DiffPanels
                          expected={prettyData(row.expected_data)}
                          actual={prettyActual(row.actual_output)}
                        />
                      ) : (
                        <>
                          {row.expected_data !== undefined && (
                            <DataBlock
                              label="Expected Output"
                              value={prettyData(row.expected_data)}
                              highlight="ok"
                            />
                          )}
                          {row.actual_output !== undefined && (
                            <DataBlock
                              label="Your Output"
                              value={prettyActual(row.actual_output)}
                              highlight={accepted ? "ok" : "err"}
                            />
                          )}
                        </>
                      )}
                    </div>
                  )}
                </div>
              );
            })}
        </TabsContent>

        <TabsContent
          value="console"
          className="mt-0 min-h-0 flex-1 overflow-auto p-3 data-[state=inactive]:hidden"
        >
          <pre className="h-full rounded-md border border-border bg-zinc-950 p-3 font-mono text-[11px] leading-relaxed whitespace-pre-wrap text-emerald-400">
            {consoleOutput || error || (isExecuting ? "Waiting..." : "No console output.")}
          </pre>
        </TabsContent>
      </Tabs>
    </div>
  );
}

// ─── main component ───────────────────────────────────────────────────────────

export function ProblemWorkspace({ problem }: { problem: Problem }) {
  const [language, setLanguage] = useState("python");
  const [code, setCode] = useState(() => starterCodeFor(problem, "python"));
  const [supportedLanguages, setSupportedLanguages] = useState<Language[]>([]);
  const [languagesLoading, setLanguagesLoading] = useState(true);
  const [copied, setCopied] = useState(false);
  const [submission, setSubmission] = useState<Submission | null>(null);
  const [executionError, setExecutionError] = useState<string | null>(null);
  const [isExecuting, setIsExecuting] = useState(false);
  const pollTimeoutRef = useRef<number | null>(null);
  const editorRef = useRef<Parameters<OnMount>[0] | null>(null);

  // Track dark mode by watching the html class list
  const [isDark, setIsDark] = useState(() =>
    document.documentElement.classList.contains("dark"),
  );
  useEffect(() => {
    const observer = new MutationObserver(() =>
      setIsDark(document.documentElement.classList.contains("dark")),
    );
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class"],
    });
    return () => observer.disconnect();
  }, []);

  useEffect(() => {
    return () => {
      if (pollTimeoutRef.current != null) {
        window.clearTimeout(pollTimeoutRef.current);
      }
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    setLanguagesLoading(true);
    listProblemLanguages(problem.id)
      .then((languages) => {
        if (cancelled) return;
        setSupportedLanguages(languages);
        const nextLanguage = languages.some((item) => item.key === language)
          ? language
          : languages[0]?.key;
        if (nextLanguage && nextLanguage !== language) {
          setLanguage(nextLanguage);
          setCode(starterCodeFor(problem, nextLanguage));
        }
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          toast.error(
            err instanceof ApiError
              ? err.message
              : "Failed to load supported languages.",
          );
          setSupportedLanguages([]);
        }
      })
      .finally(() => {
        if (!cancelled) setLanguagesLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [problem]);

  function handleLanguageChange(lang: string) {
    setLanguage(lang);
    setCode(starterCodeFor(problem, lang));
  }

  // Hydrate editor + result panel with the user's most recent submit for this
  // (problem, language). Runs after languages are loaded and on language switch.
  useEffect(() => {
    if (languagesLoading) return;
    if (!supportedLanguages.some((l) => l.key === language)) return;

    let cancelled = false;
    (async () => {
      try {
        const list = await listSubmissions({
          problemId: problem.id,
          limit: 50,
        });
        if (cancelled) return;
        const latest = list.submissions.find(
          (s) => s.language_key === language,
        );
        if (!latest) return;
        const full = await getSubmission(latest.id);
        if (cancelled) return;
        if (full.source_text) setCode(full.source_text);
        setSubmission(full);
        setExecutionError(null);
      } catch {
        // Keep starter code on failure — non-fatal.
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [problem.id, language, languagesLoading, supportedLanguages]);

  function handleCopy() {
    navigator.clipboard.writeText(code);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  function handleReset() {
    setCode(starterCodeFor(problem, language));
  }

  const pollSubmission = useCallback(async (submissionId: string) => {
    try {
      const nextSubmission = await getSubmission(submissionId);
      setSubmission(nextSubmission);

      if (isTerminalSubmissionStatus(nextSubmission.status)) {
        setIsExecuting(false);
        return;
      }

      pollTimeoutRef.current = window.setTimeout(() => {
        void pollSubmission(submissionId);
      }, 300);
    } catch (err) {
      setIsExecuting(false);
      const message =
        err instanceof ApiError
          ? err.message
          : "Failed to refresh execution status.";
      setExecutionError(message);
      toast.error(message);
    }
  }, []);

  async function handleExecute(kind: "run" | "submit") {
    const sourceText = code.trimEnd();
    if (!sourceText.trim()) {
      toast.error("Code cannot be empty.");
      return;
    }

    if (pollTimeoutRef.current != null) {
      window.clearTimeout(pollTimeoutRef.current);
      pollTimeoutRef.current = null;
    }

    setIsExecuting(true);
    setExecutionError(null);
    setSubmission(null);

    try {
      const fn = kind === "run" ? runSolution : submitSolution;
      const created = await fn(problem.id, {
        language_key: language,
        source_text: sourceText,
      });
      setSubmission(created);
      toast.success(kind === "run" ? "Run queued." : "Submission queued.");

      if (isTerminalSubmissionStatus(created.status)) {
        setIsExecuting(false);
        return;
      }

      pollTimeoutRef.current = window.setTimeout(() => {
        void pollSubmission(created.id);
      }, 200);
    } catch (err) {
      setIsExecuting(false);
      const message =
        err instanceof ApiError ? err.message : "Failed to start execution.";
      setExecutionError(message);
      toast.error(message);
    }
  }

  const handleEditorMount: OnMount = useCallback((editor) => {
    editorRef.current = editor;
  }, []);

  return (
    <div className="flex h-[calc(100dvh-3.5rem)] flex-col overflow-hidden">
      {/* ── workspace top bar ─────────────────────────────────────────────── */}
      <header className="flex h-11 shrink-0 items-center gap-2 border-b border-border bg-background/80 px-3 backdrop-blur-sm sm:gap-2.5 sm:px-4">
        <Button
          variant="ghost"
          size="sm"
          className="h-7 gap-1 px-2 text-xs sm:h-8 sm:text-sm"
          asChild
        >
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
            <ArrowLeft className="size-3.5 sm:size-4" />
            Problems
          </Link>
        </Button>

        <Separator orientation="vertical" className="hidden h-4 sm:block" />

        <h1 className="min-w-0 flex-1 truncate text-sm font-bold leading-tight tracking-tight sm:text-base">
          {problem.title}
        </h1>

        <div className="flex shrink-0 items-center gap-2">
          <DifficultyBadge difficulty={problem.difficulty} />
          {problem.is_solved && (
            <Badge className="h-5 gap-1 px-1.5 py-0 text-[10px] bg-emerald-500/15 text-emerald-700 dark:text-emerald-300">
              <CheckCircle2 className="size-2.5" />
              Solved
            </Badge>
          )}
        </div>
      </header>

      {/* ── panels ────────────────────────────────────────────────────────── */}
      <ResizablePanelGroup
        orientation="horizontal"
        className="flex-1 min-h-0 overflow-hidden"
      >
        {/* ── left: problem statement + submissions ──────────────────────── */}
        <ResizablePanel
          defaultSize={35}
          minSize={24}
          className="h-full min-h-0"
        >
          <div className="flex h-full min-h-0 flex-col">
            <Tabs
              defaultValue="description"
              className="flex min-h-0 flex-1 flex-col gap-0"
            >
              <div className="flex shrink-0 items-center border-b border-border bg-muted/40 px-4 py-0">
                <TabsList
                  variant="line"
                  className="h-10 w-auto justify-start gap-5 bg-transparent p-0"
                >
                  <TabsTrigger
                    value="description"
                    className="px-0 text-xs font-medium"
                  >
                    Description
                  </TabsTrigger>
                  <TabsTrigger
                    value="submissions"
                    className="px-0 text-xs font-medium"
                  >
                    Submissions
                  </TabsTrigger>
                </TabsList>
              </div>

              <TabsContent
                value="description"
                className="mt-0 min-h-0 flex-1 overflow-auto p-0 data-[state=inactive]:hidden"
              >
                <ScrollArea className="flex-1">
                  <div className="space-y-5 p-4 sm:p-5">
                    <ProblemMeta problem={problem} />
                    <Separator />
                    <StatementMarkdown source={problem.statement_markdown} />
                  </div>
                </ScrollArea>
              </TabsContent>

              <TabsContent
                value="submissions"
                className="mt-0 min-h-0 flex-1 overflow-hidden p-0 data-[state=inactive]:hidden"
              >
                <SubmissionsTab problemId={problem.id} />
              </TabsContent>
            </Tabs>
          </div>
        </ResizablePanel>

        <ResizableHandle withHandle />

        {/* ── right: editor + results ──────────────────────────────────── */}
        <ResizablePanel
          defaultSize={65}
          minSize={40}
          className="h-full min-h-0"
        >
          <ResizablePanelGroup
            orientation="vertical"
            className="h-full min-h-0"
          >
            {/* editor pane */}
            <ResizablePanel defaultSize={58} minSize={32} className="min-h-0">
              <div className="flex h-full min-h-0 flex-col">
                {/* toolbar */}
                <div className="flex shrink-0 items-center gap-2 border-b border-border bg-muted/60 px-3 py-2">
                  <Select value={language} onValueChange={handleLanguageChange}>
                    <SelectTrigger
                      size="sm"
                      className="h-7 w-32 text-xs font-medium"
                      disabled={
                        languagesLoading || supportedLanguages.length === 0
                      }
                    >
                      <SelectValue
                        placeholder={
                          languagesLoading ? "Loading..." : "Language"
                        }
                      />
                    </SelectTrigger>
                    <SelectContent>
                      {supportedLanguages.map((item) => (
                        <SelectItem key={item.id} value={item.key}>
                          {item.display_name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>

                  <Separator orientation="vertical" className="h-4" />

                  <div className="flex items-center gap-1">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-7"
                          onClick={handleCopy}
                        >
                          <Copy className="size-3.5" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent side="bottom" className="text-xs">
                        {copied ? "Copied!" : "Copy code"}
                      </TooltipContent>
                    </Tooltip>

                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="size-7"
                          onClick={handleReset}
                        >
                          <RotateCcw className="size-3.5" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent side="bottom" className="text-xs">
                        Reset to starter code
                      </TooltipContent>
                    </Tooltip>
                  </div>

                  <div className="ml-auto flex items-center gap-1.5">
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          type="button"
                          size="sm"
                          className="h-7 gap-1.5 px-3 font-semibold"
                          onClick={() => void handleExecute("run")}
                          disabled={
                            isExecuting ||
                            languagesLoading ||
                            supportedLanguages.length === 0
                          }
                        >
                          {isExecuting ? (
                            <Loader2 className="size-3 animate-spin" />
                          ) : (
                            <Play className="size-3" />
                          )}
                          {isExecuting ? "Running" : "Run"}
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent side="bottom" className="text-xs">
                        Run code{" "}
                        <kbd className="ml-1 rounded bg-muted px-1 font-mono text-[10px]">
                          ⌘ ↵
                        </kbd>
                      </TooltipContent>
                    </Tooltip>

                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          type="button"
                          size="sm"
                          variant="outline"
                          className="h-7 gap-1.5 px-3"
                          onClick={() => void handleExecute("submit")}
                          disabled={
                            isExecuting ||
                            languagesLoading ||
                            supportedLanguages.length === 0
                          }
                        >
                          <Send className="size-3" />
                          Submit
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent side="bottom" className="text-xs">
                        Submit solution
                      </TooltipContent>
                    </Tooltip>
                  </div>
                </div>

                {/* Monaco editor */}
                <div className="min-h-0 flex-1 overflow-hidden">
                  <Editor
                    language={MONACO_LANG[language] ?? "plaintext"}
                    value={code}
                    theme={isDark ? "capstone-dark" : "capstone-light"}
                    beforeMount={defineMonacoThemes}
                    onMount={handleEditorMount}
                    onChange={(val) => setCode(val ?? "")}
                    options={{
                      fontSize: 13,
                      fontFamily:
                        "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
                      fontLigatures: true,
                      lineHeight: 1.7,
                      minimap: { enabled: false },
                      scrollBeyondLastLine: false,
                      padding: { top: 12, bottom: 12 },
                      renderLineHighlight: "line",
                      overviewRulerBorder: false,
                      hideCursorInOverviewRuler: true,
                      scrollbar: {
                        verticalScrollbarSize: 6,
                        horizontalScrollbarSize: 6,
                      },
                      bracketPairColorization: { enabled: true },
                      guides: { bracketPairs: "active" },
                      renderWhitespace: "none",
                      smoothScrolling: true,
                      cursorBlinking: "smooth",
                      cursorSmoothCaretAnimation: "on",
                    }}
                  />
                </div>
              </div>
            </ResizablePanel>

            <ResizableHandle withHandle />

            {/* results pane */}
            <ResizablePanel defaultSize={40} minSize={20} className="min-h-0">
              <TestResultsPane
                submission={submission}
                isExecuting={isExecuting}
                error={executionError}
              />
            </ResizablePanel>
          </ResizablePanelGroup>
        </ResizablePanel>
      </ResizablePanelGroup>
    </div>
  );
}
