import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { Template } from '@/types/models'

export function list(params?: PaginationParams) {
  return api.get<PaginatedResponse<Template[]>>('/api/templates', { params })
}

export function create(data: { name: string; description: string; tech_stack: string; deploy_mode: string; config: Record<string, any> }) {
  return api.post<ApiResponse<Template>>('/api/templates', data)
}

export function update(id: number, data: Partial<Template>) {
  return api.put<ApiResponse<Template>>(`/api/templates/${id}`, data)
}

export function deleteTemplate(id: number) {
  return api.delete<ApiResponse<void>>(`/api/templates/${id}`)
}
