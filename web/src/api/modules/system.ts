import api from '@/api'
import type { ApiResponse } from '@/types/api'

export function getVersion() {
  return api.get<ApiResponse<{ version: string; build_time: string; git_commit: string }>>('/api/system/version')
}

export function getHealth() {
  return api.get<ApiResponse<{ status: string; uptime: number; components: Record<string, string> }>>('/api/system/health')
}

export function checkUpdate() {
  return api.get<ApiResponse<{ current: string; latest: string; has_update: boolean }>>('/api/system/update')
}
