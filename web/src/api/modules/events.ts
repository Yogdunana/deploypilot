import api from '@/api'

export function listEvents(params?: { event_type?: string; page?: number; page_size?: number }) {
  return api.get('/events', { params })
}

export function getEventStats() {
  return api.get('/events/stats')
}
