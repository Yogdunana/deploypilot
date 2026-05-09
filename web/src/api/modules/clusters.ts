import api from '@/api'

export function listClusters(tenantId?: string) {
  return api.get('/clusters', { params: { tenant_id: tenantId } })
}

export function getCluster(id: string) {
  return api.get(`/clusters/${id}`)
}

export function createCluster(data: Record<string, unknown>) {
  return api.post('/clusters', data)
}

export function updateCluster(id: string, data: Record<string, unknown>) {
  return api.put(`/clusters/${id}`, data)
}

export function deleteCluster(id: string) {
  return api.delete(`/clusters/${id}`)
}

export function testClusterConnection(id: string) {
  return api.post(`/clusters/${id}/test`)
}
