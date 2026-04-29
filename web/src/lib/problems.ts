import { apiDelete, apiGet, apiPost, apiPut } from "./api";

export type Difficulty = "easy" | "medium" | "hard";
export type Visibility = "draft" | "published" | "archived";

export type ParamType =
  | "int"
  | "float"
  | "string"
  | "bool"
  | "int[]"
  | "float[]"
  | "string[]"
  | "bool[]"
  | "int[][]"
  | "string[][]"
  | "ListNode"
  | "TreeNode";

export const ALL_PARAM_TYPES: ParamType[] = [
  "int", "float", "string", "bool",
  "int[]", "float[]", "string[]", "bool[]",
  "int[][]", "string[][]",
  "ListNode", "TreeNode",
];

export interface Parameter {
  name: string;
  type: ParamType;
}

export interface FunctionSpec {
  function_name: string;
  parameters: Parameter[];
  return_type: ParamType;
}

export interface Tag {
  id: string;
  name: string;
  created_at: string;
}

export interface Problem {
  id: string;
  slug: string;
  title: string;
  /** Plain-text blurb for list cards (not markdown). */
  summary: string;
  statement_markdown: string;
  time_limit_ms: number;
  memory_limit_mb: number;
  tests_ref: string;
  tests_hash: string;
  visibility: Visibility;
  difficulty: Difficulty;
  created_by_user_id: string;
  created_at: string;
  updated_at: string;
  tags?: Tag[];
  acceptance_rate?: number;
  is_solved?: boolean;
  function_spec?: FunctionSpec;
}

export interface TestCase {
  id: string;
  problem_id: string;
  external_id: string;
  input_data: unknown;
  expected_data: unknown;
  order_index: number;
  is_active: boolean;
  is_hidden: boolean;
  created_at: string;
}

export interface ListProblemsResponse {
  problems: Problem[];
  total: number;
  limit: number;
  offset: number;
}

export interface ListProblemsParams {
  q?: string;
  difficulty?: string;
  tags?: string[];
  page?: number;
  limit?: number;
  visibility?: string;
}

export interface CreateProblemRequest {
  title: string;
  summary?: string;
  statement_markdown: string;
  time_limit_ms: number;
  memory_limit_mb: number;
  tests_ref?: string;
  visibility: Visibility;
  difficulty: Difficulty;
  function_spec?: FunctionSpec;
}

export interface CreateTestCaseRequest {
  input_data: unknown;
  expected_data: unknown;
  order_index: number;
  is_hidden: boolean;
}

export interface UpdateTestCaseRequest {
  input_data?: unknown;
  expected_data?: unknown;
  order_index?: number;
  is_hidden?: boolean;
}

const PAGE_SIZE = 20;

/** Page size for admin problems table */
export const ADMIN_PAGE_SIZE = 15;

export function listProblems(
  params: ListProblemsParams = {},
): Promise<ListProblemsResponse> {
  const {
    q,
    difficulty,
    tags,
    page = 1,
    limit = PAGE_SIZE,
    visibility,
  } = params;
  const offset = (page - 1) * limit;

  const qs = new URLSearchParams();
  qs.set("limit", String(limit));
  qs.set("offset", String(offset));
  if (visibility) qs.set("visibility", visibility);
  if (difficulty) qs.set("difficulty", difficulty);
  if (q) qs.set("search", q);
  if (tags && tags.length > 0) {
    for (const tag of tags) {
      qs.append("tags[]", tag);
    }
  }

  return apiGet<ListProblemsResponse>(`/problems/?${qs.toString()}`);
}

export function createProblem(body: CreateProblemRequest): Promise<Problem> {
  return apiPost<Problem>("/internal/problems/", body);
}

export function getProblemById(id: string): Promise<Problem> {
  return apiGet<Problem>(`/problems/${encodeURIComponent(id)}`);
}

export function updateProblem(
  id: string,
  body: Partial<CreateProblemRequest>,
): Promise<Problem> {
  return apiPut<Problem>(`/internal/problems/${encodeURIComponent(id)}`, body);
}

export function deleteProblem(id: string): Promise<void> {
  return apiDelete(`/internal/problems/${encodeURIComponent(id)}`);
}

export async function getProblemBySlug(slug: string): Promise<Problem> {
  return apiGet<Problem>(`/problems/slug/${encodeURIComponent(slug)}`);
}

export async function listTags(): Promise<Tag[]> {
  const res = await apiGet<{ tags: Tag[] }>("/tags/");
  return res.tags ?? [];
}

export function createTag(name: string): Promise<Tag> {
  return apiPost<Tag>("/internal/tags/", { name: name.trim() });
}

export function updateProblemTags(
  problemId: string,
  tagIds: string[],
): Promise<void> {
  return apiPut(`/internal/problems/${problemId}/tags`, {
    tag_ids: tagIds,
  });
}

export async function listTestCases(problemId: string): Promise<TestCase[]> {
  const res = await apiGet<{ test_cases: TestCase[] }>(
    `/internal/problems/${encodeURIComponent(problemId)}/test-cases`,
  );
  return res.test_cases ?? [];
}

export function createTestCase(
  problemId: string,
  req: CreateTestCaseRequest,
): Promise<TestCase> {
  return apiPost<TestCase>(
    `/internal/problems/${encodeURIComponent(problemId)}/test-cases`,
    req,
  );
}

export function updateTestCase(
  problemId: string,
  tcId: string,
  req: UpdateTestCaseRequest,
): Promise<TestCase> {
  return apiPut<TestCase>(
    `/internal/problems/${encodeURIComponent(problemId)}/test-cases/${encodeURIComponent(tcId)}`,
    req,
  );
}

export function deleteTestCase(
  problemId: string,
  tcId: string,
): Promise<void> {
  return apiDelete(
    `/internal/problems/${encodeURIComponent(problemId)}/test-cases/${encodeURIComponent(tcId)}`,
  );
}

export { PAGE_SIZE };
