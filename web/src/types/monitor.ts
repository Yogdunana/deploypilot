export interface UptimeMonitor {
  id: string
  tenant_id?: string
  name: string
  type: 'http' | 'tcp' | 'ping'
  target: string
  interval: number
  timeout: number
  status: 'up' | 'down' | 'pending'
  enabled: boolean
  last_check?: string
  last_status?: string
  uptime: number
  total_checks: number
  up_checks: number
  avg_latency: number
  created_at: string
  updated_at: string
}

export interface MonitorCheckResult {
  id: string
  monitor_id: string
  status: 'up' | 'down'
  status_code?: number
  latency: number
  message: string
  created_at: string
}

export interface HeartbeatMonitor {
  id: string
  tenant_id?: string
  name: string
  token: string
  interval: number
  timeout: number
  status: 'up' | 'down'
  last_beat?: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface MonitorSLA {
  monitor_id: string
  uptime_pct: number
  avg_latency: number
  total_checks: number
  up_checks: number
  period_days: number
}

export interface StatusPageData {
  monitors: Array<{
    id: string
    name: string
    type: string
    status: 'up' | 'down'
    uptime: number
    avg_latency: number
  }>
  overall_uptime: number
  total_monitors: number
  up_monitors: number
}
