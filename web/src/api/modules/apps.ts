import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams, DeployConfig, BuildConfig, RollbackConfig, RestoreConfig } from '@/types/api'
import type { App, DeploymentRecord } from '@/types/models'

export function list(params?: PaginationParams) {
  return api.get<PaginatedResponse<App[]>>('/apps', { params })
}

export function get(id: number) {
  return api.get<ApiResponse<App>>(`/apps/${id}`)
}

export function create(data: Partial<App>) {
  return api.post<ApiResponse<App>>('/apps', data)
}

export function update(id: number, data: Partial<App>) {
  return api.put<ApiResponse<App>>(`/apps/${id}`, data)
}

export function deleteApp(id: number) {
  return api.delete<ApiResponse<void>>(`/apps/${id}`)
}

export function deploy(id: number, config?: DeployConfig, async = true) {
  return api.post<ApiResponse<DeploymentRecord>>(`/apps/${id}/deploy`, { ...config, async })
}

export function build(id: number, data: BuildConfig) {
  return api.post<ApiResponse<any>>(`/apps/${id}/build`, data)
}

export function getStatus(id: number) {
  return api.get<ApiResponse<any>>(`/apps/${id}/status`)
}

export function rollback(id: number, data: RollbackConfig) {
  return api.post<ApiResponse<DeploymentRecord>>(`/apps/${id}/rollback`, data)
}

export function getLogs(id: number, tail?: number) {
  return api.get<ApiResponse<string>>(`/apps/${id}/logs`, { params: { tail } })
}

export function backup(id: number) {
  return api.post<ApiResponse<any>>(`/apps/${id}/backup`)
}

export function listBackups(id: number) {
  return api.get<ApiResponse<any[]>>(`/apps/${id}/backups`)
}

export function restore(id: number, data: RestoreConfig) {
  return api.post<ApiResponse<void>>(`/apps/${id}/restore`, data)
}

export function deleteBackup(id: number, backupId: string) {
  return api.delete<ApiResponse<void>>(`/apps/${id}/backups/${backupId}`)
}

export function getEnv(id: number) {
  return api.get<ApiResponse<Record<string, string>>>(`/apps/${id}/env`)
}

export function updateEnv(id: number, data: Record<string, string>) {
  return api.put<ApiResponse<Record<string, string>>>(`/apps/${id}/env`, data)
}
