import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { SystemMetrics, ContainerMetrics, Alert, AlertRule } from '@/types/models'

export function getSystemMetrics() {
  return api.get<ApiResponse<SystemMetrics>>('/api/monitor/metrics')
}

export function getContainerMetrics(name: string) {
  return api.get<ApiResponse<ContainerMetrics>>(`/api/monitor/containers/${name}/metrics`)
}

export function listAlerts(params?: PaginationParams) {
  return api.get<PaginatedResponse<Alert[]>>('/api/monitor/alerts', { params })
}

export function listAlertRules(params?: PaginationParams) {
  return api.get<PaginatedResponse<AlertRule[]>>('/api/monitor/alert-rules', { params })
}

export function heal(name: string) {
  return api.post<ApiResponse<void>>(`/api/monitor/containers/${name}/heal`)
}

export function check(name: string) {
  return api.post<ApiResponse<{ healthy: boolean; message: string }>>(`/api/monitor/containers/${name}/check`)
}
