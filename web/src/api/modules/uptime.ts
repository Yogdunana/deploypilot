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
  return api.post<ApiResponse<UptimeMonitor>>('/monitors', data)
}

export function listMonitors() {
  return api.get<ApiResponse<UptimeMonitor[]>>('/monitors')
}

export function getMonitor(id: string) {
  return api.get<ApiResponse<UptimeMonitor>>(`/monitors/${id}`)
}

export function updateMonitor(id: string, data: Partial<UptimeMonitor>) {
  return api.put<ApiResponse<UptimeMonitor>>(`/monitors/${id}`, data)
}

export function deleteMonitor(id: string) {
  return api.delete<ApiResponse<void>>(`/monitors/${id}`)
}

export function checkMonitor(id: string) {
  return api.post<ApiResponse<MonitorCheckResult>>(`/monitors/${id}/check`)
}

export function checkAllMonitors() {
  return api.post<ApiResponse<void>>('/monitors/check-all')
}

export function getMonitorResults(id: string, limit?: number) {
  return api.get<ApiResponse<MonitorCheckResult[]>>(`/monitors/${id}/results`, {
    params: limit ? { limit } : undefined,
  })
}

export function getMonitorSLA(id: string, days?: number) {
  return api.get<ApiResponse<MonitorSLA>>(`/monitors/${id}/sla`, {
    params: days ? { days } : undefined,
  })
}

// ── Status Page (public) ─────────────────────────────────────────

export function getStatusPage() {
  return api.get<ApiResponse<StatusPageData>>('/status')
}

// ── Heartbeats ───────────────────────────────────────────────────

export function createHeartbeat(data: Partial<HeartbeatMonitor>) {
  return api.post<ApiResponse<HeartbeatMonitor>>('/heartbeats', data)
}

export function listHeartbeats() {
  return api.get<ApiResponse<HeartbeatMonitor[]>>('/heartbeats')
}

export function deleteHeartbeat(id: string) {
  return api.delete<ApiResponse<void>>(`/heartbeats/${id}`)
}
