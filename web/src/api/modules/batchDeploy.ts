import api from '@/api'

export function batchDeploy(data: {
  apps: Record<string, unknown>[]
  strategy?: 'sequential' | 'parallel' | 'rolling'
  max_concurrent?: number
  batch_size?: number
  server_ids?: string[]
}) {
  return api.post('/batch-deploy', data)
}
