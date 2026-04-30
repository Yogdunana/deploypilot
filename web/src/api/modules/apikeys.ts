import api from '@/api'
import type { ApiResponse } from '@/types/api'
import type { ApiKey } from '@/types/models'
import type { CreateApiKeyRequest } from '@/types/api'

export function list() {
  return api.get<ApiResponse<ApiKey[]>>('/api/v1/api-keys')
}

export function create(data: CreateApiKeyRequest) {
  return api.post<ApiResponse<{ id: string; name: string; key: string; key_prefix: string; scopes: string[]; expires_at?: string; created_at: string }>>('/api/v1/api-keys', data)
}

export function remove(id: string) {
  return api.delete<ApiResponse<{ message: string; id: string }>>(`/api/v1/api-keys/${id}`)
}
