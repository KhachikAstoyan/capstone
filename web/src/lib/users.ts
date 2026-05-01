import { apiGet } from "./api";

export interface PublicUserProfile {
  id: string;
  handle: string;
  display_name?: string;
  avatar_url?: string;
  status: "ACTIVE" | "BANNED";
  created_at: string;
  solved_problems: PublicSolvedProblem[];
}

export interface PublicSolvedProblem {
  id: string;
  slug: string;
  title: string;
  summary: string;
  difficulty: "easy" | "medium" | "hard";
  solution: PublicSolution;
  solved_at: string;
}

export interface PublicSolution {
  id: string;
  language_id: string;
  language_key: string;
  language_display_name: string;
  source_text?: string;
  status: string;
  created_at: string;
}

export interface UserStats {
  total_solved: number;
  solved_by_difficulty: Record<string, number>;
  solved_by_tag: TagStat[];
  recent_submissions: RecentSubmission[];
  submission_stats: SubmissionStats;
}

export interface TagStat {
  tag_id: string;
  tag_name: string;
  count: number;
  problems: number[];
}

export interface RecentSubmission {
  problem_id: string;
  problem_slug: string;
  problem_title: string;
  language_key: string;
  status: string;
  created_at: string;
}

export interface SubmissionStats {
  total_submissions: number;
  total_test_runs: number;
  acceptance_rate: number;
  most_used_languages: LanguageStat[];
}

export interface LanguageStat {
  language_key: string;
  language_name: string;
  count: number;
}

export function getPublicProfile(userRef: string): Promise<PublicUserProfile> {
  const encoded = encodeURIComponent(userRef);
  return apiGet<PublicUserProfile>(`/users/${encoded}`);
}

export function getUserStats(): Promise<UserStats> {
  return apiGet<UserStats>(`/auth/me/stats`);
}
