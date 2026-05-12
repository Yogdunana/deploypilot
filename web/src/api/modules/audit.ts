import api from '@/api'
import type { AuditLog } from '@/types/models'
import type { ApiResponse, PaginatedResponse } from '@/types/api'

export interface AuditLogParams {
  page?: number
  page_size?: number
  username?: string
  action?: string
  resource_type?: string
  start_date?: string
  end_date?: string
}

export function list(params?: AuditLogParams) {
  return api.get<ApiResponse<PaginatedResponse<AuditLog>>>('/audit', { params })
}

// Alias for backward compatibility
export const listLogs = list
