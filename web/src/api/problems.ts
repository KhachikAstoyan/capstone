import { apiClient } from './client';
import type {
  Problem,
  CreateProblemRequest,
  UpdateProblemRequest,
  ListProblemsResponse,
  ProblemVisibility,
} from '../types/problem';

export const problemsApi = {
  list: async (params?: {
    limit?: number;
    offset?: number;
    visibility?: ProblemVisibility;
  }): Promise<ListProblemsResponse> => {
    const searchParams = new URLSearchParams();
    if (params?.limit) searchParams.append('limit', params.limit.toString());
    if (params?.offset) searchParams.append('offset', params.offset.toString());
    if (params?.visibility) searchParams.append('visibility', params.visibility);

    const query = searchParams.toString();
    return apiClient.get<ListProblemsResponse>(
      `/api/v1/problems${query ? `?${query}` : ''}`
    );
  },

  get: async (id: string): Promise<Problem> => {
    return apiClient.get<Problem>(`/api/v1/problems/${id}`);
  },

  getBySlug: async (slug: string): Promise<Problem> => {
    return apiClient.get<Problem>(`/api/v1/problems/slug/${slug}`);
  },

  create: async (data: CreateProblemRequest): Promise<Problem> => {
    return apiClient.post<Problem>('/api/v1/internal/problems', data);
  },

  update: async (id: string, data: UpdateProblemRequest): Promise<Problem> => {
    return apiClient.put<Problem>(`/api/v1/internal/problems/${id}`, data);
  },

  delete: async (id: string): Promise<void> => {
    return apiClient.delete(`/api/v1/internal/problems/${id}`);
  },
};
