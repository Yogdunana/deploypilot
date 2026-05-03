import api from '@/api'

export function listRegistries(tenantId?: string) {
  return api.get('/api/v1/registries', { params: { tenant_id: tenantId } })
}

export function getRegistry(id: string) {
  return api.get(`/api/v1/registries/${id}`)
}

export function createRegistry(data: Record<string, unknown>) {
  return api.post('/api/v1/registries', data)
}

export function updateRegistry(id: string, data: Record<string, unknown>) {
  return api.put(`/api/v1/registries/${id}`, data)
}

export function deleteRegistry(id: string) {
  return api.delete(`/api/v1/registries/${id}`)
}
