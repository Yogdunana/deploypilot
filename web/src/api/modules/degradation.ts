import api from '@/api'

export function getDegradationStatus() {
  return api.get('/degradation/status')
}

export function getDegradationAudits(limit?: number) {
  return api.get('/degradation/audits', { params: { limit } })
}

export function getExportSummary() {
  return api.get('/degradation/export-summary')
}
