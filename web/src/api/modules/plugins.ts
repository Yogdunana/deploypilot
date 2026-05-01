import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'

export interface Plugin {
  id: number
  name: string
  display_name: string
  version: string
  description: string
  author: string
  status: 'installed' | 'available' | 'error'
  enabled: boolean
  config: Record<string, any>
  created_at: string
  updated_at: string
}

export function list(params?: PaginationParams) {
  return api.get<PaginatedResponse<Plugin[]>>('/api/plugins', { params })
}

export function get(id: number) {
  return api.get<ApiResponse<Plugin>>(`/api/plugins/${id}`)
}

export function install(id: number) {
  return api.post<ApiResponse<Plugin>>(`/api/plugins/${id}/install`)
}

export function uninstall(id: number) {
  return api.post<ApiResponse<void>>(`/api/plugins/${id}/uninstall`)
}

export function toggle(id: number, enabled: boolean) {
  return api.put<ApiResponse<Plugin>>(`/api/plugins/${id}`, { enabled })
}

export function updateConfig(id: number, config: Record<string, any>) {
  return api.put<ApiResponse<Plugin>>(`/api/plugins/${id}/config`, { config })
}
