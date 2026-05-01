import { apiGet } from './api'

export interface Permission {
  id: string
  key: string
  description?: string
  created_at?: string
  updated_at?: string
}

export function getMyPermissions(): Promise<Permission[]> {
  return apiGet<Permission[]>('/auth/me/permissions')
}

export function canManageProblems(perms: Pick<Permission, 'key'>[]): boolean {
  const keys = new Set(perms.map((p) => p.key))
  return keys.has('admin.access') && keys.has('problems.manage')
}
