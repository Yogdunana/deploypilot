import api from '@/api'

export function getFeatureFlags() {
  return api.get('/api/v1/feature-flags')
}

export function getFeatureFlag(key: string) {
  return api.get(`/api/v1/feature-flags/${key}`)
}

export function updateFeatureFlag(key: string, data: Record<string, unknown>) {
  return api.put(`/api/v1/feature-flags/${key}`, data)
}

export function setFeatureFlagOverride(key: string, data: { tenant_id: string; enabled: boolean; reason?: string }) {
  return api.post(`/api/v1/feature-flags/${key}/override`, data)
}

export function deleteFeatureFlagOverride(key: string, tenantId: string) {
  return api.delete(`/api/v1/feature-flags/${key}/override`, { params: { tenant_id: tenantId } })
}

export function getFeatureFlagOverrides(key: string) {
  return api.get(`/api/v1/feature-flags/${key}/overrides`)
}

export function getFeatureFlagsForTenant(tenantId: string) {
  return api.get(`/api/v1/feature-flags/tenant/${tenantId}`)
}
