import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { Credential } from '@/types/models'

export function list(tenantId?: number, params?: PaginationParams) {
  return api.get<PaginatedResponse<Credential[]>>('/credentials', { params: { tenant_id: tenantId, ...params } })
}

export function create(data: { name: string; type: string; value: string; expires_in_days?: number }) {
  return api.post<ApiResponse<Credential>>('/credentials', data)
}

export function update(id: number, data: { name?: string; type?: string; value?: string }) {
  return api.put<ApiResponse<Credential>>(`/credentials/${id}`, data)
}

export function deleteCredential(id: number) {
  return api.delete<ApiResponse<void>>(`/credentials/${id}`)
}

export function rotate(id: number, data: { value: string }) {
  return api.post<ApiResponse<Credential>>(`/credentials/${id}/rotate`, data)
}
