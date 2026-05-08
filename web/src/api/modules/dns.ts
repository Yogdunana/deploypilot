import api from '@/api'
import type { ApiResponse, PaginatedResponse, PaginationParams } from '@/types/api'
import type { DNSRecord } from '@/types/models'

export function listRecords(domain: string, params?: PaginationParams) {
  return api.get<PaginatedResponse<DNSRecord[]>>(`/dns/records`, { params: { domain, ...params } })
}

export function createRecord(data: { domain: string; subdomain: string; type: string; value: string; ttl?: number; provider_id?: number }) {
  return api.post<ApiResponse<DNSRecord>>('/dns/records', data)
}

export function updateRecord(id: number, data: Partial<DNSRecord>) {
  return api.put<ApiResponse<DNSRecord>>(`/dns/records/${id}`, data)
}

export function deleteRecord(id: number) {
  return api.delete<ApiResponse<void>>(`/dns/records/${id}`)
}
