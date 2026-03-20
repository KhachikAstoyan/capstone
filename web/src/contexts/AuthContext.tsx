import { createContext, useContext, useState, useEffect } from 'react';
import type { ReactNode } from 'react';
import { authApi } from '../api/auth';
import { apiClient } from '../api/client';
import type { User, LoginRequest, RegisterRequest } from '../types/auth';

interface AuthContextType {
  user: User | null;
  loading: boolean;
  permissions: string[];
  login: (data: LoginRequest) => Promise<void>;
  register: (data: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  isAuthenticated: boolean;
  hasPermission: (permission: string) => boolean;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [permissions, setPermissions] = useState<string[]>([]);

  useEffect(() => {
    const token = apiClient.getToken();
    if (token) {
      loadUser();
    } else {
      setLoading(false);
    }

    const handleAuthLogout = () => {
      setUser(null);
      setPermissions([]);
      apiClient.setToken(null);
    };

    window.addEventListener('auth:logout', handleAuthLogout);
    return () => window.removeEventListener('auth:logout', handleAuthLogout);
  }, []);

  const loadUser = async () => {
    try {
      const [userData, userPermissions] = await Promise.all([
        authApi.getCurrentUser(),
        authApi.getMyPermissions().catch(() => []),
      ]);
      setUser(userData);
      setPermissions(userPermissions || []);
    } catch (error) {
      console.error('Failed to load user:', error);
      apiClient.setToken(null);
      setPermissions([]);
    } finally {
      setLoading(false);
    }
  };

  const login = async (data: LoginRequest) => {
    const response = await authApi.login(data);
    apiClient.setToken(response.access_token);
    setUser(response.user);
    const userPermissions = await authApi.getMyPermissions().catch(() => []);
    setPermissions(userPermissions || []);
  };

  const register = async (data: RegisterRequest) => {
    const response = await authApi.register(data);
    apiClient.setToken(response.access_token);
    setUser(response.user);
    const userPermissions = await authApi.getMyPermissions();
    setPermissions(userPermissions);
  };

  const logout = async () => {
    try {
      await authApi.logout();
    } catch (error) {
      console.error('Logout error:', error);
    } finally {
      apiClient.setToken(null);
      setUser(null);
      setPermissions([]);
    }
  };

  const hasPermission = (permission: string): boolean => {
    return permissions.includes(permission);
  };

  return (
    <AuthContext.Provider
      value={{
        user,
        loading,
        permissions,
        login,
        register,
        logout,
        isAuthenticated: !!user,
        hasPermission,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
}
