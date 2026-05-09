import api from '@/api'
import type { ApiResponse } from '@/types/api'

export function getVersion() {
  return api.get<ApiResponse<{ version: string; build_time: string; git_commit: string; go: string; goos: string; goarch: string; num_cpu: number }>>('/system/version')
}

export function getHealth() {
  return api.get<ApiResponse<{ status: string; database: { status: string } }>>('/system/health')
}

export function checkUpdate() {
  return api.get<ApiResponse<{ current_version: string; latest_version: string; update_available: boolean; message: string; release_notes?: string; published_at?: string }>>('/system/update/check')
}

export function performUpdate(targetVersion?: string) {
  return api.post<ApiResponse<{ success: boolean; old_version: string; new_version: string; message: string; rollback_path?: string }>>('/system/update/perform', {
    version: targetVersion || 'latest'
  })
}
