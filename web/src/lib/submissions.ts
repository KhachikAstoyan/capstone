import { apiGet, apiPost } from "./api";

export type SubmissionStatus =
  | "pending"
  | "queued"
  | "running"
  | "accepted"
  | "wrong_answer"
  | "time_limit_exceeded"
  | "memory_limit_exceeded"
  | "runtime_error"
  | "compilation_error"
  | "internal_error"
  | "blocked";

export interface TestcaseResultEntry {
  testcase_id: string;
  verdict: string;
  time_ms?: number;
  memory_kb?: number;
  actual_output?: string;
  stdout_output?: string;
  input_data?: unknown;
  expected_data?: unknown;
}

export interface SubmissionResult {
  submission_id: string;
  overall_verdict: string;
  total_time_ms?: number;
  max_memory_kb?: number;
  wall_time_ms?: number;
  compiler_output?: string;
  testcase_results: TestcaseResultEntry[];
  created_at: string;
}

export type SubmissionKind = "run" | "submit";

export interface CodeValidation {
  is_allowed: boolean;
  severity: string;
  reason: string;
  details?: Record<string, unknown>;
}

export interface Submission {
  id: string;
  user_id: string;
  problem_id: string;
  language_id: string;
  language_key: string;
  source_text?: string;
  status: SubmissionStatus;
  kind: SubmissionKind;
  cp_job_id?: string;
  result?: SubmissionResult;
  validation?: CodeValidation;
  created_at: string;
}

export function submitSolution(
  problemId: string,
  body: { language_key: string; source_text: string },
): Promise<Submission> {
  return apiPost<Submission>(
    `/problems/${encodeURIComponent(problemId)}/submit`,
    body,
  );
}

export function runSolution(
  problemId: string,
  body: { language_key: string; source_text: string },
): Promise<Submission> {
  return apiPost<Submission>(
    `/problems/${encodeURIComponent(problemId)}/run`,
    body,
  );
}

export function getSubmission(id: string): Promise<Submission> {
  return apiGet<Submission>(`/submissions/${encodeURIComponent(id)}`);
}

export interface ListSubmissionsResponse {
  submissions: Submission[];
  total: number;
  limit: number;
  offset: number;
}

export function listSubmissions(opts: {
  problemId?: string;
  limit?: number;
  offset?: number;
}): Promise<ListSubmissionsResponse> {
  const params = new URLSearchParams();
  if (opts.problemId) params.set("problem_id", opts.problemId);
  if (opts.limit != null) params.set("limit", String(opts.limit));
  if (opts.offset != null) params.set("offset", String(opts.offset));
  const qs = params.toString();
  return apiGet<ListSubmissionsResponse>(
    `/submissions${qs ? `?${qs}` : ""}`,
  );
}

export function isTerminalSubmissionStatus(status: SubmissionStatus): boolean {
  return [
    "accepted",
    "wrong_answer",
    "time_limit_exceeded",
    "memory_limit_exceeded",
    "runtime_error",
    "compilation_error",
    "internal_error",
    "blocked",
  ].includes(status);
}
