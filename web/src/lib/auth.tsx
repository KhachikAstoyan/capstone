import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useRef,
  useState,
} from 'react'
import { configureApiClient, apiPost, ApiError } from './api'

export interface User {
  id: string
  handle: string
  email: string
  email_verified: boolean
  display_name: string
  avatar_url: string
  status: 'ACTIVE' | 'BANNED'
  created_at: string
  updated_at: string
}

interface AuthResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  user: User
}

interface RegisterResponse {
  message: string
}

interface AuthState {
  user: User | null
  accessToken: string | null
  loading: boolean
}

interface AuthContextValue extends AuthState {
  login: (email: string, password: string) => Promise<User>
  register: (
    handle: string,
    email: string,
    password: string,
    displayName?: string,
  ) => Promise<void>
  logout: () => Promise<void>
  /** Re-fetch user from refresh cookie + access token (e.g. after email verification). No-op if unauthenticated. */
  refreshSession: () => Promise<void>
}

const AuthContext = createContext<AuthContextValue | null>(null)

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [state, setState] = useState<AuthState>({
    user: null,
    accessToken: null,
    loading: true,
  })

  // Use a ref so the api client's callbacks always read the latest token
  // without needing to re-inject on every render.
  const tokenRef = useRef<string | null>(null)

  const setToken = useCallback((token: string | null) => {
    tokenRef.current = token
    setState((s) => ({ ...s, accessToken: token }))
  }, [])

  const clearAuth = useCallback(() => {
    tokenRef.current = null
    setState({ user: null, accessToken: null, loading: false })
  }, [])

  // Wire the api client once on mount.
  useEffect(() => {
    configureApiClient({
      getToken: () => tokenRef.current,
      setToken,
      onUnauthenticated: clearAuth,
    })
  }, [setToken, clearAuth])

  // Attempt to restore session from the HttpOnly refresh token cookie.
  useEffect(() => {
    let cancelled = false

    async function restoreSession() {
      try {
        const data = await apiPost<AuthResponse>('/auth/refresh')
        if (!cancelled) {
          tokenRef.current = data.access_token
          setState({
            user: data.user,
            accessToken: data.access_token,
            loading: false,
          })
        }
      } catch {
        if (!cancelled) {
          setState({ user: null, accessToken: null, loading: false })
        }
      }
    }

    restoreSession()
    return () => {
      cancelled = true
    }
  }, [])

  const login = useCallback(
    async (email: string, password: string): Promise<User> => {
      const data = await apiPost<AuthResponse>('/auth/login', {
        email,
        password,
      })
      tokenRef.current = data.access_token
      setState({ user: data.user, accessToken: data.access_token, loading: false })
      return data.user
    },
    [],
  )

  const register = useCallback(
    async (
      handle: string,
      email: string,
      password: string,
      displayName?: string,
    ): Promise<void> => {
      await apiPost<RegisterResponse>('/auth/register', {
        handle,
        email,
        password,
        ...(displayName ? { display_name: displayName } : {}),
      })
    },
    [],
  )

  const logout = useCallback(async () => {
    try {
      await apiPost('/auth/logout')
    } catch (err) {
      // Ignore API errors on logout — clear local state regardless.
      if (!(err instanceof ApiError)) throw err
    } finally {
      clearAuth()
    }
  }, [clearAuth])

  const refreshSession = useCallback(async () => {
    try {
      const data = await apiPost<AuthResponse>('/auth/refresh')
      tokenRef.current = data.access_token
      setState({
        user: data.user,
        accessToken: data.access_token,
        loading: false,
      })
    } catch {
      // No valid session — leave state unchanged.
    }
  }, [])

  return (
    <AuthContext.Provider
      value={{ ...state, login, register, logout, refreshSession }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return ctx
}
