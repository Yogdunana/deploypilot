import api from '@/api'
import type { ApiResponse } from '@/types/api'

export interface OAuth2Client {
  id: string
  name: string
  client_id: string
  redirect_uris: string[]
  scopes: string[]
  grant_types: string[]
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface OAuth2TokenResponse {
  access_token: string
  token_type: string
  expires_in: number
  scope: string
  refresh_token?: string
}

export function listOAuth2Clients() {
  return api.get<ApiResponse<OAuth2Client[]>>('/oauth/clients')
}

export function createOAuth2Client(data: { name: string; redirect_uris?: string[]; scopes?: string[]; grant_types?: string[] }) {
  return api.post<ApiResponse<OAuth2Client & { client_secret: string }>>('/oauth/clients', data)
}

export function getOAuth2Client(id: string) {
  return api.get<ApiResponse<OAuth2Client>>(`/oauth/clients/${id}`)
}

export function updateOAuth2Client(id: string, data: Partial<OAuth2Client>) {
  return api.put<ApiResponse<OAuth2Client>>(`/oauth/clients/${id}`, data)
}

export function deleteOAuth2Client(id: string) {
  return api.delete<ApiResponse<void>>(`/oauth/clients/${id}`)
}

export function regenerateOAuth2Secret(id: string) {
  return api.post<ApiResponse<{ client_secret: string }>>(`/oauth/clients/${id}/secret`)
}
