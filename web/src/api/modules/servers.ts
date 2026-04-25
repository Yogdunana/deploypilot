import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { Server } from '@/types/models'

export function list(params?: PaginationParams) {
  return api.get<PaginatedResponse<Server[]>>('/api/servers', { params })
}

export function create(data: Partial<Server>) {
  return api.post<ApiResponse<Server>>('/api/servers', data)
}

export function update(id: number, data: Partial<Server>) {
  return api.put<ApiResponse<Server>>(`/api/servers/${id}`, data)
}

export function deleteServer(id: number) {
  return api.delete<ApiResponse<void>>(`/api/servers/${id}`)
}

export function test(id: number) {
  return api.post<ApiResponse<{ success: boolean; message: string }>>(`/api/servers/${id}/test`)
}

export function detect(id: number, data: { host: string; port?: number }) {
  return api.post<ApiResponse<Server['detected_info']>>(`/api/servers/${id}/detect`, data)
}

export function getEnvironment(id: number) {
  return api.get<ApiResponse<Server['detected_info']>>(`/api/servers/${id}/environment`)
}
