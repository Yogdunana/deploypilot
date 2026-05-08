import api from '@/api'

export function listRegistries(tenantId?: string) {
  return api.get('/registries', { params: { tenant_id: tenantId } })
}

export function getRegistry(id: string) {
  return api.get(`/registries/${id}`)
}

export function createRegistry(data: Record<string, unknown>) {
  return api.post('/registries', data)
}

export function updateRegistry(id: string, data: Record<string, unknown>) {
  return api.put(`/registries/${id}`, data)
}

export function deleteRegistry(id: string) {
  return api.delete(`/registries/${id}`)
}
