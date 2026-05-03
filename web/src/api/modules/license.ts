import api from '@/api'
import type { ApiResponse } from '@/types/api'
import type { LicenseInfo } from '@/types/models'

export function getLicenseStatus() {
  return api.get<ApiResponse<LicenseInfo>>('/api/v1/license/status')
}

export function activateLicense(data: { license_key?: string; use_type?: string; agree_terms?: boolean }) {
  return api.post<ApiResponse<LicenseInfo>>('/api/v1/license/activate', data)
}

export function deactivateLicense() {
  return api.post<ApiResponse<void>>('/api/v1/license/deactivate')
}

export function purchaseAddon(data: { addon_key: string; amount?: number; duration_days?: number }) {
  return api.post<ApiResponse<LicenseInfo>>('/api/v1/license/addon', data)
}

export function listLicenses() {
  return api.get<ApiResponse<any[]>>('/api/v1/license/list')
}

export function issueLicense(data: any) {
  return api.post<ApiResponse<any>>('/api/v1/license/issue', data)
}

export function revokeLicense(id: string, reason: string) {
  return api.post<ApiResponse<void>>(`/api/v1/license/${id}/revoke`, { reason })
}
