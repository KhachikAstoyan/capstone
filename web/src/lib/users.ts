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

export function getPublicProfile(userRef: string): Promise<PublicUserProfile> {
  const encoded = encodeURIComponent(userRef);
  return apiGet<PublicUserProfile>(`/users/${encoded}`);
}
