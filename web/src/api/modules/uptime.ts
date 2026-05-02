import api from '@/api'
import type { ApiResponse } from '@/types/api'
import type {
  UptimeMonitor,
  MonitorCheckResult,
  HeartbeatMonitor,
  MonitorSLA,
  StatusPageData,
} from '@/types/monitor'

// ── Uptime Monitors ──────────────────────────────────────────────

export function createMonitor(data: Partial<UptimeMonitor>) {
  return api.post<ApiResponse<UptimeMonitor>>('/api/v1/monitors', data)
}

export function listMonitors() {
  return api.get<ApiResponse<UptimeMonitor[]>>('/api/v1/monitors')
}

export function getMonitor(id: string) {
  return api.get<ApiResponse<UptimeMonitor>>(`/api/v1/monitors/${id}`)
}

export function updateMonitor(id: string, data: Partial<UptimeMonitor>) {
  return api.put<ApiResponse<UptimeMonitor>>(`/api/v1/monitors/${id}`, data)
}

export function deleteMonitor(id: string) {
  return api.delete<ApiResponse<void>>(`/api/v1/monitors/${id}`)
}

export function checkMonitor(id: string) {
  return api.post<ApiResponse<MonitorCheckResult>>(`/api/v1/monitors/${id}/check`)
}

export function checkAllMonitors() {
  return api.post<ApiResponse<void>>('/api/v1/monitors/check-all')
}

export function getMonitorResults(id: string, limit?: number) {
  return api.get<ApiResponse<MonitorCheckResult[]>>(`/api/v1/monitors/${id}/results`, {
    params: limit ? { limit } : undefined,
  })
}

export function getMonitorSLA(id: string, days?: number) {
  return api.get<ApiResponse<MonitorSLA>>(`/api/v1/monitors/${id}/sla`, {
    params: days ? { days } : undefined,
  })
}

// ── Status Page (public) ─────────────────────────────────────────

export function getStatusPage() {
  return api.get<ApiResponse<StatusPageData>>('/api/v1/status')
}

// ── Heartbeats ───────────────────────────────────────────────────

export function createHeartbeat(data: Partial<HeartbeatMonitor>) {
  return api.post<ApiResponse<HeartbeatMonitor>>('/api/v1/heartbeats', data)
}

export function listHeartbeats() {
  return api.get<ApiResponse<HeartbeatMonitor[]>>('/api/v1/heartbeats')
}

export function deleteHeartbeat(id: string) {
  return api.delete<ApiResponse<void>>(`/api/v1/heartbeats/${id}`)
}
