import MainLayout from './layouts/main.vue'

export default [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/auth/login.vue'),
    meta: { requiresAuth: false, title: 'Login' }
  },
  {
    path: '/access-denied',
    name: 'AccessDenied',
    component: () => import('@/views/auth/access-denied.vue'),
    meta: { requiresAuth: false, title: 'Access Denied' }
  },
  {
    path: '/',
    redirect: '/dashboard'
  },
  {
    path: '/',
    component: MainLayout,
    children: [
      {
        path: 'dashboard',
        name: 'Dashboard',
        component: () => import('@/views/dashboard/index.vue'),
        meta: { title: 'Dashboard' }
      },
      {
        path: 'security',
        name: 'SecurityCenter',
        component: () => import('@/views/security/index.vue'),
        meta: { title: 'Security Center', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'services',
        name: 'Services',
        component: () => import('@/views/services/index.vue'),
        meta: { title: 'Services', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'firewall',
        name: 'Firewall',
        component: () => import('@/views/firewall/index.vue'),
        meta: { title: 'Firewall', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'monitoring',
        name: 'Monitoring',
        component: () => import('@/views/monitoring/index.vue'),
        meta: { title: 'Monitoring' }
      },
      {
        path: 'containers',
        name: 'Containers',
        component: () => import('@/views/containers/index.vue'),
        meta: { title: 'Containers', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'logs',
        name: 'Logs',
        component: () => import('@/views/logs/index.vue'),
        meta: { title: 'Logs' }
      },
      {
        path: 'tasks',
        name: 'Tasks',
        component: () => import('@/views/tasks/index.vue'),
        meta: { title: 'Tasks', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'alerts',
        name: 'Alerts',
        component: () => import('@/views/alerts/index.vue'),
        meta: { title: 'Alerts' }
      },
      {
        path: 'users',
        name: 'Users',
        component: () => import('@/views/users/index.vue'),
        meta: { title: 'Users', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'settings',
        redirect: '/settings/general'
      },
      {
        path: 'settings/:section',
        name: 'Settings',
        component: () => import('@/views/settings/index.vue'),
        meta: { title: 'Settings', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'terminal',
        name: 'Terminal',
        component: () => import('@/views/terminal/index.vue'),
        meta: { title: 'Terminal', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'audit-logs',
        name: 'AuditLogs',
        component: () => import('@/views/audit-logs/index.vue'),
        meta: { title: 'Audit Logs', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'updates',
        name: 'Updates',
        component: () => import('@/views/updates/index.vue'),
        meta: { title: 'Updates', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'security-tools',
        name: 'SecurityTools',
        component: () => import('@/views/security-tools/index.vue'),
        meta: { title: 'Security Tools', roles: ['admin', 'superadmin'] }
      },
      {
        path: 'apps',
        name: 'Apps',
        component: () => import('@/views/apps/index.vue'),
        meta: { title: 'Apps', roles: ['admin', 'superadmin'] }
      }
    ]
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/errors/404.vue'),
    meta: { requiresAuth: false, title: 'Page Not Found' }
  }
]
