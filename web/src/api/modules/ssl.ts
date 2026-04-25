import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { SSLCertificate } from '@/types/models'

export function listCertificates(params?: PaginationParams) {
  return api.get<PaginatedResponse<SSLCertificate[]>>('/api/ssl/certificates', { params })
}

export function requestCertificate(data: { domain: string; email: string; provider?: string; auto_renew?: boolean }) {
  return api.post<ApiResponse<SSLCertificate>>('/api/ssl/certificates', data)
}

export function deleteCertificate(id: number) {
  return api.delete<ApiResponse<void>>(`/api/ssl/certificates/${id}`)
}

export function renewCertificate(id: number) {
  return api.post<ApiResponse<SSLCertificate>>(`/api/ssl/certificates/${id}/renew`)
}
