import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { DeploymentRecord } from '@/types/models'

export function list(appId?: string, status?: string, params?: PaginationParams) {
  return api.get<PaginatedResponse<DeploymentRecord[]>>('/deployments', { params: { app_id: appId, status, ...params } })
}

export function get(id: string) {
  return api.get<ApiResponse<DeploymentRecord>>(`/deployments/${id}`)
}
