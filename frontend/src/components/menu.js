export const menuItems = [
  {
    id: 1,
    label: 'Overview',
    isTitle: true
  },
  {
    id: 2,
    label: 'Dashboard',
    icon: 'mdi mdi-view-dashboard-outline',
    link: '/dashboard'
  },

  {
    id: 10,
    label: 'Security',
    isTitle: true
  },
  {
    id: 11,
    label: 'Security Center',
    icon: 'mdi mdi-shield-lock-outline',
    link: '/security',
    badge: { variant: 'danger', text: '3' }
  },
  {
    id: 12,
    label: 'Firewall',
    icon: 'mdi mdi-wall',
    link: '/firewall'
  },
  {
    id: 121,
    label: 'Services',
    icon: 'mdi mdi-cog-refresh-outline',
    link: '/services'
  },
  {
    id: 13,
    label: 'Alerts',
    icon: 'mdi mdi-bell-alert-outline',
    link: '/alerts',
    badge: { variant: 'warning', text: '7' }
  },
  {
    id: 14,
    label: 'Audit Logs',
    icon: 'mdi mdi-clipboard-text-clock-outline',
    link: '/audit-logs'
  },
  {
    id: 20,
    label: 'Infrastructure',
    isTitle: true
  },
  {
    id: 21,
    label: 'Monitoring',
    icon: 'mdi mdi-chart-areaspline',
    link: '/monitoring'
  },
  {
    id: 22,
    label: 'Containers',
    icon: 'mdi mdi-docker',
    link: '/containers'
  },
  {
    id: 25,
    label: 'Apps',
    icon: 'mdi mdi-apps',
    link: '/apps'
  },
  {
    id: 23,
    label: 'Logs',
    icon: 'mdi mdi-text-box-multiple-outline',
    link: '/logs'
  },
  {
    id: 24,
    label: 'Terminal',
    icon: 'mdi mdi-console',
    link: '/terminal'
  },

  {
    id: 30,
    label: 'Operations',
    isTitle: true
  },
  {
    id: 31,
    label: 'Tasks',
    icon: 'mdi mdi-playlist-check',
    link: '/tasks'
  },
  {
    id: 32,
    label: 'Updates',
    icon: 'mdi mdi-update',
    link: '/updates'
  },
  {
    id: 35,
    label: 'Security Ops',
    isTitle: true
  },
  {
    id: 36,
    label: 'Security Tools',
    icon: 'mdi mdi-shield-bug-outline',
    link: '/security-tools'
  },

  {
    id: 40,
    label: 'Administration',
    isTitle: true
  },
  {
    id: 41,
    label: 'Users',
    icon: 'mdi mdi-account-group-outline',
    link: '/users'
  },
  {
    id: 42,
    label: 'Settings',
    icon: 'mdi mdi-cog-outline',
    link: '/settings'
  }
]
