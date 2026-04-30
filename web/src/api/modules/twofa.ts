import api from '@/api'
import type { ApiResponse } from '@/types/api'
import type { TwoFASetupResponse } from '@/types/models'
import type { TwoFAVerifyRequest, TwoFACodeRequest } from '@/types/api'

export function setup() {
  return api.post<ApiResponse<TwoFASetupResponse>>('/api/v1/2fa/setup')
}

export function confirm(data: TwoFACodeRequest) {
  return api.post<ApiResponse<{ enabled: boolean }>>('/api/v1/2fa/confirm', data)
}

export function disable(data: TwoFACodeRequest) {
  return api.post<ApiResponse<{ enabled: boolean }>>('/api/v1/2fa/disable', data)
}

export function verify(data: TwoFAVerifyRequest) {
  return api.post<ApiResponse<{ user: any; token: string }>>('/api/v1/auth/2fa/verify', data)
}
