import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'

export interface Cluster {
  id: number
  name: string
  provider: string
  region: string
  server_count: number
  status: string
  created_at: string
  updated_at: string
}

export function list(params?: PaginationParams) {
  return api.get<PaginatedResponse<Cluster[]>>('/api/clusters', { params })
}

export function get(id: number) {
  return api.get<ApiResponse<Cluster>>(`/api/clusters/${id}`)
}

export function create(data: Partial<Cluster>) {
  return api.post<ApiResponse<Cluster>>('/api/clusters', data)
}

export function update(id: number, data: Partial<Cluster>) {
  return api.put<ApiResponse<Cluster>>(`/api/clusters/${id}`, data)
}

export function remove(id: number) {
  return api.delete<ApiResponse<void>>(`/api/clusters/${id}`)
}
