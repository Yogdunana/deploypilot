import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { Server } from '@/types/models'

export function list(params?: PaginationParams) {
  return api.get<PaginatedResponse<Server[]>>('/servers', { params })
}

export function create(data: Partial<Server>) {
  return api.post<ApiResponse<Server>>('/servers', data)
}

export function update(id: string, data: Partial<Server>) {
  return api.put<ApiResponse<Server>>(`/servers/${id}`, data)
}

export function deleteServer(id: string) {
  return api.delete<ApiResponse<void>>(`/servers/${id}`)
}

export function test(id: string) {
  return api.post<ApiResponse<{ success: boolean; message: string }>>(`/servers/${id}/test`)
}

export function detect(id: string, data: { host: string; port?: number }) {
  return api.post<ApiResponse<Server['detected_info']>>(`/servers/${id}/detect`, data)
}

export function getEnvironment(id: string) {
  return api.get<ApiResponse<Server['detected_info']>>(`/servers/${id}/environment`)
}
