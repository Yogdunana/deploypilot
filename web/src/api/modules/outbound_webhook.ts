import api from '@/api'
import type { ApiResponse } from '@/types/api'

export interface OutboundWebhook {
  id: string
  name: string
  url: string
  format: 'json' | 'slack' | 'discord' | 'teams'
  event_types: string[]
  severity_filter: string[]
  app_filter: string[]
  server_filter: string[]
  enabled: boolean
  max_retries: number
  timeout: number
  description: string
  last_delivery_at: string | null
  last_status: string
  created_at: string
  updated_at: string
}

export interface WebhookDelivery {
  id: string
  webhook_id: string
  event_id: string
  event_type: string
  status_code: number
  latency_ms: number
  attempt: number
  success: boolean
  error_response: string
  request_body: string
  created_at: string
}

export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
}

export function listWebhooks(page = 1, pageSize = 20) {
  return api.get<ApiResponse<PaginatedResponse<OutboundWebhook>>>('/api/v1/webhooks', {
    params: { page, page_size: pageSize },
  })
}

export function getWebhook(id: string) {
  return api.get<ApiResponse<OutboundWebhook>>(`/api/v1/webhooks/${id}`)
}

export function createWebhook(data: Partial<OutboundWebhook>) {
  return api.post<ApiResponse<OutboundWebhook>>('/api/v1/webhooks', data)
}

export function updateWebhook(id: string, data: Partial<OutboundWebhook>) {
  return api.put<ApiResponse<OutboundWebhook>>(`/api/v1/webhooks/${id}`, data)
}

export function deleteWebhook(id: string) {
  return api.delete(`/api/v1/webhooks/${id}`)
}

export function testWebhook(id: string) {
  return api.post<ApiResponse<WebhookDelivery>>(`/api/v1/webhooks/${id}/test`)
}

export function listDeliveries(webhookId: string, page = 1, pageSize = 20) {
  return api.get<ApiResponse<PaginatedResponse<WebhookDelivery>>>(`/api/v1/webhooks/${webhookId}/deliveries`, {
    params: { page, page_size: pageSize },
  })
}

export function getDelivery(webhookId: string, deliveryId: string) {
  return api.get<ApiResponse<WebhookDelivery>>(`/api/v1/webhooks/${webhookId}/deliveries/${deliveryId}`)
}
