import api from '@/api'
import type { AuditLog } from '@/types/models'

export interface AuditQuery {
  user_id?: string
  action?: string
  resource_type?: string
  start_time?: string
  end_time?: string
  page?: number
  page_size?: number
}

export function list(params?: AuditQuery) {
  return api.get<AuditLog[]>('/audit', { params })
}
