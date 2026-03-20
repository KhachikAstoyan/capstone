import { apiClient } from './client';
import type { AuthResponse, LoginRequest, RegisterRequest, User } from '../types/auth';

export const authApi = {
  register: async (data: RegisterRequest): Promise<AuthResponse> => {
    return apiClient.post<AuthResponse>('/api/v1/auth/register', data);
  },

  login: async (data: LoginRequest): Promise<AuthResponse> => {
    return apiClient.post<AuthResponse>('/api/v1/auth/login', data);
  },

  logout: async (): Promise<void> => {
    return apiClient.post('/api/v1/auth/logout');
  },

  getCurrentUser: async (): Promise<User> => {
    return apiClient.get<User>('/api/v1/auth/me');
  },

  getMyPermissions: async (): Promise<string[]> => {
    const permissions = await apiClient.get<Array<{ key: string }>>('/api/v1/auth/me/permissions');
    return permissions.map(p => p.key);
  },
};
