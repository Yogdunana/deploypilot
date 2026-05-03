import api from '@/api'

export function getTrialStatus() {
  return api.get('/api/v1/trial/status')
}

export function extendTrial(data: { machine_id: string; days: number; reason?: string }) {
  return api.post('/api/v1/trial/extend', data)
}

export function listTrialPeriods() {
  return api.get('/api/v1/trial/list')
}
