const sidebarSections = [
  {
    id: 'overview',
    label: 'Overview',
    jumpKey: 'd',
    items: [
      {
        id: 'dashboard',
        label: 'Dashboard',
        icon: 'mdi mdi-view-dashboard-outline',
        link: '/dashboard',
        shortcut: 'd',
        keywords: ['overview', 'home', 'summary']
      }
    ]
  },
  {
    id: 'security',
    label: 'Security',
    jumpKey: 'a',
    items: [
      {
        id: 'alerts',
        label: 'Alerts',
        icon: 'mdi mdi-bell-alert-outline',
        link: '/alerts',
        shortcut: 'a',
        badgeKey: 'alerts',
        keywords: ['incidents', 'detections', 'notifications']
      },
      {
        id: 'security-center',
        label: 'Security Center',
        icon: 'mdi mdi-shield-lock-outline',
        link: '/security',
        shortcut: 's',
        badgeKey: 'security',
        statusKey: 'health',
        keywords: ['hardening', 'policies', 'security']
      },
      {
        id: 'firewall',
        label: 'Firewall',
        icon: 'mdi mdi-wall',
        link: '/firewall',
        shortcut: 'f',
        statusKey: 'health',
        keywords: ['rules', 'ufw', 'network']
      },
      {
        id: 'security-tools',
        label: 'Security Tools',
        icon: 'mdi mdi-shield-bug-outline',
        link: '/security-tools',
        shortcut: 'x',
        tag: 'experimental',
        keywords: ['scanner', 'tooling', 'ops']
      }
    ]
  },
  {
    id: 'infrastructure',
    label: 'Infrastructure',
    jumpKey: 'm',
    items: [
      {
        id: 'monitoring',
        label: 'Monitoring',
        icon: 'mdi mdi-chart-areaspline',
        link: '/monitoring',
        shortcut: 'm',
        statusKey: 'live',
        keywords: ['metrics', 'telemetry', 'graphs']
      },
      {
        id: 'services',
        label: 'Services',
        icon: 'mdi mdi-cog-refresh-outline',
        link: '/services',
        shortcut: 's',
        statusKey: 'services',
        keywords: ['systemd', 'daemons', 'service manager']
      },
      {
        id: 'containers',
        label: 'Containers',
        icon: 'mdi mdi-docker',
        link: '/containers',
        shortcut: 'c',
        keywords: ['docker', 'pods', 'images']
      },
      {
        id: 'apps',
        label: 'Apps',
        icon: 'mdi mdi-apps',
        link: '/apps',
        shortcut: 'p',
        tag: 'beta',
        keywords: ['packages', 'catalog', 'software']
      },
      {
        id: 'terminal',
        label: 'Terminal',
        icon: 'mdi mdi-console',
        link: '/terminal',
        shortcut: 't',
        keywords: ['shell', 'console', 'ssh']
      }
    ]
  },
  {
    id: 'operations',
    label: 'Operations',
    jumpKey: 'l',
    items: [
      {
        id: 'logs',
        label: 'Logs',
        icon: 'mdi mdi-text-box-multiple-outline',
        shortcut: 'l',
        keywords: ['events', 'audit', 'journals'],
        children: [
          {
            id: 'system-logs',
            label: 'System Logs',
            link: '/logs',
            shortcut: 'l',
            keywords: ['journal', 'syslog']
          },
          {
            id: 'audit-logs',
            label: 'Audit Logs',
            link: '/audit-logs',
            shortcut: 'u',
            keywords: ['trail', 'changes', 'audit']
          }
        ]
      },
      {
        id: 'tasks',
        label: 'Tasks',
        icon: 'mdi mdi-playlist-check',
        link: '/tasks',
        shortcut: 'k',
        keywords: ['jobs', 'automation', 'runs']
      },
      {
        id: 'updates',
        label: 'Updates',
        icon: 'mdi mdi-update',
        link: '/updates',
        shortcut: 'u',
        tag: 'new',
        keywords: ['packages', 'upgrades', 'patches']
      }
    ]
  },
  {
    id: 'administration',
    label: 'Administration',
    jumpKey: 'u',
    items: [
      {
        id: 'users',
        label: 'Users',
        icon: 'mdi mdi-account-group-outline',
        link: '/users',
        shortcut: 'u',
        keywords: ['rbac', 'roles', 'accounts']
      },
      {
        id: 'settings',
        label: 'Settings',
        icon: 'mdi mdi-cog-outline',
        link: '/settings/general',
        shortcut: 'g',
        keywords: ['preferences', 'configuration', 'config']
      }
    ]
  }
]

const settingsCommandEntries = [
  { id: 'settings-general', label: 'Settings: General', route: '/settings/general', group: 'Settings', keywords: ['hostname', 'theme', 'admin email', 'timezone'] },
  { id: 'settings-security', label: 'Settings: Security', route: '/settings/security', group: 'Settings', keywords: ['allowlist', 'login attempts', 'hardening'] },
  { id: 'settings-access-control', label: 'Settings: Access Control', route: '/settings/access-control', group: 'Settings', keywords: ['totp', 'secret path', 'lock screen', 'pin'] },
  { id: 'settings-notifications', label: 'Settings: Notifications', route: '/settings/notifications', group: 'Settings', keywords: ['smtp', 'mail', 'routing'] },
  { id: 'settings-integrations', label: 'Settings: Integrations', route: '/settings/integrations', group: 'Settings', keywords: ['recaptcha', 'ip lookup', 'api key'] },
  { id: 'settings-data-storage', label: 'Settings: Data & Storage', route: '/settings/data-storage', group: 'Settings', keywords: ['database', 'backup', 'prune'] },
  { id: 'settings-danger-zone', label: 'Settings: Danger Zone', route: '/settings/danger-zone', group: 'Settings', keywords: ['disable 2fa', 'remove pin', 'restart'] }
]

function flattenItems(items, section, parent = null) {
  return items.flatMap(item => {
    const base = {
      ...item,
      sectionId: section.id,
      sectionLabel: section.label,
      parentId: parent?.id || null,
      parentLabel: parent?.label || null
    }

    if (item.children?.length) {
      return [base, ...flattenItems(item.children, section, item)]
    }

    return [base]
  })
}

function flattenSidebarItems() {
  return sidebarSections.flatMap(section => flattenItems(section.items, section))
}

function routeItems() {
  return flattenSidebarItems().filter(item => item.link)
}

function findSidebarItemById(id) {
  return flattenSidebarItems().find(item => item.id === id) || null
}

function findSidebarItemByRoute(route) {
  return routeItems().find(item => item.link === route) || null
}

function navigationSearchEntries() {
  return routeItems().map(item => ({
    id: item.id,
    label: item.parentLabel ? `${item.parentLabel}: ${item.label}` : item.label,
    route: item.link,
    icon: item.icon || null,
    group: 'Navigation',
    sectionLabel: item.sectionLabel,
    keywords: item.keywords || []
  }))
}

export {
  sidebarSections,
  settingsCommandEntries,
  flattenSidebarItems,
  findSidebarItemById,
  findSidebarItemByRoute,
  navigationSearchEntries
}
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
