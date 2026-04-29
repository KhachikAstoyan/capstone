const BASE_URL = (import.meta.env.VITE_API_URL as string | undefined) ?? '/api/v1'

// Injected by AuthProvider so the client stays decoupled from the React context.
let _getToken: () => string | null = () => null
let _setToken: (token: string | null) => void = () => {}
let _onUnauthenticated: () => void = () => {}

export function configureApiClient(opts: {
  getToken: () => string | null
  setToken: (token: string | null) => void
  onUnauthenticated: () => void
}) {
  _getToken = opts.getToken
  _setToken = opts.setToken
  _onUnauthenticated = opts.onUnauthenticated
}

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function parseError(res: Response): Promise<string> {
  try {
    const body = await res.json()
    return (body as { error?: string; message?: string }).error ??
      (body as { error?: string; message?: string }).message ??
      res.statusText
  } catch {
    return res.statusText
  }
}

async function refreshAccessToken(): Promise<string | null> {
  const res = await fetch(`${BASE_URL}/auth/refresh`, {
    method: 'POST',
    credentials: 'include',
  })
  if (!res.ok) return null
  const data = (await res.json()) as { access_token: string }
  return data.access_token
}

async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
  isRetry = false,
): Promise<T> {
  const token = _getToken()
  const headers = new Headers(options.headers)

  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  if (!headers.has('Content-Type') && options.body) {
    headers.set('Content-Type', 'application/json')
  }

  const res = await fetch(`${BASE_URL}${path}`, {
    ...options,
    headers,
    credentials: 'include',
  })

  if (res.status === 401 && !isRetry) {
    const newToken = await refreshAccessToken()
    if (newToken) {
      _setToken(newToken)
      return apiFetch<T>(path, options, true)
    }
    _onUnauthenticated()
    throw new ApiError(401, 'Session expired. Please log in again.')
  }

  if (!res.ok) {
    const message = await parseError(res)
    throw new ApiError(res.status, message)
  }

  if (res.status === 204) return undefined as T

  return res.json() as Promise<T>
}

export function apiGet<T>(path: string, options?: RequestInit): Promise<T> {
  return apiFetch<T>(path, { ...options, method: 'GET' })
}

export function apiPost<T>(
  path: string,
  body?: unknown,
  options?: RequestInit,
): Promise<T> {
  return apiFetch<T>(path, {
    ...options,
    method: 'POST',
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
}

export function apiPatch<T>(
  path: string,
  body?: unknown,
  options?: RequestInit,
): Promise<T> {
  return apiFetch<T>(path, {
    ...options,
    method: 'PATCH',
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
}

export function apiPut<T>(
  path: string,
  body?: unknown,
  options?: RequestInit,
): Promise<T> {
  return apiFetch<T>(path, {
    ...options,
    method: 'PUT',
    body: body !== undefined ? JSON.stringify(body) : undefined,
  })
}

export function apiDelete<T>(path: string, options?: RequestInit): Promise<T> {
  return apiFetch<T>(path, { ...options, method: 'DELETE' })
}
