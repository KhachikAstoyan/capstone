import { apiGet, apiPost, apiPut } from "./api";

export interface Language {
  id: string;
  key: string;
  display_name: string;
  is_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export async function listLanguages(search?: string): Promise<Language[]> {
  const qs = new URLSearchParams();
  if (search?.trim()) qs.set("search", search.trim());
  const suffix = qs.toString() ? `?${qs.toString()}` : "";
  const res = await apiGet<{ languages: Language[] }>(
    `/internal/languages/${suffix}`,
  );
  return res.languages ?? [];
}

export function createLanguage(body: {
  key: string;
  display_name: string;
  is_enabled?: boolean;
}): Promise<Language> {
  return apiPost<Language>("/internal/languages/", body);
}

export async function listProblemLanguages(problemId: string): Promise<Language[]> {
  const res = await apiGet<{ languages: Language[] }>(
    `/problems/${encodeURIComponent(problemId)}/languages`,
  );
  return res.languages ?? [];
}

export async function listInternalProblemLanguages(
  problemId: string,
): Promise<Language[]> {
  const res = await apiGet<{ languages: Language[] }>(
    `/internal/problems/${encodeURIComponent(problemId)}/languages`,
  );
  return res.languages ?? [];
}

export async function updateProblemLanguages(
  problemId: string,
  languageIds: string[],
): Promise<Language[]> {
  const res = await apiPut<{ languages: Language[] }>(
    `/internal/problems/${encodeURIComponent(problemId)}/languages`,
    { language_ids: languageIds },
  );
  return res.languages ?? [];
}
