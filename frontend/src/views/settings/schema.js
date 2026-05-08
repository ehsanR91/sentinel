export const SETTINGS_SECTIONS = [
  {
    id: 'general',
    label: 'General',
    icon: 'mdi mdi-cog-outline',
    description: 'Runtime identity, defaults, and base server presentation.',
    groups: [
      { id: 'runtime', label: 'Runtime' },
      { id: 'preferences', label: 'Preferences' }
    ]
  },
  {
    id: 'security',
    label: 'Security',
    icon: 'mdi mdi-shield-lock-outline',
    description: 'Authentication pressure, gate controls, and safety baselines.',
    groups: [
      { id: 'login-protection', label: 'Login Protection' },
      { id: 'tls', label: 'TLS' }
    ]
  },
  {
    id: 'access-control',
    label: 'Access Control',
    icon: 'mdi mdi-key-chain-variant',
    description: '2FA, lock screen access, and the secret-link gate.',
    groups: [
      { id: 'two-factor', label: 'Two-Factor Authentication' },
      { id: 'secret-gate', label: 'Secret Link Gate' },
      { id: 'lock-screen', label: 'Lock Screen' }
    ]
  },
  {
    id: 'notifications',
    label: 'Notifications',
    icon: 'mdi mdi-bell-ring-outline',
    description: 'Email delivery, routing, and operator quiet hours.',
    groups: [
      { id: 'email', label: 'Email Delivery' },
      { id: 'routing', label: 'Routing Rules' }
    ]
  },
  {
    id: 'integrations',
    label: 'Integrations',
    icon: 'mdi mdi-connection',
    description: 'Third-party verification and enrichment providers.',
    groups: [
      { id: 'captcha', label: 'reCAPTCHA' },
      { id: 'ip-intelligence', label: 'IP Intelligence' }
    ]
  },
  {
    id: 'data-storage',
    label: 'Data & Storage',
    icon: 'mdi mdi-database-outline',
    description: 'Database administration, retention, export, and import.',
    groups: [
      { id: 'database', label: 'Database' },
      { id: 'prune', label: 'Retention & Prune' }
    ]
  },
  {
    id: 'master-key',
    label: 'Master Key',
    icon: 'mdi mdi-key-variant',
    description: 'Encryption key posture and recovery operations.',
    groups: [
      { id: 'key-status', label: 'Key Status' }
    ]
  },
  {
    id: 'audit-logs',
    label: 'Audit & Logs',
    icon: 'mdi mdi-clipboard-text-clock-outline',
    description: 'Audit trail controls and log-management handoffs.',
    groups: [
      { id: 'audit-overview', label: 'Audit Overview' }
    ]
  },
  {
    id: 'updates',
    label: 'Updates',
    icon: 'mdi mdi-update',
    description: 'Release channel, agent updates, and change review.',
    groups: [
      { id: 'update-policy', label: 'Update Policy' }
    ]
  },
  {
    id: 'advanced',
    label: 'Advanced',
    icon: 'mdi mdi-tune-vertical-variant',
    description: 'Raw configuration and feature-flag surfaces.',
    groups: [
      { id: 'raw-config', label: 'Raw Config' }
    ]
  },
  {
    id: 'danger-zone',
    label: 'Danger Zone',
    icon: 'mdi mdi-alert-octagon-outline',
    description: 'High-impact actions isolated behind explicit confirmation.',
    groups: [
      { id: 'destructive-actions', label: 'Destructive Actions' }
    ]
  }
]

export const SETTINGS_SEARCH_ENTRIES = [
  { id: 'general-admin-email', section: 'general', anchor: 'general-runtime', label: 'Admin email', description: 'Primary operator email used for notifications.', key: 'alert_email', keywords: ['admin email', 'alert email', 'smtp recipient'] },
  { id: 'general-theme', section: 'general', anchor: 'general-preferences', label: 'Default theme', description: 'UI default for users on this browser.', key: 'ui_theme_default', keywords: ['theme', 'light', 'dark', 'system'] },
  { id: 'general-language', section: 'general', anchor: 'general-preferences', label: 'Language', description: 'Local preference for the future localized interface.', key: 'ui_language', keywords: ['locale', 'language'] },
  { id: 'general-timezone', section: 'general', anchor: 'general-preferences', label: 'Timezone', description: 'Preferred timezone for local rendering.', key: 'ui_timezone', keywords: ['timezone', 'tz', 'time zone'] },
  { id: 'security-login-attempts', section: 'security', anchor: 'security-login-protection', label: 'Max login attempts', description: 'Auto-ban threshold over a rolling 10 minute window.', key: 'brute_force_threshold', keywords: ['max login', 'brute force', 'threshold'] },
  { id: 'security-ip-allowlist', section: 'security', anchor: 'security-login-protection', label: 'IP allowlist', description: 'Model and validate CIDR entries for restricted access.', key: 'ip_allowlist', keywords: ['cidr', 'allowlist', 'ip list'] },
  { id: 'security-tls', section: 'security', anchor: 'security-tls', label: 'TLS / HTTPS', description: 'Transport security state for the current agent.', key: 'tls', keywords: ['tls', 'https', 'cert'] },
  { id: 'access-2fa', section: 'access-control', anchor: 'access-two-factor', label: 'Two-factor authentication', description: 'Manage TOTP enrollment and recovery posture.', key: 'totp', keywords: ['2fa', 'totp', 'authenticator'] },
  { id: 'access-secret-gate', section: 'access-control', anchor: 'access-secret-gate', label: 'Secret link gate', description: 'Hidden entry path and cookie expiry policy.', key: 'secret_path', keywords: ['secret path', 'gate', 'hidden login'] },
  { id: 'access-lock-screen', section: 'access-control', anchor: 'access-lock-screen', label: 'Lock screen PIN', description: 'Local lock controls for active operator sessions.', key: 'lock_pin', keywords: ['pin', 'lock screen', 'idle timeout'] },
  { id: 'notifications-email', section: 'notifications', anchor: 'notifications-email', label: 'Email notifications', description: 'SMTP transport and severity threshold.', key: 'email_alerts_enabled', keywords: ['smtp', 'mail', 'notifications'] },
  { id: 'notifications-routing', section: 'notifications', anchor: 'notifications-routing', label: 'Routing rules', description: 'Targeted delivery controls for alert classes.', key: 'notification_routing', keywords: ['routing', 'telegram', 'webhook', 'slack'] },
  { id: 'integrations-recaptcha', section: 'integrations', anchor: 'integrations-captcha', label: 'reCAPTCHA', description: 'Login bot protection provider keys.', key: 'recaptcha_enabled', keywords: ['captcha', 'recaptcha', 'site key'] },
  { id: 'integrations-ip-provider', section: 'integrations', anchor: 'integrations-ip-intelligence', label: 'IP lookup provider', description: 'Select the reverse lookup provider for IP intelligence.', key: 'ip_lookup_provider', keywords: ['ipify', 'ip-api', 'lookup provider'] },
  { id: 'data-db-export', section: 'data-storage', anchor: 'data-database', label: 'Database export', description: 'Export and import the SentinelCore database.', key: 'db_export', keywords: ['export db', 'import db', 'backup'] },
  { id: 'data-prune', section: 'data-storage', anchor: 'data-prune', label: 'Prune records', description: 'Preview and delete retained records by age.', key: 'db_prune', keywords: ['retention', 'prune', 'delete records'] },
  { id: 'master-key-status', section: 'master-key', anchor: 'master-key-status', label: 'Master key status', description: 'View the configured secrets key path and rotation timestamp.', key: 'secrets_key_path', keywords: ['master key', 'rotation', 'secrets key'] },
  { id: 'audit-overview', section: 'audit-logs', anchor: 'audit-overview', label: 'Audit overview', description: 'Jump to audit logs and retention controls.', key: 'audit_logs', keywords: ['audit', 'logs', 'history'] },
  { id: 'updates-policy', section: 'updates', anchor: 'updates-policy', label: 'Update policy', description: 'Review the current agent update surface.', key: 'updates', keywords: ['updates', 'channel', 'release'] },
  { id: 'advanced-raw-config', section: 'advanced', anchor: 'advanced-raw-config', label: 'Raw config editor', description: 'Preview the raw settings surface and unsupported keys.', key: 'raw_config', keywords: ['advanced', 'raw config', 'feature flags'] },
  { id: 'danger-zone', section: 'danger-zone', anchor: 'danger-destructive-actions', label: 'Danger zone', description: 'Restart, revoke, regenerate, or remove sensitive access.', key: 'danger_zone', keywords: ['restart', 'disable 2fa', 'remove pin', 'factory reset'] }
]

export const SECTION_MAP = Object.fromEntries(SETTINGS_SECTIONS.map(section => [section.id, section]))

export function getSection(sectionId) {
  return SECTION_MAP[sectionId] || SECTION_MAP.general
}