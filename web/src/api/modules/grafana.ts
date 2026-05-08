import api from '@/api'
import type { ApiResponse } from '@/types/api'

export interface GrafanaStatus {
  connected: boolean
  url: string
  version?: string
  datasource_uid?: string
  last_sync?: string
  annotations_enabled: boolean
}

export interface GrafanaDashboard {
  id: string
  name: string
  uid: string
  description: string
  tags: string
  is_built_in: boolean
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface GrafanaExport {
  datasource: Record<string, unknown>
  dashboards: { name: string; json: Record<string, unknown> }[]
}

export function getGrafanaStatus() {
  return api.get<ApiResponse<GrafanaStatus>>('/grafana/status')
}

export function testGrafanaConnection() {
  return api.post<ApiResponse<{ success: boolean; version: string }>>('/grafana/test')
}

export function syncGrafana() {
  return api.post<ApiResponse<{ synced: number }>>('/grafana/sync')
}

export function listGrafanaDashboards() {
  return api.get<ApiResponse<GrafanaDashboard[]>>('/grafana/dashboards')
}

export function getGrafanaDashboard(id: string) {
  return api.get<ApiResponse<GrafanaDashboard>>(`/grafana/dashboards/${id}`)
}

export function createGrafanaDashboard(data: { name: string; json: string; tags?: string }) {
  return api.post<ApiResponse<GrafanaDashboard>>('/grafana/dashboards', data)
}

export function updateGrafanaDashboard(id: string, data: { name?: string; json?: string; tags?: string; enabled?: boolean }) {
  return api.put<ApiResponse<GrafanaDashboard>>(`/grafana/dashboards/${id}`, data)
}

export function deleteGrafanaDashboard(id: string) {
  return api.delete<ApiResponse<void>>(`/grafana/dashboards/${id}`)
}

export function exportGrafana() {
  return api.get<ApiResponse<GrafanaExport>>('/grafana/export')
}
