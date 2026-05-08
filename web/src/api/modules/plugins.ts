import api from '@/api'

export function listPlugins(tenantId?: string, provider?: string) {
  return api.get('/plugins', { params: { tenant_id: tenantId, provider } })
}

export function getPlugin(id: string) {
  return api.get(`/plugins/${id}`)
}

export function createPlugin(data: Record<string, unknown>) {
  return api.post('/plugins', data)
}

export function updatePlugin(id: string, data: Record<string, unknown>) {
  return api.put(`/plugins/${id}`, data)
}

export function deletePlugin(id: string) {
  return api.delete(`/plugins/${id}`)
}

export function enablePlugin(id: string) {
  return api.post(`/plugins/${id}/enable`)
}

export function disablePlugin(id: string) {
  return api.post(`/plugins/${id}/disable`)
}

export function reloadPlugin(id: string) {
  return api.post(`/plugins/${id}/reload`)
}
