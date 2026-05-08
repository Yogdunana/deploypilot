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
  return api.get<ApiResponse<any>>('/monitors/overview', {
    params: period ? { period } : undefined,
  })
}

export function getMonitorHistory(id: string, params?: MonitorHistoryParams) {
  return api.get<ApiResponse<any>>(`/monitors/${id}/history`, { params })
}

export function listAlertHistory(params?: AlertHistoryParams) {
  return api.get<ApiResponse<any>>('/alerts/history', { params })
}

export function getAlertHistory(id: string) {
  return api.get<ApiResponse<any>>(`/alerts/history/${id}`)
}

export function getAlertStats(period?: string) {
  return api.get<ApiResponse<any>>('/alerts/stats', {
    params: period ? { period } : undefined,
  })
}

export function exportMonitorData(id: string, format: string = 'csv', params?: MonitorHistoryParams) {
  return api.get(`/monitors/${id}/export`, {
    params: { format, ...params },
    responseType: 'blob',
  })
}

export function exportAlertHistory(format: string = 'csv', params?: AlertHistoryParams) {
  return api.get('/alerts/export', {
    params: { format, ...params },
    responseType: 'blob',
  })
}
