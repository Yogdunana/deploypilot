import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { User, Role } from '@/types/models'

export function getMe() {
  return api.get<ApiResponse<User>>('/users/me')
}

export function list(params?: PaginationParams) {
  return api.get<PaginatedResponse<User[]>>('/users', { params })
}

export function deleteUser(id: number) {
  return api.delete<ApiResponse<void>>(`/users/${id}`)
}

export function updateRole(id: number, roleId: number) {
  return api.put<ApiResponse<User>>(`/users/${id}/role`, { role_id: roleId })
}

export function getRoles() {
  return api.get<ApiResponse<Role[]>>('/roles')
}
