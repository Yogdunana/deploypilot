export interface App {
  id: number
  tenant_id: number
  server_id: number
  name: string
  repo_url: string
  branch: string
  domain: string
  tech_stack: string
  deploy_mode: string
  status: string
  current_version: string
  container_name: string
  env_vars: Record<string, string>
  resource_limits: ResourceLimits
  created_at: string
  updated_at: string
}

export interface ResourceLimits {
  memory: string
  cpu: string
}

export interface Server {
  id: number
  tenant_id: number
  credential_id: number
  provider_id: number
  name: string
  host: string
  port: number
  tags: string[]
  status: string
  detected_info: DetectedInfo
  created_at: string
  updated_at: string
}

export interface DetectedInfo {
  os: string
  arch: string
  docker_version: string
  docker_compose_version: string
  kernel_version: string
  memory_total: string
  cpu_cores: number
  disk_total: string
}

export interface Credential {
  id: number
  tenant_id: number
  name: string
  type: string
  created_at: string
  updated_at: string
}

export interface Provider {
  id: number
  tenant_id: number
  type: string
  name: string
  config: Record<string, string>
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface DeploymentRecord {
  id: number
  tenant_id: number
  server_id: number
  app_name: string
  container_name: string
  image: string
  status: string
  preflight_code: number
  preflight_message: string
  preflight_checks: PreflightCheck[]
  error_message: string
  created_at: string
  updated_at: string
}

export interface PreflightCheck {
  name: string
  status: string
  message: string
}

export interface AuditLog {
  id: number
  user_id: number
  username: string
  action: string
  resource_type: string
  resource_id: string
  detail: string
  ip_address: string
  user_agent: string
  created_at: string
}

export interface SSLCertificate {
  id: number
  domain: string
  email: string
  provider: string
  status: string
  cert_path: string
  key_path: string
  issued_at: string
  expires_at: string
  auto_renew: boolean
  last_renewed: string
  retry_count: number
  created_at: string
  updated_at: string
}

export interface User {
  id: number
  tenant_id: number
  role_id: number
  username: string
  email: string
  created_at: string
  updated_at: string
  tenant?: Tenant
  role?: Role
}

export interface Role {
  id: number
  name: string
  permissions: string[]
  created_at: string
}

export interface Tenant {
  id: number
  name: string
  created_at: string
  updated_at: string
}

export interface DNSRecord {
  id: number
  domain: string
  subdomain: string
  type: string
  value: string
  ttl: number
  provider_id: number
  created_at: string
  updated_at: string
}

export interface Notification {
  id: number
  tenant_id: number
  type: string
  name: string
  config: Record<string, string>
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface Template {
  id: number
  tenant_id: number
  name: string
  description: string
  tech_stack: string
  deploy_mode: string
  config: Record<string, any>
  created_at: string
  updated_at: string
}

export interface SystemMetrics {
  cpu_usage: number
  memory_usage: number
  memory_total: number
  memory_used: number
  disk_usage: number
  disk_total: number
  disk_used: number
  network_in: number
  network_out: number
  uptime: number
  timestamp: string
}

export interface ContainerMetrics {
  name: string
  cpu_usage: number
  memory_usage: number
  memory_limit: number
  network_in: number
  network_out: number
  status: string
  created: string
}

export interface AlertRule {
  id: number
  name: string
  type: string
  condition: Record<string, any>
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface Alert {
  id: number
  rule_id: number
  server_id: number
  level: string
  message: string
  resolved: boolean
  created_at: string
  resolved_at: string
}

export interface CICDBuild {
  id: number
  provider: string
  run_id: string
  status: string
  trigger_type: string
  started_at: string
  finished_at: string
  created_at: string
}
