import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { Notification } from '@/types/models'

export function list(params?: PaginationParams) {
  return api.get<PaginatedResponse<Notification[]>>('/notifications', { params })
}

export function create(data: { type: string; name: string; config: Record<string, string>; enabled?: boolean }) {
  return api.post<ApiResponse<Notification>>('/notifications', data)
}

export function update(id: number, data: Partial<Notification>) {
  return api.put<ApiResponse<Notification>>(`/notifications/${id}`, data)
}

export function deleteNotification(id: number) {
  return api.delete<ApiResponse<void>>(`/notifications/${id}`)
}
