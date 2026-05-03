import api from '@/api'

export function listClusters(tenantId?: string) {
  return api.get('/api/v1/clusters', { params: { tenant_id: tenantId } })
}

export function getCluster(id: string) {
  return api.get(`/api/v1/clusters/${id}`)
}

export function createCluster(data: Record<string, unknown>) {
  return api.post('/api/v1/clusters', data)
}

export function updateCluster(id: string, data: Record<string, unknown>) {
  return api.put(`/api/v1/clusters/${id}`, data)
}

export function deleteCluster(id: string) {
  return api.delete(`/api/v1/clusters/${id}`)
}

export function testClusterConnection(id: string) {
  return api.post(`/api/v1/clusters/${id}/test`)
}
