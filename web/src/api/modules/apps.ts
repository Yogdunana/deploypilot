import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams, DeployConfig, BuildConfig, RollbackConfig, RestoreConfig } from '@/types/api'
import type { App, DeploymentRecord } from '@/types/models'

export function list(params?: PaginationParams) {
  return api.get<PaginatedResponse<App[]>>('/apps', { params })
}

export function get(id: string) {
  return api.get<ApiResponse<App>>(`/apps/${id}`)
}

export function create(data: Partial<App>) {
  return api.post<ApiResponse<App>>('/apps', data)
}

export function update(id: string, data: Partial<App>) {
  return api.put<ApiResponse<App>>(`/apps/${id}`, data)
}

export function deleteApp(id: string) {
  return api.delete<ApiResponse<void>>(`/apps/${id}`)
}

export function deploy(id: string, config?: DeployConfig, async = true) {
  return api.post<ApiResponse<DeploymentRecord>>(`/apps/${id}/deploy`, { ...config, async })
}

export function build(id: string, data: BuildConfig) {
  return api.post<ApiResponse<any>>(`/apps/${id}/build`, data)
}

export function getStatus(id: string) {
  return api.get<ApiResponse<any>>(`/apps/${id}/status`)
}

export function rollback(id: string, data: RollbackConfig) {
  return api.post<ApiResponse<DeploymentRecord>>(`/apps/${id}/rollback`, data)
}

export function getLogs(id: string, tail?: number) {
  return api.get<ApiResponse<string>>(`/apps/${id}/logs/container`, { params: { tail } })
}

export function backup(id: string) {
  return api.post<ApiResponse<any>>(`/apps/${id}/backup`)
}

export function listBackups(id: string) {
  return api.get<ApiResponse<any[]>>(`/apps/${id}/backups`)
}

export function restore(id: string, data: RestoreConfig) {
  return api.post<ApiResponse<void>>(`/apps/${id}/restore`, data)
}

export function deleteBackup(id: string, backupId: string) {
  return api.delete<ApiResponse<void>>(`/apps/${id}/backups/${backupId}`)
}

export function getEnv(id: string) {
  return api.get<ApiResponse<Record<string, string>>>(`/apps/${id}/env`)
}

export function updateEnv(id: string, data: Record<string, string>) {
  return api.put<ApiResponse<Record<string, string>>>(`/apps/${id}/env`, data)
}
