import api from '@/api'

export function listEvents(params?: { event_type?: string; page?: number; page_size?: number }) {
  return api.get('/api/v1/events', { params })
}

export function getEventStats() {
  return api.get('/api/v1/events/stats')
}
