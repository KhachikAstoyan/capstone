export type UserStatus = 'ACTIVE' | 'BANNED';

export interface User {
  id: string;
  handle: string;
  email?: string;
  email_verified: boolean;
  display_name?: string;
  avatar_url?: string;
  status: UserStatus;
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: User;
}

export interface RegisterRequest {
  handle: string;
  email: string;
  password: string;
  display_name?: string;
}

export interface LoginRequest {
  email: string;
  password: string;
}

export interface ApiError {
  error: string;
  message: string;
}
