import api from '@/api'

export function getDegradationStatus() {
  return api.get('/api/v1/degradation/status')
}

export function getDegradationAudits(limit?: number) {
  return api.get('/api/v1/degradation/audits', { params: { limit } })
}

export function getExportSummary() {
  return api.get('/api/v1/degradation/export-summary')
}
