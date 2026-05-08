import api from '@/api'

export function getFeatureFlags() {
  return api.get('/feature-flags')
}

export function getFeatureFlag(key: string) {
  return api.get(`/feature-flags/${key}`)
}

export function updateFeatureFlag(key: string, data: Record<string, unknown>) {
  return api.put(`/feature-flags/${key}`, data)
}

export function setFeatureFlagOverride(key: string, data: { tenant_id: string; enabled: boolean; reason?: string }) {
  return api.post(`/feature-flags/${key}/override`, data)
}

export function deleteFeatureFlagOverride(key: string, tenantId: string) {
  return api.delete(`/feature-flags/${key}/override`, { params: { tenant_id: tenantId } })
}

export function getFeatureFlagOverrides(key: string) {
  return api.get(`/feature-flags/${key}/overrides`)
}

export function getFeatureFlagsForTenant(tenantId: string) {
  return api.get(`/feature-flags/tenant/${tenantId}`)
}
