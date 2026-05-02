import api from '@/api'
import type { ApiResponse } from '@/types/api'

export interface MonitorSettings {
  check_interval: number
  timeout: number
  retries: number
  heartbeat_timeout: number
  scheduler_enabled: boolean
  metrics_public: boolean
}

export function getMonitorSettings() {
  return api.get<ApiResponse<MonitorSettings>>('/api/v1/monitor/settings')
}

export function updateMonitorSettings(data: Partial<MonitorSettings>) {
  return api.put<ApiResponse<MonitorSettings>>('/api/v1/monitor/settings', data)
}
