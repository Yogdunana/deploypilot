import api from '@/api'
import type { ApiResponse } from '@/types/api'

export interface PluginInfo {
  name: string
  version: string
  description: string
  status: 'registered' | 'initialized' | 'running' | 'stopped' | 'error'
  enabled: boolean
  error?: string
  config?: Record<string, unknown>
}

export function listPlugins() {
  return api.get<ApiResponse<PluginInfo[]>>('/event-plugins')
}

export function getPlugin(name: string) {
  return api.get<ApiResponse<PluginInfo>>(`/event-plugins/${name}`)
}

export function updatePlugin(name: string, data: { enabled?: boolean; config?: Record<string, unknown> }) {
  return api.put<ApiResponse<PluginInfo>>(`/event-plugins/${name}`, data)
}

export function startPlugin(name: string) {
  return api.post<ApiResponse<void>>(`/event-plugins/${name}/start`)
}

export function stopPlugin(name: string) {
  return api.post<ApiResponse<void>>(`/event-plugins/${name}/stop`)
}
