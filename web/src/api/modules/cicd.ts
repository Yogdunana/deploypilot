import api from '@/api'
import type { ApiResponse } from '@/types/api'
import type { CICDBuild } from '@/types/models'

export function triggerBuild(data: { provider: string; repo_url: string; branch: string; app_name?: string }) {
  return api.post<ApiResponse<CICDBuild>>('/cicd/builds', data)
}

export function getBuildStatus(runId: string, provider: string) {
  return api.get<ApiResponse<CICDBuild>>(`/cicd/builds/${provider}/${runId}`)
}
