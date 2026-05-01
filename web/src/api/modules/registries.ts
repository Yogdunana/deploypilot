import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'

export interface Registry {
  id: number
  name: string
  provider: string
  url: string
  username: string
  is_connected: boolean
  image_count: number
  created_at: string
  updated_at: string
}

export function list(params?: PaginationParams) {
  return api.get<PaginatedResponse<Registry[]>>('/api/registries', { params })
}

export function get(id: number) {
  return api.get<ApiResponse<Registry>>(`/api/registries/${id}`)
}

export function create(data: Partial<Registry>) {
  return api.post<ApiResponse<Registry>>('/api/registries', data)
}

export function update(id: number, data: Partial<Registry>) {
  return api.put<ApiResponse<Registry>>(`/api/registries/${id}`, data)
}

export function remove(id: number) {
  return api.delete<ApiResponse<void>>(`/api/registries/${id}`)
}

export function testConnection(id: number) {
  return api.post<ApiResponse<{ success: boolean; message: string }>>(`/api/registries/${id}/test`)
}
