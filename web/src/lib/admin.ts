import { apiGet } from './api'

export interface AdminUser {
  id: string
  handle: string
  email?: string
  display_name?: string
  status: 'ACTIVE' | 'BANNED'
  created_at: string
  violation_count: number
  submission_count: number
}

export interface AdminUsersResponse {
  users: AdminUser[]
  total: number
  page: number
  limit: number
}

export interface SecurityEvent {
  id: string
  submission_id?: string
  category: string
  severity: string
  detail_json: Record<string, unknown>
  source_text?: string
  created_at: string
}

export interface SecurityEventsResponse {
  events: SecurityEvent[]
  total: number
  page: number
  limit: number
}

export type AdminUserSortKey = 'violation_count' | 'submissions' | 'handle' | 'created_at'

export function listAdminUsers(params: {
  q?: string
  sort?: AdminUserSortKey
  page?: number
  limit?: number
}): Promise<AdminUsersResponse> {
  const search = new URLSearchParams()
  if (params.q) search.set('q', params.q)
  if (params.sort) search.set('sort', params.sort)
  if (params.page) search.set('page', String(params.page))
  if (params.limit) search.set('limit', String(params.limit))
  const qs = search.toString()
  return apiGet<AdminUsersResponse>(`/internal/admin/users${qs ? `?${qs}` : ''}`)
}

export function getUserSecurityEvents(
  userID: string,
  params: { page?: number; limit?: number } = {},
): Promise<SecurityEventsResponse> {
  const search = new URLSearchParams()
  if (params.page) search.set('page', String(params.page))
  if (params.limit) search.set('limit', String(params.limit))
  const qs = search.toString()
  return apiGet<SecurityEventsResponse>(
    `/internal/admin/users/${userID}/security-events${qs ? `?${qs}` : ''}`,
  )
}
