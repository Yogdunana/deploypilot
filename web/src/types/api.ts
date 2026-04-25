export interface ApiResponse<T> {
  status: 'success' | 'error'
  data: T
  message?: string
}

export interface PaginatedResponse<T> {
  status: 'success'
  data: T
  pagination: {
    total: number
    page: number
    page_size: number
    total_pages: number
  }
}

export interface PaginationParams {
  page?: number
  page_size?: number
}

export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username: string
  email: string
  password: string
}

export interface LoginResponse {
  token: string
  user: {
    id: number
    username: string
    email: string
    role: string
  }
}

export interface DeployConfig {
  branch?: string
  env_vars?: Record<string, string>
}

export interface BuildConfig {
  dockerfile?: string
  build_args?: Record<string, string>
}

export interface RollbackConfig {
  version: string
}

export interface RestoreConfig {
  backup_id: string
}
