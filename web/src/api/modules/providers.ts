import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { Provider } from '@/types/models'

export function list(type?: string, params?: PaginationParams) {
  return api.get<PaginatedResponse<Provider[]>>('/api/providers', { params: { type, ...params } })
}

export function create(data: { type: string; name: string; config: Record<string, string>; enabled?: boolean }) {
  return api.post<ApiResponse<Provider>>('/api/providers', data)
}

export function update(id: number, data: Partial<Provider>) {
  return api.put<ApiResponse<Provider>>(`/api/providers/${id}`, data)
}

export function deleteProvider(id: number) {
  return api.delete<ApiResponse<void>>(`/api/providers/${id}`)
}
