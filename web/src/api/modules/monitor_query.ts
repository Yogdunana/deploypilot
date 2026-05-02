import api from '@/api'
import type { ApiResponse } from '@/types/api'

export interface MonitorHistoryParams {
  start?: string
  end?: string
  interval?: string
  limit?: number
}

export interface AlertHistoryParams {
  page?: number
  page_size?: number
  status?: string
  severity?: string
  rule_id?: string
}

export function getMonitorOverview(period?: string) {
  return api.get<ApiResponse<any>>('/api/v1/monitors/overview', {
    params: period ? { period } : undefined,
  })
}

export function getMonitorHistory(id: string, params?: MonitorHistoryParams) {
  return api.get<ApiResponse<any>>(`/api/v1/monitors/${id}/history`, { params })
}

export function listAlertHistory(params?: AlertHistoryParams) {
  return api.get<ApiResponse<any>>('/api/v1/alerts/history', { params })
}

export function getAlertHistory(id: string) {
  return api.get<ApiResponse<any>>(`/api/v1/alerts/history/${id}`)
}

export function getAlertStats(period?: string) {
  return api.get<ApiResponse<any>>('/api/v1/alerts/stats', {
    params: period ? { period } : undefined,
  })
}

export function exportMonitorData(id: string, format: string = 'csv', params?: MonitorHistoryParams) {
  return api.get(`/api/v1/monitors/${id}/export`, {
    params: { format, ...params },
    responseType: 'blob',
  })
}

export function exportAlertHistory(format: string = 'csv', params?: AlertHistoryParams) {
  return api.get('/api/v1/alerts/export', {
    params: { format, ...params },
    responseType: 'blob',
  })
}
