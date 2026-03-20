export type ProblemVisibility = 'draft' | 'published' | 'archived';

export interface Problem {
  id: string;
  slug: string;
  title: string;
  statement_markdown: string;
  time_limit_ms: number;
  memory_limit_mb: number;
  tests_ref: string;
  tests_hash?: string;
  visibility: ProblemVisibility;
  created_by_user_id?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateProblemRequest {
  title: string;
  statement_markdown: string;
  time_limit_ms: number;
  memory_limit_mb: number;
  tests_ref: string;
  visibility: ProblemVisibility;
}

export interface UpdateProblemRequest {
  slug?: string;
  title?: string;
  statement_markdown?: string;
  time_limit_ms?: number;
  memory_limit_mb?: number;
  tests_ref?: string;
  visibility?: ProblemVisibility;
}

export interface ListProblemsResponse {
  problems: Problem[];
  total: number;
  limit: number;
  offset: number;
}
