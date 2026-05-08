import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'

export interface ActivityItem {
  id: number
  type: string
  action: string
  resource_type: string
  resource_id: string
  resource_name: string
  username: string
  detail: string
  ip_address: string
  created_at: string
}

export function list(params?: PaginationParams & { type?: string }) {
  return api.get<PaginatedResponse<ActivityItem[]>>('/activity', { params })
}

export function getStats() {
  return api.get<ApiResponse<{
    today_count: number
    week_count: number
    by_type: Record<string, number>
  }>>('/activity/stats')
}
