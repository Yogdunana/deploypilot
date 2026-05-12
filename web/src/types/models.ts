export interface App {
  id: string
  tenant_id: string
  server_id: string
  name: string
  repo_url: string
  branch: string
  domain: string
  tech_stack: string
  deploy_mode: string
  status: string
  current_version: string
  container_name: string
  env_vars: string
  resource_limits: string
  created_at: string
  updated_at: string
}

export interface Server {
  id: string
  tenant_id: string
  credential_id: string
  provider_id: string
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
  id: string
  tenant_id: string
  name: string
  type: string
  expires_at?: string
  last_rotated?: string
  rotation_days: number
  is_expired?: boolean
  days_until_expiry?: number
  created_at: string
  updated_at: string
}

export interface Provider {
  id: string
  tenant_id: string
  type: string
  name: string
  config: Record<string, string>
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface DeploymentRecord {
  id: string
  tenant_id: string
  server_id: string
  app_name: string
  app_id?: string
  container_name: string
  image: string
  status: string
  preflight_code: string
  preflight_message: string
  preflight_checks: string // JSON string from backend
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
  id: string
  user_id: string
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
  totp_enabled?: boolean
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
  id: string
  tenant_id: string
  role_id: string
  username: string
  email: string
  totp_enabled?: boolean
  created_at: string
  updated_at: string
  tenant?: Tenant
  role?: Role
}

export interface Role {
  id: string
  name: string
  permissions: string[]
  created_at: string
}

export interface Tenant {
  id: string
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
  id: string
  tenant_id: string
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

export interface ApiKey {
  id: string
  tenant_id: string
  user_id: string
  name: string
  key_prefix: string
  scopes: string[] | string
  expires_at?: string
  last_used_at?: string
  created_at: string
  updated_at: string
}

export interface TwoFASetupResponse {
  secret: string
  qr_code_url: string
  backup_codes: string[]
}

export interface LicenseInfo {
  tier: string
  use_type: string
  status: string
  features: string[]
  limits: {
    max_servers: number
    max_apps: number
    max_users: number
  }
  addons: Array<{
    key: string
    amount: number
    purchased_at: string
    expires_at: string
  }>
  expires_at: string | null
  issued_at: string
  machine_id: string
  grace_days: number
}
