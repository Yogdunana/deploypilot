import api from '@/api'
import type { ApiResponse, LoginRequest, LoginResponse, RegisterRequest } from '@/types/api'

export function login(data: LoginRequest) {
  return api.post<ApiResponse<LoginResponse>>('/api/auth/login', data)
}

export function register(data: RegisterRequest) {
  return api.post<ApiResponse<LoginResponse>>('/api/auth/register', data)
}
