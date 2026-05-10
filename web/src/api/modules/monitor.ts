import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { SystemMetrics, ContainerMetrics, Alert, AlertRule } from '@/types/models'

export function getSystemMetrics() {
  return api.get<ApiResponse<SystemMetrics>>('/monitor/system')
}

export function getContainerMetrics(name: string) {
  return api.get<ApiResponse<ContainerMetrics>>(`/monitor/container/${name}`)
}

export function listAlerts(params?: PaginationParams) {
  return api.get<PaginatedResponse<Alert[]>>('/monitor/alerts', { params })
}

export function listAlertRules(params?: PaginationParams) {
  return api.get<PaginatedResponse<AlertRule[]>>('/monitor/alert-rules', { params })
}

export function heal(name: string) {
  return api.post<ApiResponse<void>>(`/monitor/heal/${name}`)
}

export function check(name: string) {
  return api.post<ApiResponse<{ healthy: boolean; message: string }>>(`/monitor/check/${name}`)
}
