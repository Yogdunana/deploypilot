import api from '@/api'
import type { PaginatedResponse, PaginationParams } from '@/types/api'
import type { AuditLog } from '@/types/models'

export interface AuditLogParams extends PaginationParams {
  user_id?: number
  action?: string
  resource_type?: string
  start_date?: string
  end_date?: string
}

export function listLogs(params?: AuditLogParams) {
  return api.get<PaginatedResponse<AuditLog[]>>('/api/audit/logs', { params })
}
