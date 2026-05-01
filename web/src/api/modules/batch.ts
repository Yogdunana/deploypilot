import api from '@/api'
import type { ApiResponse } from '@/types/api'

export interface BatchOperation {
  id: number
  operation_type: string
  resource_type: string
  target_ids: number[]
  status: 'pending' | 'running' | 'completed' | 'failed' | 'partial'
  total: number
  completed: number
  failed: number
  error_message: string
  created_at: string
  updated_at: string
}

export function deployApps(data: { app_ids: number[]; branch?: string }) {
  return api.post<ApiResponse<BatchOperation>>('/api/batch/deploy', data)
}

export function restartApps(data: { app_ids: number[] }) {
  return api.post<ApiResponse<BatchOperation>>('/api/batch/restart', data)
}

export function stopApps(data: { app_ids: number[] }) {
  return api.post<ApiResponse<BatchOperation>>('/api/batch/stop', data)
}

export function deleteServers(data: { server_ids: number[] }) {
  return api.post<ApiResponse<BatchOperation>>('/api/batch/delete-servers', data)
}

export function getStatus(id: number) {
  return api.get<ApiResponse<BatchOperation>>(`/api/batch/${id}`)
}

export function listOperations(params?: { page?: number; page_size?: number }) {
  return api.get<ApiResponse<BatchOperation[]>>('/api/batch', { params })
}
