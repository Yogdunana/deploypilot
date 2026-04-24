import axios from 'axios'
import { getToken, removeToken } from '../utils/auth'
import { ElMessage } from 'element-plus'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 30000,
})

api.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const msg = error.response?.data?.error || error.message || 'Request failed'
    if (error.response?.status === 401) {
      removeToken()
      window.location.href = '/login'
    } else {
      ElMessage.error(msg)
    }
    return Promise.reject(error)
  }
)

// Auth
export const authApi = {
  login: (data) => api.post('/auth/login', data),
  register: (data) => api.post('/auth/register', data),
}

// Users
export const usersApi = {
  me: () => api.get('/users/me'),
  list: () => api.get('/users'),
  updateRole: (id, role) => api.put(`/users/${id}/role`, { role }),
  delete: (id) => api.delete(`/users/${id}`),
  getRoles: () => api.get('/roles'),
}

// Apps
export const appsApi = {
  list: () => api.get('/apps'),
  create: (data) => api.post('/apps', data),
  get: (id) => api.get(`/apps/${id}`),
  update: (id, data) => api.put(`/apps/${id}`, data),
  delete: (id) => api.delete(`/apps/${id}`),
  deploy: (id, data, config = {}) => api.post(`/apps/${id}/deploy`, data, config),
  status: (id) => api.get(`/apps/${id}/status`),
  rollback: (id) => api.post(`/apps/${id}/rollback`),
  logs: (id) => api.get(`/apps/${id}/logs/container`),
  backup: (id) => api.post(`/apps/${id}/backup`),
  restore: (id) => api.post(`/apps/${id}/restore`),
  build: (id, data) => api.post(`/apps/${id}/build`, data),
}

// Servers
export const serversApi = {
  list: () => api.get('/servers'),
  create: (data) => api.post('/servers', data),
  update: (id, data) => api.put(`/servers/${id}`, data),
  delete: (id) => api.delete(`/servers/${id}`),
  test: (id) => api.post(`/servers/${id}/test`),
}

// Credentials
export const credentialsApi = {
  list: () => api.get('/credentials'),
  create: (data) => api.post('/credentials', data),
  update: (id, data) => api.put(`/credentials/${id}`, data),
  delete: (id) => api.delete(`/credentials/${id}`),
}

// DNS
export const dnsApi = {
  listRecords: () => api.get('/dns/records'),
  createRecord: (data) => api.post('/dns/records', data),
  updateRecord: (id, data) => api.put(`/dns/records/${id}`, data),
  deleteRecord: (id) => api.delete(`/dns/records/${id}`),
}

// Providers
export const providersApi = {
  list: () => api.get('/providers'),
  create: (data) => api.post('/providers', data),
  update: (id, data) => api.put(`/providers/${id}`, data),
  delete: (id) => api.delete(`/providers/${id}`),
}

// Notifications
export const notificationsApi = {
  list: () => api.get('/notifications'),
  create: (data) => api.post('/notifications', data),
  update: (id, data) => api.put(`/notifications/${id}`, data),
  delete: (id) => api.delete(`/notifications/${id}`),
}

// Templates
export const templatesApi = {
  list: () => api.get('/templates'),
}

// System
export const systemApi = {
  version: () => api.get('/system/version'),
  health: () => api.get('/system/health'),
  checkUpdate: () => api.get('/system/update/check'),
}

// Deployments
export const deploymentsApi = {
  list: () => api.get('/deployments'),
}

export default api
