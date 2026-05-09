import api from '@/api'

export function getTrialStatus() {
  return api.get('/trial/status')
}

export function extendTrial(data: { machine_id: string; days: number; reason?: string }) {
  return api.post('/trial/extend', data)
}

export function listTrialPeriods() {
  return api.get('/trial/list')
}
