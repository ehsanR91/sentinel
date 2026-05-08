import axios from 'axios'
import store from '@/state/store'
import router from '@/router/index'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
  headers: { 'Content-Type': 'application/json' },
  withCredentials: true
})

function getCookie(name) {
  const match = document.cookie.match(new RegExp('(^| )' + name + '=([^;]+)'))
  return match ? decodeURIComponent(match[2]) : null
}

// CSRF + cookie auth
api.interceptors.request.use(config => {
  const method = (config.method || 'get').toUpperCase()
  const unsafe = ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)
  if (unsafe) {
    const csrf = getCookie('sc_csrf')
    if (csrf) config.headers['X-CSRF-Token'] = csrf
  }
  return config
})

// Enhanced error handling with detailed messages
api.interceptors.response.use(
  res => res,
  err => {
    const status = err.response?.status
    const data = err.response?.data
    
    // Extract detailed error message from response
    let detailedMessage = ''
    if (data) {
      if (typeof data === 'string') {
        detailedMessage = data
      } else if (data.message) {
        detailedMessage = data.message
      } else if (data.error) {
        detailedMessage = data.error
      }
    }
    
    // Attach detailed message to error for components to use
    if (detailedMessage) {
      err.detailedMessage = detailedMessage
    }
    
    if (status === 401) {
      // Only redirect if not already on the login page to prevent infinite reload loops
      if (!window.location.pathname.startsWith('/login')) {
        store.dispatch('auth/logout')
        router.push('/login')
      }
    } else if (status === 403) {
      // Set Vuex notification for permission denied
      store.commit('notifications/ADD', {
        type: 'error',
        message: 'Permission denied',
        timeout: 5000
      })
    }
    
    return Promise.reject(err)
  }
)

export default {
  // Auth
  login: (username, password) => api.post('/auth/login', { username, password }),
  logout: () => api.post('/auth/logout'),
  verify2fa: (pending_token, code) => api.post('/auth/2fa/verify', { pending_token, code }),
  setup2fa: () => api.get('/auth/2fa/setup'),
  enable2fa: (secret, code) => api.post('/auth/2fa/enable', { secret, code }),
  disable2fa: (code) => api.delete('/auth/2fa/disable', { data: { code } }),
  getMe: () => api.get('/me'),

  // Alerts
  getAlerts: (params = {}) => api.get('/alerts', { params }),
  getAlertCount: () => api.get('/alerts/count'),
  markAlertRead: (id) => api.put(`/alerts/${id}/read`),
  markAlertAsRead: (id) => api.put(`/alerts/${id}/read`),
  markAlertsAsRead: (ids) => api.put('/alerts/read', { ids }),

  // System
  getMetrics: () => api.get('/system/metrics'),
  getProcesses: (limit = 50) => api.get(`/system/processes?limit=${limit}`),
  getServices: () => api.get('/system/services'),
  getHealth: () => api.get('/system/health'),
  fixHealthIssue: (data) => api.post('/system/health/fix', data),
  getCleanupStats: () => api.get('/system/cleanup/stats'),
  getCleanupLogs: () => api.get('/system/cleanup/logs'),
  runCleanup: () => api.post('/system/cleanup/run'),
  getManagedServices: () => api.get('/services'),
  installService: (name) => api.post(`/services/${name}/install`),
  getServiceInstallLogs: () => api.get('/services/install/logs'),
  serviceAction: (name, action) => api.post(`/services/${name}/${action}`),
  getServiceConfig: () => api.get('/services/config'),
  updateServiceConfig: (cfg) => api.put('/services/config', cfg),
  getSuspiciousProcesses: () => api.get('/system/suspicious'),
  getDiskUsage: (path, depth = 2, limit = 25) =>
    api.get('/system/disk-usage', { params: { path, depth, limit } }),

  // Docker
  getDockerInfo: () => api.get('/docker/info'),
  getContainers: () => api.get('/docker/containers'),
  getContainerStats: (id) => api.get(`/docker/containers/${id}/stats`),
  getContainerLogs: (id, params) => api.get(`/docker/containers/${id}/logs`, { params }),
  getContainerLogsWsUrl: (id, params = {}) => {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = import.meta.env.VITE_WS_HOST || window.location.host
    const query = new URLSearchParams(params).toString()
    return `${proto}//${host}/api/v1/docker/containers/${id}/logs/stream${query ? `?${query}` : ''}`
  },
  dockerPrune: (kind) => api.post(`/docker/prune/${kind}`),
  startContainer: (id) => api.post(`/containers/${id}/start`),
  stopContainer: (id) => api.post(`/containers/${id}/stop`),
  restartContainer: (id) => api.post(`/containers/${id}/restart`),

  // Security
  getSecurityStatus: () => api.get('/security/status'),
  getBans: () => api.get('/security/bans'),
  banIp: (ip) => api.post(`/security/bans/${ip}`),
  unban: (ip) => api.delete(`/security/bans/${ip}`),
  getSecurityTools: () => api.get('/security-tools'),
  installSecurityTool: (name) => api.post(`/security-tools/${name}/install`),
  getTunnelableApps: () => api.get('/system/tunnelable-apps'),
  grantTunnelAccess: (port, durationHours) => api.post('/system/tunnelable-apps/grant', { port, duration_hours: durationHours || 3 }),
  getSecurityToolLogs: (name) => api.get(`/security-tools/${name}/logs`),
  runSecurityTool: (name) => api.post(`/security-tools/${name}/run`),

  // Dashboard
  getDashboardLoginAttempts: () => api.get('/dashboard/login-attempts'),
  getDashboardLayout: () => api.get('/dashboard/layout'),
  saveDashboardLayout: (layout) => api.put('/dashboard/layout', layout),

  // Terminal
  getTerminalWsUrl: () => {
    const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const host = import.meta.env.VITE_WS_HOST || window.location.host
    return `${proto}//${host}/api/v1/terminal/ws`
  },

  // Firewall
  getFirewallRules: () => api.get('/firewall/rules'),
  addFirewallRule: (rule) => api.post('/firewall/rules', rule),
  deleteFirewallRule: (id) => api.delete(`/firewall/rules/${id}`),

  // Logs
  getLogs: (source, lines = 200) => api.get(`/logs?source=${source}&lines=${lines}`),

  // Alerts
  getAlerts: (params = {}) => api.get('/alerts', { params }),
  markAlertRead: (id) => api.put(`/alerts/${id}/read`),
  markAlertAsRead: (id) => api.put(`/alerts/${id}/read`),
  markAlertsAsRead: (ids) => api.put('/alerts/read', { ids }),

  // Tasks
  getTasks: () => api.get('/tasks'),
  createTask: (task) => api.post('/tasks', task),
  updateTask: (id, task) => api.put(`/tasks/${id}`, task),
  deleteTask: (id) => api.delete(`/tasks/${id}`),
  runTaskNow: (id) => api.post(`/tasks/${id}/run`),

  // Users
  getUsers: () => api.get('/users'),
  createUser: (user) => api.post('/users', user),
  updateUser: (id, data) => api.put(`/users/${id}`, data),
  deleteUser: (id) => api.delete(`/users/${id}`),

  // Audit Logs
  getAuditLogs: (params = {}) => api.get('/audit-logs', { params }),

  // Updates
  getUpdates: () => api.get('/updates'),
  installUpdates: (packages) => api.post('/updates/install', { packages }),
  getUpdateLogs: () => api.get('/updates/logs'),

  // Settings
  getSettings: () => api.get('/settings'),
  updateSettings: (settings) => api.put('/settings', settings),

  // DB Admin
  getDbStats: () => api.get('/db/stats'),
  exportDb: () => api.get('/db/export', { responseType: 'blob' }),
  importDb: (formData) => api.post('/db/import', formData, { headers: { 'Content-Type': 'multipart/form-data' } }),
  pruneDb: (type, days) => api.post('/db/prune', { type, days }),

  // Lock screen PIN
  getLockSettings: () => api.get('/lock/settings'),
  saveLockPin: (pin, enabled) => api.post('/lock/pin', { pin, enabled }),
  clearLockPin: () => api.delete('/lock/pin'),
  verifyLockPin: (pin) => api.post('/lock/verify', { pin }),

  // Service config file editor
  getServiceConfigFile: (name) => api.get(`/services/${name}/configfile`),
  saveServiceConfigFile: (name, content) => api.put(`/services/${name}/configfile`, { content }),
  backupServiceConfigFile: (name) => api.post(`/services/${name}/configfile/backup`),
  verifyServiceConfigFile: (name) => api.post(`/services/${name}/configfile/verify`),
  restoreServiceConfigFile: (name) => api.post(`/services/${name}/configfile/restore`),

  // Apps
  getApps: () => api.get('/apps'),
  installApp: (name) => api.post(`/apps/${name}/install`),
  updateApp: (name) => api.post(`/apps/${name}/update`),
  uninstallApp: (name) => api.delete(`/apps/${name}`),
  getAppOpLogs: () => api.get('/apps/op/logs')
}
